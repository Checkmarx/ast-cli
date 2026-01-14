package wrappers

import (
	"encoding/json"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	failedToParseEnvironments = "Failed to parse environments"
)

// EnvironmentsHTTPWrapper implements the EnvironmentsWrapper interface
type EnvironmentsHTTPWrapper struct {
	path string
}

// NewHTTPEnvironmentsWrapper creates a new HTTP environments wrapper
func NewHTTPEnvironmentsWrapper(path string) EnvironmentsWrapper {
	return &EnvironmentsHTTPWrapper{
		path: path,
	}
}

// Get retrieves environments with optional query parameters
func (e *EnvironmentsHTTPWrapper) Get(params map[string]string) (*EnvironmentsCollectionResponseModel, *ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)

	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, e.path, params, nil, clientTimeout)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseEnvironments)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := EnvironmentsCollectionResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseEnvironments)
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
