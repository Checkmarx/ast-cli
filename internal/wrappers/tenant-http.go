package wrappers

import (
	"encoding/json"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type TenantConfigurationHTTPWrapper struct {
	path string
}

func NewHTTPTenantConfigurationWrapper(path string) *TenantConfigurationHTTPWrapper {
	return &TenantConfigurationHTTPWrapper{
		path: path,
	}
}

func (r *TenantConfigurationHTTPWrapper) GetTenantConfiguration() (
	*[]*TenantConfigurationResponse,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	// add the path parameter to the path
	resp, err := SendHTTPRequest(http.MethodGet, r.path, nil, true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}

	decoder := json.NewDecoder(resp.Body)

	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, err
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := []*TenantConfigurationResponse{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, err
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("Response status code %d", resp.StatusCode)
	}
}
