package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
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

func (a *ApplicationsHTTPWrapper) CreateProjectAssociation(applicationID string, projectAssociationModel *AssociateProjectModel) (*ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(*projectAssociationModel)
	if err != nil {
		return nil, err
	}
	associationPath := fmt.Sprintf("%s/%s/%s", a.path, applicationID, "projects")
	resp, err := SendHTTPRequest(http.MethodPost, associationPath, bytes.NewBuffer(jsonBytes), true, clientTimeout)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(resp.Body)
	defer func() {
		_ = resp.Body.Close()
	}()
	switch resp.StatusCode {
	case http.StatusBadRequest:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, errors.Errorf("failed to parse application response for project updation: %s ", err)
		}
		return &errorModel, nil

	case http.StatusCreated:
		return nil, nil

	case http.StatusForbidden:
		return nil, errors.New(errorConstants.NoPermissionToUpdateApplication)

	case http.StatusUnauthorized:
		return nil, errors.New(errorConstants.StatusUnauthorized)
	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (a *ApplicationsHTTPWrapper) Update(applicationID string, applicationBody *ApplicationConfiguration) (*ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(applicationBody)
	updatePath := fmt.Sprintf("%s/%s", a.path, applicationID)
	if err != nil {
		return nil, err
	}
	resp, err := SendHTTPRequest(http.MethodPut, updatePath, bytes.NewBuffer(jsonBytes), true, clientTimeout)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(resp.Body)
	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusBadRequest:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, errors.Errorf("failed to parse application response: %s ", err)
		}
		return &errorModel, nil

	case http.StatusNoContent:
		return nil, nil

	case http.StatusForbidden:
		return nil, errors.New(errorConstants.NoPermissionToUpdateApplication)

	case http.StatusUnauthorized:
		return nil, errors.New(errorConstants.StatusUnauthorized)
	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
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
			return nil, errors.Errorf(errorConstants.FailedToGetApplication)
		}
		return nil, nil
	case http.StatusForbidden:
		return nil, errors.Errorf(errorConstants.ApplicationDoesntExistOrNoPermission)
	case http.StatusOK:
		model := ApplicationsResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, errors.Errorf(errorConstants.FailedToGetApplication)
		}
		return &model, nil
	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
