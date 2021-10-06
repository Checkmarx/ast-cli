package mock

import (
	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	resultsReader "github.com/checkmarxDev/sast-results/pkg/reader"
	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"
	resultsRaw "github.com/checkmarxDev/sast-results/pkg/web/path/raw"
)

type ResultsMockWrapper struct{}

func (r ResultsMockWrapper) GetByScanID(_ map[string]string) (*resultsRaw.ResultsCollection, *resultsHelpers.WebError, error) {
	const mock = "MOCK"
	return &resultsRaw.ResultsCollection{
		Results: []*resultsReader.Result{
			{
				ResultQuery: resultsReader.ResultQuery{
					QueryID:   0,
					QueryName: mock,
					Severity:  mock,
				},
				PathSystemID: mock,
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

func (r ResultsMockWrapper) GetAllResultsByScanID(_ map[string]string) (*wrappers.ScanResultsCollection, *resultsHelpers.WebError, error) {
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
				Type:     "dependency",
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
				Type:     "infrastructure",
				Severity: "low",
			},
		},
	}, nil, nil
}
