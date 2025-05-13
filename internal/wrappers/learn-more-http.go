package wrappers

import (
	"encoding/json"
	"fmt"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	failedToGetDescriptions = "Failed to get the descriptions for id: %s"
)

type LearnMoreHTTPWrapper struct {
	path string
}

func NewHTTPLearnMoreWrapper(path string) *LearnMoreHTTPWrapper {
	return &LearnMoreHTTPWrapper{
		path: path,
	}
}

func (r *LearnMoreHTTPWrapper) GetLearnMoreDetails(params map[string]string) (
	*[]*LearnMoreResponse,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	// add the path parameter to the path
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, r.path, params, nil, clientTimeout)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	return handleResponse(resp, err, params[commonParams.QueryIDQueryParam])
}

func handleResponse(resp *http.Response, err error, queryID string) (*[]*LearnMoreResponse, *WebError, error) {
	if err != nil {
		return nil, nil, err
	}

	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrap(err, fmt.Sprintf(failedToGetDescriptions, queryID))
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := []*LearnMoreResponse{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrap(err, fmt.Sprintf(failedToGetDescriptions, queryID))
		}
		return &model, nil, nil
	case http.StatusNotFound:
		return nil, nil, errors.Errorf("Learn more descriptions GET not found")
	default:
		return nil, nil, errors.Errorf("Response status code %d", resp.StatusCode)
	}
}
