package mock

import (
	"fmt"

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
		TotalCount: 7,
		Results: []*wrappers.ScanResult{
			{
				Type:     "sast",
				ID:       "1",
				Severity: "high",
				ScanResultData: wrappers.ScanResultData{
					LanguageName: "JavaScript",
					QueryName:    "mock-query-name-1",
					Nodes: []*wrappers.ScanResultNode{
						{
							FileName: "dummy-file-name-1",
							Line:     10,
							Column:   10,
							Length:   20,
						},
						{
							FileName: "dummy-file-name-1",
							Line:     11,
							Column:   3,
							Length:   10,
						},
					},
				},
			},
			{
				Type:     "sast",
				ID:       "2",
				Severity: "high",
				ScanResultData: wrappers.ScanResultData{
					LanguageName: "Java",
					QueryName:    "mock-query-name-2",
					Nodes: []*wrappers.ScanResultNode{
						{
							FileName: "dummy-file-name-2",
							Line:     10,
							Column:   10,
							Length:   20,
						},
						{
							FileName: "dummy-file-name-2",
							Line:     11,
							Column:   3,
							Length:   10,
						},
					},
				},
			},
			{
				Type:     "sast",
				Severity: "high",
				ID:       "3",
				ScanResultData: wrappers.ScanResultData{
					LanguageName: "Java",
					QueryName:    "mock-query-name-2",
					Nodes: []*wrappers.ScanResultNode{
						{
							FileName: "dummy-file-name-2",
							Line:     10,
							Column:   10,
							Length:   20,
						},
						{
							FileName: "dummy-file-name-2",
							Line:     11,
							Column:   3,
							Length:   10,
						},
						{
							FileName: "dummy-file-name-2",
							Line:     12,
							Column:   3,
							Length:   10,
						},
					},
				},
			},
			{
				Type:     "sast",
				ID:       "4",
				Severity: "high",
				ScanResultData: wrappers.ScanResultData{
					LanguageName: "Java",
					QueryName:    "mock-query-name-3",
					Nodes: []*wrappers.ScanResultNode{
						{
							FileName: "dummy-file-name-3",
							Line:     10,
							Column:   10,
							Length:   20,
						},
						{
							FileName: "dummy-file-name-3",
							Line:     11,
							Column:   3,
							Length:   10,
						},
					},
				},
			},
			{
				Type:     "sast",
				ID:       "5",
				Severity: "high",
				ScanResultData: wrappers.ScanResultData{
					LanguageName: "Java",
					QueryName:    "mock-query-name-3",
					Nodes: []*wrappers.ScanResultNode{
						{
							FileName: "dummy-file-name-4",
							Line:     10,
							Column:   10,
							Length:   20,
						},
						{
							FileName: "dummy-file-name-4",
							Line:     11,
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

func (r ResultsMockWrapper) GetResultsURL(projectID string) (string, error) {
	return fmt.Sprintf("projects/%s/overview", projectID), nil
}
