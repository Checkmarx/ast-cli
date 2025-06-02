package wrappers

type ResultsJSONWrapper interface {
	GenerateJSONReport(payload *JsonReportsPayload) (*JsonReportsResponse, *WebError, error)
	CheckJSONReportStatus(reportID string) (*JsonPollingResponse, *WebError, error)
	DownloadJSONReport(url, targetFile string, useAccessToken bool) error
}
