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
	failedToParseDastAlerts = "Failed to parse DAST alerts"
)

// DastAlertsHTTPWrapper implements the DastAlertsWrapper interface
type DastAlertsHTTPWrapper struct {
	path string // Path template: api/dast/mfe-results/results/environment/%s/%s/alert_level
}

// NewHTTPDastAlertsWrapper creates a new HTTP DAST alerts wrapper
func NewHTTPDastAlertsWrapper(path string) DastAlertsWrapper {
	return &DastAlertsHTTPWrapper{
		path: path,
	}
}

// GetAlerts retrieves DAST alerts for a specific environment and scan
func (a *DastAlertsHTTPWrapper) GetAlerts(environmentID, scanID string, params map[string]string) (*DastAlertsCollectionResponseModel, *ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)

	// Build the full path by filling in environment_id and scan_id
	path := fmt.Sprintf(a.path, environmentID, scanID)

	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, path, params, nil, clientTimeout)
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
			return nil, nil, errors.Wrapf(err, failedToParseDastAlerts)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := DastAlertsCollectionResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseDastAlerts)
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

