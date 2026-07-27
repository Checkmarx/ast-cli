package filtering

import (
	"fmt"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// AntMatcher implements [Matcher] using an ordered list of Ant-style glob
// rules following the Apache Ant patternset convention.
//
// # Pattern syntax
//
// Patterns use the Ant glob dialect as implemented by bmatcuk/doublestar:
//
//   - "*" matches any sequence of non-separator characters within one path
//     segment ("*.go" matches "main.go" but not "a/b.go")
//   - "**" matches any sequence of characters including path separators,
//     including zero segments ("src/**" matches "src" and everything under
//     it; "**/*.go" matches any .go file at any depth)
//   - "?" matches exactly one non-separator character
//   - "[abc]" is a character class
//   - "{a,b}" is an alternation ("*.{js,ts}" matches .js and .ts files)
//
// # Include / exclude semantics (Ant convention)
//
// Rules are evaluated in the order they are provided. The last matching rule
// wins, consistent with Apache Ant patternset evaluation.
//
//   - A bare pattern (no prefix)  → INCLUDE entries that match
//   - A "!" prefix                → EXCLUDE entries that match
//
// Default state:
//   - When at least one bare (include) rule is present, the default is
//     EXCLUDED: only entries that match an include rule are kept, subject to
//     later exclude rules.
//   - When only "!" (exclude) rules are present and no bare rules exist, the
//     default is INCLUDED: everything passes through unless an exclude rule
//     matches it.
//   - Empty rule list: everything is INCLUDED (no restriction).
//
// Examples:
//
//	["**/*.java", "!**/Test*.java"]
//	  → include all .java files, then exclude those whose name starts with Test
//
//	["!**/node_modules/**"]
//	  → include everything except node_modules (exclude-only, no bare rules)
//
// # Directory precedence and sub-tree pruning
//
// A directory is pruned (not descended into) only when:
//  1. It is currently excluded (Excluded returns true), AND
//  2. No subsequent include rule could possibly match any descendant
//     ([MustDescend] returns false).
//
// When condition 2 is not met, the traversal descends and applies per-entry
// evaluation. This is the correct behaviour for patterns such as:
//
//	["**/*.java", "!**/generated/**"]
//
// where .java files are included but those under generated/ are excluded;
// the traversal must descend into generated/ to evaluate per-file.
//
// # Implicit depth anchoring
//
// A pattern that contains no "/" (ignoring a trailing "/") is implicitly
// treated as "**/pattern", matching at any depth. "*.java" means "any .java
// file anywhere in the tree."
//
// # Trailing slash
//
// A pattern ending with "/" matches directories only and will never match a
// file, even one with the same name.
//
// # Relative-path contract
//
// All relPath arguments must use forward slashes and must not start with "/".
// On Windows, callers must normalise paths with filepath.ToSlash before
// passing them to this matcher.
type AntMatcher struct {
	rules          []antRule
	hasIncludeRule bool // true when at least one bare (include) rule exists
}

// antRule is a compiled, normalised pattern with its intent.
type antRule struct {
	pattern string // normalised, anchored, ready for doublestar.Match
	include bool   // true = bare pattern (include); false = "!" pattern (exclude)
	dirOnly bool   // true = trailing "/" was present; only matches directories
}

// NewAntMatcher parses and compiles patterns into an AntMatcher.
//
// Patterns are trimmed of whitespace; blank patterns are silently skipped.
// A leading "!" makes the rule an exclusion. A trailing "/" restricts the
// rule to directories only. Both may be combined: "!src/test/" excludes the
// src/test directory but not files named "src/test".
//
// An error is returned only for patterns that are syntactically invalid for
// the doublestar engine (e.g. unclosed bracket "src/[invalid").
func NewAntMatcher(patterns []string) (*AntMatcher, error) {
	rules := make([]antRule, 0, len(patterns))
	hasIncludeRule := false

	for _, raw := range patterns {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		// "!" prefix marks this as an exclusion rule.
		exclude := strings.HasPrefix(raw, "!")
		norm := raw
		if exclude {
			norm = norm[1:] // strip "!" before further processing
		}

		// Normalise OS path separators to forward slash.
		norm = toSlash(norm)

		// Normalise to lowercase for case-insensitive matching on all platforms.
		norm = strings.ToLower(norm)

		dirOnly := strings.HasSuffix(norm, "/")
		norm = strings.TrimSuffix(norm, "/")

		if norm == "" {
			// Pattern was just "!" or "!/" — meaningless, skip.
			continue
		}

		// Validate syntax against doublestar before storing.
		if _, err := doublestar.Match(norm, "probe"); err != nil {
			return nil, fmt.Errorf("filtering: invalid pattern %q: %w", raw, err)
		}

		// Implicit depth anchoring: patterns without "/" match at any depth.
		if !strings.Contains(norm, "/") {
			norm = "**/" + norm
		}

		include := !exclude
		if include {
			hasIncludeRule = true
		}

		rules = append(rules, antRule{
			pattern: norm,
			include: include,
			dirOnly: dirOnly,
		})
	}
	return &AntMatcher{rules: rules, hasIncludeRule: hasIncludeRule}, nil
}

// Excluded implements [Matcher].
//
// The default state depends on the rule set:
//   - If any bare (include) rule exists: default is excluded.
//   - If only "!" (exclude) rules exist: default is included.
//   - No rules: included.
//
// Rules are evaluated in order; the last matching rule wins.
func (m *AntMatcher) Excluded(relPath string, isDir bool) bool {
	relPath = normalise(relPath)

	// When include rules are present, entries start as excluded and must be
	// explicitly included. When only exclude rules exist, entries start as
	// included and are explicitly excluded.
	excluded := m.hasIncludeRule
	matchedAny := false

	for _, r := range m.rules {
		if r.dirOnly && !isDir {
			continue // directory-only rule cannot match a file
		}
		if antMatches(r.pattern, relPath) {
			matchedAny = true
			if r.include {
				excluded = false // bare rule: include this entry
			} else {
				excluded = true // "!" rule: exclude this entry
			}
		}
	}

	// A directory that is a required stepping stone toward a literal-prefixed
	// include pattern (e.g. "src" for pattern "src/**/*.test.*") must not be
	// pruned by the default include-only exclusion just because it doesn't
	// itself match. This only applies when no rule explicitly matched the
	// directory; an explicit "!" exclusion still stands and is left to
	// MustDescend to resolve.
	if isDir && excluded && !matchedAny && m.hasIncludeRule {
		for _, r := range m.rules {
			if !r.include {
				continue
			}
			if requiredAncestorPrefixMatch(r.pattern, relPath) {
				excluded = false
				break
			}
		}
	}

	return excluded
}

// MustDescend implements [Matcher].
//
// It returns true when the directory at dirRelPath is currently excluded but a
// subsequent include rule in the list could potentially match a descendant.
// In that case the caller must descend into the directory and evaluate each
// child with [Excluded] rather than pruning the sub-tree.
//
// The algorithm:
//  1. Find the index of the last exclusion rule ("!" rule) that matched
//     dirRelPath. If the directory is excluded because no include rule
//     matched it (hasIncludeRule=true, no bare rule hit), search for any
//     include rule that could match a child.
//  2. Check every include rule that appears after the last exclusion match.
//     Use [couldMatchUnder] to determine whether it could reach a descendant.
//
// The check is conservative: it may return true when no actual descendant
// would be included (causing unnecessary descent). It will never return false
// when a descendant would genuinely be included.
func (m *AntMatcher) MustDescend(dirRelPath string) bool {
	dirRelPath = normalise(dirRelPath)

	// Find the last exclusion ("!" rule) that matched this directory.
	lastExcludeIdx := -1
	for i, r := range m.rules {
		if r.include {
			continue // not an exclusion rule
		}
		if antMatches(r.pattern, dirRelPath) {
			lastExcludeIdx = i
		}
	}

	if lastExcludeIdx >= 0 {
		// Directory was explicitly excluded by a "!" rule.
		// Check for any include rule AFTER that exclusion that could reach a child.
		for i := lastExcludeIdx + 1; i < len(m.rules); i++ {
			r := m.rules[i]
			if !r.include {
				continue // not an include rule
			}
			if couldMatchUnder(r.pattern, dirRelPath) {
				return true
			}
		}
		return false
	}

	// The directory is excluded because no bare include rule matched it
	// (hasIncludeRule is true but no include rule hit dirRelPath). Check
	// whether any include rule could match a descendant — if so, we must
	// descend to give those children a chance to be included.
	if m.hasIncludeRule {
		for _, r := range m.rules {
			if !r.include {
				continue
			}
			if couldMatchUnder(r.pattern, dirRelPath) {
				return true
			}
		}
	}

	return false
}

// couldMatchUnder returns true when pattern could match at least one path of
// the form dirRelPath+"/"+<anything>. It is used by [MustDescend] to decide
// whether an include rule can reach descendants of an excluded directory.
//
// The algorithm is conservative (may return true when no real match exists)
// but never returns false when a match genuinely exists:
//
//   - Patterns starting with "**/" (or equal to "**") can reach any directory
//     → always return true.
//   - Otherwise extract the pattern's static prefix (segments before the first
//     wildcard) and check whether it is compatible with dirRelPath: either the
//     static prefix starts inside dirRelPath, or dirRelPath is inside the
//     static prefix (wildcards may bridge the gap).
//   - Patterns whose first segment is a wildcard (static prefix empty) could
//     match under any single-level directory → return true conservatively.
func couldMatchUnder(pattern, dirRelPath string) bool {
	// Fast path: "**/" prefix or bare "**" can reach any directory.
	if pattern == "**" || strings.HasPrefix(pattern, "**/") {
		return true
	}

	staticPrefix := patternStaticPrefix(pattern)

	if staticPrefix == "" {
		// Pattern starts with a wildcard at segment 0 — conservative true.
		return true
	}

	// Normalise to trailing-slash form for clean prefix comparison.
	prefixWithSlash := staticPrefix
	if !strings.HasSuffix(prefixWithSlash, "/") {
		prefixWithSlash += "/"
	}
	dirWithSlash := dirRelPath + "/"

	// Pattern targets something inside dirRelPath.
	if strings.HasPrefix(prefixWithSlash, dirWithSlash) {
		return true
	}

	// dirRelPath is inside the static prefix area — wildcards after the
	// static prefix may reach children of dirRelPath.
	if strings.HasPrefix(dirWithSlash, prefixWithSlash) {
		return true
	}

	return false
}

// requiredAncestorPrefixMatch reports whether dirRelPath is necessarily on the
// path to a match of pattern, based solely on pattern's literal (non-wildcard)
// static prefix. Unlike [couldMatchUnder], it does not conservatively return
// true for patterns with no static prefix (e.g. "**/src") — those provide no
// concrete evidence that a specific, unrelated directory is relevant, so they
// must not override the default exclusion for arbitrary directories.
func requiredAncestorPrefixMatch(pattern, dirRelPath string) bool {
	staticPrefix := patternStaticPrefix(pattern)
	if staticPrefix == "" {
		return false
	}

	prefixWithSlash := staticPrefix
	dirWithSlash := dirRelPath + "/"

	if strings.HasPrefix(prefixWithSlash, dirWithSlash) {
		return true
	}
	if strings.HasPrefix(dirWithSlash, prefixWithSlash) {
		return true
	}
	return false
}

// patternStaticPrefix returns the slash-separated path segments of pattern
// that precede the first glob metacharacter (*, ?, [, {).
//
// Examples:
//
//	"src/test/fixtures/**" → "src/test/fixtures/"
//	"src/*/test"           → "src/"
//	"**/*.test.js"         → ""  (first segment is **)
//	"*.go"                 → ""  (first char is *)
func patternStaticPrefix(pattern string) string {
	segments := strings.Split(pattern, "/")
	var staticSegs []string
	for _, seg := range segments {
		if containsGlobMeta(seg) {
			break
		}
		staticSegs = append(staticSegs, seg)
	}
	if len(staticSegs) == 0 {
		return ""
	}
	return strings.Join(staticSegs, "/") + "/"
}

// containsGlobMeta reports whether s contains any doublestar glob metacharacter.
func containsGlobMeta(s string) bool {
	return strings.ContainsAny(s, "*?[{")
}

// antMatches reports whether pattern matches relPath directly OR whether
// relPath is a descendant of a path that pattern matches (sub-path pruning).
//
// Sub-path pruning handles patterns like "src/test" matching files inside
// "src/test/Foo.java". It is suppressed for patterns whose last segment
// contains a dot (file-extension and dotfile patterns) or a glob
// metacharacter (the pattern already fully specifies its own depth via
// doublestar semantics, e.g. "*/src/test/*" or "test/**"), preventing a
// pattern like "**/*.test.js", "Mexico/.gitignore", or "*/src/test/*" from
// accidentally matching paths that are descendants of what it actually
// matches.
//
// Examples:
//
//	antMatches("**/test",      "src/test")           → true  (direct)
//	antMatches("**/test",      "src/test/Foo.java")  → true  (sub-path)
//	antMatches("**/*.test.js", "src/Foo.test.js")    → true  (direct)
//	antMatches("**/*.test.js", "src/Foo.go")         → false
//	antMatches("*/src/test/*", "a/src/test/sub/x")   → false (last segment is a wildcard)
func antMatches(pattern, relPath string) bool {
	if matched, _ := doublestar.Match(pattern, relPath); matched {
		return true
	}

	// Sub-path check: does the pattern match any ancestor directory of relPath?
	// Suppressed for file/dotfile/wildcard-terminated patterns to avoid false
	// positives.
	if !looksLikeFilePattern(pattern) {
		parts := strings.Split(relPath, "/")
		for i := len(parts) - 1; i > 0; i-- {
			ancestor := strings.Join(parts[:i], "/")
			if matched, _ := doublestar.Match(pattern, ancestor); matched {
				return true
			}
		}
	}

	return false
}

// looksLikeFilePattern returns true when the last segment of the pattern
// contains a dot anywhere or a glob metacharacter, indicating an
// extension-based, dotfile, or explicitly depth-bounded pattern. Such
// patterns suppress sub-path ancestor matching because they already fully
// specify their own matching depth.
//
// Examples:
//
//	"**/*.test.js"      → true  (has dot → suppress sub-path)
//	"Mexico/.gitignore" → true  (has dot → suppress sub-path)
//	"*/src/test/*"      → true  (last segment is "*" → suppress sub-path)
//	"**/test"           → false (no dot, literal segment → allow sub-path)
//	"node_modules"      → false (no dot, literal segment → allow sub-path)
func looksLikeFilePattern(pattern string) bool {
	lastSlash := strings.LastIndex(pattern, "/")
	lastSeg := pattern[lastSlash+1:]
	return strings.ContainsRune(lastSeg, '.') || containsGlobMeta(lastSeg)
}

// normalise converts backslashes to forward slashes and strips any leading slash.
func normalise(s string) string {
	s = toSlash(s)
	s = strings.TrimPrefix(s, "/")
	// Normalise to lowercase for case-insensitive matching on all platforms.
	return strings.ToLower(s)
}

// toSlash converts backslashes to forward slashes.
func toSlash(s string) string {
	return strings.ReplaceAll(s, "\\", "/")
}
