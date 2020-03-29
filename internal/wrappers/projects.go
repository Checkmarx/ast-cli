package wrappers

import (
	projectsRESTApi "github.com/checkmarxDev/scans/api/v1/rest/projects"
)

type ProjectsWrapper interface {
	Create(model *projectsRESTApi.Project) (*projectsRESTApi.ProjectResponseModel, *projectsRESTApi.ErrorModel, error)
	Get() (*projectsRESTApi.SlicedProjectsResponseModel, *projectsRESTApi.ErrorModel, error)
	GetByID(projectID string) (*projectsRESTApi.ProjectResponseModel, *projectsRESTApi.ErrorModel, error)
	Delete(projectID string) (*projectsRESTApi.ErrorModel, error)
	Tags() (*[]string, *projectsRESTApi.ErrorModel, error)
}
