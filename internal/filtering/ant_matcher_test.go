package filtering_test

import (
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/filtering"
)

// ── Construction ─────────────────────────────────────────────────────────────

func TestNewAntMatcher_ValidPatterns(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		patterns []string
	}{
		{"empty list", []string{}},
		{"nil list", nil},
		{"blank strings skipped", []string{"", "  ", "\t"}},
		{"bare include", []string{"**/*.java"}},
		{"exclude with !", []string{"!**/test*"}},
		{"exclude directory with !", []string{"!src/test/"}},
		{"mixed include and exclude", []string{"**/*.java", "!**/Test*.java"}},
		{"trailing slash dir-only include", []string{"src/"}},
		{"trailing slash dir-only exclude", []string{"!node_modules/"}},
		{"just exclamation is skipped", []string{"!"}},
		{"just exclamation-slash is skipped", []string{"!/"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m, err := filtering.NewAntMatcher(tc.patterns)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m == nil {
				t.Fatal("expected non-nil matcher")
			}
		})
	}
}

func TestNewAntMatcher_InvalidSyntax(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		pattern string
		wantErr string
	}{
		{"unclosed bracket", "src/[invalid", "invalid pattern"},
		{"exclude with unclosed bracket", "![bad", "invalid pattern"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := filtering.NewAntMatcher([]string{tc.pattern})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tc.wantErr, err)
			}
		})
	}
}

// ── Default state ─────────────────────────────────────────────────────────────

func TestAntMatcher_DefaultState(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		patterns []string
		relPath  string
		isDir    bool
		want     bool // want Excluded
	}{
		{
			// No rules → everything included.
			name:     "no rules: everything included",
			patterns: nil,
			relPath:  "src/Foo.java",
			want:     false,
		},
		{
			// Only exclude rules → default included; exclusion applied.
			name:     "only exclude rules: unmatched entry is included",
			patterns: []string{"!**/node_modules/**"},
			relPath:  "src/Foo.java",
			want:     false,
		},
		{
			// Only exclude rules → matched entry is excluded.
			name:     "only exclude rules: matched entry is excluded",
			patterns: []string{"!**/node_modules/**"},
			relPath:  "node_modules/lodash/index.js",
			want:     true,
		},
		{
			// Include rules present → default excluded; unmatched entry excluded.
			name:     "include rules present: unmatched entry is excluded",
			patterns: []string{"**/*.java"},
			relPath:  "src/Foo.go",
			want:     true,
		},
		{
			// Include rules present → matched entry is included.
			name:     "include rules present: matched entry is included",
			patterns: []string{"**/*.java"},
			relPath:  "src/Foo.java",
			want:     false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m, err := filtering.NewAntMatcher(tc.patterns)
			if err != nil {
				t.Fatalf("unexpected construction error: %v", err)
			}
			got := m.Excluded(tc.relPath, tc.isDir)
			if got != tc.want {
				t.Errorf("Excluded(%q, isDir=%v) = %v, want %v", tc.relPath, tc.isDir, got, tc.want)
			}
		})
	}
}

// ── Include / exclude with requirement samples ────────────────────────────────

