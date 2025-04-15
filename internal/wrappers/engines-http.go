package wrappers

import (
	"encoding/json"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const ()

type EnginesHTTPWrapper struct {
	path        string
	contentType string
}

func NewHTTPEnginesWrapper(path string) EnginesWrapper {
	return &EnginesHTTPWrapper{
		path:        path,
		contentType: "application/json",
	}
}

func (s *EnginesHTTPWrapper) Get(engineName string) (*EnginesAPIInventoryResponseModel, *ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)

	var mockApi bool = true

	if mockApi {
		return MockedAPIData(engineName)
	}

	fn := func() (*http.Response, error) {
		return SendHTTPRequestWithQueryParams(http.MethodGet, s.path, nil, nil, clientTimeout)
	}
	resp, err := retryHTTPRequest(fn, retryAttempts, retryDelay)
	if err != nil || (resp != nil && resp.Body != nil) {
		return nil, nil, err
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
			return nil, nil, errors.Wrapf(err, failedToParseGetAll)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := EnginesAPIInventoryResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetAll)
		}
		return &model, nil, nil
	case http.StatusNotFound:
		return nil, nil, errors.Errorf("Engine not found")
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func MockedAPIData(engineName string) (*EnginesAPIInventoryResponseModel, *ErrorModel, error) {

	enginesApis := &EnginesAPIInventoryResponseModel{
		Engines: []EnginesAPIResponseModel{
			{
				Engine_Id:   "1",
				Engine_Name: "sast",
				Apis: []EnginesAPIData{

					{
						Api_Url:     "https://eu.ast.checkmarx.net/api/sast-results",
						Api_Name:    "sast-results",
						Description: "Get comprehensive results for each of the vulnerabilities detected in a specific SAST scan (by scan ID)",
					},
					{
						Api_Url:     "https://eu.ast.checkmarx.net/api/sast-metadata/",
						Api_Name:    "sast-results-metadat",
						Description: "Get SAST scanner metadata for one or more scans.",
					},
				},
			},
		},
	}
	return enginesApis, nil, nil

}
