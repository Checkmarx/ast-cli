package wrappers

import (
	"bytes"
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
	return &RiskManagementHTTPWrapper{
		path: path,
	}
}

func (r *RiskManagementHTTPWrapper) GetTopVulnerabilitiesByProjectID(projectID string, scanID string) (*ASPMResult, *WebError, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)

	model := GetASPMResultRequest{ScanId: scanID}
	jsonBytes, err := json.Marshal(model)

	path := fmt.Sprintf(r.path, projectID)
	resp, err := SendHTTPRequest(http.MethodGet, path, bytes.NewBuffer(jsonBytes), true, clientTimeout)
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
