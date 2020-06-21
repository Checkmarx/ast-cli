package wrappers

import (
	"bytes"
	"encoding/json"
	"net/http"

	scansApi "github.com/checkmarxDev/scans/pkg/api/scans/v1/rest"
	"github.com/pkg/errors"
)

const (
	failedToParseGetAll = "Failed to parse list response"
	failedToParseTags   = "Failed to parse tags response"
)

type ScansHTTPWrapper struct {
	path        string
	contentType string
}

func NewHTTPScansWrapper(path string) ScansWrapper {
	return &ScansHTTPWrapper{
		path:        path,
		contentType: "application/json",
	}
}

func (s *ScansHTTPWrapper) Create(model *scansApi.Scan) (*scansApi.ScanResponseModel, *scansApi.ErrorModel, error) {
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		return nil, nil, err
	}
	resp, err := SendHTTPRequest(http.MethodPost, s.path, bytes.NewBuffer(jsonBytes), true)
	if err != nil {
		return nil, nil, err
	}
	if err != nil {
		return nil, nil, err
	}
	return handleScanResponseWithBody(resp, err, http.StatusCreated)
}

func (s *ScansHTTPWrapper) Get(params map[string]string) (*scansApi.ScansCollectionResponseModel, *scansApi.ErrorModel, error) {
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, s.path, params, nil)
	if err != nil {
		return nil, nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := scansApi.ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetAll)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := scansApi.ScansCollectionResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetAll)
		}
		return &model, nil, nil

	default:
		return nil, nil, errors.Errorf("Unknown response status code %d", resp.StatusCode)
	}
}

func (s *ScansHTTPWrapper) GetByID(scanID string) (*scansApi.ScanResponseModel, *scansApi.ErrorModel, error) {
	resp, err := SendHTTPRequest(http.MethodGet, s.path+"/"+scanID, nil, true)
	if err != nil {
		return nil, nil, err
	}
	return handleScanResponseWithBody(resp, err, http.StatusOK)
}

func (s *ScansHTTPWrapper) Delete(scanID string) (*scansApi.ErrorModel, error) {
	resp, err := SendHTTPRequest(http.MethodDelete, s.path+"/"+scanID, nil, true)
	if err != nil {
		return nil, err
	}
	return handleScanResponseWithNoBody(resp, err, http.StatusNoContent)
}

func (s *ScansHTTPWrapper) Tags() (map[string][]string, *scansApi.ErrorModel, error) {
	resp, err := SendHTTPRequest(http.MethodGet, s.path+"/tags", nil, true)
	if err != nil {
		return nil, nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := scansApi.ErrorModel{}
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
		return nil, nil, errors.Errorf("Unknown response status code %d", resp.StatusCode)
	}
}
