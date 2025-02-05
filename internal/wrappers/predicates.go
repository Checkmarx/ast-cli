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
}

type PredicateRequest struct {
	SimilarityID  string `json:"similarityId"`
	ProjectID     string `json:"projectId"`
	State         string `json:"state"`
	CustomStateID string `json:"customStateId"`
	Comment       string `json:"comment"`
	Severity      string `json:"severity"`
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
	PredicateSeverityAndState(predicate *PredicateRequest, scanType string) (*WebError, error)
	GetAllPredicatesForSimilarityID(
		similarityID string, projectID string, scannerType string,
	) (*PredicatesCollectionResponseModel, *WebError, error)
}
