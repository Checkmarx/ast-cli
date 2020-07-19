package wrappers

import (
	scansRESTApi "github.com/checkmarxDev/scans/pkg/api/scans/rest/v1"
)

type ScansWrapper interface {
	Create(model *scansRESTApi.Scan) (*scansRESTApi.ScanResponseModel, *scansRESTApi.ErrorModel, error)
	Get(params map[string]string) (*scansRESTApi.ScansCollectionResponseModel, *scansRESTApi.ErrorModel, error)
	GetByID(scanID string) (*scansRESTApi.ScanResponseModel, *scansRESTApi.ErrorModel, error)
	Delete(scanID string) (*scansRESTApi.ErrorModel, error)
	Tags() (map[string][]string, *scansRESTApi.ErrorModel, error)
}
