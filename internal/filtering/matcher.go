// Package filtering provides composable, path-aware file and directory
// filtering logic for source-tree traversal.
//
// # Design principles
//
//   - The [Matcher] interface is the single extension point. New strategies
//     (regex-based, policy-file-driven, per-directory gitignore) implement
//     [Matcher] and compose via [MultiMatcher] without touching any call site
//     in scan.go.
//
//   - All matching operates on forward-slash relative paths from the source
//     root (e.g. "src/test/Foo.java"), never on bare filenames. This makes
//     path-segment patterns such as "src/**" meaningful and correct across
//     platforms.
//
//   - Directories are evaluated before files. When a directory is excluded
//     AND no later negation rule can re-include anything under it, the entire
//     sub-tree is pruned without descent ([Matcher.MustDescend] returns
//     false). When a later negation rule may re-include children, the caller
//     must descend and apply per-entry [Matcher.Excluded] checks.
//
//   - This package is pure logic; it has no knowledge of zip writers, cobra
//     commands, or any other infrastructure concern.
package filtering

// Matcher decides whether a given filesystem entry should be excluded from a
// scan, and whether a currently-excluded directory must still be descended
// into because a later negation rule may re-include entries within it.
//
// relPath is the slash-separated path relative to the source root, e.g.
// "src/test/Foo.java" or "node_modules". It never starts with "/".
// isDir is true when the entry is a directory (or a symlink that resolves
// to one).
type Matcher interface {
	// Excluded returns true when the entry at relPath should be skipped.
	// For a directory, callers must also check [MustDescend] before deciding
	// whether to prune the entire sub-tree.
	Excluded(relPath string, isDir bool) bool

	// MustDescend returns true when the directory at dirRelPath is excluded
	// by the current rule set BUT a later negation rule may re-include one or
	// more of its descendants. When true, callers must descend into the
	// directory and evaluate each child with [Excluded] rather than pruning
	// the sub-tree wholesale.
	//
	// Implementations should err on the side of descending (returning true)
	// when uncertain; the cost of unnecessary descent is performance, while
	// the cost of incorrect pruning is silently missing included files.
	MustDescend(dirRelPath string) bool
}

// MultiMatcher composes multiple Matchers. An entry is excluded when ANY
// constituent Matcher excludes it (logical OR). Descent is required when ANY
// constituent Matcher requires it (logical OR), which preserves the
// conservative "descend when in doubt" invariant.
type MultiMatcher struct {
	matchers []Matcher
}

// NewMultiMatcher returns a MultiMatcher that composes all provided
// Matchers. The zero-value (no matchers) excludes nothing.
func NewMultiMatcher(matchers ...Matcher) *MultiMatcher {
	return &MultiMatcher{matchers: matchers}
}

// Excluded implements [Matcher].
func (m *MultiMatcher) Excluded(relPath string, isDir bool) bool {
	for _, matcher := range m.matchers {
		if matcher.Excluded(relPath, isDir) {
			return true
		}
	}
	return false
}

// MustDescend implements [Matcher].
func (m *MultiMatcher) MustDescend(dirRelPath string) bool {
	for _, matcher := range m.matchers {
		if matcher.MustDescend(dirRelPath) {
			return true
		}
	}
	return false
}

// Add appends additional Matchers to an existing MultiMatcher. This allows
// incremental composition (e.g. per-directory gitignore files discovered at
// walk time).
func (m *MultiMatcher) Add(matchers ...Matcher) {
	m.matchers = append(m.matchers, matchers...)
}

// NopMatcher is a Matcher that never excludes anything and never requires
// forced descent. Useful as a safe zero-value default when no patterns are
// configured.
type NopMatcher struct{}

// Excluded implements [Matcher]; always returns false.
func (NopMatcher) Excluded(_ string, _ bool) bool { return false }

// MustDescend implements [Matcher]; always returns false.
func (NopMatcher) MustDescend(_ string) bool { return false }
