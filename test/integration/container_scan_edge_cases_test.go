//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"gotest.tools/assert"
)

// TestContainerScan_ErrorHandling_InvalidAndValidScenarios tests error handling improvements
func TestContainerScan_ErrorHandling_InvalidAndValidScenarios(t *testing.T) {
	t.Run("ValidScanWithAllParameters", func(t *testing.T) {
		createASTIntegrationTestCommand(t)
		testArgs := []string{
			"scan", "create",
			flag(params.ProjectName), getProjectNameForScanTests(),
			flag(params.SourcesFlag), "data/Dockerfile-mysql571.zip",
			flag(params.ContainerImagesFlag), "nginx:alpine,mysql:5.7",
			flag(params.BranchFlag), "dummy_branch",
			flag(params.ScanTypes), params.ContainersTypeFlag,
			flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		}
		scanID, projectID := executeCreateScan(t, testArgs)
		assert.Assert(t, scanID != "", "Scan ID should not be empty")
		assert.Assert(t, projectID != "", "Project ID should not be empty")
	})

	t.Run("InvalidImageInList", func(t *testing.T) {
		createASTIntegrationTestCommand(t)
		testArgs := []string{
			"scan", "create",
			flag(params.ProjectName), getProjectNameForScanTests(),
			flag(params.SourcesFlag), "data/Dockerfile-mysql571.zip",
			flag(params.ContainerImagesFlag), "nginx:alpine,bad-image,mysql:5.7",
			flag(params.BranchFlag), "dummy_branch",
			flag(params.ScanTypes), params.ContainersTypeFlag,
		}
		err, _ := executeCommand(t, testArgs...)
		assert.Assert(t, err != nil, "Expected error for invalid image in list")
	})
}

// TestContainerScan_TarFileValidation tests .tar file validation scenarios
func TestContainerScan_TarFileValidation(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("ExistingTarFile", func(t *testing.T) {
		// Create a dummy .tar file
		tarFile := filepath.Join(tempDir, "test-container.tar")
		f, err := os.Create(tarFile)
		assert.NilError(t, err)
		f.Close()

		createASTIntegrationTestCommand(t)
		testArgs := []string{
			"scan", "create",
			flag(params.ProjectName), getProjectNameForScanTests(),
			flag(params.SourcesFlag), "data/insecure.zip",
			flag(params.ContainerImagesFlag), tarFile,
			flag(params.BranchFlag), "dummy_branch",
			flag(params.ScanTypes), params.ContainersTypeFlag,
			flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		}
		scanID, projectID := executeCreateScan(t, testArgs)
		assert.Assert(t, scanID != "", "Scan ID should not be empty for existing tar file")
		assert.Assert(t, projectID != "", "Project ID should not be empty for existing tar file")
	})

	t.Run("NonExistentTarFile", func(t *testing.T) {
		nonExistentTar := filepath.Join(tempDir, "nonexistent.tar")

		createASTIntegrationTestCommand(t)
		testArgs := []string{
			"scan", "create",
			flag(params.ProjectName), getProjectNameForScanTests(),
			flag(params.SourcesFlag), "data/insecure.zip",
			flag(params.ContainerImagesFlag), nonExistentTar,
			flag(params.BranchFlag), "dummy_branch",
			flag(params.ScanTypes), params.ContainersTypeFlag,
		}
		err, _ := executeCommand(t, testArgs...)
		assert.Assert(t, err != nil, "Expected error for non-existent tar file")
	})

	t.Run("TarFileWithOtherImages", func(t *testing.T) {
		// Create a dummy .tar file
		tarFile := filepath.Join(tempDir, "another-test.tar")
		f, err := os.Create(tarFile)
		assert.NilError(t, err)
		f.Close()

		createASTIntegrationTestCommand(t)
		testArgs := []string{
			"scan", "create",
			flag(params.ProjectName), getProjectNameForScanTests(),
			flag(params.SourcesFlag), "data/insecure.zip",
			flag(params.ContainerImagesFlag), "nginx:alpine," + tarFile,
			flag(params.BranchFlag), "dummy_branch",
			flag(params.ScanTypes), params.ContainersTypeFlag,
			flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		}
		scanID, projectID := executeCreateScan(t, testArgs)
		assert.Assert(t, scanID != "", "Scan ID should not be empty for tar file with other images")
		assert.Assert(t, projectID != "", "Project ID should not be empty for tar file with other images")
	})
}

