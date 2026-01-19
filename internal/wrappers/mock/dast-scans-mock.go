package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

// DastScansMockWrapper is a mock implementation of DastScansWrapper
type DastScansMockWrapper struct{}

// Get mocks the Get method
func (s *DastScansMockWrapper) Get(params map[string]string) (*wrappers.DastScansCollectionResponseModel, *wrappers.ErrorModel, error) {
	return &wrappers.DastScansCollectionResponseModel{
		Scans: []wrappers.DastScanResponseModel{
			{
				ScanID:     "test-scan-id",
				Initiator:  "Test User",
				ScanType:   "DAST",
				Created:    "2024-01-01T00:00:00Z",
				RiskRating: "Medium risk",
				RiskLevel: wrappers.RiskLevel{
					CriticalCount: 0,
					HighCount:     0,
					MediumCount:   3,
					LowCount:      10,
					InfoCount:     40,
				},
				AlertRiskLevel: wrappers.RiskLevel{
					CriticalCount: 0,
					HighCount:     0,
					MediumCount:   1,
					LowCount:      2,
					InfoCount:     5,
				},
				StartTime:         "2024-01-01T00:01:00Z",
				UpdateTime:        "2024-01-01T00:10:00Z",
				ScanDuration:      540,
				LastStatus:        "Finished@Scan finished successfully",
				Statistics:        "Completed",
				HasResults:        true,
				ScannedPathsCount: 10,
				HasLog:            true,
				Source:            "PLATFORM",
			},
		},
		TotalScans: 1,
	}, nil, nil
}

