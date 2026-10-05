package guardrails

import (
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// hasGlobMeta reports whether s contains glob metacharacters (*, ?, [).
func hasGlobMeta(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

// normalizeForMatch lower-cases path and converts separators to forward slashes,
// mirroring how the rest of the guardrails package normalises paths.
func normalizeForMatch(s string) string {
	return strings.ToLower(filepath.ToSlash(s))
}

// matchFilePattern reports whether target matches pattern.
//
// A match is any of:
//   - literal equality of the normalized forms
//   - basename equality (so a policy entry like "kubeconfig" matches any path whose leaf is kubeconfig)
//   - suffix equality after "/" (so ".env" matches "/app/.env") — preserved for back-compat
//   - doublestar glob match against the full normalized target
//   - doublestar glob match against just the basename (so "*.pem" matches "foo.pem")
//
// Normalization is lowercase + forward-slash; patterns authored with backslashes
// on Windows still work.
func matchFilePattern(pattern, target string) bool {
	p := normalizeForMatch(pattern)
	t := normalizeForMatch(target)
	base := filepath.Base(t)

	if p == t || p == base {
		return true
	}
	if strings.HasSuffix(t, "/"+p) {
		return true
	}
	if hasGlobMeta(p) {
		if ok, _ := doublestar.Match(p, t); ok {
			return true
		}
		if ok, _ := doublestar.Match(p, base); ok {
			return true
		}
	}
	return false
}

// anyPatternMatchesFile returns true when target matches at least one entry in patterns
// via matchFilePattern. Convenience wrapper for call sites that just need a boolean.
func anyPatternMatchesFile(patterns []string, target string) bool {
	for _, p := range patterns {
		if matchFilePattern(p, target) {
			return true
		}
	}
	return false
}

// matchDirContains reports whether target is pattern itself or sits under it.
//
// For literal patterns this is the classic "target == dir OR target starts with dir/".
// For glob patterns it additionally accepts target when doublestar matches
// either the pattern directly (target is the dir) or "pattern/**"
// (target is a file inside the glob-matched dir).
func matchDirContains(pattern, target string) bool {
	p := strings.TrimSuffix(normalizeForMatch(pattern), "/")
	t := normalizeForMatch(target)

	if t == p || strings.HasPrefix(t, p+"/") {
		return true
	}
	if hasGlobMeta(p) {
		if ok, _ := doublestar.Match(p, t); ok {
			return true
		}
		if ok, _ := doublestar.Match(p+"/**", t); ok {
			return true
		}
	}
	return false
}
