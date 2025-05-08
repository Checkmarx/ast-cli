package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

type RealtimeScannerMockWrapper struct {
}

func NewRealtimeScannerMockWrapper() *RealtimeScannerMockWrapper {
	return &RealtimeScannerMockWrapper{}
}

func (r RealtimeScannerMockWrapper) Scan(packages []wrappers.OssPackageRequest) (*wrappers.OssPackageResponse, error) {
	return &wrappers.OssPackageResponse{
		Packages: []wrappers.OssResults{
			{
				PackageManager: "npm",
				PackageName:    "lodash",
				Version:        "4.17.21",
				Status:         "OK",
			},
			{
				PackageManager: "npm",
				PackageName:    "express",
				Version:        "4.17.1",
				Status:         "OK",
			},
		},
	}, nil
}
