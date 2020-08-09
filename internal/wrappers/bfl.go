package wrappers

import (
	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"
	resultsBfl "github.com/checkmarxDev/sast-results/pkg/web/path/bfl"
)

type BFLWrapper interface {
	GetByScanID(params map[string]string) (*resultsBfl.Forest, *resultsHelpers.WebError, error)
}
