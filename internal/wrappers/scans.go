package wrappers

import (
	scansRESTApi "github.com/checkmarxDev/scans/pkg/api/scans/rest/v1"
)

type ScanTaskResponseModel struct {
	Source    string `json:"source"`
	Timestamp string `json:"timestamp"`
	Info      string `json:"info"`
}

type ScansWrapper interface {
	Create(model *scansRESTApi.Scan) (*scansRESTApi.ScanResponseModel, *scansRESTApi.ErrorModel, error)
	Get(params map[string]string) (*scansRESTApi.ScansCollectionResponseModel, *scansRESTApi.ErrorModel, error)
	GetByID(scanID string) (*scansRESTApi.ScanResponseModel, *scansRESTApi.ErrorModel, error)
	GetWorkflowByID(scanID string) ([]*ScanTaskResponseModel, *scansRESTApi.ErrorModel, error)
	Delete(scanID string) (*scansRESTApi.ErrorModel, error)
	Tags() (map[string][]string, *scansRESTApi.ErrorModel, error)
}