func TestAntMatcher_IncludeExclude_RequirementSamples(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		patterns []string
		relPath  string
		isDir    bool
		want     bool // want Excluded
	}{
		// ── bare *.test.tsx means INCLUDE those files ────────────────────
		// With include rules present, everything else is excluded by default.
		{
			name:     "*.test.tsx includes matching file",
			patterns: []string{"*.test.tsx"},
			relPath:  "Button.test.tsx",
			want:     false, // included
		},
		{
			name:     "*.test.tsx includes in subdirectory (implicit **/)",
			patterns: []string{"*.test.tsx"},
			relPath:  "src/components/Button.test.tsx",
			want:     false, // included
		},
		{
			name:     "*.test.tsx excludes non-test file (default excluded when include rules present)",
			patterns: []string{"*.test.tsx"},
			relPath:  "src/components/Button.tsx",
			want:     true, // excluded by default
		},

		// ── !*.test.tsx means EXCLUDE those files ───────────────────────
		// With only exclude rules, everything else is included by default.
		{
			name:     "!*.test.tsx excludes matching file",
			patterns: []string{"!*.test.tsx"},
			relPath:  "src/components/Button.test.tsx",
			want:     true, // excluded
		},
		{
			name:     "!*.test.tsx does not exclude non-test file",
			patterns: []string{"!*.test.tsx"},
			relPath:  "src/components/Button.tsx",
			want:     false, // included by default
		},

		// ── !**/*.test.js (exclude test files, include everything else) ──
		{
			name:     "!**/*.test.js excludes at any depth",
			patterns: []string{"!**/*.test.js"},
			relPath:  "deep/src/utils/helper.test.js",
			want:     true,
		},
		{
			name:     "!**/*.test.js does not exclude plain .js",
			patterns: []string{"!**/*.test.js"},
			relPath:  "src/utils/helper.js",
			want:     false,
		},

		// ── **/*.java, !**/Test*.java (Ant patternset idiom) ─────────────
		{
			name:     "include java exclude test classes: Foo.java included",
			patterns: []string{"**/*.java", "!**/Test*.java"},
			relPath:  "src/main/Foo.java",
			want:     false,
		},
		{
			name:     "include java exclude test classes: TestFoo.java excluded",
			patterns: []string{"**/*.java", "!**/Test*.java"},
			relPath:  "src/test/TestFoo.java",
			want:     true,
		},
		{
			name:     "include java exclude test classes: Foo.go excluded (not java)",
			patterns: []string{"**/*.java", "!**/Test*.java"},
			relPath:  "src/main/Foo.go",
			want:     true,
		},

		// ── src/**/*.test.* include ──────────────────────────────────────
		{
			name:     "src/**/*.test.* includes test files under src",
			patterns: []string{"src/**/*.test.*"},
			relPath:  "src/utils/helper.test.ts",
			want:     false,
		},
		{
			name:     "src/**/*.test.* excludes files outside src (default)",
			patterns: []string{"src/**/*.test.*"},
			relPath:  "lib/utils/helper.test.ts",
			want:     true,
		},

		// ── !Mexico/.gitignore (exclude that specific file) ──────────────
		{
			name:     "!Mexico/.gitignore excludes that exact file",
			patterns: []string{"!Mexico/.gitignore"},
			relPath:  "Mexico/.gitignore",
			want:     true,
		},
		{
			name:     "!Mexico/.gitignore does not exclude other .gitignore",
			patterns: []string{"!Mexico/.gitignore"},
			relPath:  "src/.gitignore",
			want:     false,
		},
		// Sub-path pruning must not fire for dotfile patterns.
		{
			name:     "!Mexico/.gitignore does not sub-path match a path under it",
			patterns: []string{"!Mexico/.gitignore"},
			relPath:  "Mexico/.gitignore/impostor",
			want:     false,
		},

		// ── !Mexico/src/test (exclude path-anchored directory) ───────────
		{
			name:     "!Mexico/src/test excludes file inside via sub-path pruning",
			patterns: []string{"!Mexico/src/test"},
			relPath:  "Mexico/src/test/Foo.java",
			want:     true,
		},

		// ── Windows backslash normalisation ─────────────────────────────
		{
			name:     "backslash relPath is normalised for include pattern",
			patterns: []string{"src/**/*.test.ts"},
			relPath:  `src\utils\helper.test.ts`,
			want:     false, // included
		},
		{
			name:     "backslash relPath is normalised for exclude pattern",
			patterns: []string{"!src/**/*.test.ts"},
			relPath:  `src\utils\helper.test.ts`,
			want:     true, // excluded
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m, err := filtering.NewAntMatcher(tc.patterns)
			if err != nil {
				t.Fatalf("unexpected construction error: %v", err)
			}
			got := m.Excluded(tc.relPath, tc.isDir)
			if got != tc.want {
				t.Errorf("Excluded(%q, isDir=%v) = %v, want %v", tc.relPath, tc.isDir, got, tc.want)
			}
		})
	}
}

// ── Directory precedence ──────────────────────────────────────────────────────

