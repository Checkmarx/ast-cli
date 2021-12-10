package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/checkmarx/ast-cli/internal/params"
	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"net/http"
	"strings"
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

func (r *ResultsPredicatesHTTPWrapper) GetAllPredicatesForSimilarityId(similarityId string, projectID string, scannerType string) (*PredicatesCollectionResponseModel, *resultsHelpers.WebError, error) {

	clientTimeout := viper.GetUint(params.ClientTimeoutKey)

	var triageApiPath = ""
	if strings.EqualFold(strings.TrimSpace(scannerType), KICS) {
		triageApiPath = viper.GetString(params.KicsResultsPredicatesPathKey)
	} else if strings.EqualFold(strings.TrimSpace(scannerType), SAST) {
		triageApiPath = viper.GetString(params.SastResultsPredicatesPathKey)
	}
	fmt.Println("Fetching the predicate history for SimilarityId : " + similarityId)
	r.SetPath(triageApiPath)

	var request = "/" + similarityId + "?project-ids=" + projectID
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

	triageApiPath := ""
	if strings.EqualFold(strings.TrimSpace(predicate.ScannerType), SAST) {
		triageApiPath = viper.GetString(params.SastResultsPredicatesPathKey)
	} else {
		triageApiPath = viper.GetString(params.KicsResultsPredicatesPathKey)
	}
	PrintIfVerbose(fmt.Sprintf("Sending POST request to  %s", triageApiPath))
	PrintIfVerbose(fmt.Sprintf("Request Payload:  %s", string(jsonBytes)))

	r.SetPath(triageApiPath)

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
