package wrappers

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

const (
	failedToParseErr = "Failed to parse error response"
)

func handleScanResponseWithNoBody(resp *http.Response, err error,
	successStatusCode int) (*ErrorModel, error) {
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError, http.StatusNotFound:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, errors.Wrapf(err, failedToParseErr)
		}
		return &errorModel, nil
	case successStatusCode:
		return nil, nil

	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func handleScanResponseWithBody(resp *http.Response, err error,
	successStatusCode int) (*ScanResponseModel, *ErrorModel, error) {
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
			return responseScanParsingFailed(err)
		}
		return nil, &errorModel, nil
	case successStatusCode:
		model := ScanResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return responseScanParsingFailed(err)
		}
		return &model, nil, nil
	case http.StatusNotFound:
		return nil, nil, errors.Errorf("scan not found")
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func handleProjectResponseWithNoBody(resp *http.Response, err error,
	successStatusCode int) (*ErrorModel, error) {
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, errors.Wrapf(err, failedToParseErr)
		}
		return &errorModel, nil
	case successStatusCode:
		return nil, nil

	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func handleProjectResponseWithBody(resp *http.Response, err error,
	successStatusCode int) (*ProjectResponseModel, *ErrorModel, error) {
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
			return responseProjectParsingFailed(err)
		}
		return nil, &errorModel, nil
	case http.StatusNotFound:
		return nil, nil, errors.Errorf("project not found")
	case successStatusCode:
		model := ProjectResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return responseProjectParsingFailed(err)
		}
		return &model, nil, nil

	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func responseScanParsingFailed(err error) (*ScanResponseModel, *ErrorModel, error) {
	msg := "Failed to parse scan response"
	return nil, nil, errors.Wrapf(err, msg)
}
func responseProjectParsingFailed(err error) (*ProjectResponseModel, *ErrorModel, error) {
	msg := "Failed to parse project response"
	return nil, nil, errors.Wrapf(err, msg)
}
