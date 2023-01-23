package wrappers

type ResultsPdfReportsWrapper interface {
	GeneratePdfReport(payload PdfReportsPayload) (*PdfReportsResponse, *WebError, error)
	PoolingForPdfReport(reportId string) (*PdfReportsPoolingResponse, *WebError, error)
}
