package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/checkmarx/ast-cli/internal/params"
	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	SAST                    = "sast"
	KICS                    = "kics"
	failedToParsePredicates = "Failed to parse predicates response."
)

type ResultsPredicatesHTTPWrapper struct {
	path string
}

func NewResultsPredicatesHTTPWrapper() ResultsPredicatesWrapper {
	return &ResultsPredicatesHTTPWrapper{}
}

func (r *ResultsPredicatesHTTPWrapper) GetAllPredicatesForSimilarityID(similarityID, projectID, scannerType string) (*PredicatesCollectionResponseModel, *resultsHelpers.WebError, error) {

	clientTimeout := viper.GetUint(params.ClientTimeoutKey)

	var triageAPIPath = ""
	if strings.EqualFold(strings.TrimSpace(scannerType), KICS) {
		triageAPIPath = viper.GetString(params.KicsResultsPredicatesPathKey)
	} else if strings.EqualFold(strings.TrimSpace(scannerType), SAST) {
		triageAPIPath = viper.GetString(params.SastResultsPredicatesPathKey)
	}
	fmt.Println("Fetching the predicate history for SimilarityId : " + similarityID)
	r.SetPath(triageAPIPath)

	var request = "/" + similarityID + "?project-ids=" + projectID
	PrintIfVerbose(fmt.Sprintf("Sending GET request to %s", r.path+request))
	resp, err := SendHTTPRequest(http.MethodGet, r.path+request, nil, true, clientTimeout)
	PrintIfVerbose(fmt.Sprintf("Response : %s", resp.Status))
	if err != nil {
		return nil, nil, err
	}
	return handleResponseWithBody(resp, err, http.StatusOK)

}

func (r *ResultsPredicatesHTTPWrapper) SetPath(newPath string) {
	r.path = newPath
}

func (r ResultsPredicatesHTTPWrapper) PredicateSeverityAndState(predicate *PredicateRequest) (*resultsHelpers.WebError, error) {

	clientTimeout := viper.GetUint(params.ClientTimeoutKey)
	b := [...]PredicateRequest{*predicate}
	jsonBytes, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}

	triageAPIPath := ""
	if strings.EqualFold(strings.TrimSpace(predicate.ScannerType), SAST) {
		triageAPIPath = viper.GetString(params.SastResultsPredicatesPathKey)
	} else {
		triageAPIPath = viper.GetString(params.KicsResultsPredicatesPathKey)
	}
	PrintIfVerbose(fmt.Sprintf("Sending POST request to  %s", triageAPIPath))
	PrintIfVerbose(fmt.Sprintf("Request Payload:  %s", string(jsonBytes)))

	r.SetPath(triageAPIPath)

	resp, err2 := SendHTTPRequest(http.MethodPost, r.path, bytes.NewBuffer(jsonBytes), true, clientTimeout)
	PrintIfVerbose(fmt.Sprintf("Response : %s", resp.Status))

	if err2 != nil {
		return nil, err2
	}
	if resp.StatusCode == http.StatusOK {
		fmt.Println("Predicate updated successfully.")
	}
	return nil, nil
}

func handleResponseWithBody(resp *http.Response, err error,
	successStatusCode int) (*PredicatesCollectionResponseModel, *resultsHelpers.WebError, error) {
	if err != nil {
		return nil, nil, err
	}
	decoder := json.NewDecoder(resp.Body)

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := resultsHelpers.WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return responsePredicateParsingFailed(err)
		}
		return nil, &errorModel, nil
	case successStatusCode:
		model := PredicatesCollectionResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return responsePredicateParsingFailed(err)
		}
		return &model, nil, nil
	case http.StatusNotFound:
		return nil, nil, errors.Errorf("Predicate not found.")
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func responsePredicateParsingFailed(err error) (*PredicatesCollectionResponseModel, *resultsHelpers.WebError, error) {
	return nil, nil, errors.Wrapf(err, failedToParsePredicates)
}
