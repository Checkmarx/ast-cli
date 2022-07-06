package wrappers

import (
	"encoding/json"
	"fmt"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"net/http"
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
	*[]LearnMoreResponseModel,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	// add the path parameter to the path
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, r.path, params, nil, clientTimeout)
	return handleResponse(resp, err, params[commonParams.QueryIDQueryParam])
}

func handleResponse(resp *http.Response, err error, queryId string) (*[]LearnMoreResponseModel, *WebError, error) {
	if err != nil {
		return nil, nil, err
	}

	decoder := json.NewDecoder(resp.Body)

	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, fmt.Sprintf(failedToGetDescriptions, queryId))
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		var model []LearnMoreResponseModel
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, fmt.Sprintf(failedToGetDescriptions, queryId))
		}
		return &model, nil, nil
	case http.StatusNotFound:
		return nil, nil, errors.Errorf("Learn more descriptions GET not found")
	default:
		return nil, nil, errors.Errorf("Response status code %d", resp.StatusCode)
	}
}
