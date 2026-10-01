//go:build integration

package integration

import (
	"path/filepath"
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
