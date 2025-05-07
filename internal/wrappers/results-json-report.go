package wrappers

type ResultsJsonWrapper interface {
	GenerateJsonReport(payload *JsonReportsPayload) (*JsonReportsResponse, *WebError, error)
	CheckJsonReportStatus(reportID string) (*JsonPollingResponse, *WebError, error)
	DownloadJsonReport(url, targetFile string, useAccessToken bool) error
}
