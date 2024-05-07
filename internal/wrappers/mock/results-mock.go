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

var containersResults = &wrappers.ScanResult{
	Type:     "containers",
	Severity: "medium",
	ScanResultData: wrappers.ScanResultData{
		PackageName:       "image-mock",
		PackageVersion:    "1.1",
		ImageName:         "image-mock",
		ImageTag:          "1.1",
		ImageFilePath:     "DockerFile",
		ImageOrigin:       "Docker",
		PackageIdentifier: "mock",
		QueryID:           12.4,
		QueryName:         "mock-query-name",
	},
	Description: "mock-description",
	VulnerabilityDetails: wrappers.VulnerabilityDetails{
		CvssScore: 4.5,
		CveName:   "CVE-2021-1234",
		CweID:     "CWE-1234",
	},
}

func (r ResultsMockWrapper) GetAllResultsByScanID(params map[string]string) (
	*wrappers.ScanResultsCollection,
	*wrappers.WebError,
	error,
) {
	if params["scan-id"] == "CONTAINERS_ONLY" {
		return &wrappers.ScanResultsCollection{
			TotalCount: 1,
			Results: []*wrappers.ScanResult{
				containersResults,
			},
		}, nil, nil
	}
	const mock = "mock"
	var dependencyPath = wrappers.DependencyPath{ID: mock, Name: mock, Version: mock, IsResolved: true, IsDevelopment: false, Locations: nil}
	var dependencyArray = [][]wrappers.DependencyPath{{dependencyPath}}
	return &wrappers.ScanResultsCollection{
		TotalCount: 8,
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
			containersResults,
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

func (r ResultsMockWrapper) GetScanSummariesByScanIDS(params map[string]string) (*wrappers.ScanSummariesModel, *wrappers.WebError, error) {
	if params["scan-ids"] == "MOCKWEBERR" {
		return nil, &wrappers.WebError{
			Message: "web error",
		}, nil
	}
	if params["scan-ids"] == "MOCKERR" {
		return nil, nil, fmt.Errorf("mock error")
	}
	return &wrappers.ScanSummariesModel{
		ScansSummaries: []wrappers.ScanSumaries{
			{
				SastCounters: wrappers.SastCounters{
					SeverityCounters: []wrappers.SeverityCounters{
						{
							Severity: "info",
							Counter:  1,
						},
						{
							Severity: "low",
							Counter:  1,
						},
						{
							Severity: "medium",
							Counter:  1,
						},
						{
							Severity: "high",
							Counter:  1,
						},
					},
					TotalCounter:        4,
					FilesScannedCounter: 1,
				},
				KicsCounters: wrappers.KicsCounters{
					SeverityCounters: []wrappers.SeverityCounters{
						{
							Severity: "info",
							Counter:  1,
						},
						{
							Severity: "low",
							Counter:  1,
						},
						{
							Severity: "medium",
							Counter:  1,
						},
						{
							Severity: "high",
							Counter:  1,
						},
					},

					TotalCounter:        4,
					FilesScannedCounter: 1,
				},
				ScaCounters: wrappers.ScaCounters{
					SeverityCounters: []wrappers.SeverityCounters{
						{
							Severity: "info",
							Counter:  1,
						},
						{
							Severity: "low",
							Counter:  1,
						},
						{
							Severity: "medium",
							Counter:  1,
						},
						{
							Severity: "high",
							Counter:  1,
						},
					},

					TotalCounter:        4,
					FilesScannedCounter: 1,
				},
			},
		},
	}, nil, nil
}
