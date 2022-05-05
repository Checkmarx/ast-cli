package wrappers

import (
	"encoding/json"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"

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

func (r *ResultsHTTPWrapper) GetAllResultsByScanID(params map[string]string) (
	*ScanResultsCollection,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	// AST has a limit of 10000 results, this makes it get all of them
	params["limit"] = "10000"
	resp, err := SendPrivateHTTPRequestWithQueryParams(http.MethodGet, r.path, params, nil, clientTimeout)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := WebError{}
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
