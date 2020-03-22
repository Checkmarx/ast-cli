package wrappers

import (
	"fmt"

	scansApi "github.com/checkmarxDev/scans/api/v1/rest/scans"
	scansModels "github.com/checkmarxDev/scans/pkg/scans"
)

type ScansMockWrapper struct {
}

func (m *ScansMockWrapper) Create(model *scansApi.Scan) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error) {
	fmt.Println("Called Create in ScansMockWrapper")
	return &scansModels.ScanResponseModel{
		ID:     model.ScanID,
		Status: "MOCK",
	}, nil, nil
}

func (m *ScansMockWrapper) Get() (*scansModels.ResponseModel, *scansModels.ErrorModel, error) {
	fmt.Println("Called Get in ScansMockWrapper")
	return &scansModels.ResponseModel{
		Scans: []scansModels.ScanResponseModel{
			{
				ID:     "MOCK",
				Status: "STATUS",
			},
		},
	}, nil, nil
}

func (m *ScansMockWrapper) GetByID(scanID string) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error) {
	fmt.Println("Called GetByID in ScansMockWrapper")
	return &scansModels.ScanResponseModel{
		ID:     scanID,
		Status: "STATUS",
	}, nil, nil
}

func (m *ScansMockWrapper) Delete(scanID string) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error) {
	fmt.Println("Called Delete in ScansMockWrapper")
	return &scansModels.ScanResponseModel{
		ID:     scanID,
		Status: "STATUS",
	}, nil, nil
}
