package wrappers

import (
	"time"
)

type Project struct {
	Name       string            `json:"name,omitempty"`
	RepoURL    string            `json:"repoUrl,omitempty"`
	MainBranch string            `json:"mainBranch,omitempty"`
	Origin     string            `json:"origin,omitempty"`
	ScmRepoID  string            `json:"scmRepoId,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Groups     []string          `json:"groups,omitempty"`
}

type ProjectsCollectionResponseModel struct {
	TotalCount         uint                   `json:"totalCount"`
	FilteredTotalCount uint                   `json:"filteredTotalCount"`
	Projects           []ProjectResponseModel `json:"projects"`
}

type ProjectResponseModel struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
	Groups     []string          `json:"groups"`
	Tags       map[string]string `json:"tags"`
	RepoURL    string            `json:"repoUrl"`
	MainBranch string            `json:"mainBranch"`
	Origin     string            `json:"origin,omitempty"`
	ScmRepoID  string            `json:"scmRepoId,omitempty"`
}

type ProjectConfiguration struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Category      string `json:"category"`
	OriginLevel   string `json:"originLevel"`
	Value         string `json:"value"`
	ValueType     string `json:"valuetype"`
	AllowOverride bool   `json:"allowOverride"`
}

type ProjectsWrapper interface {
	Create(model *Project) (*ProjectResponseModel, *ErrorModel, error)
	Get(params map[string]string) (*ProjectsCollectionResponseModel, *ErrorModel, error)
	GetByID(projectID string) (*ProjectResponseModel, *ErrorModel, error)
	GetBranchesByID(projectID string, params map[string]string) ([]string, *ErrorModel, error)
	Delete(projectID string) (*ErrorModel, error)
	Tags() (map[string][]string, *ErrorModel, error)
	UpdateConfiguration(projectID string, configuration []ProjectConfiguration) (*ErrorModel, error)
}
