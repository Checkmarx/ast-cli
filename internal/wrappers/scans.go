package wrappers

import (
	scansRESTApi "github.com/checkmarxDev/scans/pkg/api/scans/v1/rest"
)

type ScansWrapper interface {
	Create(model *scansRESTApi.Scan) (*scansRESTApi.ScanResponseModel, *scansRESTApi.ErrorModel, error)
	Get(limit, offset uint64) (*scansRESTApi.SlicedScansResponseModel, *scansRESTApi.ErrorModel, error)
	GetByID(scanID string) (*scansRESTApi.ScanResponseModel, *scansRESTApi.ErrorModel, error)
	Delete(scanID string) (*scansRESTApi.ErrorModel, error)
	Tags() (*[]string, *scansRESTApi.ErrorModel, error)
}
