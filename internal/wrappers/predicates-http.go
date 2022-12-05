package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	failedToParsePredicates = "Failed to parse predicates response."
	invalidScanType         = "Invalid scan type %s"
)

type ResultsPredicatesHTTPWrapper struct {
	path string
}

func NewResultsPredicatesHTTPWrapper() ResultsPredicatesWrapper {
	return &ResultsPredicatesHTTPWrapper{}
}

func (r *ResultsPredicatesHTTPWrapper) GetAllPredicatesForSimilarityID(similarityID, projectID, scannerType string) (
	*PredicatesCollectionResponseModel, *WebError, error,
) {
	clientTimeout := viper.GetUint(params.ClientTimeoutKey)

	var triageAPIPath string
	if strings.EqualFold(strings.TrimSpace(scannerType), params.KicsType) || strings.EqualFold(strings.TrimSpace(scannerType), params.IacType) {
		triageAPIPath = viper.GetString(params.KicsResultsPredicatesPathKey)
	} else if strings.EqualFold(strings.TrimSpace(scannerType), params.SastType) {
		triageAPIPath = viper.GetString(params.SastResultsPredicatesPathKey)
	} else if strings.EqualFold(strings.TrimSpace(scannerType), params.ScaType) {
		return &PredicatesCollectionResponseModel{}, nil, nil
	} else {
		return nil, nil, errors.Errorf(invalidScanType, scannerType)
	}

	logger.PrintIfVerbose(fmt.Sprintf("Fetching the predicate history for SimilarityID : %s", similarityID))
	r.SetPath(triageAPIPath)

	var request = "/" + similarityID + "?project-ids=" + projectID
	logger.PrintIfVerbose(fmt.Sprintf("Sending GET request to %s", r.path+request))

	return handleResponseWithBody(SendHTTPRequest(http.MethodGet, r.path+request, nil, true, clientTimeout))
}

func (r *ResultsPredicatesHTTPWrapper) SetPath(newPath string) {
	r.path = newPath
}

func (r ResultsPredicatesHTTPWrapper) PredicateSeverityAndState(predicate *PredicateRequest) (
	*WebError, error,
) {
	clientTimeout := viper.GetUint(params.ClientTimeoutKey)
	b := [...]PredicateRequest{*predicate}
	jsonBytes, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}

	var triageAPIPath string
	if strings.EqualFold(strings.TrimSpace(predicate.ScannerType), params.SastType) {
		triageAPIPath = viper.GetString(params.SastResultsPredicatesPathKey)
	} else if strings.EqualFold(strings.TrimSpace(predicate.ScannerType), params.KicsType) || strings.EqualFold(strings.TrimSpace(predicate.ScannerType), params.IacType) {
		triageAPIPath = viper.GetString(params.KicsResultsPredicatesPathKey)
	} else {
		return nil, errors.Errorf(invalidScanType, predicate.ScannerType)
	}

	logger.PrintIfVerbose(fmt.Sprintf("Sending POST request to  %s", triageAPIPath))
	logger.PrintIfVerbose(fmt.Sprintf("Request Payload:  %s", string(jsonBytes)))

	r.SetPath(triageAPIPath)

	resp, err := SendHTTPRequest(http.MethodPost, r.path, bytes.NewBuffer(jsonBytes), true, clientTimeout)
	if err != nil {
		return nil, err
	}

	logger.PrintIfVerbose(fmt.Sprintf("Response : %s ", resp.Status))

	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		return nil, errors.Errorf("Predicate bad request.")
	case http.StatusOK:
		fmt.Println("Predicate updated successfully.")
		return nil, nil
	case http.StatusNotModified:
		return nil, errors.Errorf("No changes to update.")
	case http.StatusForbidden:
		return nil, errors.Errorf("No permission to update predicate.")
	case http.StatusNotFound:
		return nil, errors.Errorf("Predicate not found.")
	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func handleResponseWithBody(resp *http.Response, err error) (*PredicatesCollectionResponseModel, *WebError, error) {
	if err != nil {
		return nil, nil, err
	}

	logger.PrintIfVerbose(fmt.Sprintf("Response : %s", resp.Status))

	decoder := json.NewDecoder(resp.Body)

	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return responsePredicateParsingFailed(err)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := PredicatesCollectionResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return responsePredicateParsingFailed(err)
		}
		return &model, nil, nil
	case http.StatusForbidden:
		return nil, nil, errors.Errorf("No permission to show predicate.")
	case http.StatusNotFound:
		return nil, nil, errors.Errorf("Predicate not found.")
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func responsePredicateParsingFailed(err error) (*PredicatesCollectionResponseModel, *WebError, error) {
	return nil, nil, errors.Wrapf(err, failedToParsePredicates)
}
