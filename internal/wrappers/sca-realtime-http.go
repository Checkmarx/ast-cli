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
	scaVulnerabilitiesPackagesURL string
}

func NewHTTPScaRealTimeWrapper() ScaRealTimeWrapper {
	return &ScaRealTimeHTTPWrapper{
		scaVulnerabilitiesPackagesURL: "https://api-sca.checkmarx.net/public/vulnerabilities/packages",
	}
}

func (s ScaRealTimeHTTPWrapper) GetScaVulnerabilitiesPackages(scaRequest []ScaDependencyBodyRequest) ([]ScaVulnerabilitiesResponseModel, *WebError, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(scaRequest)
	if err != nil {
		return nil, nil, err
	}

	resp, err := SendHTTPRequestByFullURL(http.MethodPost, s.scaVulnerabilitiesPackagesURL, bytes.NewReader(jsonBytes), false, clientTimeout, "", true)
	if err != nil {
		return nil, nil, errors.Errorf("Invoking HTTP request to get sca vulnerabilities failed - %s", err.Error())
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := GetWebError(decoder)
		return nil, &errorModel, nil
	case http.StatusOK:
		var model []ScaVulnerabilitiesResponseModel
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, nil
		}

		return model, nil, nil
	default:
		return nil, nil, errors.Errorf("Failed to get SCA vulnerabilities. response status code %d", resp.StatusCode)
	}
}
