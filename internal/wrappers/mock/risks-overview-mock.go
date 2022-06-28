package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

type RisksOverviewMockWrapper struct{}

func (r RisksOverviewMockWrapper) GetAllApiSecRisksByScanID(scanID string) (
	*wrappers.ApiSecResult,
	*wrappers.WebError,
	error,
) {
	const mock = "MOCK"
	return &wrappers.ApiSecResult{
		APICount:        0,
		TotalRisksCount: 0,
		Risks:           []int{},
	}, nil, nil
}
