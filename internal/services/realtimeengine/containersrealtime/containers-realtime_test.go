package containersrealtime

import (
	"errors"
	"testing"

	"github.com/Checkmarx/containers-types/types"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

const testImageName = "nginx"

func TestRunContainersRealtimeScan_ValidLicenseAndFile_Success(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	service := NewContainersRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{
			CustomScanImages: func(images *wrappers.ContainerImageRequest) (*wrappers.ContainerImageResponse, error) {
				return &wrappers.ContainerImageResponse{
					Images: []wrappers.ContainerImageResponseItem{{
						ImageName: "nginx",
						ImageTag:  "latest",
						Status:    "OK",
						Vulnerabilities: []wrappers.ContainerImageVulnerability{{
							CVE:         "CVE-1234-5678",
							Description: "Mock vuln",
							Severity:    "High",
						}},
					}},
				}, nil
			},
		},
	)
	result, err := service.RunContainersRealtimeScan("../../../commands/data/containers/testdata/Dockerfile", "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, len(result.Images), 0)
}

func TestRunContainersRealtimeScan_EmptyFilePath_Fails(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	service := NewContainersRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)
	result, err := service.RunContainersRealtimeScan("", "")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "realtime engine error: file path is required")
}

func TestRunContainersRealtimeScan_InvalidLicense_Fails(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	service := NewContainersRealtimeService(
		nil, // Invalid JWT wrapper
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)
	result, err := service.RunContainersRealtimeScan("../../../commands/data/containers/testdata/Dockerfile", "")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "realtime engine error: failed to ensure license")
}

func TestRunContainersRealtimeScan_FeatureFlagDisabled_Fails(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: false}
	service := NewContainersRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)
	result, err := service.RunContainersRealtimeScan("../../../commands/data/containers/testdata/Dockerfile", "")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "realtime engine error: Realtime engine is not available for this tenant")
}

func TestRunContainersRealtimeScan_InvalidFilePath_Fails(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	service := NewContainersRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)
	result, err := service.RunContainersRealtimeScan("/non/existent/Dockerfile", "")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "realtime engine error: invalid file path")
}

func TestRunContainersRealtimeScan_NoImagesFound_ReturnsEmpty(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	service := NewContainersRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)
	result, err := service.RunContainersRealtimeScan("../../../commands/data/containers/emptytestdata/Dockerfile", "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result.Images))
}

func TestRunContainersRealtimeScan_ScanError_ReturnsError(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	service := NewContainersRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{
			CustomScanImages: func(images *wrappers.ContainerImageRequest) (*wrappers.ContainerImageResponse, error) {
				return nil, errors.New("mock scan error")
			},
		},
	)
	result, err := service.RunContainersRealtimeScan("../../../commands/data/containers/testdata/Dockerfile", "")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "realtime engine error: Realtime scanner engine failed")
}

func TestRunContainersRealtimeScan_ImageVulnerabilityMapping(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}
	service := NewContainersRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{
			CustomScanImages: func(images *wrappers.ContainerImageRequest) (*wrappers.ContainerImageResponse, error) {
				return &wrappers.ContainerImageResponse{
					Images: []wrappers.ContainerImageResponseItem{{
						ImageName: "nginx",
						ImageTag:  "latest",
						Status:    "OK",
						Vulnerabilities: []wrappers.ContainerImageVulnerability{{
							CVE:         "CVE-9999-0000",
							Description: "Test vuln",
							Severity:    "Medium",
						}},
					}},
				}, nil
			},
		},
	)
	result, err := service.RunContainersRealtimeScan("../../../commands/data/containers/testdata/Dockerfile", "")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "nginx", result.Images[0].ImageName)
	assert.Equal(t, "CVE-9999-0000", result.Images[0].Vulnerabilities[0].CVE)
	assert.Equal(t, "Medium", result.Images[0].Vulnerabilities[0].Severity)
	assert.Equal(t, 0, result.Images[0].Locations[0].Line)
	assert.Equal(t, 5, result.Images[0].Locations[0].StartIndex)
	assert.Equal(t, 17, result.Images[0].Locations[0].EndIndex)
}

func TestSplitToImageAndSha_whenHasShaAndTag_shouldReturnImageAndShaAndTag(t *testing.T) {
	image := testImageName + ":latest@sha256:1234567890abcdef"
	expectedImage := testImageName
	expectedSha := "latest@sha256:1234567890abcdef"

	imageName, sha := splitToImageAndSha(image)

	assert.Equal(t, expectedImage, imageName)
	assert.Equal(t, expectedSha, sha)
}

func TestSplitToImageAndSha_whenHasShaNoTag_shouldReturnImageAndShaNoTag(t *testing.T) {
	image := testImageName + "@sha256:1234567890abcdef"
	expectedImage := testImageName
	expectedSha := "sha256:1234567890abcdef"

	imageName, sha := splitToImageAndSha(image)

	assert.Equal(t, expectedImage, imageName)
	assert.Equal(t, expectedSha, sha)
}

func TestSplitLocationsToSeparateResults_MultipleLocations(t *testing.T) {
	img := types.ImageModel{
		Name: testImageName,
		ImageLocations: []types.ImageLocation{
			{Line: 1, StartIndex: 0, EndIndex: 5},
			{Line: 2, StartIndex: 6, EndIndex: 11},
		},
	}
	result := splitLocationsToSeparateResults([]types.ImageModel{img})
	assert.Equal(t, 2, len(result))
	assert.Equal(t, 1, len(result[0].ImageLocations))
	assert.Equal(t, 1, result[0].ImageLocations[0].Line)
	assert.Equal(t, 1, len(result[1].ImageLocations))
	assert.Equal(t, 2, result[1].ImageLocations[0].Line)
}

func TestSplitLocationsToSeparateResults_SingleLocation(t *testing.T) {
	img := types.ImageModel{
		Name: testImageName,
		ImageLocations: []types.ImageLocation{
			{Line: 3, StartIndex: 0, EndIndex: 5},
		},
	}
	result := splitLocationsToSeparateResults([]types.ImageModel{img})
	assert.Equal(t, 1, len(result))
	assert.Equal(t, 1, len(result[0].ImageLocations))
	assert.Equal(t, 3, result[0].ImageLocations[0].Line)
}

func TestSplitLocationsToSeparateResults_NoLocations(t *testing.T) {
	img := types.ImageModel{
		Name:           testImageName,
		ImageLocations: []types.ImageLocation{},
	}
	result := splitLocationsToSeparateResults([]types.ImageModel{img})
	assert.Equal(t, 1, len(result))
	assert.Equal(t, 0, len(result[0].ImageLocations))
}

func TestRunContainersRealtimeScan_WithIgnoreFile_FiltersResult(t *testing.T) {
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.OssRealtimeEnabled, Status: true}

	ignoreFile := "../../../commands/data/containers/testdata/ignoredContainers.json"

	service := NewContainersRealtimeService(
		&mock.JWTMockWrapper{},
		&mock.FeatureFlagsMockWrapper{},
		&mock.RealtimeScannerMockWrapper{},
	)

	result, err := service.RunContainersRealtimeScan("../../../commands/data/containers/testdata/Dockerfile", ignoreFile)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result.Images))
}

func TestBuildContainerIgnoreMap_Basic(t *testing.T) {
	ignored := []IgnoredContainersFinding{
		{
			ImageName: "nginx",
			ImageTag:  "latest",
		},
	}
	result := buildContainerIgnoreMap(ignored)

	key := "nginx_latest"
	assert.Contains(t, result, key)
}
