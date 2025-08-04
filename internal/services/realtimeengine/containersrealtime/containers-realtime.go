package containersrealtime

import (
	"encoding/json"
	"fmt"
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

const defaultTag = "latest"

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

func loadIgnoredContainerFindings(path string) ([]IgnoredContainersFinding, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ignored []IgnoredContainersFinding
	err = json.Unmarshal(data, &ignored)
	if err != nil {
		return nil, err
	}
	return ignored, nil
}

func buildContainerIgnoreMap(ignored []IgnoredContainersFinding) map[string]bool {
	m := make(map[string]bool)
	for _, f := range ignored {
		key := fmt.Sprintf("%s_%s_%s", f.ImageName, f.ImageTag, f.FilePath)
		m[key] = true
	}
	return m
}

func filterIgnoredContainers(results []ContainerImage, ignoreMap map[string]bool) []ContainerImage {
	filtered := make([]ContainerImage, 0, len(results))
	for _, r := range results {
		key := fmt.Sprintf("%s_%s_%s", r.ImageName, r.ImageTag, r.FilePath)
		if !ignoreMap[key] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// RunContainersRealtimeScan performs a containers real-time scan on the given file.
func (c *ContainersRealtimeService) RunContainersRealtimeScan(filePath string, ignoredFilePath string) (*ContainerImageResults, error) {

	if err := realtimeengine.EnsureLicense(c.JwtWrapper); err != nil {
		return nil, errorconstants.NewRealtimeEngineError("failed to ensure license").Error()
	}

	if err := realtimeengine.ValidateFilePath(filePath); err != nil {
		return nil, errorconstants.NewRealtimeEngineError("invalid file path").Error()
	}

	images, err := parseContainersFile(filePath)
	if err != nil {
		return nil, errorconstants.NewRealtimeEngineError("failed to parse containers file").Error()
	}

	if len(images) == 0 {
		return &ContainerImageResults{Images: []ContainerImage{}}, nil
	}

	images = splitLocationsToSeparateResults(images)

	results, err := c.scanImages(images, filePath)
	if err != nil {
		return nil, errorconstants.NewRealtimeEngineError("Realtime scanner engine failed").Error()
	}

	if ignoredFilePath != "" {
		ignored, err := loadIgnoredContainerFindings(ignoredFilePath)
		if err != nil {
			return nil, errorconstants.NewRealtimeEngineError("failed to load ignored containers").Error()
		}
		ignoreMap := buildContainerIgnoreMap(ignored)
		results.Images = filterIgnoredContainers(results.Images, ignoreMap)
	}

	return results, nil
}

func splitLocationsToSeparateResults(images []types.ImageModel) []types.ImageModel {
	for i := 0; i < len(images); {
		if len(images[i].ImageLocations) > 1 {
			for _, loc := range images[i].ImageLocations {
				newImage := images[i]
				newImage.ImageLocations = []types.ImageLocation{loc}
				images = append(images, newImage)
			}
			images = append(images[:i], images[i+1:]...)
		} else {
			i++
		}
	}
	return images
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

	files, envVars, _, err := extractor.ExtractFiles(scanPath, false)
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
	var imagesWithSha []wrappers.ContainerImageResponseItem
	for _, img := range images {
		if img.IsSha {
			logger.PrintfIfVerbose("Skipping image with SHA: %s", img.Name)
			addShaImage(&imagesWithSha, img)
			continue
		}
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

	result := c.buildContainerImageResults(response.Images, imagesWithSha, images, filePath)
	return &result, nil
}

func addShaImage(images *[]wrappers.ContainerImageResponseItem, img types.ImageModel) {
	imageName, imageTag := splitToImageAndSha(img.Name)

	*images = append(*images, wrappers.ContainerImageResponseItem{
		ImageName:       imageName,
		ImageTag:        imageTag,
		Status:          "Unknown",
		Vulnerabilities: []wrappers.ContainerImageVulnerability{},
	})
}

func splitToImageAndSha(image string) (imageName, imageTag string) {
	atIndex := strings.Index(image, "@")
	if atIndex == -1 {
		return splitToImageAndTag(image)
	}

	nameAndTag := image[:atIndex]
	shaPart := image[atIndex+1:]

	colonIndex := strings.LastIndex(nameAndTag, ":")
	if colonIndex != -1 {
		imageName = nameAndTag[:colonIndex]
		tag := nameAndTag[colonIndex+1:]
		imageTag = tag + "@" + shaPart
	} else {
		imageName = nameAndTag
		imageTag = shaPart
	}
	return
}

// buildContainerImageResults builds ContainerImageResults from response and images
func (c *ContainersRealtimeService) buildContainerImageResults(responseImages, imagesWithSha []wrappers.ContainerImageResponseItem, images []types.ImageModel, filePath string) ContainerImageResults {
	var result ContainerImageResults

	result = mergeImagesToResults(responseImages, result, &images, filePath)
	result = mergeImagesToResults(imagesWithSha, result, &images, filePath)
	return result
}

func mergeImagesToResults(listOfImages []wrappers.ContainerImageResponseItem, result ContainerImageResults, images *[]types.ImageModel, filePath string) ContainerImageResults {
	for _, respImg := range listOfImages {
		locations, specificFilePath := getImageLocations(images, respImg.ImageName, respImg.ImageTag)
		if specificFilePath == "" {
			specificFilePath = filePath
		}
		containerImage := ContainerImage{
			ImageName:       respImg.ImageName,
			ImageTag:        respImg.ImageTag,
			FilePath:        specificFilePath,
			Locations:       locations,
			Status:          respImg.Status,
			Vulnerabilities: convertVulnerabilities(respImg.Vulnerabilities),
		}
		result.Images = append(result.Images, containerImage)
	}
	return result
}

func getImageLocations(images *[]types.ImageModel, imageName, imageTag string) (location []realtimeengine.Location, filePath string) {
	for i, img := range *images {
		if !isSameImage(img.Name, imageName, imageTag) {
			continue
		}
		location := convertLocations(&img.ImageLocations)
		filePath := ""
		if len(img.ImageLocations) > 0 {
			filePath = img.ImageLocations[0].Path
		}
		*images = append((*images)[:i], (*images)[i+1:]...)
		return location, filePath
	}
	return []realtimeengine.Location{}, ""
}

func isSameImage(curImage, imageName, imageTag string) bool {
	return curImage == imageName+":"+imageTag || curImage == imageName+"@"+imageTag || curImage == imageName && imageTag == defaultTag
}

// splitToImageAndTag splits the image string into name and tag components.
func splitToImageAndTag(image string) (imageName, imageTag string) {
	// Split the image string by the last colon to separate name and tag
	lastColonIndex := strings.LastIndex(image, ":")

	if lastColonIndex == len(image)-1 || lastColonIndex == -1 {
		return image, defaultTag // No tag specified, default to "latest"
	}

	imageName = image[:lastColonIndex]
	imageTag = image[lastColonIndex+1:]

	return imageName, imageTag
}

// convertLocations converts types locations to realtimeengine locations.
func convertLocations(locations *[]types.ImageLocation) []realtimeengine.Location {
	var result []realtimeengine.Location
	for _, loc := range *locations {
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
