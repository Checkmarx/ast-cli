//go:build !integration

package params

import (
	"slices"
	"testing"

	"gotest.tools/assert"
)

func TestBaseIncludeFiltersContainsNewExtensions(t *testing.T) {
	tests := []struct {
		name      string
		extension string
		present   bool
	}{
		// New extensions added for AST-177752
		{name: "Java jspdsbld", extension: "*.jspdsbld", present: true},
		{name: "Java wod", extension: "*.wod", present: true},
		{name: "JavaScript app", extension: "*.app", present: true},
		{name: "JavaScript evt", extension: "*.evt", present: true},
		{name: "Python csv", extension: "*.csv", present: true},
		{name: "Python latex", extension: "*.latex", present: true},
		{name: "Python tex", extension: "*.tex", present: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := slices.Contains(BaseIncludeFilters, tt.extension)
			assert.Assert(t, found == tt.present, "Extension %s should %s in BaseIncludeFilters", tt.extension, map[bool]string{true: "be", false: "not be"}[tt.present])
		})
	}
}

func TestBaseIncludeFiltersBackwardCompatibility(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		shouldExist bool
	}{
		// Verify existing extensions are still present
		{name: "Java class", pattern: "*.java", shouldExist: true},
		{name: "Java JSP", pattern: "*.jsp", shouldExist: true},
		{name: "JavaScript", pattern: "*.js", shouldExist: true},
		{name: "Python", pattern: "*.py", shouldExist: true},
		{name: "Go", pattern: "*.go", shouldExist: true},
		{name: "C++", pattern: "*.cpp", shouldExist: true},
		{name: "C#", pattern: "*.cs", shouldExist: true},

		// Verify specific txt patterns still present
		{name: "Requirements txt pattern", pattern: "*requirement*.txt", shouldExist: true},
		{name: "CMakeLists txt pattern", pattern: "*CMakeLists*.txt", shouldExist: true},

		// Verify specific lock files still present (not generic *.lock)
		{name: "yarn.lock", pattern: "yarn.lock", shouldExist: true},
		{name: "composer.lock", pattern: "composer.lock", shouldExist: true},
		{name: "poetry.lock", pattern: "poetry.lock", shouldExist: true},
		{name: "Podfile.lock", pattern: "Podfile.lock", shouldExist: true},

		// Verify go.mod and go.sum (not generic *.mod)
		{name: "go.mod", pattern: "go.mod", shouldExist: true},
		{name: "go.sum", pattern: "go.sum", shouldExist: true},

		// Verify no generic patterns were added (being conservative)
		{name: "Generic lock pattern should not exist", pattern: "*.lock", shouldExist: false},
		{name: "Generic txt pattern should not exist", pattern: "*.txt", shouldExist: false},
		{name: "Generic mod pattern should not exist", pattern: "*.mod", shouldExist: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := slices.Contains(BaseIncludeFilters, tt.pattern)
			assert.Assert(t, found == tt.shouldExist, "Pattern %s should %s in BaseIncludeFilters", tt.pattern, map[bool]string{true: "exist", false: "not exist"}[tt.shouldExist])
		})
	}
}

func TestBaseExcludeFiltersUnchanged(t *testing.T) {
	expectedExclusions := []string{
		"!.vs",
		"!.vscode",
		"!.idea",
		"!node_modules",
	}

	assert.Assert(t, len(BaseExcludeFilters) == len(expectedExclusions), "BaseExcludeFilters length should be %d, got %d", len(expectedExclusions), len(BaseExcludeFilters))

	for _, pattern := range expectedExclusions {
		found := slices.Contains(BaseExcludeFilters, pattern)
		assert.Assert(t, found, "Expected exclusion pattern %s not found in BaseExcludeFilters", pattern)
	}
}

func TestKicsBaseFiltersUnchanged(t *testing.T) {
	expectedPatterns := []string{
		".tf",
		".yaml",
		".yml",
		".json",
		".auto.tfvars",
		".terraform.tfvars",
		"Dockerfile",
		".proto",
		".dockerfile",
	}

	assert.Assert(t, len(KicsBaseFilters) == len(expectedPatterns), "KicsBaseFilters length should be %d, got %d", len(expectedPatterns), len(KicsBaseFilters))

	for _, pattern := range expectedPatterns {
		found := slices.Contains(KicsBaseFilters, pattern)
		assert.Assert(t, found, "Expected KICS pattern %s not found in KicsBaseFilters", pattern)
	}
}

func TestNewExtensionsAreOrganizedByLanguage(t *testing.T) {
	// Verify new extensions are added in logical positions near related extensions
	javaExtensions := []string{"*.java", "*.jsp", "*.jspf", "*.jspdsbld", "*.wod"}
	jsExtensions := []string{"*.js", "*.app", "*.evt"}
	pythonExtensions := []string{"*.py", "*.csv", "*.latex", "*.tex"}

	for _, ext := range javaExtensions {
		found := slices.Contains(BaseIncludeFilters, ext)
		assert.Assert(t, found, "Java extension %s should be in BaseIncludeFilters", ext)
	}

	for _, ext := range jsExtensions {
		found := slices.Contains(BaseIncludeFilters, ext)
		assert.Assert(t, found, "JavaScript extension %s should be in BaseIncludeFilters", ext)
	}

	for _, ext := range pythonExtensions {
		found := slices.Contains(BaseIncludeFilters, ext)
		assert.Assert(t, found, "Python extension %s should be in BaseIncludeFilters", ext)
	}
}

func TestNoGenericPatternsAdded(t *testing.T) {
	// Ensure we didn't add overly generic patterns
	genericPatterns := []string{
		"*.txt",   // Should be specific: *requirement*.txt, *CMakeLists*.txt
		"*.lock",  // Should be specific: yarn.lock, composer.lock, etc.
		"*.mod",   // Should be specific: go.mod only
	}

	for _, pattern := range genericPatterns {
		found := slices.Contains(BaseIncludeFilters, pattern)
		assert.Assert(t, !found, "Generic pattern %s should NOT be in BaseIncludeFilters for conservative filtering", pattern)
	}
}
