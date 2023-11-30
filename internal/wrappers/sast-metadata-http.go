package wrappers

import (
	"encoding/json"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"net/http"
)

type SastIncrementalHTTPWrapper struct {
	path        string
	contentType string
}

func NewSastIncrementalHTTPWrapper(path string) SastMetadataWrapper {
	return &SastIncrementalHTTPWrapper{
		path:        path,
		contentType: "application/json",
	}
}

func (s *SastIncrementalHTTPWrapper) GetSastMetadataByIDs(params map[string]string) (*SastMetadataModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)

	resp, err := SendPrivateHTTPRequestWithQueryParams(http.MethodGet, s.path, params, http.NoBody, clientTimeout)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, errors.Wrapf(err, failedToParseGetAll)
		}
		return nil, err
	case http.StatusOK:
		model := SastMetadataModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, errors.Wrapf(err, failedToParseGetAll)
		}
		return &model, nil
	case http.StatusNotFound:
		return nil, errors.Errorf("scan not found")
	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
