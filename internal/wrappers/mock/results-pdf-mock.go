package mock

import (
	"os"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

type ResultsPdfWrapper struct{}

// GeneratePdfReport mock for tests
func (*ResultsPdfWrapper) GeneratePdfReport(_ *wrappers.PdfReportsPayload) (*wrappers.PdfReportsResponse, *wrappers.WebError, error) {
	return &wrappers.PdfReportsResponse{
		ReportID: "reportId",
	}, nil, nil
}

// CheckPdfReportStatus mock for tests
func (*ResultsPdfWrapper) CheckPdfReportStatus(_ string) (*wrappers.PdfPollingResponse, *wrappers.WebError, error) {
	return &wrappers.PdfPollingResponse{
		Status: "completed",
	}, nil, nil
}

// DownloadPdfReport mock for tests
func (*ResultsPdfWrapper) DownloadPdfReport(_, targetFile string, b bool) error {
	file, err := os.Create(targetFile)
	defer func() {
		err = file.Close()
		if err != nil {
			panic(err)
		}
	}()
	if err != nil {
		return errors.Wrapf(err, "Failed to create file %s", targetFile)
	}
	return nil
}
