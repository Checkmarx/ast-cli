package mock

import (
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type ResultsMockWrapper struct{}

func (r ResultsMockWrapper) GetAllResultsTypeByScanID(params map[string]string) (*[]wrappers.ScaTypeCollection, *wrappers.WebError, error) {
	const mock = "mock"
	var scaTypes = []wrappers.ScaTypeCollection{{
		ID:   mock,
		Type: mock,
	}, {
		ID:   mock,
		Type: mock,
	}}
	return &scaTypes, nil, nil
}

func (r ResultsMockWrapper) GetAllResultsPackageByScanID(params map[string]string) (*[]wrappers.ScaPackageCollection, *wrappers.WebError, error) {
	const mock = "mock"
	var dependencyPath = wrappers.DependencyPath{ID: mock, Name: mock, Version: mock, IsResolved: true, IsDevelopment: false, Locations: nil}
	var dependencyArray = [][]wrappers.DependencyPath{{dependencyPath}}

	dependencyArray[0][0] = dependencyPath
	var scaPackages = []wrappers.ScaPackageCollection{{
		ID:                  mock,
		FixLink:             mock,
		Locations:           nil,
		DependencyPathArray: dependencyArray,
		Outdated:            false,
	}, {
		ID:                  mock,
		FixLink:             mock,
		Locations:           nil,
		DependencyPathArray: dependencyArray,
		Outdated:            false,
	}}
	return &scaPackages, nil, nil
}

func (r ResultsMockWrapper) GetAllResultsByScanID(_ map[string]string) (
	*wrappers.ScanResultsCollection,
	*wrappers.WebError,
	error,
) {
	const mock = "mock"
	var dependencyPath = wrappers.DependencyPath{ID: mock, Name: mock, Version: mock, IsResolved: true, IsDevelopment: false, Locations: nil}
	var dependencyArray = [][]wrappers.DependencyPath{{dependencyPath}}
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
					ScaPackageCollection: &wrappers.ScaPackageCollection{
						ID:                  "mock",
						FixLink:             "mock",
						Locations:           nil,
						DependencyPathArray: dependencyArray,
						Outdated:            false,
					},
					PackageIdentifier: "mock",
					QueryID:           12.4,
					QueryName:         "mock-query-name",
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
