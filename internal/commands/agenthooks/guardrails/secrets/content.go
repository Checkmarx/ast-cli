package secrets

import (
	"os"
	"strings"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
)

func normLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// proposedContent returns the file content that would exist after changes are applied,
// plus the content currently on disk (empty when the file does not exist yet).
func proposedContent(filePath string, changes []agenthooks.FileDiff) (newContent, originalContent string) {
	diskBytes, readErr := os.ReadFile(filePath)
	if readErr == nil {
		originalContent = string(diskBytes)
	}

	if len(changes) == 1 && changes[0].Before == "" {
		return changes[0].After, originalContent
	}

	current := normLF(originalContent)
	for _, diff := range changes {
		before := normLF(diff.Before)
		after := normLF(diff.After)
		idx := strings.Index(current, before)
		if idx < 0 {
			continue
		}
		current = current[:idx] + after + current[idx+len(before):]
	}

	return current, normLF(originalContent)
}
