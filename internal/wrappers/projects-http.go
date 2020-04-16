package wrappers

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	projectsRESTApi "github.com/checkmarxDev/scans/api/v1/rest/projects"
)

const (
	limitQueryParam  = "limit"
	offsetQueryParam = "offset"
)

type ProjectsHTTPWrapper struct {
	url string
}

func NewHTTPProjectsWrapper(url string) ProjectsWrapper {
	return &ProjectsHTTPWrapper{
		url: url,
	}
}

func (p *ProjectsHTTPWrapper) Create(model *projectsRESTApi.Project) (
	*projectsRESTApi.ProjectResponseModel,
	*projectsRESTApi.ErrorModel, error) {
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		return nil, nil, err
	}

	resp, err := SendHTTPRequest(http.MethodPost, p.url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, nil, err
	}
	return handleProjectResponseWithBody(resp, err, http.StatusCreated)
}

func (p *ProjectsHTTPWrapper) Get(limit, offset uint64) (
	*projectsRESTApi.SlicedProjectsResponseModel,
	*projectsRESTApi.ErrorModel, error) {
	resp, err := SendHTTPRequestWithLimitAndOffset(http.MethodGet, p.url, make(map[string]string), limit, offset, nil)
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
		model := projectsRESTApi.SlicedProjectsResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetAll)
		}
		return &model, nil, nil

	default:
		return nil, nil, errors.Errorf("Unknown response status code %d", resp.StatusCode)
	}
}

func (p *ProjectsHTTPWrapper) GetByID(projectID string) (
	*projectsRESTApi.ProjectResponseModel,
	*projectsRESTApi.ErrorModel,
	error) {
	resp, err := SendHTTPRequest(http.MethodGet, p.url+"/"+projectID, nil)
	if err != nil {
		return nil, nil, err
	}
	return handleProjectResponseWithBody(resp, err, http.StatusOK)
}

func (p *ProjectsHTTPWrapper) Delete(projectID string) (*projectsRESTApi.ErrorModel, error) {
	resp, err := SendHTTPRequest(http.MethodDelete, p.url+"/"+projectID, nil)
	if err != nil {
		return nil, err
	}
	return handleProjectResponseWithNoBody(resp, err, http.StatusNoContent)
}

func (p *ProjectsHTTPWrapper) Tags() (
	*[]string,
	*projectsRESTApi.ErrorModel,
	error) {
	resp, err := SendHTTPRequest(http.MethodGet, p.url+"/tags", nil)
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
		tags := []string{}
		err = decoder.Decode(&tags)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseTags)
		}
		return &tags, nil, nil

	default:
		return nil, nil, errors.Errorf("Unknown response status code %d", resp.StatusCode)
	}
}
