package wrappers

type ScanSummaryWrapper interface {
	GetScanSummaryByScanID(scanID string) (*ScanSummariesModel, *WebError, error)
}

