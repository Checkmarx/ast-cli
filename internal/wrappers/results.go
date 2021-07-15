package wrappers

import (
	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"
)

type ResultsWrapper interface {
	GetAllResultsByScanID(params map[string]string) (*ScanResultsCollection, *resultsHelpers.WebError, error)
	GetScaAPIPath() string
}
