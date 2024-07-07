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

func (e *ExportMockWrapper) GetExportPackage(scanID string) (*wrappers.ScaPackageCollectionExport, error) {
	return &wrappers.ScaPackageCollectionExport{}, nil
}
