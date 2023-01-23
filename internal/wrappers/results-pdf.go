package wrappers

type ResultsPdfReportsWrapper interface {
	GeneratePdfReport(payload PdfReportsPayload) (*PdfReportsResponse, *WebError, error)
	CheckPdfReportStatus(reportId string) (*PdfReportsPoolingResponse, *WebError, error)
	DownloadPdfReport(reportID, targetFile string) error
}
