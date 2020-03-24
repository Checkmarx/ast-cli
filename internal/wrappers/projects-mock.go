package wrappers

import (
	"fmt"

	projApi "github.com/checkmarxDev/scans/api/v1/rest/projects"
	projModels "github.com/checkmarxDev/scans/pkg/projects"
)

type ProjectsMockWrapper struct{}

func (p *ProjectsMockWrapper) Create(model *projApi.Project) (*projModels.ProjectResponseModel, *projModels.ErrorModel, error) {
	fmt.Println("Called Create in ProjectsMockWrapper")
	return &projModels.ProjectResponseModel{
		ID: model.ID,
	}, nil, nil
}

func (p *ProjectsMockWrapper) Get() (*projModels.SlicedProjectsResponseModel, *projModels.ErrorModel, error) {
	fmt.Println("Called Get in ProjectsMockWrapper")
	return &projModels.SlicedProjectsResponseModel{
		Projects: []projModels.ProjectResponseModel{
			{
				ID: "MOCK",
			},
		},
	}, nil, nil
}

func (p *ProjectsMockWrapper) GetByID(projectID string) (*projModels.ProjectResponseModel, *projModels.ErrorModel, error) {
	fmt.Println("Called GetByID in ProjectsMockWrapper")
	return &projModels.ProjectResponseModel{
		ID: projectID,
	}, nil, nil
}

func (p *ProjectsMockWrapper) Delete(projectID string) (*projModels.ProjectResponseModel, *projModels.ErrorModel, error) {
	fmt.Println("Called Delete in ProjectsMockWrapper")
	return &projModels.ProjectResponseModel{
		ID: projectID,
	}, nil, nil
}

func (p *ProjectsMockWrapper) Tags() (*[]string, *projModels.ErrorModel, error) {
	fmt.Println("Called Tags in ProjectsMockWrapper")
	return &[]string{"t1"}, nil, nil
}
