package wrappers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	queries "github.com/checkmarxDev/sast-queries/pkg/v1/queriesobjects"
	queriesHelpers "github.com/checkmarxDev/sast-queries/pkg/web/helpers"
	"github.com/pkg/errors"
)

const (
	failedToParseImportResult   = "failed to parse import result"
	failedToParseCloneResult    = "failed to parse clone result"
	failedToParseListResult     = "failed to parse list results"
	failedToParseDeleteResult   = "failed to parse delete result"
	failedToParseActivateResult = "failed to parse activate result"
)

const cloneNameParamsFieldName = "name"

type queriesHTTPWrapper struct {
	path      string
	clonePath string
}

func NewQueriesHTTPWrapper(queryPath, queriesClonePath string) QueriesWrapper {
	return &queriesHTTPWrapper{
		queryPath,
		queriesClonePath,
	}
}

func (q *queriesHTTPWrapper) Clone(name string) (io.ReadCloser, *queriesHelpers.WebError, error) {
	params := make(map[string]string)
	if name != "" {
		params[cloneNameParamsFieldName] = name
	}

	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, q.clonePath, params, nil)
	if err != nil {
		return nil, nil, err
	}

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError:
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		errorModel := queriesHelpers.WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseCloneResult)
		}

		return nil, &errorModel, nil
	case http.StatusOK:
		return resp.Body, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (q *queriesHTTPWrapper) Import(uploadURL, name string) (*queriesHelpers.WebError, error) {
	body := &queries.NewQueriesRepoRequest{
		Name:      name,
		UploadURL: uploadURL,
	}

	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	resp, err := SendHTTPRequest(http.MethodPost, q.path, bytes.NewBuffer(jsonBytes), true, DefaultTimeoutSeconds)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		errorModel := queriesHelpers.WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, errors.Wrapf(err, failedToParseImportResult)
		}

		return &errorModel, nil
	case http.StatusCreated:
		return nil, nil
	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (q *queriesHTTPWrapper) List() ([]*queries.QueriesRepo, error) {
	resp, err := SendHTTPRequest(http.MethodGet, q.path, nil, true, DefaultTimeoutSeconds)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	if resp.StatusCode == http.StatusOK {
		var queriesRepos []*queries.QueriesRepo
		err = decoder.Decode(&queriesRepos)
		if err != nil {
			return nil, errors.Wrapf(err, failedToParseListResult)
		}

		return queriesRepos, nil
	}

	return nil, errors.Errorf("response status code %d", resp.StatusCode)
}

func (q *queriesHTTPWrapper) Activate(name string) (*queriesHelpers.WebError, error) {
	body := queries.ActiveQueriesRepoRequest{
		Activate: true,
	}

	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	resp, err := SendHTTPRequest(http.MethodPut, q.path+"/"+name, bytes.NewBuffer(jsonBytes), true, DefaultTimeoutSeconds)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError:
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		errorModel := queriesHelpers.WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, errors.Wrapf(err, failedToParseActivateResult)
		}

		return &errorModel, nil
	case http.StatusOK:
		return nil, nil
	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (q *queriesHTTPWrapper) Delete(name string) (*queriesHelpers.WebError, error) {
	resp, err := SendHTTPRequest(http.MethodDelete, q.path+"/"+name, nil, true, DefaultTimeoutSeconds)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError:
		defer resp.Body.Close()
		decoder := json.NewDecoder(resp.Body)
		errorModel := queriesHelpers.WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, errors.Wrapf(err, failedToParseDeleteResult)
		}

		return &errorModel, nil
	case http.StatusOK:
		return nil, nil
	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
