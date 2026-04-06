package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

// ScanSummaryMockWrapper is a mock implementation of ScanSummaryWrapper.
type ScanSummaryMockWrapper struct{}

// GetScanSummaryByScanID returns mock scan summary data with empty AISC counters.
func (s *ScanSummaryMockWrapper) GetScanSummaryByScanID(scanID string) (*wrappers.ScanSummariesModel, *wrappers.WebError, error) {
	// Return mock scan summary data with empty AISC counters
	return &wrappers.ScanSummariesModel{
		ScansSummaries: []wrappers.ScanSumaries{
			{
				ScanID: scanID,
				AiscCounters: wrappers.AiscCounters{
					AssetsCounter:     0,
					AssetTypesCounter: 0,
				},
			},
		},
		TotalCount: 1,
	}, nil, nil
}
