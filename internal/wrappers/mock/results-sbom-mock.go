package mock

import (
	"os"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

type ResultsSbomWrapper struct{}

// GenerateSbomReport mock for tests
func (*ResultsSbomWrapper) GenerateSbomReport(_ *wrappers.SbomReportsPayload) (*wrappers.SbomReportsResponse, error) {
	return &wrappers.SbomReportsResponse{
		ExportID: "id123456",
	}, nil
}

// GetSbomReportStatus mock for tests
func (*ResultsSbomWrapper) GetSbomReportStatus(_ string) (*wrappers.SbomPollingResponse, error) {
	return &wrappers.SbomPollingResponse{
		ExportID:     "id1234",
		ExportStatus: "Completed",
		FileURL:      "url",
	}, nil
}

// DownloadSbomReport mock for tests
func (*ResultsSbomWrapper) DownloadSbomReport(_, targetFile string) error {
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
