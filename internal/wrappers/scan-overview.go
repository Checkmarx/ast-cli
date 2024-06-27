package wrappers

type ScanOverviewWrapper interface {
	GetSCSOverviewByScanID(scanID string) (*SCSOverview, *WebError, error)
}
