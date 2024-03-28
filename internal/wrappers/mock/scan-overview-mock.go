package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type ScanOverviewMockWrapper struct{}

func (r ScanOverviewMockWrapper) GetSCSOverviewByScanID(scanID string) (
	*wrappers.SCSOverview,
	*wrappers.WebError,
	error,
) {
	return &wrappers.SCSOverview{
		Status:          "Partial",
		TotalRisksCount: 10,
		RiskSummary: map[string]int{
			"critical": 0,
			"high":     5,
			"medium":   3,
			"low":      2,
			"info":     0,
		},
		MicroEngineOverviews: []*wrappers.MicroEngineOverview{
			{
				Name:       "2ms",
				FullName:   "Secret Detection",
				Status:     "Completed",
				TotalRisks: 10,
				RiskSummary: map[string]int{
					"critical": 0,
					"high":     5,
					"medium":   3,
					"low":      2,
					"info":     0,
				},
			},
			{
				Name:        "Scorecard",
				FullName:    "Scorecard",
				Status:      "",
				TotalRisks:  0,
				RiskSummary: map[string]int{},
			},
		},
	}, nil, nil
}
