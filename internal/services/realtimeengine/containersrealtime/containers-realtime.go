package containersrealtime

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Checkmarx/containers-images-extractor/pkg/imagesExtractor"
	"github.com/Checkmarx/containers-types/types"

	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

// ContainersRealtimeService is the service responsible for performing real-time container scanning.
type ContainersRealtimeService struct {
	JwtWrapper             wrappers.JWTWrapper
	FeatureFlagWrapper     wrappers.FeatureFlagsWrapper
	RealtimeScannerWrapper wrappers.RealtimeScannerWrapper
}

// NewContainersRealtimeService creates a new ContainersRealtimeService.
func NewContainersRealtimeService(
	jwtWrapper wrappers.JWTWrapper,
	featureFlagWrapper wrappers.FeatureFlagsWrapper,
	realtimeScannerWrapper wrappers.RealtimeScannerWrapper,
) *ContainersRealtimeService {
	return &ContainersRealtimeService{
		JwtWrapper:             jwtWrapper,
		FeatureFlagWrapper:     featureFlagWrapper,
		RealtimeScannerWrapper: realtimeScannerWrapper,
	}
}

// RunContainersRealtimeScan performs a containers real-time scan on the given file.
func (c *ContainersRealtimeService) RunContainersRealtimeScan(filePath string) (results *ContainerImageResults, err error) {
	if filePath == "" {
		return nil, errorconstants.NewRealtimeEngineError("file path is required").Error()
	}

	if enabled, err := c.isFeatureFlagEnabled(); err != nil || !enabled {
		logger.PrintfIfVerbose("Containers Realtime scan is not available (feature flag disabled or error: %v)", err)
		return nil, errorconstants.NewRealtimeEngineError(errorconstants.RealtimeEngineNotAvailable).Error()
	}

	if err := c.ensureLicense(); err != nil {
		return nil, errorconstants.NewRealtimeEngineError("failed to ensure license").Error()
	}

	if err := validateFilePath(filePath); err != nil {
		return nil, errorconstants.NewRealtimeEngineError("invalid file path").Error()
	}

	images, err := parseContainersFile(filePath)
	if err != nil {
		logger.PrintfIfVerbose("Failed to parse containers file %s: %v", filePath, err)
		return nil, errorconstants.NewRealtimeEngineError("failed to parse containers file").Error()
	}

	if len(images) == 0 {
		return &ContainerImageResults{Images: []ContainerImage{}}, nil
	}

	result, err := c.scanImages(images, filePath)
	if err != nil {
		logger.PrintfIfVerbose("Failed to scan images via realtime service: %v", err)
		return nil, errorconstants.NewRealtimeEngineError("Realtime scanner engine failed").Error()
	}

	return result, nil
}

// isFeatureFlagEnabled checks if the Containers Realtime feature flag is enabled.
func (c *ContainersRealtimeService) isFeatureFlagEnabled() (bool, error) {
	enabled, err := c.FeatureFlagWrapper.GetSpecificFlag(wrappers.OssRealtimeEnabled)
	if err != nil {
		return false, errors.Wrap(err, "failed to get feature flag")
	}
	return enabled.Status, nil
}

// ensureLicense validates that a valid JWT wrapper is available.
func (c *ContainersRealtimeService) ensureLicense() error {
	if c.JwtWrapper == nil {
		return errors.New("JWT wrapper is not initialized, cannot ensure license")
	}
	return nil
}

// validateFilePath validates that the file path exists and is accessible.
func validateFilePath(filePath string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return errors.Errorf("file does not exist: %s", filePath)
	}

	// Check if it's a regular file
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return errors.Errorf("cannot access file: %s", filePath)
	}

	if fileInfo.IsDir() {
		return errors.Errorf("path is a directory, expected a file: %s", filePath)
	}

	return nil
}

