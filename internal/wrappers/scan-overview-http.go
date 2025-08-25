package wrappers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const defaultPath = "api/micro-engines/read/scans/%s/scan-overview"

type ScanOverviewHTTPWrapper struct {
	path string
}

func NewHTTPScanOverviewWrapper(path string) ScanOverviewWrapper {
	validPath := setDefaultPath(path)
	return &ScanOverviewHTTPWrapper{
		path: validPath,
	}
}

func (r *ScanOverviewHTTPWrapper) GetSCSOverviewByScanID(scanID string) (
	*SCSOverview,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	path := fmt.Sprintf(r.path, scanID)
	resp, err := SendHTTPRequest(http.MethodGet, path, http.NoBody, true, clientTimeout)
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
		model := SCSOverview{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

// setDefaultPath checks if the path is the default path, if not it writes the default path to the config file
func setDefaultPath(path string) string {
	if path != defaultPath {
		configFilePath, err := configuration.GetConfigFilePath()
		if err != nil {
			logger.PrintfIfVerbose("Error getting config file path: %v", err)
		}
		err = configuration.SafeWriteSingleConfigKeyString(configFilePath, commonParams.ScsScanOverviewPathKey, defaultPath)
		if err != nil {
			logger.PrintfIfVerbose("Error writing Scan Overview path to config file: %v", err)
		}
	}
	return defaultPath
}
