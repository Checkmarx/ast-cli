package wrappers

import (
	"fmt"

	scansRESTApi "github.com/checkmarxDev/scans/pkg/api/scans/v1/rest"
)

type ScansMockWrapper struct {
}

func (m *ScansMockWrapper) Create(model *scansRESTApi.Scan) (*scansRESTApi.ScanResponseModel, *scansRESTApi.ErrorModel, error) {
	fmt.Println("Called Create in ScansMockWrapper")
	return &scansRESTApi.ScanResponseModel{
		ID:     model.ScanID,
		Status: "MOCK",
	}, nil, nil
}

func (m *ScansMockWrapper) Get(limit, offset uint64) (*scansRESTApi.SlicedScansResponseModel, *scansRESTApi.ErrorModel, error) {
	fmt.Println("Called Get in ScansMockWrapper")
	return &scansRESTApi.SlicedScansResponseModel{
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

func (m *ScansMockWrapper) Tags() (*[]string, *scansRESTApi.ErrorModel, error) {
	fmt.Println("Called Tags in ScansMockWrapper")
	return &[]string{"t1"}, nil, nil
}
