package wrappers

type ExportWrapper interface {
	GenerateSbomReport(payload *ExportRequestPayload) (*ExportResponse, error)
	GetSbomReportStatus(reportID string) (*ExportPollingResponse, error)
	DownloadSbomReport(reportID, targetFile string) error
}
