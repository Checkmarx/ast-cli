package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type ResultsPdfWrapper struct{}

// GeneratePdfReport mock for tests
func (*ResultsPdfWrapper) GeneratePdfReport(_ *wrappers.PdfReportsPayload) (*wrappers.PdfReportsResponse, *wrappers.WebError, error) {
	return &wrappers.PdfReportsResponse{
		ReportID: "reportId",
	}, nil, nil
}

// CheckPdfReportStatus mock for tests
func (*ResultsPdfWrapper) CheckPdfReportStatus(_ string) (*wrappers.PdfPoolingResponse, *wrappers.WebError, error) {
	return &wrappers.PdfPoolingResponse{
		Status: "completed",
	}, nil, nil
}

// DownloadPdfReport mock for tests
func (*ResultsPdfWrapper) DownloadPdfReport(_, _ string) error {
	return nil
}
