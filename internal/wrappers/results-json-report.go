package wrappers

type ResultsJSONWrapper interface {
	GenerateJSONReport(payload *JSONReportsPayload) (*JSONReportsResponse, *WebError, error)
	CheckJSONReportStatus(reportID string) (*JSONPollingResponse, *WebError, error)
	DownloadJSONReport(url, targetFile string, useAccessToken bool) error
}
