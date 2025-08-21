package wrappers

import (
	"time"
)

type BasePredicate struct {
	SimilarityID string `json:"similarityId"`
	ProjectID    string `json:"projectId"`
	State        string `json:"state"`
	Severity     string `json:"severity"`
	Comment      string `json:"comment"`
	StateID      int    `json:"stateId"`
}

type PredicateRequest struct {
	SimilarityID  string  `json:"similarityId"`
	ProjectID     string  `json:"projectId"`
	State         *string `json:"state,omitempty"`
	CustomStateID *int    `json:"customStateId,omitempty"`
	Comment       string  `json:"comment"`
	Severity      string  `json:"severity"`
}

type ScaPredicateRequest struct {
	PackageName     string      `json:"packageName"`
	PackageVersion  string      `json:"packageVersion"`
	PackageManager  string      `json:"packageManager"`
	VulnerabilityId string      `json:"vulnerabilityId"`
	ProjectIds      []string    `json:"projectIds"`
	Actions         []ScaAction `json:"actions"`
}

type State string

const (
	ToVerify               string = "ToVerify"
	Confirmed              string = "Confirmed"
	NotExploitable         string = "NotExploitable"
	ProposedNotExploitable string = "ProposedNotExploitable"
	Urgent                 string = "Urgent"
)

type ScaAction struct {
	ActionType string `json:"actionType"`
	Value      string `json:"value"`
	Comment    string `json:"comment"`
}

type Predicate struct {
	BasePredicate
	ID        string    `json:"ID"`
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
}

type PredicateHistory struct {
	ProjectID    string      `json:"projectId"`
	SimilarityID string      `json:"similarityId"`
	Predicates   []Predicate `json:"predicates"`
	TotalCount   int         `json:"totalCount"`
}

type PredicatesCollectionResponseModel struct {
	PredicateHistoryPerProject []PredicateHistory `json:"predicateHistoryPerProject"`
	TotalCount                 int                `json:"totalCount"`
}

type CustomState struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type CustomStatesWrapper interface {
	GetAllCustomStates(includeDeleted bool) ([]CustomState, error)
}

type ResultsPredicatesWrapper interface {
	PredicateSeverityAndState(predicate interface{}, scanType string) (*WebError, error)
	GetAllPredicatesForSimilarityID(
		similarityID string, projectID string, scannerType string,
	) (*PredicatesCollectionResponseModel, *WebError, error)
}
