package wrappers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/spf13/viper"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
)

type FeatureFlagsHTTPWrapper struct {
	path string
}

func NewFeatureFlagsHTTPWrapper(path string) FeatureFlagsWrapper {
	return &FeatureFlagsHTTPWrapper{
		path: path,
	}
}

func (f FeatureFlagsHTTPWrapper) GetAll() (*FeatureFlagsResponseModel, error) {
	accessToken, tokenError := GetAccessToken()
	if tokenError != nil {
		return nil, tokenError
	}

	tenantIDFromClaims, extractError := extractFromTokenClaims(accessToken, tenantIDClaimKey)
	if extractError != nil {
		return nil, extractError
	}

	tenantID := strings.Split(tenantIDFromClaims, "::")[1]
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	params := map[string]string{
		"filter": tenantID,
	}
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, f.path, params, nil, clientTimeout)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	if resp != nil {
		defer resp.Body.Close()
	}

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		return nil, err
	case http.StatusOK:
		model := FeatureFlagsResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, err
		}
		return &model, nil

	default:
		return nil, nil
	}
}
