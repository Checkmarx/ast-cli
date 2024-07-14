package mock

import (
	"os"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

type ExportMockWrapper struct{}

// GenerateSbomReport mock for tests
func (*ExportMockWrapper) GenerateSbomReport(payload *wrappers.ExportRequestPayload) (*wrappers.ExportResponse, error) {
	if payload.ScanID == "err-scan-id" {
		return nil, errors.New("error")
	}
	return &wrappers.ExportResponse{
		ExportID: "id123456",
	}, nil
}

// GetSbomReportStatus mock for tests
func (*ExportMockWrapper) GetSbomReportStatus(_ string) (*wrappers.ExportPollingResponse, error) {
	return &wrappers.ExportPollingResponse{
		ExportID:     "id1234",
		ExportStatus: "Completed",
		FileURL:      "url",
	}, nil
}

// DownloadSbomReport mock for tests
func (*ExportMockWrapper) DownloadSbomReport(_, targetFile string) error {
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
