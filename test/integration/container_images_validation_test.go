//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

// TestContainerImageValidation_ValidFormats tests that valid container image formats are accepted
func TestContainerImageValidation_ValidFormats(t *testing.T) {
	tests := []struct {
		name        string
		imageFormat string
		description string
	}{
		{
			name:        "SimpleImageWithTag",
			imageFormat: "nginx:alpine",
			description: "Simple image with tag should be valid",
		},
		{
			name:        "ImageWithRegistryAndTag",
			imageFormat: "docker.io/nginx:1.21",
			description: "Image with registry and tag should be valid",
		},
		{
			name:        "ImageWithPortAndTag",
			imageFormat: "localhost:5000/myapp:latest",
			description: "Image with port and tag should be valid",
		},
		{
			name:        "ImageWithNamespaceAndTag",
			imageFormat: "checkmarx/kics:v2.1.11",
			description: "Image with namespace and tag should be valid",
		},
		{
			name:        "FullyQualifiedImageWithTag",
			imageFormat: "registry.example.com:8080/namespace/myapp:v1.0.0",
			description: "Fully qualified image with tag should be valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createASTIntegrationTestCommand(t)
			testArgs := []string{
				"scan", "create",
				flag(params.ProjectName), getProjectNameForScanTests(),
				flag(params.SourcesFlag), "data/insecure.zip",
				flag(params.ContainerImagesFlag), tt.imageFormat,
				flag(params.BranchFlag), "dummy_branch",
				flag(params.ScanTypes), params.ContainersTypeFlag,
				flag(params.ScanInfoFormatFlag), printer.FormatJSON,
			}
			scanID, projectID := executeCreateScan(t, testArgs)
			assert.Assert(t, scanID != "", "Scan ID should not be empty for: "+tt.description)
			assert.Assert(t, projectID != "", "Project ID should not be empty for: "+tt.description)
		})
	}
}

// TestContainerImageValidation_InvalidFormats tests that invalid container image formats are rejected
func TestContainerImageValidation_InvalidFormats(t *testing.T) {
	tests := []struct {
		name          string
		imageFormat   string
		description   string
		expectedError string
	}{
		{
			name:          "ImageWithoutTag",
			imageFormat:   "nginx",
			description:   "Image without tag should be invalid",
			expectedError: "image does not have a tag",
		},
		{
			name:          "ImageWithEmptyTag",
			imageFormat:   "nginx:",
			description:   "Image with empty tag should be invalid",
			expectedError: "Image name and tag cannot be empty",
		},
		{
			name:          "ImageWithEmptyName",
			imageFormat:   ":alpine",
			description:   "Image with empty name should be invalid",
			expectedError: "Image name and tag cannot be empty",
		},
		{
			name:          "OnlyColon",
			imageFormat:   ":",
			description:   "Only colon should be invalid",
			expectedError: "Image name and tag cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createASTIntegrationTestCommand(t)
			testArgs := []string{
				"scan", "create",
				flag(params.ProjectName), getProjectNameForScanTests(),
				flag(params.SourcesFlag), "data/insecure.zip",
				flag(params.ContainerImagesFlag), tt.imageFormat,
				flag(params.BranchFlag), "dummy_branch",
				flag(params.ScanTypes), params.ContainersTypeFlag,
				flag(params.ScanInfoFormatFlag), printer.FormatJSON,
			}
			err, _ := executeCommand(t, testArgs...)
			assert.Assert(t, err != nil, "Expected error for: "+tt.description)
			if tt.expectedError != "" {
				assertError(t, err, tt.expectedError)
			}
		})
	}
}

// TestContainerImageValidation_MultipleImagesValidation tests validation with multiple container images
func TestContainerImageValidation_MultipleImagesValidation(t *testing.T) {
	tests := []struct {
		name          string
		imageList     string
		shouldSucceed bool
		description   string
	}{
		{
			name:          "MultipleValidImages",
			imageList:     "nginx:alpine,mysql:5.7,debian:9",
			shouldSucceed: true,
			description:   "Multiple valid images should pass",
		},
		{
			name:          "OneInvalidAmongValid",
			imageList:     "nginx:alpine,invalid-image,mysql:5.7",
			shouldSucceed: false,
			description:   "One invalid image among valid ones should fail",
		},
		{
			name:          "LastImageInvalid",
			imageList:     "nginx:alpine,mysql:5.7,invalid:",
			shouldSucceed: false,
			description:   "Last image being invalid should fail",
		},
		{
			name:          "FirstImageInvalid",
			imageList:     ":invalid,nginx:alpine,mysql:5.7",
			shouldSucceed: false,
			description:   "First image being invalid should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createASTIntegrationTestCommand(t)
			testArgs := []string{
				"scan", "create",
				flag(params.ProjectName), getProjectNameForScanTests(),
				flag(params.SourcesFlag), "data/insecure.zip",
				flag(params.ContainerImagesFlag), tt.imageList,
				flag(params.BranchFlag), "dummy_branch",
				flag(params.ScanTypes), params.ContainersTypeFlag,
				flag(params.ScanInfoFormatFlag), printer.FormatJSON,
			}

			if tt.shouldSucceed {
				scanID, projectID := executeCreateScan(t, testArgs)
				assert.Assert(t, scanID != "", "Scan ID should not be empty for: "+tt.description)
				assert.Assert(t, projectID != "", "Project ID should not be empty for: "+tt.description)
			} else {
				err, _ := executeCommand(t, testArgs...)
				assert.Assert(t, err != nil, "Expected error for: "+tt.description)
			}
		})
	}
}

