package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
)

type ProjectsHTTPWrapper struct {
	path string
}

func NewHTTPProjectsWrapper(path string) ProjectsWrapper {
	return &ProjectsHTTPWrapper{
		path: path,
	}
}

func (p *ProjectsHTTPWrapper) Create(model *Project) (*ProjectResponseModel, *ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		return nil, nil, err
	}

	resp, err := SendHTTPRequest(http.MethodPost, p.path, bytes.NewBuffer(jsonBytes), true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	return handleProjectResponseWithBody(resp, err, http.StatusCreated)
}

func (p *ProjectsHTTPWrapper) Update(projectID string, model *Project) error {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		return nil
	}

	resp, err := SendHTTPRequest(http.MethodPut, fmt.Sprintf("%s/%s", p.path, projectID), bytes.NewBuffer(jsonBytes), true, clientTimeout)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	default:
		return errors.Errorf("failed to update project %s, status - %d", projectID, resp.StatusCode)
	}
}

func (p *ProjectsHTTPWrapper) UpdateConfiguration(projectID string, configuration []ProjectConfiguration) (*ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(configuration)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		commonParams.ProjectIDFlag: projectID,
	}

	resp, err := SendHTTPRequestWithQueryParams(http.MethodPatch, "api/configuration/project", params, bytes.NewBuffer(jsonBytes), clientTimeout)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return handleProjectResponseWithNoBody(resp, err, http.StatusNoContent)
}

func (p *ProjectsHTTPWrapper) Get(params map[string]string) (
	*ProjectsCollectionResponseModel,
	*ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)

	if _, ok := params[limit]; !ok {
		params[limit] = limitValue
	}

	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, p.path, params, nil, clientTimeout)
	if err != nil {
		return nil, nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetAll)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := ProjectsCollectionResponseModel{}
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
	*ProjectResponseModel,
	*ErrorModel,
	error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	resp, err := SendHTTPRequest(http.MethodGet, p.path+"/"+projectID, http.NoBody, true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	return handleProjectResponseWithBody(resp, err, http.StatusOK)
}

func (p *ProjectsHTTPWrapper) GetBranchesByID(projectID string, params map[string]string) ([]string, *ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)

	var request = "/branches?project-id=" + projectID

	params["limit"] = limitValue
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, p.path+request, params, nil, clientTimeout)
	if err != nil {
		return nil, nil, err
	}

	decoder := json.NewDecoder(resp.Body)
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseBranches)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		var branches []string
		err = decoder.Decode(&branches)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseBranches)
		}
		return branches, nil, nil

	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (p *ProjectsHTTPWrapper) Delete(projectID string) (*ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	resp, err := SendHTTPRequest(http.MethodDelete, p.path+"/"+projectID, http.NoBody, true, clientTimeout)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return handleProjectResponseWithNoBody(resp, err, http.StatusNoContent)
}

func (p *ProjectsHTTPWrapper) Tags() (
	map[string][]string,
	*ErrorModel,
	error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	resp, err := SendHTTPRequest(http.MethodGet, p.path+"/tags", http.NoBody, true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := ErrorModel{}
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
