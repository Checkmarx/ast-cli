package wrappers

import (
	"encoding/json"
	"fmt"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const riskManagementDefaultPath = "api/micro-engines/read/scans/%s/scan-overview"

type RiskManagementHTTPWrapper struct {
	path string
}

func NewHTTPRiskManagementWrapper(path string) RiskManagementWrapper {
	validPath := setRMDefaultPath(path)
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

func setRMDefaultPath(path string) string {
	if path != riskManagementDefaultPath {
		configFilePath, err := configuration.GetConfigFilePath()
		if err != nil {
			logger.PrintfIfVerbose("Error getting config file path: %v", err)
		}
		err = configuration.SafeWriteSingleConfigKeyString(configFilePath, commonParams.RiskManagementPathKey, riskManagementDefaultPath)
		if err != nil {
			logger.PrintfIfVerbose("Error writing Scan Overview path to config file: %v", err)
		}
	}
	return riskManagementDefaultPath
}
