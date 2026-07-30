package asca

import (
	"os"
	"strings"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
)

// normLF normalises any line-ending variant (CRLF, bare CR, LF) to LF.
// Applied only for agents that send LF-only payloads against CRLF disk files
// (e.g. Copilot CLI on Windows) to ensure strings.Index finds a match.
func normLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

// ProposedContent returns the file content that would exist after ev.Changes are applied.
// Returns (newContent, originalContent, err).
//   - Full-file write: Changes = [{Before:"", After:X}] → newContent=X, originalContent=<disk or "">
//   - String-replace edit: read disk, apply each FileDiff.Before→After in order
//   - File doesn't exist on disk: originalContent=""
func ProposedContent(filePath string, changes []agenthooks.FileDiff, agent agenthooks.AgentID) (newContent, originalContent string, err error) {
	diskBytes, readErr := os.ReadFile(filePath)
	if readErr == nil {
		originalContent = string(diskBytes)
	}
	// readErr means file doesn't exist yet — originalContent stays ""

	// Full-file write: single diff with empty Before
	if len(changes) == 1 && changes[0].Before == "" {
		return changes[0].After, originalContent, nil
	}

	// Copilot CLI sends LF-only old_str/new_str regardless of the OS, while the
	// file on disk may have CRLF (Windows) or bare CR (classic macOS). Normalise
	// both the disk content and the diff strings to LF before matching so
	// strings.Index reliably finds the substring on all three OSes.
	normalize := agent == agenthooks.AgentCopilotCLI
	current := originalContent
	if normalize {
		current = normLF(current)
	}

	for _, diff := range changes {
		before := diff.Before
		after := diff.After
		if normalize {
			before = normLF(before)
			after = normLF(after)
		}
		idx := strings.Index(current, before)
		if idx < 0 {
			// Before not found — malformed edit; fail-open, let agent's tool surface it
			continue
		}
		current = current[:idx] + after + current[idx+len(before):]
	}

	if normalize {
		originalContent = normLF(originalContent)
	}
	return current, originalContent, nil
}
