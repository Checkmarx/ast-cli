package wrappers

import (
	"bytes"
	"encoding/json"
	"net/http"

	scansApi "github.com/checkmarxDev/scans/api/v1/rest/scans"
	scansModels "github.com/checkmarxDev/scans/pkg/scans"
	"github.com/pkg/errors"
)

type ScansHTTPWrapper struct {
	url         string
	contentType string
}

func (s *ScansHTTPWrapper) Create(model *scansApi.Scan) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error) {
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		return nil, nil, err
	}

	resp, err := http.Post(s.url, s.contentType, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := scansModels.ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return responseParsingFailed(err, resp.StatusCode)
		}
		return nil, &errorModel, nil
	case http.StatusCreated:
		model := scansModels.ScanResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return responseParsingFailed(err, resp.StatusCode)
		}
		return &model, nil, nil

	default:
		return nil, nil, errors.Errorf("Unknown response status code %d", resp.StatusCode)
	}
}

func (s *ScansHTTPWrapper) Get() (*scansModels.ResponseModel, *scansModels.ErrorModel, error) {
	panic("implement me")
}

func (s *ScansHTTPWrapper) GetByID(scanID string) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error) {
	resp, err := http.Get(s.url + "/" + scanID)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := scansModels.ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return responseParsingFailed(err, resp.StatusCode)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := scansModels.ScanResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return responseParsingFailed(err, resp.StatusCode)
		}
		return &model, nil, nil

	default:
		return nil, nil, errors.Errorf("Unknown response status code %d", resp.StatusCode)
	}
}

func (s *ScansHTTPWrapper) Delete(scanID string) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error) {
	panic("implement me")
}

func NewHTTPScansWrapper(url string) ScansWrapper {
	return &ScansHTTPWrapper{
		url:         url,
		contentType: "application/json",
	}
}

func responseParsingFailed(err error, statusCode int) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error) {
	msg := "Failed to parse a scan response"
	return nil, nil, errors.Wrapf(err, msg)
}
