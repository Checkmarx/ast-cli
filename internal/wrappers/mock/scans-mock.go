package mock

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/wrappers"

	"github.com/google/uuid"

	scansRESTApi "github.com/checkmarxDev/scans/pkg/api/scans/rest/v1"
)

type ScansMockWrapper struct {
}

func (m *ScansMockWrapper) GetWorkflowByID(_ string) ([]*wrappers.ScanTaskResponseModel, *scansRESTApi.ErrorModel, error) {
	return nil, nil, nil
}

func (m *ScansMockWrapper) Create(_ *scansRESTApi.Scan) (*scansRESTApi.ScanResponseModel, *scansRESTApi.ErrorModel, error) {
	fmt.Println("Called Create in ScansMockWrapper")
	return &scansRESTApi.ScanResponseModel{
		ID:     uuid.New().String(),
		Status: "MOCK",
	}, nil, nil
}

func (m *ScansMockWrapper) Get(_ map[string]string) (*scansRESTApi.ScansCollectionResponseModel, *scansRESTApi.ErrorModel, error) {
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
		Status: "Completed",
	}, nil, nil
}

func (m *ScansMockWrapper) Delete(_ string) (*scansRESTApi.ErrorModel, error) {
	fmt.Println("Called Delete in ScansMockWrapper")
	return nil, nil
}

func (m *ScansMockWrapper) Cancel(string) (*scansRESTApi.ErrorModel, error) {
	fmt.Println("Called Cancel in ScansMockWrapper")
	return nil, nil
}

func (m *ScansMockWrapper) Tags() (map[string][]string, *scansRESTApi.ErrorModel, error) {
	fmt.Println("Called Tags in ScansMockWrapper")
	return map[string][]string{"t1": {"v1"}}, nil, nil
}
