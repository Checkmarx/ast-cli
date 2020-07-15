package wrappers

import (
	"time"

	scansRESTApi "github.com/checkmarxDev/scans/pkg/api/scans/v1/rest"
)

type ScanTaskResponseModel struct {
	Id          string     `json:"id"`
	State       string     `json:"state"`
	TaskType    string     `json:"taskType"`
	StartedTime *time.Time `json:"startedTime"`
	EndTime     *time.Time `json:"endTime"`
	Info        string     `json:"info"`
}

type ScansWrapper interface {
	Create(model *scansRESTApi.Scan) (*scansRESTApi.ScanResponseModel, *scansRESTApi.ErrorModel, error)
	Get(params map[string]string) (*scansRESTApi.ScansCollectionResponseModel, *scansRESTApi.ErrorModel, error)
	GetByID(scanID string) (*scansRESTApi.ScanResponseModel, *scansRESTApi.ErrorModel, error)
	GetWorkflowByID(scanID string) ([]*ScanTaskResponseModel, *scansRESTApi.ErrorModel, error)
	Delete(scanID string) (*scansRESTApi.ErrorModel, error)
	Tags() (map[string][]string, *scansRESTApi.ErrorModel, error)
}