// parseContainersFile parses the containers file and returns a list of images.
func parseContainersFile(filePath string) ([]types.ImageModel, error) {
	extractor := imagesExtractor.NewImagesExtractor()

	// Extract files from the scan path (directory containing the file)
	scanPath := filepath.Dir(filePath)

	logger.PrintfIfVerbose("Scanning directory for container images: %s", scanPath)

	// Ensure the directory exists
	if _, statErr := os.Stat(scanPath); os.IsNotExist(statErr) {
		return nil, errors.Errorf("directory does not exist: %s", scanPath)
	}

	var files types.FileImages
	var envVars map[string]map[string]string
	var err error
	files, envVars, _, err = extractor.ExtractFiles(scanPath)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting files")
	}

	images, err := extractor.ExtractAndMergeImagesFromFilesWithLineInfo(files, []types.ImageModel{}, envVars)
	if err != nil {
		return nil, errors.Wrap(err, "error merging images")
	}

	logger.PrintfIfVerbose("Extracted %d container images", len(images))
	return images, nil
}

// scanImages scans the extracted images using the realtime scanner.
func (c *ContainersRealtimeService) scanImages(images []types.ImageModel, filePath string) (results *ContainerImageResults, err error) {
	logger.PrintfIfVerbose("Scanning %d images for vulnerabilities", len(images))

	var requestImages []wrappers.ContainerImageRequestItem
	for _, img := range images {
		imageName, imageTag := splitToImageAndTag(img.Name)

		logger.PrintfIfVerbose("Processing image: %s:%s", imageName, imageTag)

		requestImages = append(requestImages, wrappers.ContainerImageRequestItem{
			ImageName: imageName,
			ImageTag:  imageTag,
		})
	}

	request := &wrappers.ContainerImageRequest{
		Images: requestImages,
	}

	response, err := c.RealtimeScannerWrapper.ScanImages(request)
	if err != nil {
		return nil, err
	}

	logger.PrintfIfVerbose("Received scan results for %d images", len(response.Images))

	result := c.buildContainerImageResults(response.Images, images, filePath)
	return &result, nil
}

// buildContainerImageResults builds ContainerImageResults from response and images
func (c *ContainersRealtimeService) buildContainerImageResults(responseImages []wrappers.ContainerImageResponseItem, images []types.ImageModel, filePath string) ContainerImageResults {
	var result ContainerImageResults
	for i, respImg := range responseImages {
		var locations []realtimeengine.Location
		if i < len(images) {
			locations = convertLocations(images[i].ImageLocations)
		}

		containerImage := ContainerImage{
			ImageName:       respImg.ImageName,
			ImageTag:        respImg.ImageTag,
			FilePath:        filePath,
			Locations:       locations,
			Status:          respImg.Status,
			Vulnerabilities: convertVulnerabilities(respImg.Vulnerabilities),
		}
		result.Images = append(result.Images, containerImage)
	}
	return result
}

// splitToImageAndTag splits the image string into name and tag components.
func splitToImageAndTag(image string) (string, string) {
	// Split the image string by the last colon to separate name and tag
	lastColonIndex := strings.LastIndex(image, ":")

	if lastColonIndex == len(image)-1 {
		return image, "latest" // No tag specified, default to "latest"
	}

	imageName := image[:lastColonIndex]
	imageTag := image[lastColonIndex+1:]

	return imageName, imageTag
}

// convertLocations converts types locations to realtimeengine locations.
func convertLocations(locations []types.ImageLocation) []realtimeengine.Location {
	var result []realtimeengine.Location
	for _, loc := range locations {
		line := loc.Line
		startIndex := loc.StartIndex
		endIndex := loc.EndIndex

		result = append(result, realtimeengine.Location{
			Line:       line,
			StartIndex: startIndex,
			EndIndex:   endIndex,
		})
	}
	return result
}

// convertVulnerabilities converts wrapper vulnerabilities to service vulnerabilities.
func convertVulnerabilities(vulns []wrappers.ContainerImageVulnerability) []Vulnerability {
	var result []Vulnerability
	for _, vuln := range vulns {
		result = append(result, Vulnerability{
			CVE:      vuln.CVE,
			Severity: vuln.Severity,
		})
	}
	return result
}
