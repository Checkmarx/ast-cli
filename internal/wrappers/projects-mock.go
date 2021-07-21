package wrappers

import (
	"fmt"

	projectsRESTApi "github.com/checkmarxDev/scans/pkg/api/projects/v1/rest"
)

type ProjectsMockWrapper struct{}

func (p *ProjectsMockWrapper) Create(model *projectsRESTApi.Project) (
	*projectsRESTApi.ProjectResponseModel,
	*projectsRESTApi.ErrorModel,
	error) {
	fmt.Println("Called Create in ProjectsMockWrapper")
	return &projectsRESTApi.ProjectResponseModel{
		Name: model.Name,
	}, nil, nil
}

func (p *ProjectsMockWrapper) Get(_ map[string]string) (
	*projectsRESTApi.ProjectsCollectionResponseModel,
	*projectsRESTApi.ErrorModel,
	error) {
	fmt.Println("Called Get in ProjectsMockWrapper")
	return &projectsRESTApi.ProjectsCollectionResponseModel{
		Projects: []projectsRESTApi.ProjectResponseModel{
			{
				ID: "MOCK",
			},
		},
	}, nil, nil
}

func (p *ProjectsMockWrapper) GetByID(projectID string) (
	*projectsRESTApi.ProjectResponseModel,
	*projectsRESTApi.ErrorModel,
	error) {
	fmt.Println("Called GetByID in ProjectsMockWrapper")
	return &projectsRESTApi.ProjectResponseModel{
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

func (p *ProjectsMockWrapper) Delete(_ string) (
	*projectsRESTApi.ErrorModel,
	error) {
	fmt.Println("Called Delete in ProjectsMockWrapper")
	return nil, nil
}

func (p *ProjectsMockWrapper) Tags() (
	map[string][]string,
	*projectsRESTApi.ErrorModel,
	error) {
	fmt.Println("Called Tags in ProjectsMockWrapper")
	return map[string][]string{
		"t1": {
			"v1",
		},
	}, nil, nil
}