func TestAntMatcher_DirectoryPrecedence(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		patterns []string
		relPath  string
		isDir    bool
		want     bool // want Excluded
	}{
		// !test/** — exclude-only: default included, test/ and its contents excluded
		{"!test/** excludes the dir itself", []string{"!test/**"}, "test", true, true},
		{"!test/** excludes files inside", []string{"!test/**"}, "test/Foo.java", false, true},
		{"!test/** does not exclude unrelated dir", []string{"!test/**"}, "src", true, false},

		// !src/test — exclude-only: sub-path pruning applies to files inside
		{"!src/test excludes the directory", []string{"!src/test"}, "src/test", true, true},
		{"!src/test excludes files under it (sub-path)", []string{"!src/test"}, "src/test/Foo.java", false, true},
		{"!src/test does not exclude src/main", []string{"!src/test"}, "src/main", true, false},

		// !**/node_modules/**
		{"!**/node_modules/** excludes node_modules at root", []string{"!**/node_modules/**"}, "node_modules", true, true},
		{"!**/node_modules/** excludes nested node_modules", []string{"!**/node_modules/**"}, "packages/lib/node_modules", true, true},
		{"!**/node_modules/** excludes file deep inside", []string{"!**/node_modules/**"}, "packages/lib/node_modules/lodash/index.js", false, true},

		// !*/src/test/*
		{"!*/src/test/* excludes single-segment top dir", []string{"!*/src/test/*"}, "CICD/src/test/Foo.java", false, true},
		{"!*/src/test/* does not exclude two levels deep", []string{"!*/src/test/*"}, "CICD/src/test/sub/Foo.java", false, false},

		// !*test*/**
		{"!*test*/** excludes test-named dir", []string{"!*test*/**"}, "mytest_utils", true, true},
		{"!*test*/** excludes files inside test-named dir", []string{"!*test*/**"}, "mytest_utils/helper.go", false, true},

		// trailing slash — directory-only exclude
		{"!test/ excludes directory", []string{"!test/"}, "test", true, true},
		{"!test/ does not exclude file named test", []string{"!test/"}, "test", false, false},

		// trailing slash — directory-only include
		{"src/ includes the directory itself", []string{"src/"}, "src", true, false},
		{"src/ excludes unrelated dir (default)", []string{"src/"}, "lib", true, true},

		// file pattern must not prune parent dir
		{"src/**/*.test.* does not prune src dir itself", []string{"src/**/*.test.*"}, "src", true, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m, err := filtering.NewAntMatcher(tc.patterns)
			if err != nil {
				t.Fatalf("unexpected construction error: %v", err)
			}
			got := m.Excluded(tc.relPath, tc.isDir)
			if got != tc.want {
				t.Errorf("Excluded(%q, isDir=%v) = %v, want %v", tc.relPath, tc.isDir, got, tc.want)
			}
		})
	}
}

// ── Ordered evaluation (last match wins) ──────────────────────────────────────

