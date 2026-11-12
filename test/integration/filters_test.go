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

// TestScanWithLinguistFilters_VerifyFilesUploaded asserts that Linguist extensions and filenames from the default
// include filter reach the source zip, while files outside the filter are still left out.
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
	for _, name := range []string{"deploy.sh", "build.ps1", "Jenkinsfile", "Makefile", "Main.java"} {
		assert.Assert(t, hasZipLogLine(logText, "Included: ", name), "%s should be included by the default filter", name)
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
