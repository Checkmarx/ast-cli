package wrappers

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	projApi "github.com/checkmarxDev/scans/api/v1/rest/projects"
	projModels "github.com/checkmarxDev/scans/pkg/projects"
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

func (p *ProjectsHTTPWrapper) Create(model *projApi.Project) (*projModels.ProjectResponseModel, *projModels.ErrorModel, error) {
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		return nil, nil, err
	}

	resp, err := http.Post(p.url, p.contentType, bytes.NewBuffer(jsonBytes))
	return handleProjectsResponse(resp, err, http.StatusCreated)
}

func (p *ProjectsHTTPWrapper) Get() (*projModels.ResponseModel, *projModels.ErrorModel, error) {
	resp, err := http.Get(p.url)
	if err != nil {
		return nil, nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := projModels.ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetAll)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := projModels.ResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetAll)
		}
		return &model, nil, nil

	default:
		return nil, nil, errors.Errorf("Unknown response status code %d", resp.StatusCode)
	}
}

func (p *ProjectsHTTPWrapper) GetByID(projectID string) (*projModels.ProjectResponseModel, *projModels.ErrorModel, error) {
	resp, err := http.Get(p.url + "/" + projectID)
	if err != nil {
		return nil, nil, err
	}
	return handleProjectsResponse(resp, err, http.StatusOK)
}

func (p *ProjectsHTTPWrapper) Delete(projectID string) (*projModels.ProjectResponseModel, *projModels.ErrorModel, error) {
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", p.url+"/"+projectID, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	return handleProjectsResponse(resp, err, http.StatusOK)
}

func (p *ProjectsHTTPWrapper) Tags() (*[]string, *projModels.ErrorModel, error) {
	resp, err := http.Get(p.url + "/tags")
	if err != nil {
		return nil, nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := projModels.ErrorModel{}
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

func handleProjectsResponse(
	resp *http.Response,
	err error,
	successStatusCode int) (*projModels.ProjectResponseModel, *projModels.ErrorModel, error) {
	if err != nil {
		return nil, nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := projModels.ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return responseProjectsParsingFailed(err)
		}
		return nil, &errorModel, nil
	case successStatusCode:
		model := projModels.ProjectResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return responseProjectsParsingFailed(err)
		}
		return &model, nil, nil

	default:
		return nil, nil, errors.Errorf("Unknown response status code %d", resp.StatusCode)
	}
}

func responseProjectsParsingFailed(err error) (*projModels.ProjectResponseModel, *projModels.ErrorModel, error) {
	msg := "Failed to parse a project response"
	return nil, nil, errors.Wrapf(err, msg)
}
