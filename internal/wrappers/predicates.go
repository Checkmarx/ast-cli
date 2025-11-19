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
	ScaResponse                interface{}        `json:"scaPredicate,omitempty"`
	PredicateHistoryPerProject []PredicateHistory `json:"predicateHistoryPerProject"`
	TotalCount                 int                `json:"totalCount"`
}

type ScaPredicateResult struct {
	ID         string    `json:"id"`
	Context    Context   `json:"context"`
	Name       string    `json:"name"`
	Actions    []Action  `json:"actions"`
	EntityType string    `json:"entityType"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Context struct {
	PackageManager  string `json:"PackageManager"`
	PackageName     string `json:"PackageName"`
	PackageVersion  string `json:"PackageVersion"`
	VulnerabilityId string `json:"VulnerabilityId"`
}

type Action struct {
	ActionType  string    `json:"actionType"`
	ActionValue string    `json:"actionValue"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UserName    string    `json:"userName"`
	Message     string    `json:"message"`
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
	ScaPredicateResult(vulnerabilityDetails []string, projectId string) (*ScaPredicateResult, error)
	PredicateSeverityAndState(predicate interface{}, scanType string) (*WebError, error)
	GetAllPredicatesForSimilarityID(
		similarityID string, projectID string, scannerType string,
	) (*PredicatesCollectionResponseModel, *WebError, error)
}
