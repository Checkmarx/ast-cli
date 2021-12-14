package wrappers

import (
	"time"

	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"
)

type BasePredicate struct {
	SimilarityId string `json:"similarityId"`
	ProjectId    string `json:"projectId"`
	State        string `json:"state"`
	Severity     string `json:"severity"`
	Comment      string `json:"comment"`
}

type PredicateRequest struct {
	SimilarityId string `json:"similarityId"`
	ProjectId    string `json:"projectId"`
	State        string `json:"state"`
	Severity     string `json:"severity"`
	Comment      string `json:"comment"`
	ScannerType  string `json:"scannerType"`
}

type Predicate struct {
	BasePredicate
	Id        string    `json:"ID"`
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
}

type PredicateHistory struct {
	ProjectId    string      `json:"projectId"`
	SimilarityId string      `json:"similarityId"`
	Predicates   []Predicate `json:"predicates"`
	TotalCount   int         `json:"totalCount"`
}

type PredicatesCollectionResponseModel struct {
	PredicateHistoryPerProject []PredicateHistory `json:"predicateHistoryPerProject"`
	TotalCount                 int                `json:"totalCount"`
}

type ResultsPredicatesWrapper interface {
	PredicateSeverityAndState(predicate *PredicateRequest) (*resultsHelpers.WebError, error)
	GetAllPredicatesForSimilarityId(similarityId string, projectID string, scannerType string) (*PredicatesCollectionResponseModel, *resultsHelpers.WebError, error)
}
