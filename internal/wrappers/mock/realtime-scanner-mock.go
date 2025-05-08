package mock

import (
	"crypto/rand"
	"math/big"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type RealtimeScannerMockWrapper struct {
	CustomScan func(packages []wrappers.OssPackageRequest) (*wrappers.OssPackageResponse, error)
}

func NewRealtimeScannerMockWrapper() *RealtimeScannerMockWrapper {
	return &RealtimeScannerMockWrapper{}
}

func (r RealtimeScannerMockWrapper) Scan(packages []wrappers.OssPackageRequest) (*wrappers.OssPackageResponse, error) {
	if r.CustomScan != nil {
		return r.CustomScan(packages)
	}
	return generateMockResponse(packages), nil
}

func generateMockResponse(packages []wrappers.OssPackageRequest) *wrappers.OssPackageResponse {
	var response wrappers.OssPackageResponse
	for _, pkg := range packages {
		response.Packages = append(response.Packages, wrappers.OssResults{
			PackageManager: pkg.PackageManager,
			PackageName:    pkg.PackageName,
			Version:        pkg.Version,
			Status:         getRandomStatus(),
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
