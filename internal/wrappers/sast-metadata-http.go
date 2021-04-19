package wrappers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/checkmarxDev/sast-metadata/pkg/api/v1/rest"
	"github.com/pkg/errors"
)

type SastMetadataHTTPWrapper struct {
	basePath          string
	metricsPathFormat string
}

const (
	failedToParseScanInfoResult = "failed to parse scan info result"
	failedToParseMetricsResult  = "failed ot parse metrics result"
)

func NewSastMetadataHTTPWrapper(basePath, metricsPathFormat string) SastMetadataWrapper {
	return &SastMetadataHTTPWrapper{
		basePath:          basePath,
		metricsPathFormat: metricsPathFormat,
	}
}

func (s *SastMetadataHTTPWrapper) GetScanInfo(scanID string) (*rest.ScanInfo, *rest.Error, error) {
	resp, err := SendHTTPRequest(http.MethodGet, fmt.Sprintf("%s/%s", s.basePath, scanID),
		nil, true, DefaultTimeoutSeconds)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusInternalServerError:
		return nil, nil, errors.New("internal server error")
	case http.StatusNotFound, http.StatusBadRequest:
		errorModel := &rest.Error{}
		err = decoder.Decode(errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseScanInfoResult)
		}

		return nil, errorModel, nil
	case http.StatusOK:
		model := &rest.ScanInfo{}
		err := decoder.Decode(model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseScanInfoResult)
		}

		return model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (s *SastMetadataHTTPWrapper) GetMetrics(scanID string) (*rest.Metrics, *rest.Error, error) {
	resp, err := SendHTTPRequest(http.MethodGet, fmt.Sprintf(s.metricsPathFormat, scanID), nil, true, DefaultTimeoutSeconds)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusInternalServerError:
		return nil, nil, errors.New("internal server error")
	case http.StatusNotFound, http.StatusBadRequest:
		errorModel := &rest.Error{}
		err = decoder.Decode(errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseMetricsResult)
		}

		return nil, errorModel, nil
	case http.StatusOK:
		model := &rest.Metrics{}
		err := decoder.Decode(model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseMetricsResult)
		}

		return model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