// TestContainerScan_SpecialCharactersInImageNames tests handling of special characters
func TestContainerScan_SpecialCharactersInImageNames(t *testing.T) {
	tests := []struct {
		name          string
		imageName     string
		shouldSucceed bool
		description   string
	}{
		{
			name:          "ImageWithHyphen",
			imageName:     "my-image:latest",
			shouldSucceed: true,
			description:   "Image with hyphen should be valid",
		},
		{
			name:          "ImageWithUnderscore",
			imageName:     "my_image:v1.0",
			shouldSucceed: true,
			description:   "Image with underscore should be valid",
		},
		{
			name:          "ImageWithDots",
			imageName:     "my.image.name:1.2.3",
			shouldSucceed: true,
			description:   "Image with dots should be valid",
		},
		{
			name:          "ImageWithSlash",
			imageName:     "namespace/image:tag",
			shouldSucceed: true,
			description:   "Image with namespace slash should be valid",
		},
		{
			name:          "ComplexRegistry",
			imageName:     "registry.example.com:5000/namespace/image:v1.0.0",
			shouldSucceed: true,
			description:   "Complex registry path should be valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createASTIntegrationTestCommand(t)
			testArgs := []string{
				"scan", "create",
				flag(params.ProjectName), getProjectNameForScanTests(),
				flag(params.SourcesFlag), "data/insecure.zip",
				flag(params.ContainerImagesFlag), tt.imageName,
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

// TestContainerScan_ImageTagVariations tests different tag formats
func TestContainerScan_ImageTagVariations(t *testing.T) {
	tests := []struct {
		name          string
		imageName     string
		shouldSucceed bool
		description   string
	}{
		{
			name:          "LatestTag",
			imageName:     "nginx:latest",
			shouldSucceed: true,
			description:   "Latest tag should be valid",
		},
		{
			name:          "NumericTag",
			imageName:     "nginx:1.21",
			shouldSucceed: true,
			description:   "Numeric tag should be valid",
		},
		{
			name:          "SemanticVersionTag",
			imageName:     "nginx:1.21.6",
			shouldSucceed: true,
			description:   "Semantic version tag should be valid",
		},
		{
			name:          "AlphanumericTag",
			imageName:     "nginx:v2.1.11",
			shouldSucceed: true,
			description:   "Alphanumeric tag with prefix should be valid",
		},
		{
			name:          "TagWithHyphen",
			imageName:     "nginx:stable-alpine",
			shouldSucceed: true,
			description:   "Tag with hyphen should be valid",
		},
		{
			name:          "SHA256Tag",
			imageName:     "nginx:sha256",
			shouldSucceed: true,
			description:   "SHA256 tag should be valid",
		},
		{
			name:          "TagWithUnderscore",
			imageName:     "nginx:stable_release",
			shouldSucceed: true,
			description:   "Tag with underscore should be valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createASTIntegrationTestCommand(t)
			testArgs := []string{
				"scan", "create",
				flag(params.ProjectName), getProjectNameForScanTests(),
				flag(params.SourcesFlag), "data/insecure.zip",
				flag(params.ContainerImagesFlag), tt.imageName,
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

// TestContainerScan_BoundaryConditions tests boundary conditions
func TestContainerScan_BoundaryConditions(t *testing.T) {
	t.Run("SingleCharacterImageName", func(t *testing.T) {
		createASTIntegrationTestCommand(t)
		testArgs := []string{
			"scan", "create",
			flag(params.ProjectName), getProjectNameForScanTests(),
			flag(params.SourcesFlag), "data/insecure.zip",
			flag(params.ContainerImagesFlag), "a:b",
			flag(params.BranchFlag), "dummy_branch",
			flag(params.ScanTypes), params.ContainersTypeFlag,
			flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		}
		scanID, projectID := executeCreateScan(t, testArgs)
		assert.Assert(t, scanID != "", "Scan ID should not be empty for single character image")
		assert.Assert(t, projectID != "", "Project ID should not be empty for single character image")
	})

	t.Run("VeryLongImageName", func(t *testing.T) {
		longName := "verylongimagenamethatshouldstillbevalidaslongasithasapropertagformat:v1.0.0"
		createASTIntegrationTestCommand(t)
		testArgs := []string{
			"scan", "create",
			flag(params.ProjectName), getProjectNameForScanTests(),
			flag(params.SourcesFlag), "data/insecure.zip",
			flag(params.ContainerImagesFlag), longName,
			flag(params.BranchFlag), "dummy_branch",
			flag(params.ScanTypes), params.ContainersTypeFlag,
			flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		}
		scanID, projectID := executeCreateScan(t, testArgs)
		assert.Assert(t, scanID != "", "Scan ID should not be empty for long image name")
		assert.Assert(t, projectID != "", "Project ID should not be empty for long image name")
	})

	t.Run("ManyImagesInList", func(t *testing.T) {
		// Test with 10 images
		images := "nginx:1,nginx:2,nginx:3,nginx:4,nginx:5,nginx:6,nginx:7,nginx:8,nginx:9,nginx:10"
		createASTIntegrationTestCommand(t)
		testArgs := []string{
			"scan", "create",
			flag(params.ProjectName), getProjectNameForScanTests(),
			flag(params.SourcesFlag), "data/insecure.zip",
			flag(params.ContainerImagesFlag), images,
			flag(params.BranchFlag), "dummy_branch",
			flag(params.ScanTypes), params.ContainersTypeFlag,
			flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		}
		scanID, projectID := executeCreateScan(t, testArgs)
		assert.Assert(t, scanID != "", "Scan ID should not be empty for many images")
		assert.Assert(t, projectID != "", "Project ID should not be empty for many images")
	})
}

// TestContainerScan_CombinedWithOtherScanTypes tests container scans combined with other scan types
func TestContainerScan_CombinedWithOtherScanTypes(t *testing.T) {
	t.Run("ContainerAndIaC", func(t *testing.T) {
		createASTIntegrationTestCommand(t)
		testArgs := []string{
			"scan", "create",
			flag(params.ProjectName), getProjectNameForScanTests(),
			flag(params.SourcesFlag), "data/iac-insecure.zip",
			flag(params.ContainerImagesFlag), "nginx:alpine",
			flag(params.BranchFlag), "dummy_branch",
			flag(params.ScanTypes), params.ContainersTypeFlag + "," + params.IacType,
			flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		}
		scanID, projectID := executeCreateScan(t, testArgs)
		assert.Assert(t, scanID != "", "Scan ID should not be empty for combined scan")
		assert.Assert(t, projectID != "", "Project ID should not be empty for combined scan")
	})

	t.Run("ContainerWithSCA", func(t *testing.T) {
		createASTIntegrationTestCommand(t)
		testArgs := []string{
			"scan", "create",
			flag(params.ProjectName), getProjectNameForScanTests(),
			flag(params.SourcesFlag), "data/insecure.zip",
			flag(params.ContainerImagesFlag), "nginx:alpine",
			flag(params.BranchFlag), "dummy_branch",
			flag(params.ScanTypes), params.ContainersTypeFlag + "," + params.ScaType,
			flag(params.ScanInfoFormatFlag), printer.FormatJSON,
		}
		scanID, projectID := executeCreateScan(t, testArgs)
		assert.Assert(t, scanID != "", "Scan ID should not be empty for container+sca scan")
		assert.Assert(t, projectID != "", "Project ID should not be empty for container+sca scan")
	})
}
