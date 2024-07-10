package mock

import (
	"os"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

type ExportMockWrapper struct {
	ExportPath string
}

func NewExportMockWrapper(exportPath string) wrappers.ExportWrapper {
	return &ExportMockWrapper{
		ExportPath: exportPath,
	}
}

func (e *ExportMockWrapper) InitiateExportRequest(scanID, format string) (string, error) {
	return "1234-5678-9111-2131", nil
}

func (e *ExportMockWrapper) CheckExportStatus(exportID string) (string, error) {
	return "Complete", nil
}

func (e *ExportMockWrapper) GetScaPackageCollectionExport(fileURL string) (*wrappers.ScaPackageCollectionExport, error) {
	return &wrappers.ScaPackageCollectionExport{}, nil
}

func (e *ExportMockWrapper) DownloadExportReport(reportURL, targetFile string) error {
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
