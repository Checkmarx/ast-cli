package wrappers

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	projectsRESTApi "github.com/checkmarxDev/scans/api/v1/rest/projects"
)

type ProjectsHTTPWrapper struct {
	url         string
	contentType string
}

func NewHTTPProjectsWrapper(url string) ProjectsWrapper {
	return &ProjectsHTTPWrapper{
		url:         url,
		contentType: "application/json",
	}
}

func (p *ProjectsHTTPWrapper) Create(model *projectsRESTApi.Project) (
	*projectsRESTApi.ProjectResponseModel,
	*projectsRESTApi.ErrorModel, error) {
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		return nil, nil, err
	}

	resp, err := http.Post(p.url, p.contentType, bytes.NewBuffer(jsonBytes))
	return handleProjectResponseWithBody(resp, err, http.StatusCreated)
}

func (p *ProjectsHTTPWrapper) Get() (
	*projectsRESTApi.SlicedProjectsResponseModel,
	*projectsRESTApi.ErrorModel, error) {
	resp, err := http.Get(p.url)
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
	resp, err := http.Get(p.url + "/" + projectID)
	if err != nil {
		return nil, nil, err
	}
	return handleProjectResponseWithBody(resp, err, http.StatusOK)
}

func (p *ProjectsHTTPWrapper) Delete(projectID string) (
	*projectsRESTApi.ErrorModel,
	error) {
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", p.url+"/"+projectID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	return handleProjectResponseWithNoBody(resp, err, http.StatusOK)
}

func (p *ProjectsHTTPWrapper) Tags() (
	*[]string,
	*projectsRESTApi.ErrorModel,
	error) {
	resp, err := http.Get(p.url + "/tags")
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
