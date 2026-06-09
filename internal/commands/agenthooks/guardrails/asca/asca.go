package asca

import (
	"os"
	"path/filepath"
	"strings"

	agenthooks "github.com/CheckmarxDev/ast-cx-hooks"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ignore"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/spf13/viper"
)

// ascaSupportedExtensions lists file extensions for languages ASCA can scan:
// Java, JavaScript (Node.js), C#, Go, and Python.
var ascaSupportedExtensions = map[string]struct{}{
	".java": {}, ".js": {}, ".jsx": {}, ".ts": {}, ".tsx": {}, ".mjs": {}, ".cjs": {},
	".cs": {}, ".go": {}, ".py": {}, ".pyw": {},
}

// isSupportedByASCA returns true when the file's extension is one ASCA can scan.
func isSupportedByASCA(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	_, ok := ascaSupportedExtensions[ext]
	return ok
}

// ScanFileEdit runs ASCA on the proposed post-edit content.
// Returns blocked=true with a formatted reason and remediation context when ASCA
// finds *new* vulnerabilities introduced by ev.Changes (delta-detection for edits;
// any-vuln for new writes). Findings the user already suppressed via
// `cx ignore-vulnerability` (the realtime ignore file) are filtered out before the
// verdict. Fail-open on infrastructure errors (ASCA install fail, engine unavailable, panic).
func ScanFileEdit(ev agenthooks.FileEditEvent) (blocked bool, reason, context string) {
	defer func() {
		if r := recover(); r != nil {
			blocked = false
			reason = ""
			context = ""
		}
	}()

	if !isSupportedByASCA(ev.FilePath) {
		return false, "", ""
	}

	newContent, originalContent, err := ProposedContent(ev.FilePath, ev.Changes)
	if err != nil || newContent == "" {
		return false, "", ""
	}

	wrapperParams := services.AscaWrappersParam{
		JwtWrapper:  wrappers.NewJwtWrapper(),
		ASCAWrapper: grpcs.NewASCAGrpcWrapper(viper.GetInt(params.ASCAPortKey)),
	}

	ascaParams := services.AscaScanParams{
		ASCAUpdateVersion: shouldUpdateVersion(),
		IsDefaultAgent:    true, // license already verified upstream in agenthooks.go
		// Honor findings the user suppressed via `cx ignore-vulnerability` so the hook
		// stops blocking them. Only set when the file exists: the ASCA service treats a
		// configured-but-missing ignore path as a scan error, which would fail-open the
		// guardrail entirely.
		IgnoredFilePath: existingIgnoreFilePath(),
	}

	// Stage and scan the proposed (new) content
	stagedNew, cleanupNew, err := stageForScan(ev.FilePath, newContent, ev.SessionID)
	if err != nil {
		return false, "", ""
	}
	defer cleanupNew()

	ascaParams.FilePath = stagedNew
	newResult, err := services.CreateASCAScanRequest(ascaParams, wrapperParams)
	if err != nil || newResult == nil {
		return false, "", ""
	}
	if newResult.Error != nil {
		return false, "", ""
	}
	if len(newResult.ScanDetails) == 0 {
		return false, "", ""
	}

	// For new files (no original content), every finding is new
	if originalContent == "" {
		r, c := formatFindings(ev.FilePath, newResult.ScanDetails)
		return true, r + guardrails.DenyMessage, c
	}

	// Delta: scan original content and find only newly introduced findings
	stagedOrig, cleanupOrig, err := stageForScan(ev.FilePath, originalContent, ev.SessionID)
	if err != nil {
		return false, "", ""
	}
	defer cleanupOrig()

	ascaParams.FilePath = stagedOrig
	origResult, err := services.CreateASCAScanRequest(ascaParams, wrapperParams)
	if err != nil || origResult == nil {
		return false, "", ""
	}
	var origDetails []grpcs.ScanDetail
	if origResult.Error == nil {
		origDetails = origResult.ScanDetails
	}

	newFindings := NewFindings(origDetails, newResult.ScanDetails)
	if len(newFindings) == 0 {
		return false, "", ""
	}

	r, c := formatFindings(ev.FilePath, newFindings)
	return true, r + guardrails.DenyMessage, c
}

// existingIgnoreFilePath returns the default realtime ignore-file path only when it
// exists on disk. The ASCA service short-circuits the scan with an error when a
// configured ignore path is missing, so we pass it only once the user has created it
// via `cx ignore-vulnerability`; otherwise the scan runs without ignore filtering.
func existingIgnoreFilePath() string {
	p := ignore.DefaultPath()
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

// shouldUpdateVersion returns whether ASCA should check for a newer version.
func shouldUpdateVersion() bool {
	v := viper.GetString(params.DisableASCALatestVersionKey)
	return v != "true"
}
