package wrappers

import (
	"encoding/json"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	failedToParseDastEnvironments = "Failed to parse DAST environments"
)

// DastEnvironmentsHTTPWrapper implements the DastEnvironmentsWrapper interface
type DastEnvironmentsHTTPWrapper struct {
	path string
}

// NewHTTPDastEnvironmentsWrapper creates a new HTTP DAST environments wrapper
func NewHTTPDastEnvironmentsWrapper(path string) DastEnvironmentsWrapper {
	return &DastEnvironmentsHTTPWrapper{
		path: path,
	}
}

// Get retrieves DAST environments with optional query parameters
func (e *DastEnvironmentsHTTPWrapper) Get(params map[string]string) (*DastEnvironmentsCollectionResponseModel, *ErrorModel, error) {
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
			return nil, nil, errors.Wrapf(err, failedToParseDastEnvironments)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := DastEnvironmentsCollectionResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseDastEnvironments)
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
