package mock

import (
	"fmt"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
)

type ProjectsMockWrapper struct{}

func (p *ProjectsMockWrapper) Create(model *wrappers.Project) (
	*wrappers.ProjectResponseModel,
	*wrappers.ErrorModel,
	error) {
	fmt.Println("Called Create in ProjectsMockWrapper")
	return &wrappers.ProjectResponseModel{
		Name: model.Name,
	}, nil, nil
}

func (p *ProjectsMockWrapper) UpdateConfiguration(projectID string, configuration []wrappers.ProjectConfiguration) (*wrappers.ErrorModel, error) {
	fmt.Println("Called Update Configuration for project", projectID, " in ProjectsMockWrapper with the configuration ", configuration)
	return nil, nil
}
func (p *ProjectsMockWrapper) GetConfiguration(projectID string) (*[]wrappers.ProjectConfiguration, *wrappers.ErrorModel, error) {
	fmt.Println("Called Get Configuration for project", projectID, " in ProjectsMockWrapper")
	projectConfig := &[]wrappers.ProjectConfiguration{wrappers.ProjectConfiguration{}}

	if projectID == "FORCE-ERROR-MOCK" {
		return projectConfig, nil, errors.New("error message")
	}
	if projectID == "FORCE-ERROR-MODEL-MOCK" {
		return projectConfig, &wrappers.ErrorModel{
			Code:    400,
			Message: "error model message",
		}, nil
	}

	if projectID == "MOCK" {
		projectConfig = &[]wrappers.ProjectConfiguration{wrappers.ProjectConfiguration{
			Key:           "scan.handler.git.repository",
			Name:          "repository",
			Category:      "git",
			OriginLevel:   "Project",
			Value:         "https://github.com/dummyuser/dummy_project.git",
			ValueType:     "String",
			AllowOverride: true,
		}}
	}

	return projectConfig, nil, nil
}

func (p *ProjectsMockWrapper) Get(params map[string]string) (
	*wrappers.ProjectsCollectionResponseModel,
	*wrappers.ErrorModel,
	error) {
	fmt.Println("Called Get in ProjectsMockWrapper")

	filteredTotalCount := 1

	if params["name"] == "MOCK-NO-FILTERED-PROJECTS" {
		filteredTotalCount = 0
	}
	if params["name"] == "FORCE-ERROR-MOCK" {
		return &wrappers.ProjectsCollectionResponseModel{
			FilteredTotalCount: uint(filteredTotalCount),
			Projects: []wrappers.ProjectResponseModel{
				{
					ID:   "FORCE-ERROR-MOCK",
					Name: "FORCE-ERROR-MOCK",
				},
			},
		}, nil, nil
	}
	if params["name"] == "FORCE-ERROR-MODEL-MOCK" {
		return &wrappers.ProjectsCollectionResponseModel{
			FilteredTotalCount: uint(filteredTotalCount),
			Projects: []wrappers.ProjectResponseModel{
				{
					ID:   "FORCE-ERROR-MODEL-MOCK",
					Name: "FORCE-ERROR-MODEL-MOCK",
				},
			},
		}, nil, nil
	}

	return &wrappers.ProjectsCollectionResponseModel{
		FilteredTotalCount: uint(filteredTotalCount),
		Projects: []wrappers.ProjectResponseModel{
			{
				ID:   "MOCK",
				Name: "MOCK",
			},
		},
	}, nil, nil
}

func (p *ProjectsMockWrapper) GetByID(projectID string) (
	*wrappers.ProjectResponseModel,
	*wrappers.ErrorModel,
	error) {
	fmt.Println("Called GetByID in ProjectsMockWrapper")
	return &wrappers.ProjectResponseModel{
		ID: projectID,
		Tags: map[string]string{
			"a": "b",
			"c": "d",
		},
		Groups: []string{
			"a",
			"b",
		},
	}, nil, nil
}

func (p *ProjectsMockWrapper) GetBranchesByID(_ string, _ map[string]string) (
	[]string,
	*wrappers.ErrorModel,
	error) {
	fmt.Println("Called GetBranchesByID in ProjectsMockWrapper")
	return []string{
		"master",
		"feature/MOCK",
	}, nil, nil
}

func (p *ProjectsMockWrapper) Delete(_ string) (
	*wrappers.ErrorModel,
	error) {
	fmt.Println("Called Delete in ProjectsMockWrapper")
	return nil, nil
}

func (p *ProjectsMockWrapper) Tags() (
	map[string][]string,
	*wrappers.ErrorModel,
	error) {
	fmt.Println("Called Tags in ProjectsMockWrapper")
	return map[string][]string{
		"t1": {
			"v1",
		},
	}, nil, nil
}
