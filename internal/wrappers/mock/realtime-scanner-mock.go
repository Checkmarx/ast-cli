package mock

import (
	"crypto/rand"
	"math/big"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type RealtimeScannerMockWrapper struct {
	CustomScan func(packages *wrappers.RealtimeScannerPackageRequest) (*wrappers.RealtimeScannerPackageResponse, error)
}

func NewRealtimeScannerMockWrapper() *RealtimeScannerMockWrapper {
	return &RealtimeScannerMockWrapper{}
}

func (r RealtimeScannerMockWrapper) Scan(packages *wrappers.RealtimeScannerPackageRequest) (*wrappers.RealtimeScannerPackageResponse, error) {
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

func getRandomStatus() string {
	statuses := []string{"OK", "Malicious", "Unknown"}
	// Randomly select a status from the list
	randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(statuses))))
	if err != nil {
		return "OK" // Fallback to "OK" in case of error
	}
	return statuses[randomIndex.Int64()]
}
