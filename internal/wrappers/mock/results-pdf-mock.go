package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type ResultsPdfReportsWrapper struct{}

// GeneratePdfReport mock for tests
func (*ResultsPdfReportsWrapper) GeneratePdfReport(payload wrappers.PdfReportsPayload) (*wrappers.PdfReportsResponse, *wrappers.WebError, error) {
	return &wrappers.PdfReportsResponse{
		ReportId: "reportID",
	}, nil, nil
}

// CheckPdfReportStatus mock for tests

func (*ResultsPdfReportsWrapper) CheckPdfReportStatus(reportId string) (*wrappers.PdfReportsPoolingResponse, *wrappers.WebError, error) {
	return &wrappers.PdfReportsPoolingResponse{
		Status: "Ready",
	}, nil, nil
}

// DownloadPdfReport mock for tests

func (*ResultsPdfReportsWrapper) DownloadPdfReport(reportID, targetFile string) error {
	return nil
}
