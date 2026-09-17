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

	t.Run("EmptyTarFile", func(t *testing.T) {
		// Empty tar is not a container image; local resolution records Status=Failed. Since it was
		// named explicitly via --container-images, scan create must fail (AST-165915).
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
		}
		scanErr, _ := executeCommand(t, testArgs...)
		assert.Assert(t, scanErr != nil, "an empty tar named explicitly must fail the scan")
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

	t.Run("EmptyTarFileWithOtherImages", func(t *testing.T) {
		// nginx:alpine still resolves, but the empty tar was also named explicitly via
		// --container-images, so it must fail the scan even mixed with a valid image (AST-165915).
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
		}
		scanErr, _ := executeCommand(t, testArgs...)
		assert.Assert(t, scanErr != nil, "an unresolved image named explicitly must fail the scan even mixed with a valid one")
	})
}

// TestContainerScan_DiscoveredOnlyFailureWarnsButSucceeds runs the real containers-resolver
// against a source directory containing a Dockerfile that references an image guaranteed to fail
// resolution. Unlike an image named through --container-images, one only discovered inside the
// scanned sources must warn, not fail the scan (AST-146648, preserved by the AST-165915 fix).
func TestContainerScan_DiscoveredOnlyFailureWarnsButSucceeds(t *testing.T) {
	sourceDir := t.TempDir()
	assert.NilError(t, os.WriteFile(filepath.Join(sourceDir, "Dockerfile"), []byte("FROM debian:non-existent-tag-999\n"), 0o600))

	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), sourceDir,
		flag(params.ContainerResolveLocallyFlag), // Enable resolve locally, no --container-images
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanTypes), params.ContainersTypeFlag,
		flag(params.ScanInfoFormatFlag), printer.FormatJSON,
	}
	scanID, projectID := executeCreateScan(t, testArgs)
	assert.Assert(t, scanID != "", "a discovered-only unresolved image must not fail scan create")
	assert.Assert(t, projectID != "", "a discovered-only unresolved image must not fail scan create")
}

// TestContainerScan_RequestedAndDiscoveredFailuresTogether runs the real containers-resolver with
// two simultaneous unresolved images: one named explicitly via --container-images (must fail the
// scan) and one only discovered through a Dockerfile in the source (must only warn). Confirms the
// two do not interfere end-to-end, against the resolver's real output rather than a synthetic file.
func TestContainerScan_RequestedAndDiscoveredFailuresTogether(t *testing.T) {
	sourceDir := t.TempDir()
	assert.NilError(t, os.WriteFile(filepath.Join(sourceDir, "Dockerfile"), []byte("FROM alpine:non-existent-discovered-tag\n"), 0o600))

	createASTIntegrationTestCommand(t)
	testArgs := []string{
		"scan", "create",
		flag(params.ProjectName), getProjectNameForScanTests(),
		flag(params.SourcesFlag), sourceDir,
		flag(params.ContainerImagesFlag), "debian:non-existent-tag-999",
		flag(params.ContainerResolveLocallyFlag), // Enable resolve locally
		flag(params.BranchFlag), "dummy_branch",
		flag(params.ScanTypes), params.ContainersTypeFlag,
	}
	scanErr, _ := executeCommand(t, testArgs...)
	assert.Assert(t, scanErr != nil, "the explicitly requested unresolved image must fail the scan even alongside a merely-discovered one")
	assertError(t, scanErr, "debian:non-existent-tag-999")
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
		// Test with 5 images (reduced from 10 to avoid timeout)
		images := "nginx:1,nginx:2,nginx:3,nginx:4,nginx:5"
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
