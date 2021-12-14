package mock

import (
	"fmt"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"
	"time"
)

type ResultsPredicatesMockWrapper struct {
}

func (r ResultsPredicatesMockWrapper) PredicateSeverityAndState(predicate *wrappers.PredicateRequest) (*resultsHelpers.WebError, error) {
	fmt.Println("Called 'PredicateSeverityAndState' in ResultsPredicatesMockWrapper")
	return nil, nil
}

func (r ResultsPredicatesMockWrapper) GetAllPredicatesForSimilarityId(similarityId string, projectID string, scannerType string) (*wrappers.PredicatesCollectionResponseModel, *resultsHelpers.WebError, error) {
	fmt.Println("Called 'GetAllPredicatesForSimilarityId' in ResultsPredicatesMockWrapper")

	totalCount := 1

	mockPredicateItem := wrappers.Predicate{
		Id:        "MOCK",
		CreatedBy: "MOCK",
		CreatedAt: time.Now(),
	}
	return &wrappers.PredicatesCollectionResponseModel{
		TotalCount: totalCount,
		PredicateHistoryPerProject: []wrappers.PredicateHistory{
			{
				ProjectId:    "MOCK",
				SimilarityId: "MOCK",
				TotalCount:   1,
				Predicates: []wrappers.Predicate{
					mockPredicateItem,
				},
			},
		},
	}, nil, nil

}
