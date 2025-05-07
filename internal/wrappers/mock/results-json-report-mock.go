package mock

import (
	"os"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

type ResultsJsonWrapper struct{}

// GenerateJsonReport mock for tests
func (*ResultsJsonWrapper) GenerateJsonReport(_ *wrappers.JsonReportsPayload) (*wrappers.JsonReportsResponse, *wrappers.WebError, error) {
	return &wrappers.JsonReportsResponse{
		ReportID: "reportId",
	}, nil, nil
}

// CheckJsonReportStatus mock for tests
func (*ResultsJsonWrapper) CheckJsonReportStatus(_ string) (*wrappers.JsonPollingResponse, *wrappers.WebError, error) {
	return &wrappers.JsonPollingResponse{
		Status: "completed",
	}, nil, nil
}

// DownloadJsonReport mock for tests
func (*ResultsJsonWrapper) DownloadJsonReport(_, targetFile string, b bool) error {
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
