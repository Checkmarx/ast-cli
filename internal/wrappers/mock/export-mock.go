package mock

import (
	"os"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

type ExportMockWrapper struct {
	CustomGetExportReportStatus         func(exportID string) (*wrappers.ExportPollingResponse, error)
	CustomGetScaPackageCollectionExport func(fileURL string, auth bool) (*wrappers.ScaPackageCollectionExport, error)
}

// GenerateSbomReport mock for tests
func (*ExportMockWrapper) InitiateExportRequest(payload *wrappers.ExportRequestPayload) (*wrappers.ExportResponse, error) {
	if payload.ScanID == "err-scan-id" {
		return nil, errors.New("error")
	}
	return &wrappers.ExportResponse{
		ExportID: "id123456",
	}, nil
}

// GetSbomReportStatus mock for tests
func (e *ExportMockWrapper) GetExportReportStatus(exportID string) (*wrappers.ExportPollingResponse, error) {
	if e.CustomGetExportReportStatus != nil {
		return e.CustomGetExportReportStatus(exportID)
	}
	return &wrappers.ExportPollingResponse{
		ExportID:     "id1234",
		ExportStatus: "Completed",
		FileURL:      "url",
	}, nil
}

// DownloadSbomReport mock for tests
func (*ExportMockWrapper) DownloadExportReport(_, targetFile string) error {
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

func (e *ExportMockWrapper) GetScaPackageCollectionExport(fileURL string, auth bool) (*wrappers.ScaPackageCollectionExport, error) {
	if e.CustomGetScaPackageCollectionExport != nil {
		return e.CustomGetScaPackageCollectionExport(fileURL, auth)
	}
	return &wrappers.ScaPackageCollectionExport{}, nil
}
