package mock

import "github.com/checkmarx/ast-cli/internal/wrappers"

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
	return nil
}
