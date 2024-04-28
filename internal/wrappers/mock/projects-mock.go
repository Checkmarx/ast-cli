package mock

import (
	"fmt"

	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type ProjectsMockWrapper struct{}

func (p *ProjectsMockWrapper) Create(model *wrappers.Project) (
	*wrappers.ProjectResponseModel,
	*wrappers.ErrorModel,
	error) {
	fmt.Println("Called Create in ProjectsMockWrapper")
	if model.Name == "mock-some-error-model" {
		return nil, &wrappers.ErrorModel{
			Message: "some error message",
			Type:    "",
			Code:    1,
		}, fmt.Errorf("some error")
	}
	return &wrappers.ProjectResponseModel{
		ID:             fmt.Sprintf("ID-%s", model.Name),
		Name:           model.Name,
		ApplicationIds: model.ApplicationIds,
	}, nil, nil
}
func (p *ProjectsMockWrapper) Update(projectID string, model *wrappers.Project) error {
	fmt.Println("Called Update in ProjectsMockWrapper")
	return nil
}

func (p *ProjectsMockWrapper) UpdateConfiguration(projectID string, configuration []wrappers.ProjectConfiguration) (*wrappers.ErrorModel, error) {
	fmt.Println("Called Update Configuration for project", projectID, " in ProjectsMockWrapper with the configuration ", configuration)
	return nil, nil
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

func (p *ProjectsMockWrapper) GetByName(name string) (
	*wrappers.ProjectResponseModel,
	*wrappers.ErrorModel,
	error) {
	fmt.Println("Called GetByName in ProjectsMockWrapper")
	if name == "mock-missing-file-path" {
		return nil, nil, fmt.Errorf(errorConstants.ImportFilePathIsRequired)
	}
	if name == "" {
		return nil, nil, fmt.Errorf(errorConstants.ProjectNameIsRequired)
	}
	return &wrappers.ProjectResponseModel{
		ID:   "MOCK",
		Name: name,
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
