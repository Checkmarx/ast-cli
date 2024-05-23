package wrappers

import (
	"encoding/json"
	"fmt"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
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

	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, fmt.Errorf("%v %s", err, failedToParseGetAll)
		}
		return nil, err
	case http.StatusOK:
		model := SastMetadataModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, fmt.Errorf("%v %s", err, failedToParseGetAll)
		}
		return &model, nil
	case http.StatusNotFound:
		return nil, fmt.Errorf("scan not found")
	default:
		return nil, fmt.Errorf("response status code %d", resp.StatusCode)
	}
}
