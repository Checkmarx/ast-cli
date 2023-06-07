package wrappers

type ResultsSbomWrapper interface {
	GenerateSbomReport(payload *SbomReportsPayload) (*SbomReportsResponse, error)
	GetSbomReportStatus(reportID string) (*SbomPollingResponse, error)
	DownloadSbomReport(reportID, targetFile string) error
}
