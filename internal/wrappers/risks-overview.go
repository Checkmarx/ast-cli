package wrappers

type RisksOverviewWrapper interface {
	GetAllAPISecRisksByScanID(scanID string) (*APISecResult, *WebError, error)
	GetFilterResultForAPISecByScanID(scanID string, queryParam map[string]string) (APISecRiskEntriesResult, *WebError, error)
}
