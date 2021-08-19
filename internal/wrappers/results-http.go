package wrappers

import (
	"encoding/json"
	"net/http"

	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"

	"github.com/pkg/errors"
)

const (
	failedToParseGetResults = "Failed to parse list results"
)

type ResultsHTTPWrapper struct {
	path      string
	sastPath  string
	kicsPath  string
	scansPath string
}

func NewHTTPResultsWrapper(path, sastPath, kicsPath, scansPath string) ResultsWrapper {
	return &ResultsHTTPWrapper{
		path:      path,
		sastPath:  sastPath,
		kicsPath:  kicsPath,
		scansPath: scansPath,
	}
}

func (r *ResultsHTTPWrapper) GetScaAPIPath() string {
	return r.scansPath
}

func (r *ResultsHTTPWrapper) GetAllResultsByScanID(params map[string]string) (*ScanResultsCollection, *resultsHelpers.WebError, error) {
	params["limit"] = "10000"
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, r.path, params, nil, DefaultTimeoutSeconds)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := resultsHelpers.WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := ScanResultsCollection{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