// TestContainerImageValidation_TarFiles tests validation of .tar file references
func TestContainerImageValidation_TarFiles(t *testing.T) {
	// Create a temporary .tar file for testing
	tempDir := t.TempDir()
	validTarFile := filepath.Join(tempDir, "test-image.tar")

	// Create an empty .tar file for testing
	f, err := os.Create(validTarFile)
	assert.NilError(t, err, "Should create temp .tar file")
	f.Close()

	tests := []struct {
		name          string
		tarFile       string
		shouldSucceed bool
		description   string
	}{
		{
			name:          "ValidTarFile",
			tarFile:       validTarFile,
			shouldSucceed: true,
			description:   "Valid .tar file path should be accepted",
		},
		{
			name:          "NonExistentTarFile",
			tarFile:       "/tmp/nonexistent-image.tar",
			shouldSucceed: false,
			description:   "Non-existent .tar file should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createASTIntegrationTestCommand(t)
			testArgs := []string{
				"scan", "create",
				flag(params.ProjectName), getProjectNameForScanTests(),
				flag(params.SourcesFlag), "data/insecure.zip",
				flag(params.ContainerImagesFlag), tt.tarFile,
				flag(params.BranchFlag), "dummy_branch",
				flag(params.ScanTypes), params.ContainersTypeFlag,
				flag(params.ScanInfoFormatFlag), printer.FormatJSON,
			}

			if tt.shouldSucceed {
				scanID, projectID := executeCreateScan(t, testArgs)
				assert.Assert(t, scanID != "", "Scan ID should not be empty for: "+tt.description)
				assert.Assert(t, projectID != "", "Project ID should not be empty for: "+tt.description)
			} else {
				err, _ := executeCommand(t, testArgs...)
				assert.Assert(t, err != nil, "Expected error for: "+tt.description)
			}
		})
	}
}

// TestContainerImageValidation_MixedTarAndRegularImages tests mixing .tar files with regular images
func TestContainerImageValidation_MixedTarAndRegularImages(t *testing.T) {
	// Create a temporary .tar file for testing
	tempDir := t.TempDir()
	validTarFile := filepath.Join(tempDir, "test-image.tar")

	f, err := os.Create(validTarFile)
	assert.NilError(t, err, "Should create temp .tar file")
	f.Close()

	t.Run("ValidTarAndRegularImage", func(t *testing.T) {
		createASTIntegrationTestCommand(t)
		imageList := fmt.Sprintf("nginx:alpine,%s", validTarFile)
		testArgs := []string{
			"scan", "create",
			flag(params.ProjectName), getProjectNameForScanTests(),
			flag(params.SourcesFlag), "data/insecure.zip",
			flag(params.ContainerImagesFlag), imageList,
			flag(params.BranchFlag), "dummy_branch",
			flag(params.ScanTypes), params.ContainersTypeFlag,
			flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		}
		scanID, projectID := executeCreateScan(t, testArgs)
		assert.Assert(t, scanID != "", "Scan ID should not be empty for mixed valid images")
		assert.Assert(t, projectID != "", "Project ID should not be empty for mixed valid images")
	})

	t.Run("ValidTarAndInvalidRegularImage", func(t *testing.T) {
		createASTIntegrationTestCommand(t)
		imageList := fmt.Sprintf("nginx:,%s", validTarFile)
		testArgs := []string{
			"scan", "create",
			flag(params.ProjectName), getProjectNameForScanTests(),
			flag(params.SourcesFlag), "data/insecure.zip",
			flag(params.ContainerImagesFlag), imageList,
			flag(params.BranchFlag), "dummy_branch",
			flag(params.ScanTypes), params.ContainersTypeFlag,
			flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		}
		err, _ := executeCommand(t, testArgs...)
		assert.Assert(t, err != nil, "Expected error for mixed images with one invalid")
	})
}
