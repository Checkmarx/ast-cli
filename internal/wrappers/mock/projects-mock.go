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

	var model *wrappers.ProjectsCollectionResponseModel
	switch name := params["names"]; name {
	case "fake-kics-scanner-fail":
		model = getProjectResponseModel(fmt.Sprintf("%s-id", name), name, filteredTotalCount)
	case "fake-multiple-scanner-fails":
		model = getProjectResponseModel(fmt.Sprintf("%s-id", name), name, filteredTotalCount)
	case "fake-sca-fail-partial":
		model = getProjectResponseModel(fmt.Sprintf("%s-id", name), name, filteredTotalCount)
	case "fake-kics-fail-sast-canceled":
		model = getProjectResponseModel(fmt.Sprintf("%s-id", name), name, filteredTotalCount)
	case "existing-project":
		model = getProjectResponseModel(fmt.Sprintf("%s-id", name), name, filteredTotalCount)
	case "non-existing-project":
		model = nil
	case "error-project":
		return nil, nil, fmt.Errorf("some error")
	case "test_project1":
		model = getProjectResponseModel("1", "test_project1", filteredTotalCount)
	case "test_project2":
		model = getProjectResponseModel("2", "test_project2", filteredTotalCount)
	case "exact_project":
		model = getProjectResponseModel("3", "exact_project", filteredTotalCount)
	case "test_project3":
		model = getProjectResponseModel("4", "test_project3", filteredTotalCount)
	default:
		model = getProjectResponseModel("MOCK", "MOCK", filteredTotalCount)
	}

	return model, nil, nil
}

func getProjectResponseModel(id, name string, filteredTotalCount int) *wrappers.ProjectsCollectionResponseModel {
	return &wrappers.ProjectsCollectionResponseModel{
		TotalCount:         1,
		FilteredTotalCount: uint(filteredTotalCount),
		Projects: []wrappers.ProjectResponseModel{
			{
				ID:   id,
				Name: name,
			},
		},
	}
}

func (p *ProjectsMockWrapper) GetByID(projectID string) (
	*wrappers.ProjectResponseModel,
	*wrappers.ErrorModel,
	error) {
	if projectID == "ID-mock-some-error-model" {
		return nil, &wrappers.ErrorModel{Code: 202, Message: "some-message"}, nil
	}
	fmt.Println("Called GetByID in ProjectsMockWrapper")
	return &wrappers.ProjectResponseModel{
		ID:   projectID,
		Name: "MOCK",
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
