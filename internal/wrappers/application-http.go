package wrappers

import (
	"encoding/json"
	"net/http"

	applicationErrors "github.com/checkmarx/ast-cli/internal/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type ApplicationsHTTPWrapper struct {
	path string
}

func NewApplicationsHTTPWrapper(path string) ApplicationsWrapper {
	return &ApplicationsHTTPWrapper{
		path: path,
	}
}

func (a *ApplicationsHTTPWrapper) Get(params map[string]string) (*ApplicationsResponseModel, error) {
	if _, ok := params[limit]; !ok {
		params[limit] = limitValue
	}

	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)

	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, a.path, params, nil, clientTimeout)
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		if err != nil {
			return nil, errors.Errorf(applicationErrors.FailedToGetApplication)
		}
		return nil, nil
	case http.StatusForbidden:
		return nil, errors.Errorf(applicationErrors.ApplicationDoesntExistOrNoPermission)
	case http.StatusOK:
		model := ApplicationsResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, errors.Errorf(applicationErrors.FailedToGetApplication)
		}
		return &model, nil
	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
