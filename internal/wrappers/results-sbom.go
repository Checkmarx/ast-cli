package wrappers

type ResultsSbomWrapper interface {
	GenerateSbomReport(payload *SbomReportsPayload) (*SbomReportsResponse, error)
	GenerateSbomReportWithProxy(payload *SbomReportsPayload, targetFile string) (bool, error)
	GetSbomReportStatus(reportID string) (*SbomPollingResponse, error)
	DownloadSbomReport(reportID, targetFile string) error
}
