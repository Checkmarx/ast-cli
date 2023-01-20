package wrappers

type ResultsPdfReportsWrapper interface {
	GeneratePdfReport(payload PdfReportsPayload) (*PdfReportsResponse, *WebError, error)
}
