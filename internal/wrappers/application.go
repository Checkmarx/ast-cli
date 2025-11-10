package wrappers

import "time"

type ApplicationsResponseModel struct {
	TotalCount         int           `json:"totalCount"`
	FilteredTotalCount int           `json:"filteredTotalCount"`
	Applications       []Application `json:"applications"`
}

type Application struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Criticality int               `json:"criticality"`
	Rules       []Rule            `json:"rules"`
	ProjectIds  []string          `json:"projectIds"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	Tags        map[string]string `json:"tags"`
	Type        string            `json:"type"`
}

type ApplicationConfiguration struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Criticality int               `json:"criticality"`
	Rules       []Rule            `json:"rules"`
	Tags        map[string]string `json:"tags"`
}

type AssociateProjectModel struct {
	ProjectIds []string `json:"projectIds"`
}

type Rule struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type ApplicationsWrapper interface {
	Get(params map[string]string) (*ApplicationsResponseModel, error)
	Update(applicationID string, applicationBody *ApplicationConfiguration) (*ErrorModel, error)
	CreateProjectAssociation(applicationID string, requestModel *AssociateProjectModel) (*ErrorModel, error)
}
