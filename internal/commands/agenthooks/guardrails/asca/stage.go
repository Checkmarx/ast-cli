package asca

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
)

// noop is a no-op cleanup func returned on error paths so callers can always defer cleanup().
var noop = func() {}

// asciiSafe replaces every non-ASCII rune with a space so ASCA's language
// parsers can tokenise the file. Non-ASCII chars appear in comments or string
// literals but never in code constructs that ASCA analyses for vulnerabilities;
// substituting them with spaces preserves line numbers and code structure.
func asciiSafe(s string) string {
	if utf8.ValidString(s) {
		allASCII := true
		for _, r := range s {
			if r > 127 {
				allASCII = false
				break
			}
		}
		if allASCII {
			return s
		}
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r > 127 {
			b.WriteByte(' ')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// stageForScan writes content to a fresh temp directory under os.TempDir(),
// preserving the original basename so ASCA's language detection works correctly
// and findings report a sensible file_name. The dir name includes a short,
// sanitized prefix of sessionID so concurrent agent sessions are visibly
// separated and orphaned dirs can be traced back to the session that created them.
//
// For AgentCopilotCLI, non-ASCII characters are replaced with spaces before
// writing — Copilot CLI may add non-ASCII chars (e.g. EM dashes in comments)
// that cause ASCA's parser to silently return 0 findings.
//
// Returns the staged path and a cleanup func. Caller must defer cleanup().
func stageForScan(originalPath, content, sessionID string, agent agenthooks.AgentID) (stagedPath string, cleanup func(), err error) {
	pattern := fmt.Sprintf("asca-hook-%s-*", safeSessionTag(sessionID))
	tempDir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return "", noop, err
	}

	base := filepath.Base(originalPath)
	if base == "." || base == ".." || base == "" || base == string(filepath.Separator) {
		_ = os.RemoveAll(tempDir)
		return "", noop, fmt.Errorf("invalid basename %q", base)
	}

	staged := filepath.Join(tempDir, base)
	// Path-traversal guard, copied from iacrealtime/file_handler.go:62
	if !strings.HasPrefix(filepath.Clean(staged), filepath.Clean(tempDir)) {
		_ = os.RemoveAll(tempDir)
		return "", noop, fmt.Errorf("path traversal in %q", base)
	}

	toWrite := content
	if agent == agenthooks.AgentCopilotCLI {
		toWrite = asciiSafe(content)
	}
	if err := os.WriteFile(staged, []byte(toWrite), 0o600); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", noop, err
	}
	return staged, func() { _ = os.RemoveAll(tempDir) }, nil
}

// safeSessionTag returns up to 8 chars of [a-zA-Z0-9_-] from sid, or "anon" if
// sid is empty or has no usable characters. Keeps the dir name short and shell-safe.
func safeSessionTag(sid string) string {
	if sid == "" {
		return "anon"
	}
	var b strings.Builder
	for _, r := range sid {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
			if b.Len() >= 8 {
				break
			}
		}
	}
	if b.Len() == 0 {
		return "anon"
	}
	return b.String()
}
