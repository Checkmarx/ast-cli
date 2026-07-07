package asca

import (
	"os"
	"strings"

	agenthooks "github.com/CheckmarxDev/ast-cx-hooks"
)

// ProposedContent returns the file content that would exist after ev.Changes are applied.
// Returns (newContent, originalContent, err).
//   - Full-file write: Changes = [{Before:"", After:X}] → newContent=X, originalContent=<disk or "">
//   - String-replace edit: read disk, apply each FileDiff.Before→After in order
//   - File doesn't exist on disk: originalContent=""
func ProposedContent(filePath string, changes []agenthooks.FileDiff) (newContent, originalContent string, err error) {
	diskBytes, readErr := os.ReadFile(filePath)
	if readErr == nil {
		originalContent = string(diskBytes)
	}
	// readErr means file doesn't exist yet — originalContent stays ""

	// Full-file write: single diff with empty Before
	if len(changes) == 1 && changes[0].Before == "" {
		return changes[0].After, originalContent, nil
	}

	// String-replace: apply each diff in order against current content
	current := originalContent
	for _, diff := range changes {
		idx := strings.Index(current, diff.Before)
		if idx < 0 {
			// Before not found — malformed edit; fail-open, let agent's tool surface it
			continue
		}
		current = current[:idx] + diff.After + current[idx+len(diff.Before):]
	}
	return current, originalContent, nil
}
