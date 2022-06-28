package wrappers

type RisksOverviewWrapper interface {
	GetAllAPISecRisksByScanID(scanID string) (*APISecResult, *WebError, error)
}
