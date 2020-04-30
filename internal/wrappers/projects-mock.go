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
		ID: model.ID,
	}, nil, nil
}

func (p *ProjectsMockWrapper) Get(params map[string]string) (
	*projectsRESTApi.SlicedProjectsResponseModel,
	*projectsRESTApi.ErrorModel,
	error) {
	fmt.Println("Called Get in ProjectsMockWrapper")
	return &projectsRESTApi.SlicedProjectsResponseModel{
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
	}, nil, nil
}

func (p *ProjectsMockWrapper) Delete(projectID string) (
	*projectsRESTApi.ErrorModel,
	error) {
	fmt.Println("Called Delete in ProjectsMockWrapper")
	return nil, nil
}

func (p *ProjectsMockWrapper) Tags() (
	*[]string,
	*projectsRESTApi.ErrorModel,
	error) {
	fmt.Println("Called Tags in ProjectsMockWrapper")
	return &[]string{"t1"}, nil, nil
}
