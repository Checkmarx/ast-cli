package wrappers

import (
	"bytes"
	"encoding/json"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type ScaRealTimeHTTPWrapper struct {
	path string
}

func NewHTTPScaRealTimeWrapper(path string) ScaRealTimeWrapper {
	return &ScaRealTimeHTTPWrapper{
		path: path,
	}
}

func (s ScaRealTimeHTTPWrapper) GetScaVulnerabilitiesPackages(scaRequest []ScaDependencyBodyRequest) ([]ScaVulnerabilitiesResponseModel, *WebError, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(scaRequest)
	if err != nil {
		return nil, nil, err
	}

	resp, err := SendHTTPRequestByFullURL(http.MethodPost, s.path, bytes.NewReader(jsonBytes), false, clientTimeout, "")
	if err != nil {
		return nil, nil, errors.Errorf("Invoking HTTP request to get sca vulnerabilities failed - %s", err.Error())
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to get SCA vulnerabilities")
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		var model []ScaVulnerabilitiesResponseModel
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, nil
		}

		return model, nil, nil
	default:
		return nil, nil, errors.Errorf("failed to get SCA vulnerabilities. response status code %d", resp.StatusCode)
	}
}
