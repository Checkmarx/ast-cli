package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type ResultsMockWrapper struct{}

func (r ResultsMockWrapper) GetAllResultsPackageByScanID(params map[string]string) (*[]wrappers.ScaPackageCollection, *wrappers.WebError, error) {
	var scaPackage []wrappers.ScaPackageCollection
	return &scaPackage, nil, nil
}

func (r ResultsMockWrapper) GetByScanID(_ map[string]string) (
	*wrappers.ScanResultsCollection,
	*wrappers.WebError,
	error,
) {
	const mock = "MOCK"
	return &wrappers.ScanResultsCollection{
		Results: []*wrappers.ScanResult{
			{
				ID:           mock,
				FirstScanID:  mock,
				FirstFoundAt: mock,
				FoundAt:      mock,
				Status:       mock,
			},
		},
		TotalCount: 1,
	}, nil, nil
}

func (r ResultsMockWrapper) GetAllResultsByScanID(_ map[string]string) (
	*wrappers.ScanResultsCollection,
	*wrappers.WebError,
	error,
) {
	return &wrappers.ScanResultsCollection{
		TotalCount: 3,
		Results: []*wrappers.ScanResult{
			{
				Type:     "sast",
				Severity: "high",
				ScanResultData: wrappers.ScanResultData{
					Nodes: []*wrappers.ScanResultNode{
						{
							FileName: "dummy-file-name",
							Line:     10,
							Column:   10,
							Length:   20,
						},
						{
							FileName: "dummy-file-name",
							Line:     0,
							Column:   3,
							Length:   10,
						},
					},
				},
			},
			{
				Type:     "sca",
				Severity: "medium",
				ScanResultData: wrappers.ScanResultData{
					QueryID:   12.4,
					QueryName: "mock-query-name",
					Nodes: []*wrappers.ScanResultNode{
						{
							FileName: "dummy-file-name",
							Line:     10,
							Column:   10,
							Length:   20,
						},
						{
							FileName: "dummy-file-name",
							Line:     0,
							Column:   3,
							Length:   10,
						},
					},
				},
			},
			{
				Type:     "kics",
				Severity: "low",
			},
		},
	}, nil, nil
}
