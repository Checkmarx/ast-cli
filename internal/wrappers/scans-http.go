package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	failedToParseGetAll   = "Failed to parse list response"
	failedToParseTags     = "Failed to parse tags response"
	failedToParseBranches = "Failed to parse branches response"
	MaxLimit              = 300
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

func (s *ScansHTTPWrapper) Create(model *Scan) (*ScanResponseModel, *ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		return nil, nil, err
	}
	resp, err := SendHTTPRequest(http.MethodPost, s.path, bytes.NewBuffer(jsonBytes), true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	return handleScanResponseWithBody(resp, err, http.StatusCreated)
}

func (s *ScansHTTPWrapper) Get(params map[string]string) (*ScansCollectionResponseModel, *ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, s.path, params, nil, clientTimeout)
	if err != nil {
		return nil, nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetAll)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := ScansCollectionResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetAll)
		}
		return &model, nil, nil
	case http.StatusNotFound:
		return nil, nil, errors.Errorf("scan not found")
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (s *ScansHTTPWrapper) GetWithPagination(params map[string]string) (*ScansCollectionResponseModel, *ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	userLimit, err := parseLimit(params[commonParams.LimitQueryParam])

	if err != nil {
		return nil, nil, err
	}

	offset := 0
	combinedResponse := &ScansCollectionResponseModel{}

	for {
		currentLimit := min(userLimit-offset, MaxLimit)
		params[commonParams.LimitQueryParam] = fmt.Sprint(currentLimit)
		params[commonParams.OffsetQueryParam] = fmt.Sprint(offset)

		resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, s.path, params, nil, clientTimeout)
		if err != nil {
			return nil, nil, err
		}

		decoder := json.NewDecoder(resp.Body)
		defer func() {
			if err == nil {
				_ = resp.Body.Close()
			}
		}()

		switch resp.StatusCode {
		case http.StatusBadRequest, http.StatusInternalServerError:
			errorModel := ErrorModel{}
			err = decoder.Decode(&errorModel)
			if err != nil {
				return nil, nil, errors.Wrapf(err, failedToParseGetAll)
			}
			return nil, &errorModel, nil
		case http.StatusOK:
			model := ScansCollectionResponseModel{}
			err = decoder.Decode(&model)
			if err != nil {
				return nil, nil, errors.Wrapf(err, failedToParseGetAll)
			}

			// Combine the current response with the combined response
			combinedResponse.Scans = append(combinedResponse.Scans, model.Scans...)

			// Update the offset for the next request
			offset += len(model.Scans)

			// Check if we need to make another request
			if len(model.Scans) == 0 || (offset >= userLimit) {
				return combinedResponse, nil, nil
			}

		case http.StatusNotFound:
			return nil, nil, errors.Errorf("scan not found")
		default:
			return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
		}
	}
}

func parseLimit(limitParam string) (int, error) {
	if limitParam == "" {
		return -1, nil
	}

	limit, err := strconv.Atoi(limitParam)
	if err != nil {
		return 0, fmt.Errorf("invalid limit parameter: %v", err)
	}

	if limit <= 0 {
		return 0, fmt.Errorf("limit must be a positive integer")
	}

	return limit, nil
}

func (s *ScansHTTPWrapper) GetByID(scanID string) (*ScanResponseModel, *ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	resp, err := SendHTTPRequest(http.MethodGet, s.path+"/"+scanID, http.NoBody, true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	return handleScanResponseWithBody(resp, err, http.StatusOK)
}

func (s *ScansHTTPWrapper) GetWorkflowByID(scanID string) ([]*ScanTaskResponseModel, *ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	path := fmt.Sprintf("%s/%s/workflow", s.path, scanID)
	resp, err := SendHTTPRequest(http.MethodGet, path, http.NoBody, true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	return handleWorkflowResponseWithBody(resp, err)
}

func handleWorkflowResponseWithBody(resp *http.Response, err error) ([]*ScanTaskResponseModel, *ErrorModel, error) {
	if err != nil {
		return nil, nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "Failed to parse workflow response")
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		var model []*ScanTaskResponseModel
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "Failed to parse workflow response")
		}
		return model, nil, nil

	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (s *ScansHTTPWrapper) Delete(scanID string) (*ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	resp, err := SendHTTPRequest(http.MethodDelete, s.path+"/"+scanID, http.NoBody, true, clientTimeout)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	return handleScanResponseWithNoBody(resp, err, http.StatusNoContent)
}

func (s *ScansHTTPWrapper) Cancel(scanID string) (*ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	b, err := json.Marshal(&CancelScanModel{
		Status: ScanCanceled,
	})
	if err != nil {
		return nil, err
	}

	resp, err := SendHTTPRequest(http.MethodPatch, s.path+"/"+scanID, bytes.NewBuffer(b), true, clientTimeout)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	return handleScanResponseWithNoBody(resp, err, http.StatusNoContent)
}

func (s *ScansHTTPWrapper) Tags() (map[string][]string, *ErrorModel, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	resp, err := SendHTTPRequest(http.MethodGet, s.path+"/tags", http.NoBody, true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
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
