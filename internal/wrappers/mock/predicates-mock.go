package mock

import (
	"fmt"
	"time"

	"github.com/checkmarx/ast-cli/internal/wrappers"
)

type ResultsPredicatesWrapper struct {
}

func (r ResultsPredicatesWrapper) PredicateSeverityAndState(predicate interface{}, scanType string) (
	*wrappers.WebError, error,
) {
	fmt.Println("Called 'PredicateSeverityAndState' in ResultsPredicatesMockWrapper")
	return nil, nil
}

func (r ResultsPredicatesWrapper) GetAllPredicatesForSimilarityID(similarityID, projectID, scannerType string) (
	*wrappers.PredicatesCollectionResponseModel, *wrappers.WebError, error,
) {
	fmt.Println("Called 'GetAllPredicatesForSimilarityID' in ResultsPredicatesMockWrapper")

	totalCount := 1

	mockPredicateItem := wrappers.Predicate{
		ID:        "MOCK",
		CreatedBy: "MOCK",
		CreatedAt: time.Now(),
	}
	return &wrappers.PredicatesCollectionResponseModel{
		TotalCount: totalCount,
		PredicateHistoryPerProject: []wrappers.PredicateHistory{
			{
				ProjectID:    "MOCK",
				SimilarityID: "MOCK",
				TotalCount:   1,
				Predicates: []wrappers.Predicate{
					mockPredicateItem,
				},
			},
		},
	}, nil, nil
}

func (r ResultsPredicatesWrapper) ScaPredicateResult(vulnerabilityDetails []string, projectId string) (*wrappers.ScaPredicateResult, error) {
	fmt.Println("Called 'ScaPredicateResult' in ResultsPredicatesMockWrapper")
	return nil, nil
}
