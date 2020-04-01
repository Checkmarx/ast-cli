package wrappers

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

const (
	failedToParseGetResults = "Failed to parse get results"
)

type ResultsHTTPWrapper struct {
	url         string
	contentType string
}

func NewHTTPResultsWrapper(url string) ResultsWrapper {
	return &ResultsHTTPWrapper{
		url:         url,
		contentType: "application/json",
	}
}

func (r *ResultsHTTPWrapper) GetByScanID(scanID string, limit, offset uint64) ([]ResultResponseModel, *ResultError, error) {
	resp, err := getRequestWithLimitAndOffset(r.url+"/scan"+scanID+"/items", limit, offset)
	if err != nil {
		return nil, nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := ResultError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := []ResultResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return model, nil, nil

	default:
		return nil, nil, errors.Errorf("Unknown response status code %d", resp.StatusCode)
	}
}
