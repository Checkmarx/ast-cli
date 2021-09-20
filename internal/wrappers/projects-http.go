package wrappers

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"
	projectsRESTApi "github.com/checkmarxDev/scans/pkg/api/projects/v1/rest"
)

type ProjectsHTTPWrapper struct {
	path string
}

func NewHTTPProjectsWrapper(path string) ProjectsWrapper {
	return &ProjectsHTTPWrapper{
		path: path,
	}
}

func (p *ProjectsHTTPWrapper) Create(model *projectsRESTApi.Project) (
	*projectsRESTApi.ProjectResponseModel,
	*projectsRESTApi.ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		return nil, nil, err
	}

	resp, err := SendHTTPRequest(http.MethodPost, p.path, bytes.NewBuffer(jsonBytes), true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}
	return handleProjectResponseWithBody(resp, err, http.StatusCreated)
}

func (p *ProjectsHTTPWrapper) Get(params map[string]string) (
	*projectsRESTApi.ProjectsCollectionResponseModel,
	*projectsRESTApi.ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, p.path, params, nil, clientTimeout)
	if err != nil {
		return nil, nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := projectsRESTApi.ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetAll)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := projectsRESTApi.ProjectsCollectionResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetAll)
		}
		return &model, nil, nil

	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (p *ProjectsHTTPWrapper) GetByID(projectID string) (
	*projectsRESTApi.ProjectResponseModel,
	*projectsRESTApi.ErrorModel,
	error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	resp, err := SendHTTPRequest(http.MethodGet, p.path+"/"+projectID, nil, true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}
	return handleProjectResponseWithBody(resp, err, http.StatusOK)
}

func (p *ProjectsHTTPWrapper) Delete(projectID string) (*projectsRESTApi.ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	resp, err := SendHTTPRequest(http.MethodDelete, p.path+"/"+projectID, nil, true, clientTimeout)
	if err != nil {
		return nil, err
	}
	return handleProjectResponseWithNoBody(resp, err, http.StatusNoContent)
}

func (p *ProjectsHTTPWrapper) Tags() (
	map[string][]string,
	*projectsRESTApi.ErrorModel,
	error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	resp, err := SendHTTPRequest(http.MethodGet, p.path+"/tags", nil, true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}

	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := projectsRESTApi.ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseTags)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		tags := map[string][]string{}
		err = decoder.Decode(&tags)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseTags)
		}
		return tags, nil, nil

	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
