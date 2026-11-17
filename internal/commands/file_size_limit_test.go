//go:build !integration

package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/filtering"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
	"gotest.tools/assert"
)

const testSizeLimitBytes = 64

// withSizeLimit lowers the default filter size limit for the duration of the test.
func withSizeLimit(t *testing.T, limit int64) {
	t.Helper()
	previous := MaxDefaultFilterFileSizeBytes
	MaxDefaultFilterFileSizeBytes = limit
	t.Cleanup(func() { MaxDefaultFilterFileSizeBytes = previous })
}

// writeSizedFiles creates each file with the given number of bytes and returns the root with a trailing separator.
func writeSizedFiles(t *testing.T, files map[string]int) string {
	t.Helper()
	root := t.TempDir()
	for rel, size := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		assert.NilError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		assert.NilError(t, os.WriteFile(p, []byte(strings.Repeat("a", size)), 0o600))
	}
	return root + string(os.PathSeparator)
}

func compressWith(t *testing.T, root, userInclude string, skipDefaultFilter bool) []string {
	t.Helper()
	zipPath, err := compressFolder(root, "", userInclude, "", filtering.NopMatcher{}, skipDefaultFilter, false, false,
		commonParams.DefaultIncludeFilters(true))
	assert.NilError(t, err)
	t.Cleanup(func() { _ = os.Remove(zipPath) })
	data, err := os.ReadFile(zipPath)
	assert.NilError(t, err)
	return zipEntryNames(t, data)
}

func TestIsScaFile(t *testing.T) {
	scaFiles := []string{
		"pom.xml", "package.json", "package-lock.json", "npm-shrinkwrap.json", "yarn.lock", "pnpm-lock.yaml",
		"go.mod", "go.sum", "build.gradle", "build.gradle.kts", "libs.versions.toml", "build.sbt",
		"requirements.txt", "requirements-dev.txt", "dev-requirements.txt", "constraints.txt", "packages.txt",
		"setup.py", "setup.cfg", "pyproject.toml", "poetry.lock", "App.csproj", "APP.CSPROJ", "packages.config",
		"packages.lock.json", "Directory.Packages.props", "Directory.Build.props", "Gemfile", "Gemfile.lock",
		"composer.json", "composer.lock", "Podfile", "Podfile.lock", "Lib.podspec", "Lib.podspec.json", "Cartfile", "Cartfile.private",
		"Package.swift", "Package@swift-5.9.swift", "pubspec.yaml", "bower.json", "lib.jar", ".cxsca-results.json", "scan.cxsca.sig",
	}
	for _, name := range scaFiles {
		assert.Assert(t, isScaFile(name), "expected an SCA file: %s", name)
	}

	otherFiles := []string{"Main.java", "app.js", "data.json", "config.yaml", "notes.txt", "Main.swift", "lib.dll", "deploy.sh", "app.war", "Pipfile"}
	for _, name := range otherFiles {
		assert.Assert(t, !isScaFile(name), "expected a non-SCA file: %s", name)
	}
}

// Every manifest the OSS realtime scanner accepts must be treated as an SCA file.
func TestScaFilePatterns_CoverOssRealtimeManifests(t *testing.T) {
	manifests := []string{
		"pom.xml", "package.json", "bower.json", "Directory.Packages.props", "packages.config", "App.csproj", "go.mod",
		"build.gradle", "build.gradle.kts", "libs.versions.toml", "build.sbt", "setup.cfg", "setup.py", "pyproject.toml",
		"requirements.txt", "packages.txt", "constraints.txt", "Podfile", "Lib.podspec", "Lib.podspec.json", "Cartfile",
		"Cartfile.private", "Gemfile", "composer.json", "pubspec.yaml", "Package.swift", "Package@swift-5.9.swift",
	}
	for _, name := range manifests {
		assert.Assert(t, ossrealtime.IsSupportedManifestFile(name), "not an OSS realtime manifest: %s", name)
		assert.Assert(t, isScaFile(name), "OSS realtime manifest not treated as an SCA file: %s", name)
	}
}

// SCA patterns must come from existing code: the OSS realtime manifest list or the default include filter.
func TestScaFilePatterns_ComeFromExistingCode(t *testing.T) {
	for _, pattern := range commonParams.ScaFilePatterns {
		name := strings.ReplaceAll(pattern, "*", "x")
		assert.Assert(t, ossrealtime.IsSupportedManifestFile(name) || filterMatched(commonParams.BaseIncludeFilters, name),
			"SCA pattern is neither an OSS realtime manifest nor in the default include filter: %s", pattern)
	}
}

func TestExceedsDefaultFilterSizeLimit(t *testing.T) {
	withSizeLimit(t, testSizeLimitBytes)
	filters := commonParams.DefaultIncludeFilters(true)
	big := int64(testSizeLimitBytes + 1)

	assert.Assert(t, exceedsDefaultFilterSizeLimit(filters, "Main.java", big))
	assert.Assert(t, !exceedsDefaultFilterSizeLimit(filters, "Main.java", testSizeLimitBytes), "a file at the limit is kept")
	assert.Assert(t, !exceedsDefaultFilterSizeLimit(filters, "pom.xml", big), "SCA files are exempt")
	assert.Assert(t, !exceedsDefaultFilterSizeLimit(filters, "custom.xyz", big), "files outside the default filter are exempt")
	assert.Assert(t, !exceedsDefaultFilterSizeLimit(nil, "Main.java", big), "no limit without the default filter")
}

func TestCompressFolder_SkipsOversizedFilesExceptSca(t *testing.T) {
	withSizeLimit(t, testSizeLimitBytes)
	big := testSizeLimitBytes + 1
	root := writeSizedFiles(t, map[string]int{
		"src/Main.java":         big,
		"src/App.java":          testSizeLimitBytes,
		"deploy.sh":             big,
		"pom.xml":               big,
		"package-lock.json":     big,
		"requirements-dev.txt":  big,
		"libs/dependency.jar":   big,
		"web/yarn.lock":         big,
		"custom.xyz":            big,
		"go.sum":                big,
		"ios/Package.resolved":  big,
		"dotnet/App.csproj":     big,
		"dotnet/data.json":      big,
		"dotnet/appsettings.cs": 1,
	})

	assert.DeepEqual(t, compressWith(t, root, "", false), []string{
		"dotnet/App.csproj", "dotnet/appsettings.cs", "go.sum", "libs/dependency.jar",
		"package-lock.json", "pom.xml", "requirements-dev.txt", "src/App.java", "web/yarn.lock",
	})

	// Files added only by --file-include are not limited.
	entries := compressWith(t, root, "*.xyz", false)
	assert.Assert(t, contains(entries, "custom.xyz"))
	assert.Assert(t, !contains(entries, "src/Main.java"))

	// --skip-default-filter turns the limit off.
	entries = compressWith(t, root, "", true)
	assert.Assert(t, contains(entries, "src/Main.java"))
	assert.Assert(t, contains(entries, "dotnet/data.json"))
}

func TestCompressFolder_AllFilesOversizedAddsPlaceholder(t *testing.T) {
	withSizeLimit(t, testSizeLimitBytes)
	root := writeSizedFiles(t, map[string]int{"Main.java": testSizeLimitBytes + 1})

	assert.DeepEqual(t, compressWith(t, root, "", false), []string{".container-scan"})
}
