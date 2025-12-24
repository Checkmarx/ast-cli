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
func (r RisksOverviewMockWrapper) GetFilterResultForAPISecByScanID(scanID string, queryParam map[string]string) (wrappers.APISecRiskEntriesResult, *wrappers.WebError, error) {
	return wrappers.APISecRiskEntriesResult{
		Entries: []wrappers.APISecRiskEntry{},
	}, nil, nil
}

type RisksOverviewMockWrapperWithEntries struct {
	Entries []wrappers.APISecRiskEntry
}

func (r *RisksOverviewMockWrapperWithEntries) GetFilterResultForAPISecByScanID(scanID string, queryParam map[string]string) (wrappers.APISecRiskEntriesResult, *wrappers.WebError, error) {
	return wrappers.APISecRiskEntriesResult{
		Entries: r.Entries,
	}, nil, nil
}
func (r RisksOverviewMockWrapperWithEntries) GetAllAPISecRisksByScanID(scanID string) (
	*wrappers.APISecResult,
	*wrappers.WebError,
	error,
) {
	return &wrappers.APISecResult{}, nil, nil
}