func TestAntMatcher_OrderedEvaluation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		patterns []string
		relPath  string
		isDir    bool
		want     bool // want Excluded
	}{
		// Include all java, then exclude generated — fixtures re-included by include rule
		{
			name:     "include java then exclude generated: generated file excluded",
			patterns: []string{"**/*.java", "!**/generated/**"},
			relPath:  "src/generated/Auto.java",
			want:     true,
		},
		{
			name:     "include java then exclude generated: normal file included",
			patterns: []string{"**/*.java", "!**/generated/**"},
			relPath:  "src/main/Foo.java",
			want:     false,
		},

		// Exclude all tests, re-include fixtures (using include rule to restore)
		{
			name:     "exclude tests re-include fixtures: fixture file included",
			patterns: []string{"!**/test/**", "**/test/fixtures/**"},
			relPath:  "src/test/fixtures/data.json",
			want:     false,
		},
		{
			name:     "exclude tests re-include fixtures: non-fixture test excluded",
			patterns: []string{"!**/test/**", "**/test/fixtures/**"},
			relPath:  "src/test/Foo.java",
			want:     true,
		},

		// Three rules: include, exclude subset, re-include sub-subset
		{
			name:     "three rules: re-included sub-subset is included",
			patterns: []string{"**/*.java", "!**/test/**", "**/test/fixtures/**"},
			relPath:  "src/test/fixtures/Data.java",
			want:     false,
		},
		{
			name:     "three rules: excluded subset remains excluded",
			patterns: []string{"**/*.java", "!**/test/**", "**/test/fixtures/**"},
			relPath:  "src/test/Foo.java",
			want:     true,
		},
		{
			name:     "three rules: non-java excluded by default",
			patterns: []string{"**/*.java", "!**/test/**", "**/test/fixtures/**"},
			relPath:  "src/main/script.py",
			want:     true,
		},

		// Order matters: reversed gives opposite result
		{
			name:     "include after exclude wins",
			patterns: []string{"!**/test/**", "**/*.java"},
			relPath:  "src/test/Foo.java",
			want:     false, // last rule includes *.java
		},
		{
			name:     "exclude after include wins",
			patterns: []string{"**/*.java", "!**/test/**"},
			relPath:  "src/test/Foo.java",
			want:     true, // last rule excludes test/**
		},

		// Trailing-slash include then specific exclude
		{
			name:     "include src/ but exclude src/test/",
			patterns: []string{"src/", "!src/test/"},
			relPath:  "src/main",
			isDir:    true,
			want:     false, // included by src/
		},
		{
			name:     "include src/ but !src/test/ excludes test dir",
			patterns: []string{"src/", "!src/test/"},
			relPath:  "src/test",
			isDir:    true,
			want:     true, // excluded by !src/test/
		},

		// dir-only exclude does not affect files
		{
			name:     "!src/test/ does not exclude a file named src/test",
			patterns: []string{"src/**", "!src/test/"},
			relPath:  "src/test",
			isDir:    false,
			want:     false, // included by src/**; dir-only exclude does not apply to files
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m, err := filtering.NewAntMatcher(tc.patterns)
			if err != nil {
				t.Fatalf("unexpected construction error: %v", err)
			}
			got := m.Excluded(tc.relPath, tc.isDir)
			if got != tc.want {
				t.Errorf("Excluded(%q, isDir=%v) = %v, want %v", tc.relPath, tc.isDir, got, tc.want)
			}
		})
	}
}

// ── MustDescend ───────────────────────────────────────────────────────────────

func TestAntMatcher_MustDescend(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		patterns     []string
		dirRelPath   string
		wantExcluded bool
		wantDescend  bool
	}{
		{
			// Exclude-only rules. "src/test" is excluded by !**/test/**.
			// A later include rule "src/test/fixtures/**" could match children.
			// The "**/" fast path in couldMatchUnder triggers → MustDescend=true.
			name:         "include rule after exclude rule requires descent",
			patterns:     []string{"!**/test/**", "**/test/fixtures/**"},
			dirRelPath:   "src/test",
			wantExcluded: true,
			wantDescend:  true,
		},
		{
			// Exclude-only, no include rules follow.
			name:         "no include rule after exclude: prune safely",
			patterns:     []string{"!**/test/**"},
			dirRelPath:   "src/test",
			wantExcluded: true,
			wantDescend:  false,
		},
		{
			// Include rules present: "src/test" is excluded by default (no include matched).
			// There IS an include rule "**/*.java" that could match children.
			name:         "include rules present: unmatched dir requires descent for child check",
			patterns:     []string{"**/*.java"},
			dirRelPath:   "src/test",
			wantExcluded: true,
			wantDescend:  true,
		},
		{
			// "src/test" is excluded by !**/test/** (last rule).
			// Because the exclude rule is last, every child of src/test will also
			// be excluded by it (last-match-wins). No include rule follows the
			// last exclude, so MustDescend=false — safe to prune.
			name:         "explicit exclude is last rule: safe to prune despite prior include",
			patterns:     []string{"**/*.java", "!**/test/**"},
			dirRelPath:   "src/test",
			wantExcluded: true,
			wantDescend:  false,
		},
		{
			// Dir not excluded at all.
			name:         "non-excluded dir: MustDescend false",
			patterns:     []string{"!**/test/**"},
			dirRelPath:   "src/main",
			wantExcluded: false,
			wantDescend:  false,
		},
		{
			// No patterns.
			name:         "no patterns: nothing excluded or descended",
			patterns:     nil,
			dirRelPath:   "src/test",
			wantExcluded: false,
			wantDescend:  false,
		},
		{
			// Static-prefix include inside excluded dir.
			name:         "static-prefix include inside excluded dir requires descent",
			patterns:     []string{"!src/**", "src/main/java/**"},
			dirRelPath:   "src",
			wantExcluded: true,
			wantDescend:  true,
		},
		{
			// Include for completely unrelated tree: "foo/**" after excluding "bar/".
			// couldMatchUnder("foo/**", "bar"): staticPrefix="foo/", not compatible → false.
			name:         "include for unrelated tree does not require descent",
			patterns:     []string{"!bar/**", "foo/**"},
			dirRelPath:   "bar",
			wantExcluded: true,
			wantDescend:  false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m, err := filtering.NewAntMatcher(tc.patterns)
			if err != nil {
				t.Fatalf("unexpected construction error: %v", err)
			}

			gotExcluded := m.Excluded(tc.dirRelPath, true)
			if gotExcluded != tc.wantExcluded {
				t.Errorf("Excluded(%q, isDir=true) = %v, want %v", tc.dirRelPath, gotExcluded, tc.wantExcluded)
			}

			gotDescend := m.MustDescend(tc.dirRelPath)
			if gotDescend != tc.wantDescend {
				t.Errorf("MustDescend(%q) = %v, want %v", tc.dirRelPath, gotDescend, tc.wantDescend)
			}
		})
	}
}

