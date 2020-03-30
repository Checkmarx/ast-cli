package wrappers

import (
	"bytes"
	"encoding/json"
	"net/http"

	scansApi "github.com/checkmarxDev/scans/api/v1/rest/scans"
	"github.com/pkg/errors"
)

const (
	failedToParseGetAll = "Failed to parse get-all response"
	failedToParseTags   = "Failed to parse tags response"
)

type ScansHTTPWrapper struct {
	url         string
	contentType string
}

func (s *ScansHTTPWrapper) Create(model *scansApi.Scan) (*scansApi.ScanResponseModel, *scansApi.ErrorModel, error) {
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		return nil, nil, err
	}

	resp, err := http.Post(s.url, s.contentType, bytes.NewBuffer(jsonBytes))
	return handleScanResponseWithBody(resp, err, http.StatusCreated)
}

func (s *ScansHTTPWrapper) Get(limit, offset uint) (*scansApi.SlicedScansResponseModel, *scansApi.ErrorModel, error) {
	resp, err := http.Get(s.url)
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
	resp, err := http.Get(s.url + "/" + scanID)
	if err != nil {
		return nil, nil, err
	}
	return handleScanResponseWithBody(resp, err, http.StatusOK)
}

func (s *ScansHTTPWrapper) Delete(scanID string) (*scansApi.ErrorModel, error) {
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", s.url+"/"+scanID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	return handleScanResponseWithNoBody(resp, err, http.StatusOK)
}

func (s *ScansHTTPWrapper) Tags() (*[]string, *scansApi.ErrorModel, error) {
	resp, err := http.Get(s.url + "/tags")
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

func NewHTTPScansWrapper(url string) ScansWrapper {
	return &ScansHTTPWrapper{
		url:         url,
		contentType: "application/json",
	}
}
