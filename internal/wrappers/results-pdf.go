package wrappers

type ResultsPdfWrapper interface {
	GeneratePdfReport(payload *PdfReportsPayload) (*PdfReportsResponse, *WebError, error)
	CheckPdfReportStatus(reportID string) (*PdfPoolingResponse, *WebError, error)
	DownloadPdfReport(reportID, targetFile string) error
}
