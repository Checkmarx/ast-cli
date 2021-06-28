package wrappers

import (
	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"
	resultsRaw "github.com/checkmarxDev/sast-results/pkg/web/path/raw"
)

type ResultsWrapper interface {
	GetSastByScanID(params map[string]string) (*resultsRaw.ResultsCollection, *resultsHelpers.WebError, error)
	GetKicsByScanID(params map[string]string) (*resultsRaw.ResultsCollection, *resultsHelpers.WebError, error)
}
