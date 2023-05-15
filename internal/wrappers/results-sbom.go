package wrappers

type ResultsSbomWrapper interface {
	GenerateSbomReport(payload *SbomReportsPayload) (*SbomReportsResponse, *WebError, error)
	CheckSbomReportStatus(reportID string) (*SbomPoolingResponse, *WebError, error)
	DownloadSbomReport(reportID, targetFile string) error
}
