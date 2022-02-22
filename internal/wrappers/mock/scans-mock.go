package mock

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/wrappers"

	"github.com/google/uuid"
)

type ScansMockWrapper struct {
	Running bool
}

func (m *ScansMockWrapper) GetWorkflowByID(_ string) ([]*wrappers.ScanTaskResponseModel, *wrappers.ErrorModel, error) {
	return nil, nil, nil
}

func (m *ScansMockWrapper) Create(_ *wrappers.Scan) (*wrappers.ScanResponseModel, *wrappers.ErrorModel, error) {
	fmt.Println("Called Create in ScansMockWrapper")
	return &wrappers.ScanResponseModel{
		ID:     uuid.New().String(),
		Status: "MOCK",
	}, nil, nil
}

func (m *ScansMockWrapper) Get(_ map[string]string) (
	*wrappers.ScansCollectionResponseModel,
	*wrappers.ErrorModel,
	error,
) {
	fmt.Println("Called Get in ScansMockWrapper")
	return &wrappers.ScansCollectionResponseModel{
		Scans: []wrappers.ScanResponseModel{
			{
				ID:     "MOCK",
				Status: "STATUS",
			},
		},
	}, nil, nil
}

func (m *ScansMockWrapper) GetByID(scanID string) (*wrappers.ScanResponseModel, *wrappers.ErrorModel, error) {
	fmt.Println("Called GetByID in ScansMockWrapper")
	var status wrappers.ScanStatus = "Completed"
	if m.Running {
		status = "Running"
	}
	m.Running = !m.Running
	return &wrappers.ScanResponseModel{
		ID:     scanID,
		Status: status,
	}, nil, nil
}

func (m *ScansMockWrapper) Delete(_ string) (*wrappers.ErrorModel, error) {
	fmt.Println("Called Delete in ScansMockWrapper")
	return nil, nil
}

func (m *ScansMockWrapper) Cancel(string) (*wrappers.ErrorModel, error) {
	fmt.Println("Called Cancel in ScansMockWrapper")
	return nil, nil
}

func (m *ScansMockWrapper) Tags() (map[string][]string, *wrappers.ErrorModel, error) {
	fmt.Println("Called Tags in ScansMockWrapper")
	return map[string][]string{"t1": {"v1"}}, nil, nil
}
