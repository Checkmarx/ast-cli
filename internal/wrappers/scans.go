package wrappers

import (
	scansRESTApi "github.com/checkmarxDev/scans/api/v1/rest/scans"
)

type ScansWrapper interface {
	Create(model *scansRESTApi.Scan) (*scansRESTApi.ScanResponseModel, *scansRESTApi.ErrorModel, error)
	Get() (*scansRESTApi.SlicedScansResponseModel, *scansRESTApi.ErrorModel, error)
	GetByID(scanID string) (*scansRESTApi.ScanResponseModel, *scansRESTApi.ErrorModel, error)
	Delete(scanID string) (*scansRESTApi.ScanResponseModel, *scansRESTApi.ErrorModel, error)
	Tags() (*[]string, *scansRESTApi.ErrorModel, error)
}
