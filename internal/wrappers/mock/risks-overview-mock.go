package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

type RisksOverviewMockWrapper struct{}

func (r RisksOverviewMockWrapper) GetAllAPISecRisksByScanID(scanID string) (
	*wrappers.APISecResult,
	*wrappers.WebError,
	error,
) {
	return &wrappers.APISecResult{
		APICount:        0,
		TotalRisksCount: 0,
		Risks:           []int{},
	}, nil, nil
}
