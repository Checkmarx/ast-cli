package wrappers

import (
	projApi "github.com/checkmarxDev/scans/api/v1/rest/projects"
	projModels "github.com/checkmarxDev/scans/pkg/projects"
)

type ProjectsWrapper interface {
	Create(model *projApi.Project) (*projModels.ProjectResponseModel, *projModels.ErrorModel, error)
	Get() (*projModels.ResponseModel, *projModels.ErrorModel, error)
	GetByID(projectID string) (*projModels.ProjectResponseModel, *projModels.ErrorModel, error)
	Delete(projectID string) (*projModels.ProjectResponseModel, *projModels.ErrorModel, error)
	Tags() (*[]string, *projModels.ErrorModel, error)
}
