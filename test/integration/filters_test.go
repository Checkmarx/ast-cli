//go:build integration

package integration

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

func TestScanWithNewFileExtensions_VerifyIncluded(t *testing.T) {
	projectName := GenerateRandomProjectNameForScan()
	branch := "test-new-extensions"
	testDataDir := filepath.Join(projectDirectory, "data", "filter")

	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.BranchFlag), branch,
		flag(params.SourcesFlag), testDataDir,
		flag(params.ScanTypes), params.SastType,
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Scan with new file extensions should succeed")
}

func TestScanWithExistingExtensions_VerifyBackwardCompatibility(t *testing.T) {
	projectName := GenerateRandomProjectNameForScan()
	branch := "test-existing-extensions"
	testDataDir := filepath.Join(projectDirectory, "data", "filter")

	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.BranchFlag), branch,
		flag(params.SourcesFlag), testDataDir,
		flag(params.ScanTypes), params.SastType,
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Scan with existing file extensions should succeed")
}

func TestScanWithConservativeFiltering_VerifyGenericPatternsExcluded(t *testing.T) {
	projectName := GenerateRandomProjectNameForScan()
	branch := "test-conservative-filtering"
	testDataDir := filepath.Join(projectDirectory, "data", "filter")

	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.BranchFlag), branch,
		flag(params.SourcesFlag), testDataDir,
		flag(params.ScanTypes), params.SastType,
	}

	err, _ := executeCommand(t, args...)
	assert.NilError(t, err, "Scan with conservative filtering should succeed")
}

// TestScanWithLinguistFilters_VerifyFilesUploaded asserts that Linguist extensions and filenames reach the source zip
// only when the tenant or project has LLM-based scanning enabled, that SAST files are always uploaded, and that files
// outside the filter are still left out.
func TestScanWithLinguistFilters_VerifyFilesUploaded(t *testing.T) {
	sourceDir := t.TempDir()
	files := map[string]string{
		"deploy.sh":   "#!/bin/sh\necho deploy\n",
		"build.ps1":   "Write-Host 'build'\n",
		"Jenkinsfile": "pipeline { agent any }\n",
		"Makefile":    "all:\n\techo build\n",
		"Main.java":   "public class Main {}\n",
		"logo.png":    "not a real image",
		"notes.bak":   "backup",
	}
	for name, content := range files {
		assert.NilError(t, os.WriteFile(filepath.Join(sourceDir, name), []byte(content), 0o600))
	}

	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), GenerateRandomProjectNameForScan(),
		flag(params.BranchFlag), "test-linguist-filters",
		flag(params.SourcesFlag), sourceDir,
		flag(params.ScanTypes), params.SastType,
		flag(params.AsyncFlag),
		flag(params.DebugFlag),
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		log.SetOutput(os.Stderr)
	}()

	cmd := createASTIntegrationTestCommand(t)
	err := execute(cmd, args...)
	assert.NilError(t, err, "Scan with Linguist files should succeed")

	logText := buf.String()
	assert.Assert(t, hasZipLogLine(logText, "Included: ", "Main.java"), "SAST files should always be included by the default filter")

	llmEnabled := strings.Contains(logText, "LLM-based scanning is enabled: adding Linguist patterns")
	linguistPrefix := "Excluded: "
	if llmEnabled {
		linguistPrefix = "Included: "
	}
	for _, name := range []string{"deploy.sh", "build.ps1", "Jenkinsfile", "Makefile"} {
		assert.Assert(t, hasZipLogLine(logText, linguistPrefix, name), "%s should be %s (LLM-based scanning enabled: %v)", name, linguistPrefix, llmEnabled)
	}
	for _, name := range []string{"logo.png", "notes.bak"} {
		assert.Assert(t, hasZipLogLine(logText, "Excluded: ", name), "%s should be excluded by the default filter", name)
	}
}

