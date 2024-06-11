package wrappers

type ResultsPdfWrapper interface {
	GeneratePdfReport(payload *PdfReportsPayload) (*PdfReportsResponse, *WebError, error)
	CheckPdfReportStatus(reportID string) (*PdfPollingResponse, *WebError, error)
	DownloadPdfReport(url, targetFile string) error
}
