package http

import (
	"bytes"
	"encoding/json"
	"net/http"

	scansModels "github.com/checkmarxDev/scans/pkg/scans"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type ScansWrapper interface {
	Create(input []byte) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error)
	Get() (*scansModels.ResponseModel, *scansModels.ErrorModel, error)
	GetByID(scanID string) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error)
	Delete(scanID string) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error)
}

type ScansHTTPWrapper struct {
	url         string
	contentType string
}

func (s *ScansHTTPWrapper) Create(input []byte) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error) {
	resp, err := http.Post(s.url, s.contentType, bytes.NewBuffer(input))
	if err != nil {
		msg := "Failed to create a scan"
		log.WithFields(log.Fields{
			"err": err,
		}).Error(msg)
		return nil, nil, errors.Wrapf(err, msg)
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	switch resp.StatusCode {
	case http.StatusBadRequest:
		errorModel := scansModels.ErrorModel{}
		decoder := json.NewDecoder(resp.Body)
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
	panic("implement me")
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