// hasZipLogLine reports whether the zip walk logged prefix for a file whose path ends with name.
func hasZipLogLine(logText, prefix, name string) bool {
	for _, line := range strings.Split(logText, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, prefix) && (strings.HasSuffix(line, "/"+name) || strings.HasSuffix(line, `\`+name)) {
			return true
		}
	}
	return false
}

// oversizedFileBytes is just over the 100MB default filter size limit; files are created sparse so they take no disk space.
const oversizedFileBytes = 101 * 1024 * 1024

// TestScanWithFileFilter_VerifyUserExclusionApplied asserts that a --file-filter exclusion removes files the default
// filter would otherwise include, and leaves the other default files in the zip.
func TestScanWithFileFilter_VerifyUserExclusionApplied(t *testing.T) {
	sourceDir := writeFilterTestFiles(t, map[string]string{
		"Main.java": "public class Main {}\n",
		"app.py":    "print('app')\n",
	})

	logText := runFilterScanWithLogs(t, sourceDir, "test-file-filter", flag(params.SourceDirFilterFlag), "!*.java")

	assert.Assert(t, hasZipLogLine(logText, "Excluded: ", "Main.java"), "Main.java should be excluded by --file-filter")
	assert.Assert(t, hasZipLogLine(logText, "Included: ", "app.py"), "app.py should still be included by the default filter")
}

// TestScanWithFileInclude_VerifyExtraExtensionIncluded asserts that --file-include adds an extension the default
// filter leaves out, without dropping the default files.
func TestScanWithFileInclude_VerifyExtraExtensionIncluded(t *testing.T) {
	sourceDir := writeFilterTestFiles(t, map[string]string{
		"Main.java": "public class Main {}\n",
		"notes.bak": "backup",
		"logo.png":  "not a real image",
	})

	logText := runFilterScanWithLogs(t, sourceDir, "test-file-include", flag(params.IncludeFilterFlag), "*.bak")

	assert.Assert(t, hasZipLogLine(logText, "Included: ", "notes.bak"), "notes.bak should be included by --file-include")
	assert.Assert(t, hasZipLogLine(logText, "Included: ", "Main.java"), "Main.java should still be included by the default filter")
	assert.Assert(t, hasZipLogLine(logText, "Excluded: ", "logo.png"), "logo.png should still be excluded by the default filter")
}

// TestScanWithSkipDefaultFilter_VerifyAllFilesIncluded asserts that --skip-default-filter uploads files the default
// filter would leave out.
func TestScanWithSkipDefaultFilter_VerifyAllFilesIncluded(t *testing.T) {
	sourceDir := writeFilterTestFiles(t, map[string]string{
		"Main.java": "public class Main {}\n",
		"notes.bak": "backup",
		"logo.png":  "not a real image",
	})

	logText := runFilterScanWithLogs(t, sourceDir, "test-skip-default-filter", flag(params.SkipDefaultFilterFlag))

	for _, name := range []string{"Main.java", "notes.bak", "logo.png"} {
		assert.Assert(t, hasZipLogLine(logText, "Included: ", name), "%s should be included with --skip-default-filter", name)
	}
	assert.Assert(t, !strings.Contains(logText, "LLM-based scanning is"), "tenant settings should not be read with --skip-default-filter")
}

// TestScanWithOversizedFiles_VerifySizeLimitApplied asserts that a default-filter file over 100MB is left out of the
// zip, while an SCA lock file of the same size is exempt and a small file is still uploaded.
func TestScanWithOversizedFiles_VerifySizeLimitApplied(t *testing.T) {
	sourceDir := writeFilterTestFiles(t, map[string]string{
		"Main.java": "public class Main {}\n",
	})
	writeSparseFile(t, filepath.Join(sourceDir, "Huge.java"), oversizedFileBytes)
	writeSparseFile(t, filepath.Join(sourceDir, "package-lock.json"), oversizedFileBytes)

	logText := runFilterScanWithLogs(t, sourceDir, "test-size-limit")

	assert.Assert(t, hasSizeLimitLogLine(logText, "Huge.java"), "Huge.java should be excluded by the size limit")
	assert.Assert(t, !hasZipLogLine(logText, "Included: ", "Huge.java"), "Huge.java should not reach the zip")
	assert.Assert(t, hasZipLogLine(logText, "Included: ", "package-lock.json"), "SCA lock files should be exempt from the size limit")
	assert.Assert(t, hasZipLogLine(logText, "Included: ", "Main.java"), "files under the size limit should be included")
}

// TestScanWithOnlyOversizedFiles_VerifyScanFailsAndLogsExclusion asserts that when every matching file is over the
// size limit the folder is treated as empty, and the excluded file is still reported.
func TestScanWithOnlyOversizedFiles_VerifyScanFailsAndLogsExclusion(t *testing.T) {
	sourceDir := t.TempDir()
	writeSparseFile(t, filepath.Join(sourceDir, "Huge.java"), oversizedFileBytes)

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	cmd := createASTIntegrationTestCommand(t)
	err := execute(cmd, filterScanArgs(sourceDir, "test-size-limit-only")...)

	assert.Assert(t, err != nil, "scan should fail when every file is over the size limit")
	assert.Assert(t, hasSizeLimitLogLine(buf.String(), "Huge.java"), "the oversized file should be reported as excluded")
}

// runFilterScanWithLogs runs an async SAST scan of sourceDir in debug mode and returns the captured log output.
func runFilterScanWithLogs(t *testing.T, sourceDir, branch string, extraArgs ...string) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	cmd := createASTIntegrationTestCommand(t)
	err := execute(cmd, append(filterScanArgs(sourceDir, branch), extraArgs...)...)
	assert.NilError(t, err, "Scan should succeed")
	return buf.String()
}

func filterScanArgs(sourceDir, branch string) []string {
	return []string{
		scanCommand, "create",
		flag(params.ProjectName), GenerateRandomProjectNameForScan(),
		flag(params.BranchFlag), branch,
		flag(params.SourcesFlag), sourceDir,
		flag(params.ScanTypes), params.SastType,
		flag(params.AsyncFlag),
		flag(params.DebugFlag),
	}
}

// writeFilterTestFiles writes files into a new temporary directory and returns its path.
func writeFilterTestFiles(t *testing.T, files map[string]string) string {
	dir := t.TempDir()
	for name, content := range files {
		assert.NilError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}
	return dir
}

// writeSparseFile creates a file of the given size without writing its contents.
func writeSparseFile(t *testing.T, path string, size int64) {
	f, err := os.Create(path)
	assert.NilError(t, err)
	defer f.Close()
	assert.NilError(t, f.Truncate(size))
}

// hasSizeLimitLogLine reports whether the zip walk logged a size limit exclusion for a file whose path ends with name.
func hasSizeLimitLogLine(logText, name string) bool {
	for _, line := range strings.Split(logText, "\n") {
		if strings.Contains(line, "Excluded (larger than") &&
			(strings.Contains(line, "/"+name+" (") || strings.Contains(line, `\`+name+" (")) {
			return true
		}
	}
	return false
}
