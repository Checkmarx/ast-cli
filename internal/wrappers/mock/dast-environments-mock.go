package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

// DastEnvironmentsMockWrapper is a mock implementation of DastEnvironmentsWrapper
type DastEnvironmentsMockWrapper struct{}

// Get mocks the Get method
func (e *DastEnvironmentsMockWrapper) Get(params map[string]string) (*wrappers.DastEnvironmentsCollectionResponseModel, *wrappers.ErrorModel, error) {
	return &wrappers.DastEnvironmentsCollectionResponseModel{
		Environments: []wrappers.DastEnvironmentResponseModel{
			{
				EnvironmentID: "test-env-id",
				Domain:        "test-domain",
				URL:           "https://test.example.com",
				ScanType:      "DAST",
				Created:       "2024-01-01T00:00:00Z",
				RiskRating:    "Low risk",
				LastScanTime:  "2024-01-02T00:00:00Z",
				LastStatus:    "Finished@Scan finished successfully",
			},
		},
		TotalItems: 1,
	}, nil, nil
}
