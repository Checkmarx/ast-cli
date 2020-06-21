package wrappers

import (
	"encoding/json"
	"net/http"

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

func (r *ResultsHTTPWrapper) GetByScanID(params map[string]string) (*ResultsResponseModel, *ErrorModel, error) {
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, r.path, params, nil)
	if err != nil {
		return nil, nil, err
	}

	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := ResultsResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return &model, nil, nil

	default:
		return nil, nil, errors.Errorf("Unknown response status code %d", resp.StatusCode)
	}
}
