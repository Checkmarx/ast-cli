package wrappers

import (
	"encoding/json"
	"fmt"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type RiskManagementHTTPWrapper struct {
	path string
}

func NewHTTPRiskManagementWrapper(path string) RiskManagementWrapper {
	validPath := setDefaultPath(path, riskManagementDefaultPath)
	return &RiskManagementHTTPWrapper{
		path: validPath,
	}
}

func (r *RiskManagementHTTPWrapper) GetTopVulnerabilitiesByProjectID(projectID string, scanID string) (*ASPMResult, *WebError, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)

	path := fmt.Sprintf(r.path, projectID, scanID)
	resp, err := SendHTTPRequest(http.MethodGet, path, nil, true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
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
		model := ASPMResult{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
