package kics

import (
	"os"
	"path/filepath"
	"strings"

	agenthooks "github.com/CheckmarxDev/ast-cx-hooks"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ignore"
)

// isSupportedByKICS returns true when the file matches a KICS-supported extension or basename.
// Mirrors params.KicsBaseFilters: basename match for Dockerfile/.dockerfile,
// extension match for .tf/.yaml/.yml/.json/.auto.tfvars/.terraform.tfvars/.proto.
func isSupportedByKICS(filePath string) bool {
	base := filepath.Base(filePath)
	baseLower := strings.ToLower(base)

	// Check basenames (Dockerfile and .dockerfile are basenames, not extensions)
	for _, filter := range params.KicsBaseFilters {
		filterLower := strings.ToLower(filter)
		// Basename matches (no dot prefix means it's a filename, not an extension)
		if !strings.HasPrefix(filterLower, ".") {
			if baseLower == filterLower {
				return true
			}
			continue
		}
		// Extension matches — check if the file path ends with the filter string
		// (handles compound extensions like .auto.tfvars and .terraform.tfvars)
		if strings.HasSuffix(strings.ToLower(filePath), filterLower) {
			return true
		}
	}
	return false
}

// ScanFileEdit runs KICS on the proposed post-edit content.
// Returns blocked=true with a formatted reason and remediation context when KICS
// finds *new* vulnerabilities introduced by ev.Changes (delta-detection for edits;
// any-vuln for new writes). Findings the user already suppressed via
// `cx ignore-vulnerability` (the realtime ignore file) are filtered out before the
// verdict. Fail-open on infrastructure errors (Docker unavailable, image pull fail, panic).
func ScanFileEdit(ev agenthooks.FileEditEvent, svc *Scanner) (blocked bool, reason, context string) {
	defer func() {
		if r := recover(); r != nil {
			blocked = false
			reason = ""
			context = ""
		}
	}()

	if !isSupportedByKICS(ev.FilePath) {
		return false, "", ""
	}

	newContent, originalContent, err := proposedContent(ev.FilePath, ev.Changes)
	if err != nil || newContent == "" {
		return false, "", ""
	}

	// Stage and scan the proposed (new) content
	stagedNew, cleanupNew, err := stageForScan(ev.FilePath, newContent, ev.SessionID)
	if err != nil {
		return false, "", ""
	}
	defer cleanupNew()

	newResults, err := svc.scan(stagedNew)
	if err != nil {
		// Fail open: Docker unavailable, image pull failure, feature flag disabled, etc.
		return false, "", ""
	}
	if len(newResults) == 0 {
		return false, "", ""
	}

	// For new files (no original content), every finding is new
	if originalContent == "" {
		r, c := formatFindings(ev.FilePath, newResults)
		return true, r, c
	}

	// Delta: scan original content and find only newly introduced findings
	stagedOrig, cleanupOrig, err := stageForScan(ev.FilePath, originalContent, ev.SessionID)
	if err != nil {
		return false, "", ""
	}
	defer cleanupOrig()

	origResults, err := svc.scan(stagedOrig)
	if err != nil {
		// Fail open on original scan error
		return false, "", ""
	}

	newFindings := NewFindings(origResults, newResults)
	if len(newFindings) == 0 {
		return false, "", ""
	}

	r, c := formatFindings(ev.FilePath, newFindings)
	return true, r, c
}

// existingIgnoreFilePath returns the default realtime ignore-file path only when it
// exists on disk. The IaC realtime service logs a warning and skips ignore filtering
// when a missing path is passed, but we keep the pattern consistent with ASCA.
func existingIgnoreFilePath() string {
	p := ignore.DefaultPath()
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}
