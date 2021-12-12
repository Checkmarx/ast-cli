package mock

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/wrappers"

	"github.com/google/uuid"
)

type ScansMockWrapper struct {
}

func (m *ScansMockWrapper) GetWorkflowByID(_ string) ([]*wrappers.ScanTaskResponseModel, *ErrorModel, error) {
	return nil, nil, nil
}

func (m *ScansMockWrapper) Create(_ *Scan) (*ScanResponseModel, *ErrorModel, error) {
	fmt.Println("Called Create in ScansMockWrapper")
	return &ScanResponseModel{
		ID:     uuid.New().String(),
		Status: "MOCK",
	}, nil, nil
}

func (m *ScansMockWrapper) Get(_ map[string]string) (*ScansCollectionResponseModel, *ErrorModel, error) {
	fmt.Println("Called Get in ScansMockWrapper")
	return &ScansCollectionResponseModel{
		Scans: []ScanResponseModel{
			{
				ID:     "MOCK",
				Status: "STATUS",
			},
		},
	}, nil, nil
}

func (m *ScansMockWrapper) GetByID(scanID string) (*ScanResponseModel, *ErrorModel, error) {
	fmt.Println("Called GetByID in ScansMockWrapper")
	return &ScanResponseModel{
		ID:     scanID,
		Status: "Completed",
	}, nil, nil
}

func (m *ScansMockWrapper) Delete(_ string) (*ErrorModel, error) {
	fmt.Println("Called Delete in ScansMockWrapper")
	return nil, nil
}

func (m *ScansMockWrapper) Cancel(string) (*ErrorModel, error) {
	fmt.Println("Called Cancel in ScansMockWrapper")
	return nil, nil
}

func (m *ScansMockWrapper) Tags() (map[string][]string, *ErrorModel, error) {
	fmt.Println("Called Tags in ScansMockWrapper")
	return map[string][]string{"t1": {"v1"}}, nil, nil
}
