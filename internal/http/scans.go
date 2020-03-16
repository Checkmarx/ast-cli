package http

import (
	"bytes"
	"encoding/json"
	"net/http"

	scansRest "github.com/checkmarxDev/scans/api/v1/rest"
	scansModels "github.com/checkmarxDev/scans/pkg/scans"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type ScansWrapper interface {
	Create(scan *scansRest.Scan) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error)
	Get() (*scansModels.ResponseModel, *scansModels.ErrorModel, error)
	GetByID(scanID string) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error)
	Delete(scanID string) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error)
}

type ScansHTTPWrapper struct {
	endpoint    string
	contentType string
}

func (s *ScansHTTPWrapper) Create(scan *scansRest.Scan) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error) {
	payload, err := json.Marshal(scan)
	if err != nil {
		msg := "Failed to serialize scan body"
		log.WithFields(log.Fields{
			"err": err,
		}).Error(msg)
		return nil, nil, errors.Wrapf(err, msg)
	}

	resp, err := http.Post(s.endpoint, s.contentType, bytes.NewBuffer(payload))
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
		log.WithFields(log.Fields{
			"status_code": resp.StatusCode,
		}).Error("Unknown response status code")
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

func NewHTTPScansWrapper(endpoint string) ScansWrapper {
	return &ScansHTTPWrapper{
		endpoint:    endpoint,
		contentType: "application/json",
	}
}

func responseParsingFailed(err error, statusCode int) (*scansModels.ScanResponseModel, *scansModels.ErrorModel, error) {
	msg := "Failed to parse a scan response"
	log.WithFields(log.Fields{
		"err":         err,
		"status_code": statusCode,
	}).Error(msg)
	return nil, nil, errors.Wrapf(err, msg)
}
