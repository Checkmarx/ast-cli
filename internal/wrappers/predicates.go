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
	SimilarityID string `json:"similarityId"`
	ProjectID    string `json:"projectId"`
	State        string `json:"state"`
	Severity     string `json:"severity"`
	Comment      string `json:"comment"`
	ScannerType  string `json:"scannerType"`
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

type ResultsPredicatesWrapper interface {
	PredicateSeverityAndState(predicate *PredicateRequest) (*WebError, error)
	GetAllPredicatesForSimilarityID(
		similarityID string, projectID string, scannerType string,
	) (*PredicatesCollectionResponseModel, *WebError, error)
}
