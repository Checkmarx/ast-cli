package wrappers

import (
	"encoding/json"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
	"net/http"
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
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	resp, err := SendHTTPRequest(http.MethodGet, f.path, nil, true, clientTimeout)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()

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
