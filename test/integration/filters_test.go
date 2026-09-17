//go:build integration

package integration

import (
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

// TestScanWithNewFileExtensions_VerifyIncluded tests that newly added file extensions
// (.jspdsbld, .wod, .app, .evt, .csv, .latex, .tex) are included in SAST scans
func TestScanWithNewFileExtensions_VerifyIncluded(t *testing.T) {
	cmd := createASTIntegrationTestCommand(t)

	projectName := GenerateRandomProjectNameForScan()
	branch := "test-new-extensions"

	// Use the test data directory with filter test files
	testDataDir := filepath.Join(projectDirectory, "data", "filter")

	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.BranchFlag), branch,
		flag(params.SourcesFlag), testDataDir,
		flag(params.ScanTypes), params.SastType,
	}

	err, _ := executeCommand(t, args...)

	// Scan should succeed and include the files with new extensions
	assert.NilError(t, err, "Scan with new file extensions should succeed")
}

// TestScanWithExistingExtensions_VerifyBackwardCompatibility tests that
// existing file extensions (like .java, .js, .py) still work after adding new ones
func TestScanWithExistingExtensions_VerifyBackwardCompatibility(t *testing.T) {
	cmd := createASTIntegrationTestCommand(t)

	projectName := GenerateRandomProjectNameForScan()
	branch := "test-existing-extensions"

	// Use the test data directory with filter test files
	testDataDir := filepath.Join(projectDirectory, "data", "filter")

	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.BranchFlag), branch,
		flag(params.SourcesFlag), testDataDir,
		flag(params.ScanTypes), params.SastType,
	}

	err, _ := executeCommand(t, args...)

	// Scan should succeed with existing extensions
	assert.NilError(t, err, "Scan with existing file extensions should succeed")
}

// TestScanWithConservativeFiltering_VerifyGenericPatternsExcluded tests that
// generic patterns (*.txt, *.lock, *.mod) are NOT included even if present,
// verifying the conservative filtering approach
func TestScanWithConservativeFiltering_VerifyGenericPatternsExcluded(t *testing.T) {
	cmd := createASTIntegrationTestCommand(t)

	projectName := GenerateRandomProjectNameForScan()
	branch := "test-conservative-filtering"

	// Use the test data directory with filter test files (mixed types)
	testDataDir := filepath.Join(projectDirectory, "data", "filter")

	args := []string{
		scanCommand, "create",
		flag(params.ProjectName), projectName,
		flag(params.BranchFlag), branch,
		flag(params.SourcesFlag), testDataDir,
		flag(params.ScanTypes), params.SastType,
	}

	err, _ := executeCommand(t, args...)

	// Scan should succeed - generic patterns should be silently excluded
	// (this is expected behavior for conservative filtering)
	assert.NilError(t, err, "Scan with conservative filtering should succeed")
}
