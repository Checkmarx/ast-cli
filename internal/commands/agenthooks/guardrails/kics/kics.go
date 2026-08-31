package kics

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/checkmarx/ast-cli/internal/logger"
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
// verdict. Fail-open on infrastructure errors (Docker unavailable, image pull fail, panic),
// returning a skippedNote so the skipped check is visible.
func ScanFileEdit(ev agenthooks.FileEditEvent, svc *Scanner) (blocked bool, reason, context, note string) {
	defer func() {
		if r := recover(); r != nil {
			logger.PrintfIfVerbose("kics guardrail: recovered from panic, failing open: %v", r)
			blocked = false
			reason = ""
			context = ""
			note = skippedNote(ev.FilePath, fmt.Errorf("internal error: %v", r))
		}
	}()

	if !isSupportedByKICS(ev.FilePath) {
		return false, "", "", ""
	}

	newContent, originalContent, err := proposedContent(ev.FilePath, ev.Changes)
	if err != nil {
		return false, "", "", skippedNote(ev.FilePath, err)
	}
	if newContent == "" {
		return false, "", "", ""
	}

	// Stage and scan the proposed (new) content
	stagedNew, cleanupNew, err := stageForScan(ev.FilePath, newContent, ev.SessionID)
	if err != nil {
		return false, "", "", skippedNote(ev.FilePath, err)
	}
	defer cleanupNew()

	ignoreFilePath := existingIgnoreFilePath(ev.WorkDir)
	newResults, err := svc.scan(stagedNew, ignoreFilePath)
	if err != nil {
		// Fail open: Docker unavailable, image pull failure, feature flag disabled, etc.
		logger.PrintfIfVerbose("kics guardrail: scan of proposed content failed, failing open: %v", err)
		return false, "", "", skippedNote(ev.FilePath, err)
	}
	if len(newResults) == 0 {
		return false, "", "", ""
	}

	// For new files (no original content), every finding is new
	if originalContent == "" {
		r, c := formatFindings(ev.FilePath, newResults, ev.Agent)
		return true, r, c, ""
	}

	// Delta: scan original content and find only newly introduced findings
	stagedOrig, cleanupOrig, err := stageForScan(ev.FilePath, originalContent, ev.SessionID)
	if err != nil {
		return false, "", "", skippedNote(ev.FilePath, err)
	}
	defer cleanupOrig()

	origResults, err := svc.scan(stagedOrig, ignoreFilePath)
	if err != nil {
		// Fail open on original scan error
		logger.PrintfIfVerbose("kics guardrail: scan of original content failed, failing open: %v", err)
		return false, "", "", skippedNote(ev.FilePath, err)
	}

	newFindings := NewFindings(origResults, newResults)
	if len(newFindings) == 0 {
		return false, "", "", ""
	}

	r, c := formatFindings(ev.FilePath, newFindings, ev.Agent)
	return true, r, c, ""
}

// skippedNote is what the user sees when the guardrail fails open. Without it a
// file edited with no container engine running is indistinguishable from a file
// that scanned clean — the edit is allowed either way, and nothing says why.
func skippedNote(filePath string, err error) string {
	return fmt.Sprintf("Checkmarx IaC guardrail skipped %s: %v. The edit was allowed without an IaC security check.",
		filepath.Base(filePath), err)
}

// existingIgnoreFilePath returns the realtime ignore-file path anchored at workDir only
// when it exists on disk. Mirrors the ASCA pattern: anchor to workDir so the hook reads
// from the same absolute path that `cx ignore-vulnerability` writes to when run from the
// project root. Returns "" (no filtering) until the user creates the file.
func existingIgnoreFilePath(workDir string) string {
	p := ignore.PathFor(workDir)
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}
