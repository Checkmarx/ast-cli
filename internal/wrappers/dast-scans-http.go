package wrappers

import (
	"encoding/json"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	failedToParseDastScans = "Failed to parse DAST scans"
)

// DastScansHTTPWrapper implements the DastScansWrapper interface
type DastScansHTTPWrapper struct {
	path string
}

// NewHTTPDastScansWrapper creates a new HTTP DAST scans wrapper
func NewHTTPDastScansWrapper(path string) DastScansWrapper {
	return &DastScansHTTPWrapper{
		path: path,
	}
}

// Get retrieves DAST scans with optional query parameters
func (s *DastScansHTTPWrapper) Get(params map[string]string) (*DastScansCollectionResponseModel, *ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)

	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, s.path, params, nil, clientTimeout)
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
			return nil, nil, errors.Wrapf(err, failedToParseDastScans)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := DastScansCollectionResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseDastScans)
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

