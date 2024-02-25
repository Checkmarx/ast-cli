package wrappers

import (
	"encoding/json"
	applicationErrors "github.com/checkmarx/ast-cli/internal/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"net/http"
)

type ApplicationsHTTPWrapper struct {
	path string
}

func NewApplicationsHTTPWrapper(path string) ApplicationsWrapper {
	return &ApplicationsHTTPWrapper{
		path: path,
	}
}

func (a *ApplicationsHTTPWrapper) Get(params map[string]string) (*ApplicationsResponseModel, *ErrorModel, error) {
	if _, ok := params[limit]; !ok {
		params[limit] = limitValue
	}

	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)

	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, a.path, params, nil, clientTimeout)
	defer resp.Body.Close()

	if err != nil {
		return nil, nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetAll)
		}
		return nil, &errorModel, nil
	case http.StatusForbidden:
		return nil, nil, errors.Errorf(applicationErrors.ApplicationNoPermission)
	case http.StatusOK:
		model := ApplicationsResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetAll)
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
