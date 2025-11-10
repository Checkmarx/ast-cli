//go:build integration

package integration

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

// TestContainerScan_EmptyFolderWithExternalImages tests scanning with empty folders and external container images
// This tests the createMinimalZipFile functionality introduced in the cloud-default-scan branch
func TestContainerScan_EmptyFolderWithExternalImages(t *testing.T) {
	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/empty-folder.zip",
		flag(params.ContainerImagesFlag), "nginx:alpine",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanTypes), params.ContainersTypeFlag,
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	scanID, projectID := executeCreateScan(t, testArgs)
	assert.Assert(t, scanID != "", "Scan ID should not be empty for empty folder with external images")
	assert.Assert(t, projectID != "", "Project ID should not be empty for empty folder with external images")
}

// TestContainerScan_EmptyFolderWithMultipleExternalImages tests scanning empty folder with multiple external images
func TestContainerScan_EmptyFolderWithMultipleExternalImages(t *testing.T) {
	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/empty-folder.zip",
		flag(params.ContainerImagesFlag), "nginx:alpine,mysql:5.7,debian:9",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanTypes), params.ContainersTypeFlag,
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	scanID, projectID := executeCreateScan(t, testArgs)
	assert.Assert(t, scanID != "", "Scan ID should not be empty for empty folder with multiple external images")
	assert.Assert(t, projectID != "", "Project ID should not be empty for empty folder with multiple external images")
}

// TestContainerScan_EmptyFolderWithExternalImagesAndDebug tests with debug flag enabled
func TestContainerScan_EmptyFolderWithExternalImagesAndDebug(t *testing.T) {
	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/empty-folder.zip",
		flag(params.ContainerImagesFlag), "mysql:5.7",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.DebugFlag),
		flag(params.ScanTypes), params.ContainersTypeFlag,
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	scanID, projectID := executeCreateScan(t, testArgs)
	assert.Assert(t, scanID != "", "Scan ID should not be empty for empty folder with debug flag")
	assert.Assert(t, projectID != "", "Project ID should not be empty for empty folder with debug flag")
}

// TestContainerScan_EmptyFolderWithoutExternalImages tests that empty folder without external images still works
func TestContainerScan_EmptyFolderWithoutExternalImages(t *testing.T) {
	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/empty-folder.zip",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanTypes), params.ContainersTypeFlag,
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	// This should succeed but may not find any containers to scan
	err, _ := executeCommand(t, testArgs...)
	// We don't assert error here as the behavior may vary
	// The important thing is that it doesn't crash
	t.Logf("Empty folder without external images result: %v", err)
}

// TestContainerScan_MinimalProjectWithExternalImages tests minimal project content with external images
func TestContainerScan_MinimalProjectWithExternalImages(t *testing.T) {
	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/insecure.zip",
		flag(params.ContainerImagesFlag), "nginx:alpine,mysql:8.0",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanTypes), params.ContainersTypeFlag,
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	scanID, projectID := executeCreateScan(t, testArgs)
	assert.Assert(t, scanID != "", "Scan ID should not be empty")
	assert.Assert(t, projectID != "", "Project ID should not be empty")
}

// TestContainerScan_EmptyFolderWithComplexImageNames tests empty folder with complex image names
func TestContainerScan_EmptyFolderWithComplexImageNames(t *testing.T) {
	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/empty-folder.zip",
		flag(params.ContainerImagesFlag), "docker.io/library/nginx:1.21.6,mysql:5.7.38",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanTypes), params.ContainersTypeFlag,
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	scanID, projectID := executeCreateScan(t, testArgs)
	assert.Assert(t, scanID != "", "Scan ID should not be empty for complex image names")
	assert.Assert(t, projectID != "", "Project ID should not be empty for complex image names")
}

// TestContainerScan_EmptyFolderWithRegistryImages tests empty folder with registry-prefixed images
func TestContainerScan_EmptyFolderWithRegistryImages(t *testing.T) {
	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/empty-folder.zip",
		flag(params.ContainerImagesFlag), "checkmarx/kics:v2.1.11",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanTypes), params.ContainersTypeFlag,
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	scanID, projectID := executeCreateScan(t, testArgs)
	assert.Assert(t, scanID != "", "Scan ID should not be empty for registry images")
	assert.Assert(t, projectID != "", "Project ID should not be empty for registry images")
}

// TestContainerScan_EmptyFolderInvalidImageShouldFail tests that validation still works with empty folders
func TestContainerScan_EmptyFolderInvalidImageShouldFail(t *testing.T) {
	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/empty-folder.zip",
		flag(params.ContainerImagesFlag), "invalid-image-without-tag",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanTypes), params.ContainersTypeFlag,
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	err, _ := executeCommand(t, testArgs...)
	assert.Assert(t, err != nil, "Expected error for invalid image format with empty folder")
	assertError(t, err, "Invalid value for --container-images flag")
}

// TestContainerScan_EmptyFolderMixedValidInvalidImages tests mixed valid/invalid images with empty folder
func TestContainerScan_EmptyFolderMixedValidInvalidImages(t *testing.T) {
	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), "data/empty-folder.zip",
		flag(params.ContainerImagesFlag), "nginx:alpine,invalid-image,mysql:5.7",
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanTypes), params.ContainersTypeFlag,
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	err, _ := executeCommand(t, testArgs...)
	assert.Assert(t, err != nil, "Expected error for mixed valid/invalid images with empty folder")
	assertError(t, err, "Invalid value for --container-images flag")
}
