package wrappers

// ScanSummaryWrapper wraps scan summary logic.
type ScanSummaryWrapper interface {
	GetScanSummaryByScanID(scanID string) (*ScanSummariesModel, *WebError, error)
}
