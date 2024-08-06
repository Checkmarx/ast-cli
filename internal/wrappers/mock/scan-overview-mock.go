package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

var ScsScanPartial bool
var ScorecardScanned bool

type ScanOverviewMockWrapper struct {
}

func (s ScanOverviewMockWrapper) GetSCSOverviewByScanID(scanID string) (
	*wrappers.SCSOverview,
	*wrappers.WebError,
	error,
) {
	if ScsScanPartial {
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
					RiskSummary: map[string]interface{}{
						"critical": 0,
						"high":     5,
						"medium":   3,
						"low":      2,
						"info":     0,
					},
				},
				{
					Name:       "Scorecard",
					FullName:   "Scorecard",
					Status:     "Failed",
					TotalRisks: 0,
					RiskSummary: map[string]interface{}{
						"critical": 0,
						"high":     0,
						"medium":   0,
						"low":      0,
						"info":     0,
					},
				},
			},
		}, nil, nil
	}
	if ScorecardScanned {
		return &wrappers.SCSOverview{
			Status:          "Completed",
			TotalRisksCount: 14,
			RiskSummary: map[string]int{
				"critical": 0,
				"high":     7,
				"medium":   4,
				"low":      3,
				"info":     0,
			},
			MicroEngineOverviews: []*wrappers.MicroEngineOverview{
				{
					Name:       "2ms",
					FullName:   "Secret Detection",
					Status:     "Completed",
					TotalRisks: 10,
					RiskSummary: map[string]interface{}{
						"critical": 0,
						"high":     5,
						"medium":   3,
						"low":      2,
						"info":     0,
					},
				},
				{
					Name:       "Scorecard",
					FullName:   "Scorecard",
					Status:     "Completed",
					TotalRisks: 4,
					RiskSummary: map[string]interface{}{
						"critical": 0,
						"high":     2,
						"medium":   1,
						"low":      1,
						"info":     0,
					},
				},
			},
		}, nil, nil
	}
	// default Overview
	return &wrappers.SCSOverview{
		Status:          "Completed",
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
				RiskSummary: map[string]interface{}{
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
				RiskSummary: map[string]interface{}{},
			},
		},
	}, nil, nil
}

// SetScsMockVarsToDefault resets the mock variables to their default values.
// Use at the end of test cases where these variables were changed to reset them. This way subsequent tests aren't affected
func SetScsMockVarsToDefault() {
	HasScs = false
	ScsScanPartial = false
	ScorecardScanned = false
}
