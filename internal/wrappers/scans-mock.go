package wrappers

import (
	"fmt"

	"github.com/google/uuid"

	scansRESTApi "github.com/checkmarxDev/scans/pkg/api/scans/v1/rest"
)

type ScansMockWrapper struct {
}

func (m *ScansMockWrapper) GetWorkflowByID(scanID string) ([]*ScanTaskResponseModel, *scansRESTApi.ErrorModel, error) {
	return nil, nil, nil
}

func (m *ScansMockWrapper) Create(model *scansRESTApi.Scan) (*scansRESTApi.ScanResponseModel, *scansRESTApi.ErrorModel, error) {
	fmt.Println("Called Create in ScansMockWrapper")
	return &scansRESTApi.ScanResponseModel{
		ID:     uuid.New().String(),
		Status: "MOCK",
	}, nil, nil
}

func (m *ScansMockWrapper) Get(params map[string]string) (*scansRESTApi.ScansCollectionResponseModel, *scansRESTApi.ErrorModel, error) {
	fmt.Println("Called Get in ScansMockWrapper")
	return &scansRESTApi.ScansCollectionResponseModel{
		Scans: []scansRESTApi.ScanResponseModel{
			{
				ID:     "MOCK",
				Status: "STATUS",
			},
		},
	}, nil, nil
}

func (m *ScansMockWrapper) GetByID(scanID string) (*scansRESTApi.ScanResponseModel, *scansRESTApi.ErrorModel, error) {
	fmt.Println("Called GetByID in ScansMockWrapper")
	return &scansRESTApi.ScanResponseModel{
		ID:     scanID,
		Status: "STATUS",
	}, nil, nil
}

func (m *ScansMockWrapper) Delete(scanID string) (*scansRESTApi.ErrorModel, error) {
	fmt.Println("Called Delete in ScansMockWrapper")
	return nil, nil
}

func (m *ScansMockWrapper) Tags() (map[string][]string, *scansRESTApi.ErrorModel, error) {
	fmt.Println("Called Tags in ScansMockWrapper")
	return map[string][]string{"t1": {"v1"}}, nil, nil
}
