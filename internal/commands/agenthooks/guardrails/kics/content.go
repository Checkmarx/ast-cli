package kics

import (
	"os"
	"strings"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
)

func normLF(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

// proposedContent returns the file content that would exist after ev.Changes are applied.
// Returns (newContent, originalContent, err).
func proposedContent(filePath string, changes []agenthooks.FileDiff) (newContent, originalContent string, err error) {
	diskBytes, readErr := os.ReadFile(filePath)
	if readErr == nil {
		originalContent = string(diskBytes)
	}

	if len(changes) == 1 && changes[0].Before == "" {
		return changes[0].After, originalContent, nil
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

	originalContent = normLF(originalContent)
	return current, originalContent, nil
}
