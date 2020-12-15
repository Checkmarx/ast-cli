package wrappers

import (
	"encoding/json"
	"net/http"

	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"
	resultsRaw "github.com/checkmarxDev/sast-results/pkg/web/path/raw"

	"github.com/pkg/errors"
)

const (
	failedToParseGetResults = "Failed to parse list results"
)

type ResultsHTTPWrapper struct {
	path string
}

func NewHTTPResultsWrapper(path string) ResultsWrapper {
	return &ResultsHTTPWrapper{
		path: path,
	}
}

func (r *ResultsHTTPWrapper) GetByScanID(params map[string]string) (*resultsRaw.ResultsCollection, *resultsHelpers.WebError, error) {
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, r.path, params, nil, DefaultTimeoutSeconds)
	if err != nil {
		return nil, nil, err
	}

	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := resultsHelpers.WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := resultsRaw.ResultsCollection{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return &model, nil, nil

	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
