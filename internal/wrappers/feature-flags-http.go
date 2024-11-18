package wrappers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
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
	params, clientTimeout, err := f.getTenantDetails()
	if err != nil {
		return nil, err
	}

	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, f.path, params, nil, clientTimeout)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
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
	case http.StatusNotFound:
		return nil, errors.New("feature flags not found")
	default:
		return nil, errors.New("failed to load feature flags for tenant")
	}
}

func (f FeatureFlagsHTTPWrapper) GetSpecificFlag(flagName string) (*FeatureFlagResponseModel, error) {
	params, clientTimeout, err := f.getTenantDetails()
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("%s/%s", f.path, flagName) // Adjusting the path to include the flag name

	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, path, params, nil, clientTimeout)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		return nil, err
	case http.StatusOK:
		model := FeatureFlagResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, err
		}
		return &model, nil
	case http.StatusNotFound:
		return nil, errors.New("feature flags not found")
	default:
		return nil, errors.New("failed to load feature flags for tenant")
	}
}

func (f FeatureFlagsHTTPWrapper) getTenantDetails() (params map[string]string, clientTimeout uint, err error) {
	accessToken, tokenError := GetAccessToken()
	if tokenError != nil {
		return nil, 0, tokenError
	}

	tenantIDFromClaims, extractError := ExtractFromTokenClaims(accessToken, tenantIDClaimKey)
	if extractError != nil {
		return nil, 0, extractError
	}

	tenantID := strings.Split(tenantIDFromClaims, "::")[1]

	clientTimeout = viper.GetUint(commonParams.ClientTimeoutKey)
	params = map[string]string{
		"filter": tenantID,
	}
	return params, clientTimeout, nil
}
