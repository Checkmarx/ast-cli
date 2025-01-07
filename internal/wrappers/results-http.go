package wrappers

import (
	"encoding/json"
	"fmt"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"

	"github.com/pkg/errors"
)

const (
	failedToParseGetResults = "Failed to parse list results"
	respStatusCode          = "response status code %d"
	sort                    = "sort"
	sortResultsDefault      = "-severity"
	offset                  = "offset"
	astAPIPageLen           = 10_000
	astAPIPagingValue       = "10000"
)

type ResultsHTTPWrapper struct {
	resultsPath     string
	scanSummaryPath string
}

func NewHTTPResultsWrapper(path, scanSummaryPath string) ResultsWrapper {
	return &ResultsHTTPWrapper{
		resultsPath:     path,
		scanSummaryPath: scanSummaryPath,
	}
}

func (r *ResultsHTTPWrapper) GetAllResultsByScanID(params map[string]string) (
	*ScanResultsCollection,
	*WebError,
	error,
) {
	var scanModelslice []ScanResultsCollection
	var scanModel ScanResultsCollection
	DefaultMapValue(params, limit, astAPIPagingValue)
	DefaultMapValue(params, sort, sortResultsDefault)

	webErr, err := getResultsWithPagination(r.resultsPath, params, &scanModelslice)
	if err != nil {
		return &scanModel, nil, err
	}
	if webErr != nil {
		return &scanModel, webErr, nil
	}
	for _, resultsPage := range scanModelslice {
		scanModel.Results = append(scanModel.Results, resultsPage.Results...)
		scanModel.TotalCount = resultsPage.TotalCount
	}
	return &scanModel, nil, nil
}
func getResultsWithPagination(resultPath string, queryParams map[string]string, slice *[]ScanResultsCollection) (*WebError, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	var currentPage = 0
	for {
		queryParams[offset] = fmt.Sprintf("%d", currentPage)
		target, hasNextPage, weberr, err := getResultsByOffset(resultPath, queryParams, clientTimeout)
		if err != nil {
			return nil, err
		}

		if weberr != nil {
			return weberr, nil
		}

		*slice = append(*slice, *target)

		if !hasNextPage {
			break
		}
		if astAPIPageLen > int(target.TotalCount) {
			break
		}
		currentPage++
	}
	return nil, nil
}
func getResultsByOffset(resultPath string, params map[string]string, clientTimeout uint) (*ScanResultsCollection, bool, *WebError, error) {
	resp, err := SendPrivateHTTPRequestWithQueryParams(http.MethodGet, resultPath, params, http.NoBody, clientTimeout)
	if err != nil {
		return nil, false, nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, false, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return nil, false, &errorModel, nil
	case http.StatusOK:
		model := ScanResultsCollection{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, false, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		if err != nil {
			return nil, false, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		if len(model.Results) == 0 {
			return &model, false, nil, nil
		}
		return &model, true, nil, nil
	default:
		return nil, false, nil, errors.Errorf(respStatusCode, resp.StatusCode)
	}
}

func (r *ResultsHTTPWrapper) GetResultsURL(projectID string) (string, error) {
	accessToken, err := GetAccessToken()
	if err != nil {
		return "", err
	}
	baseURI, err := GetURL(fmt.Sprintf("projects/%s/overview", projectID), accessToken)
	if err != nil {
		return "", err
	}

	return baseURI, nil
}

func DefaultMapValue(params map[string]string, key, value string) {
	if _, ok := params[key]; !ok {
		params[key] = value
	}
}
