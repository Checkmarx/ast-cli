package mock

import (
	"crypto/rand"
	"math/big"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type RealtimeScannerMockWrapper struct {
	CustomScan       func(packages *wrappers.RealtimeScannerPackageRequest) (*wrappers.RealtimeScannerPackageResponse, error)
	CustomScanImages func(images *wrappers.ContainerImageRequest) (*wrappers.ContainerImageResponse, error)
}

func NewRealtimeScannerMockWrapper() *RealtimeScannerMockWrapper {
	return &RealtimeScannerMockWrapper{}
}

func (r RealtimeScannerMockWrapper) ScanPackages(packages *wrappers.RealtimeScannerPackageRequest) (results *wrappers.RealtimeScannerPackageResponse, err error) {
	if r.CustomScan != nil {
		return r.CustomScan(packages)
	}
	return generateMockResponse(packages), nil
}

func generateMockResponse(packages *wrappers.RealtimeScannerPackageRequest) *wrappers.RealtimeScannerPackageResponse {
	var response wrappers.RealtimeScannerPackageResponse
	for _, pkg := range packages.Packages {
		response.Packages = append(response.Packages, wrappers.RealtimeScannerResults{
			PackageManager: pkg.PackageManager,
			PackageName:    pkg.PackageName,
			Version:        pkg.Version,
			Status:         getRandomStatus(),
		})
	}
	return &response
}

func (r RealtimeScannerMockWrapper) ScanImages(images *wrappers.ContainerImageRequest) (results *wrappers.ContainerImageResponse, err error) {
	if r.CustomScanImages != nil {
		return r.CustomScanImages(images)
	}
	return generateMockImageResponse(images), nil
}

func generateMockImageResponse(images *wrappers.ContainerImageRequest) *wrappers.ContainerImageResponse {
	var response wrappers.ContainerImageResponse
	for _, img := range images.Images {
		response.Images = append(response.Images, wrappers.ContainerImageResponseItem{
			ImageName: img.ImageName,
			ImageTag:  img.ImageTag,
			Status:    getRandomStatus(),
			Vulnerabilities: []wrappers.ContainerImageVulnerability{
				{CVE: "CVE-1234-5678", Description: "Mock vulnerability", Severity: "High"},
			},
		})
	}
	return &response
}

func getRandomStatus() string {
	statuses := []string{"OK", "Malicious", "Unknown"}
	// Randomly select a status from the list
	randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(statuses))))
	if err != nil {
		return "OK" // Fallback to "OK" in case of error
	}
	return statuses[randomIndex.Int64()]
}
