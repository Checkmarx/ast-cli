package wrappers

import (
	scansApi "github.com/checkmarxDev/scans/api/v1/rest/scans"
	scansModels "github.com/checkmarxDev/scans/pkg/scans"
)

type ScansWrapper interface {
	Create(model *scansApi.Scan) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error)
	Get() (*scansModels.SlicedScansResponseModel, *scansModels.ErrorModel, error)
	GetByID(scanID string) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error)
	Delete(scanID string) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error)
	Tags() (*[]string, *scansModels.ErrorModel, error)
}
