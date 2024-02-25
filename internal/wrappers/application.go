package wrappers

import "time"

type ApplicationsResponseModel struct {
	TotalCount         int           `json:"totalCount"`
	FilteredTotalCount int           `json:"filteredTotalCount"`
	Applications       []Application `json:"applications"`
}

type Application struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Criticality int       `json:"criticality"`
	Rules       []Rule    `json:"rules"`
	ProjectIds  []string  `json:"projectIds"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Tags        Tags      `json:"tags"`
}

type Rule struct {
	Id    string `json:"id"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Tags struct {
	Test     string `json:"test"`
	Priority string `json:"priority"`
}

type ApplicationsWrapper interface {
	Get(params map[string]string) (*ApplicationsResponseModel, *ErrorModel, error)
}
