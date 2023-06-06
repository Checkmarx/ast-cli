package wrappers

type ResultsSbomWrapper interface {
	GenerateSbomReport(payload *SbomReportsPayload) (*SbomReportsResponse, *WebError, error)
	GetSbomReportStatus(reportID string) (*SbomPoolingResponse, *WebError, error)
	DownloadSbomReport(reportID, targetFile string) error
}
