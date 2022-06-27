package wrappers

type RisksOverviewWrapper interface {
	GetAllApiSecRisksByScanID(scanID string) (*ApiSecResult, *WebError, error)
}
