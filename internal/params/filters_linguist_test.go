package params

import (
	"slices"
	"strings"
	"testing"

	"gotest.tools/assert"
)

// unsafeFilterChars are glob, negation and path characters that must not appear in a Linguist filter.
const unsafeFilterChars = "*?[" + `\` + "!/ "

func TestLinguistExtensionFiltersFormat(t *testing.T) {
	assert.Assert(t, len(linguistExtensionFilters) > 0)
	for _, f := range linguistExtensionFilters {
		assert.Assert(t, strings.HasPrefix(f, "*."), "extension filter %q must start with \"*.\"", f)
		assert.Assert(t, len(f) > len("*."), "extension filter %q has no extension", f)
		assert.Equal(t, strings.ToLower(f), f, "extension filter %q must be lower-case", f)
		assert.Assert(t, !strings.ContainsAny(f[1:], unsafeFilterChars), "extension filter %q has unexpected glob or path characters", f)
	}
}

func TestLinguistFilenameFiltersFormat(t *testing.T) {
	assert.Assert(t, len(linguistFilenameFilters) > 0)
	for _, f := range linguistFilenameFilters {
		assert.Assert(t, f != "", "filename filters must not contain empty entries")
		assert.Equal(t, strings.ToLower(f), f, "filename filter %q must be lower-case", f)
		assert.Assert(t, !strings.ContainsAny(f, unsafeFilterChars), "filename filter %q must be a plain file name", f)
	}
}

func TestLinguistFiltersHaveNoDuplicates(t *testing.T) {
	for name, list := range map[string][]string{
		"linguistExtensionFilters": linguistExtensionFilters,
		"linguistFilenameFilters":  linguistFilenameFilters,
	} {
		seen := map[string]struct{}{}
		for _, f := range list {
			_, dup := seen[f]
			assert.Assert(t, !dup, "%s contains duplicate %q", name, f)
			seen[f] = struct{}{}
		}
	}
}

func TestLinguistFiltersDoNotOverlapSastFilters(t *testing.T) {
	sast := map[string]struct{}{}
	for _, f := range sastIncludeFilters {
		sast[strings.ToLower(f)] = struct{}{}
	}
	for _, list := range [][]string{linguistExtensionFilters, linguistFilenameFilters} {
		for _, f := range list {
			_, overlap := sast[f]
			assert.Assert(t, !overlap, "%q is already in sastIncludeFilters", f)
		}
	}
}

func TestBaseIncludeFiltersCombinesAllLists(t *testing.T) {
	want := len(sastIncludeFilters) + len(linguistExtensionFilters) + len(linguistFilenameFilters)
	assert.Equal(t, want, len(BaseIncludeFilters))
	assert.Equal(t, want, cap(BaseIncludeFilters), "BaseIncludeFilters should be clipped so appends never alias")
	assert.Assert(t, slices.Equal(sastIncludeFilters, BaseIncludeFilters[:len(sastIncludeFilters)]), "SAST filters must stay first")
	for _, list := range [][]string{sastIncludeFilters, linguistExtensionFilters, linguistFilenameFilters} {
		for _, f := range list {
			assert.Assert(t, slices.Contains(BaseIncludeFilters, f), "%q missing from BaseIncludeFilters", f)
		}
	}
}

func TestBaseIncludeFiltersAppendDoesNotAlias(t *testing.T) {
	a := append(BaseIncludeFilters, "*.first")
	b := append(BaseIncludeFilters, "*.second")
	assert.Equal(t, "*.first", a[len(a)-1])
	assert.Equal(t, "*.second", b[len(b)-1])
}

func TestLinguistFiltersSamples(t *testing.T) {
	for _, f := range []string{"*.sh", "*.ps1", "*.bat", "*.ipynb", "*.j2", "jenkinsfile", "makefile", "containerfile", "gnumakefile", ".bash_history"} {
		assert.Assert(t, slices.Contains(BaseIncludeFilters, f), "%q should be in BaseIncludeFilters", f)
	}
}
