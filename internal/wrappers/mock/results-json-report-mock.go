package mock

import (
	"os"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

type ResultsJSONWrapper struct{}

// GenerateJSONReport mock for tests
func (*ResultsJSONWrapper) GenerateJSONReport(_ *wrappers.JsonReportsPayload) (*wrappers.JsonReportsResponse, *wrappers.WebError, error) {
	return &wrappers.JsonReportsResponse{
		ReportID: "reportId",
	}, nil, nil
}

// CheckJSONReportStatus mock for tests
func (*ResultsJSONWrapper) CheckJSONReportStatus(_ string) (*wrappers.JsonPollingResponse, *wrappers.WebError, error) {
	return &wrappers.JsonPollingResponse{
		Status: "completed",
	}, nil, nil
}

// DownloadJSONReport mock for tests
func (*ResultsJSONWrapper) DownloadJSONReport(_, targetFile string, b bool) error {
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
