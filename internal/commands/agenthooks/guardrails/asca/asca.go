package asca

import (
	"os"
	"path/filepath"
	"strings"

	agenthooks "github.com/CheckmarxDev/ast-cx-hooks"
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
func ScanFileEdit(ev agenthooks.FileEditEvent, telemetryWrapper wrappers.TelemetryWrapper, agent string) (blocked bool, reason, context string) {
	findingCount := 0

	defer func() {
		if r := recover(); r != nil {
			blocked = false
			reason = ""
			context = ""
		}
		logASCATelemetry(telemetryWrapper, agent, findingCount)
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
		IgnoredFilePath: existingIgnoreFilePath(ev.WorkDir),
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
		r, c := formatFindings(ev.FilePath, newResult.ScanDetails, ev.WorkDir)
		findingCount = len(newResult.ScanDetails)
		return true, r, c
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
		findingCount = 0
		return false, "", ""
	}

	r, c := formatFindings(ev.FilePath, newFindings, ev.WorkDir)
	findingCount = len(newFindings)
	return true, r, c
}

// existingIgnoreFilePath returns the default realtime ignore-file path only when it
// exists on disk. The ASCA service short-circuits the scan with an error when a
// configured ignore path is missing, so we pass it only once the user has created it
// via `cx ignore-vulnerability`; otherwise the scan runs without ignore filtering.
func existingIgnoreFilePath(workDir string) string {
	p := ignore.PathFor(workDir)
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

// logASCATelemetry sends a telemetry event for ASCA scan results.
// Called once after ASCA scan is performed with the actual finding count.
func logASCATelemetry(telemetryWrapper wrappers.TelemetryWrapper, agent string, totalCount int) {
	if telemetryWrapper == nil || totalCount == 0 {
		return
	}

	telemetryData := &wrappers.DataForAITelemetry{

		//agent = aiProvider
		//hooks-detect for detection
		//subtype = scan
		//  hooks-remeditae
		//subType = fixWithAIchet

		Agent:      agent + "-cli",
		AIProvider: agent,
		Engine:     "Asca",
		TotalCount: totalCount,
		UniqueID:   wrappers.GetUniqueID(),
		Type:       "hooks-detect",
		SubType:    "scan",
		ScanType:   "asca",
	}

	if err := telemetryWrapper.SendAIDataToLog(telemetryData); err != nil {
		// fail-open
	}
}