// ── Composition ───────────────────────────────────────────────────────────────

func TestNopMatcher(t *testing.T) {
	t.Parallel()
	var n filtering.NopMatcher
	if n.Excluded("anything", false) {
		t.Error("NopMatcher.Excluded should always return false")
	}
	if n.Excluded("anything", true) {
		t.Error("NopMatcher.Excluded should always return false for dirs")
	}
	if n.MustDescend("anything") {
		t.Error("NopMatcher.MustDescend should always return false")
	}
}

func TestMultiMatcher_Excluded_OrSemantics(t *testing.T) {
	t.Parallel()
	// Both matchers are exclude-only (! prefix) so default is included.
	m1, _ := filtering.NewAntMatcher([]string{"!**/*.test.ts"})
	m2, _ := filtering.NewAntMatcher([]string{"!**/node_modules/**"})
	multi := filtering.NewMultiMatcher(m1, m2)

	if !multi.Excluded("src/Foo.test.ts", false) {
		t.Error("m1 should have excluded .test.ts")
	}
	if !multi.Excluded("node_modules/lodash/index.js", false) {
		t.Error("m2 should have excluded node_modules file")
	}
	if multi.Excluded("src/Foo.ts", false) {
		t.Error("neither matcher should exclude src/Foo.ts")
	}
}

func TestMultiMatcher_MustDescend_OrSemantics(t *testing.T) {
	t.Parallel()
	// m1: exclude test/ but has an include rule for fixtures → must descend
	m1, _ := filtering.NewAntMatcher([]string{"!**/test/**", "**/test/fixtures/**"})
	// m2: exclude test/ only → no descent needed
	m2, _ := filtering.NewAntMatcher([]string{"!**/test/**"})
	multi := filtering.NewMultiMatcher(m1, m2)

	if !multi.MustDescend("src/test") {
		t.Error("MultiMatcher should require descent because m1 does")
	}
}

func TestMultiMatcher_Add(t *testing.T) {
	t.Parallel()
	multi := filtering.NewMultiMatcher()
	m, _ := filtering.NewAntMatcher([]string{"!**/*.test.ts"})
	multi.Add(m)

	if !multi.Excluded("src/Foo.test.ts", false) {
		t.Error("expected exclusion after Add")
	}
}

func TestAntMatcher_EmptyPatterns_ExcludesNothing(t *testing.T) {
	t.Parallel()
	m, err := filtering.NewAntMatcher(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Excluded("src/Foo.go", false) {
		t.Error("empty matcher should exclude nothing")
	}
	if m.MustDescend("src/test") {
		t.Error("empty matcher should not require descent")
	}
}
