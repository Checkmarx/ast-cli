package wrappers

import (
	"bytes"
	"encoding/json"
	"net/http"

	scansApi "github.com/checkmarxDev/scans/api/v1/rest/scans"
	"github.com/pkg/errors"
)

const (
	failedToParseGetAll = "Failed to parse list response"
	failedToParseTags   = "Failed to parse tags response"
)

type ScansHTTPWrapper struct {
	url         string
	contentType string
}

func NewHTTPScansWrapper(url string) ScansWrapper {
	return &ScansHTTPWrapper{
		url:         url,
		contentType: "application/json",
	}
}

func (s *ScansHTTPWrapper) Create(model *scansApi.Scan) (*scansApi.ScanResponseModel, *scansApi.ErrorModel, error) {
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		return nil, nil, err
	}
	resp, err := SendHTTPRequest(http.MethodPost, s.url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, nil, err
	}
	if err != nil {
		return nil, nil, err
	}
	return handleScanResponseWithBody(resp, err, http.StatusCreated)
}

func (s *ScansHTTPWrapper) Get(limit, offset uint64) (*scansApi.SlicedScansResponseModel, *scansApi.ErrorModel, error) {
	resp, err := SendHTTPRequestWithLimitAndOffset(http.MethodGet, s.url, make(map[string]string), limit, offset, nil)
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
		model := scansApi.SlicedScansResponseModel{}
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
	resp, err := SendHTTPRequest(http.MethodGet, s.url+"/"+scanID, nil)
	if err != nil {
		return nil, nil, err
	}
	return handleScanResponseWithBody(resp, err, http.StatusOK)
}

func (s *ScansHTTPWrapper) Delete(scanID string) (*scansApi.ErrorModel, error) {
	resp, err := SendHTTPRequest(http.MethodDelete, s.url+"/"+scanID, nil)
	if err != nil {
		return nil, err
	}
	return handleScanResponseWithNoBody(resp, err, http.StatusOK)
}

func (s *ScansHTTPWrapper) Tags() (*[]string, *scansApi.ErrorModel, error) {
	resp, err := SendHTTPRequest(http.MethodGet, s.url+"/tags", nil)
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
