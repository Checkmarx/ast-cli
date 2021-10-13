package wrappers

import (
	projectsRESTApi "github.com/checkmarxDev/scans/pkg/api/projects/v1/rest"
)

type ProjectsWrapper interface {
	Create(model *projectsRESTApi.Project) (*projectsRESTApi.ProjectResponseModel, *projectsRESTApi.ErrorModel, error)
	Get(params map[string]string) (*projectsRESTApi.ProjectsCollectionResponseModel, *projectsRESTApi.ErrorModel, error)
	GetByID(projectID string) (*projectsRESTApi.ProjectResponseModel, *projectsRESTApi.ErrorModel, error)
	GetBranchesByID(projectID string, params map[string]string) ([]string, *projectsRESTApi.ErrorModel, error)
	Delete(projectID string) (*projectsRESTApi.ErrorModel, error)
	Tags() (map[string][]string, *projectsRESTApi.ErrorModel, error)
}
