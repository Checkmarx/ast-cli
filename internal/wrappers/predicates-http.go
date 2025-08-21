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
	} else if strings.EqualFold(strings.TrimSpace(scannerType), params.ScsType) {
		triageAPIPath = viper.GetString(params.ScsResultsReadPredicatesPathKey)
	} else if strings.EqualFold(strings.TrimSpace(scannerType), params.ScaType) {
		return &PredicatesCollectionResponseModel{}, nil, nil
	} else {
		return nil, nil, errors.Errorf(invalidScanType, scannerType)
	}

	logger.PrintIfVerbose(fmt.Sprintf("Fetching the predicate history for SimilarityID : %s", similarityID))
	r.SetPath(triageAPIPath)

	var request = "/" + similarityID + "?project-ids=" + projectID
	logger.PrintIfVerbose(fmt.Sprintf("Sending GET request to %s", r.path+request))
	resp, err := SendHTTPRequest(http.MethodGet, r.path+request, http.NoBody, true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	return handleResponseWithBody(resp, err)
}

func (r *ResultsPredicatesHTTPWrapper) SetPath(newPath string) {
	r.path = newPath
}

func (r *ResultsPredicatesHTTPWrapper) PredicateScaState(predicate *ScaPredicateRequest) (*WebError, error) {
	clientTimeout := viper.GetUint(params.ClientTimeoutKey)
	jsonBody, err := json.Marshal(predicate)
	if err != nil {
		return nil, err
	}
	var scaTriageAPIPath string

	scaTriageAPIPath = viper.GetString(params.ScaResultsPredicatesPathEnv)
	logger.PrintIfVerbose(fmt.Sprintf("Sending POST request to  %s", scaTriageAPIPath))
	logger.PrintIfVerbose(fmt.Sprintf("Request Payload:  %s", string(jsonBody)))

	r.SetPath(scaTriageAPIPath)

	resp, err := SendHTTPRequestWithJSONContentType(http.MethodPost, r.path, bytes.NewBuffer(jsonBody), true, clientTimeout)
	if err != nil {
		return nil, err
	}
	logger.PrintIfVerbose(fmt.Sprintf("Response : %s", resp.Status))
	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		fmt.Println("Predicate updated successfully.")
		return nil, nil
	case http.StatusForbidden:
		return nil, errors.Errorf("You do not have permission to update state")
	case http.StatusNotFound:
		return nil, errors.Errorf("Predicate not found.")
	case http.StatusBadRequest, http.StatusInternalServerError:
		return nil, errors.Errorf("Predicate bad request.")

	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (r ResultsPredicatesHTTPWrapper) PredicateSeverityAndState(predicate interface{}, scanType string) (
	*WebError, error,
) {
	clientTimeout := viper.GetUint(params.ClientTimeoutKey)
	var predicateModel interface{}
	if !strings.EqualFold(strings.TrimSpace(scanType), params.ScaType) {
		predicateModel = []interface{}{predicate}
	} else {
		predicateModel = predicate
	}
	jsonBytes, err := json.Marshal(predicateModel)
	if err != nil {
		return nil, err
	}

	var triageAPIPath string
	if strings.EqualFold(strings.TrimSpace(scanType), params.SastType) {
		triageAPIPath = viper.GetString(params.SastResultsPredicatesPathKey)
	} else if strings.EqualFold(strings.TrimSpace(scanType), params.KicsType) || strings.EqualFold(strings.TrimSpace(scanType), params.IacType) {
		triageAPIPath = viper.GetString(params.KicsResultsPredicatesPathKey)
	} else if strings.EqualFold(strings.TrimSpace(scanType), params.ScsType) {
		triageAPIPath = viper.GetString(params.ScsResultsWritePredicatesPathKey)
	} else if strings.EqualFold(strings.TrimSpace(scanType), params.ScaType) {
		triageAPIPath = viper.GetString(params.ScaResultsPredicatesPathEnv)
	} else {
		return nil, errors.Errorf(invalidScanType, scanType)
	}

	logger.PrintIfVerbose(fmt.Sprintf("Sending POST request to  %s", triageAPIPath))
	logger.PrintIfVerbose(fmt.Sprintf("Request Payload:  %s", string(jsonBytes)))

	r.SetPath(triageAPIPath)

	resp, err := SendHTTPRequestWithJSONContentType(http.MethodPost, r.path, bytes.NewBuffer(jsonBytes), true, clientTimeout)
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
	case http.StatusOK, http.StatusCreated:
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

type CustomStatesHTTPWrapper struct {
	path string
}

func NewCustomStatesHTTPWrapper() CustomStatesWrapper {
	return &CustomStatesHTTPWrapper{
		path: viper.GetString(params.CustomStatesAPIPathKey),
	}
}

func (c *CustomStatesHTTPWrapper) GetAllCustomStates(includeDeleted bool) ([]CustomState, error) {
	clientTimeout := viper.GetUint(params.ClientTimeoutKey)

	if c.path == "" {
		return nil, errors.New("CustomStatesAPIPathKey is not set")
	}
	queryParams := make(map[string]string)
	if includeDeleted {
		queryParams[params.IncludeDeletedQueryParam] = params.True
	}

	logger.PrintIfVerbose(fmt.Sprintf("Fetching custom states from: %s with params: %v", c.path, queryParams))
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, c.path, queryParams, http.NoBody, clientTimeout)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Failed to fetch custom states. HTTP status: %d", resp.StatusCode)
	}
	var states []CustomState
	err = json.NewDecoder(resp.Body).Decode(&states)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to parse custom states response")
	}
	return states, nil
}
