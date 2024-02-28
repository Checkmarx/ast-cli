package wrappers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"

	"github.com/pkg/errors"
)

const (
	failedToParseGetResults    = "Failed to parse list results"
	failedTogetScanSummaries   = "Failed to get scan summaries"
	failedToParseScanSummaries = "Failed to parse scan summaries"
	respStatusCode             = "response status code %d"
	sort                       = "sort"
	sortResultsDefault         = "-severity"
	offset                     = "offset"
	astAPIPageLen              = 1000
	astAPIPagingValue          = "1000"
)

type ResultsHTTPWrapper struct {
	resultsPath     string
	scanSummaryPath string
	scaPackagePath  string
}

func NewHTTPResultsWrapper(path, scaPackagePath, scanSummaryPath string) ResultsWrapper {
	return &ResultsHTTPWrapper{
		resultsPath:     path,
		scanSummaryPath: scanSummaryPath,
		scaPackagePath:  scaPackagePath,
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
func (r *ResultsHTTPWrapper) GetAllResultsPackageByScanID(params map[string]string) (
	*[]ScaPackageCollection,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	// AST has a limit of 10000 results, this makes it get all of them
	params[limit] = limitValue
	resp, err := SendPrivateHTTPRequestWithQueryParams(
		http.MethodGet,
		r.scaPackagePath+params[commonParams.ScanIDQueryParam]+"/packages",
		params,
		http.NoBody,
		clientTimeout,
	)
	if err != nil {
		return nil, nil, err
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
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		var model []ScaPackageCollection
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return &model, nil, nil
	case http.StatusNotFound: // scan was not triggered with SCA type or SCA scan didn't start yet
		logger.PrintIfVerbose("SCA packages for enrichment not found")
		return nil, nil, nil
	default:
		return nil, nil, errors.Errorf(respStatusCode, resp.StatusCode)
	}
}

func (r *ResultsHTTPWrapper) GetAllResultsTypeByScanID(params map[string]string) (
	*[]ScaTypeCollection,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	// AST has a limit of 10000 results, this makes it get all of them
	params[limit] = limitValue
	resp, err := SendPrivateHTTPRequestWithQueryParams(
		http.MethodGet,
		r.scaPackagePath+params[commonParams.ScanIDQueryParam]+"/vulnerabilities",
		params,
		http.NoBody,
		clientTimeout,
	)
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
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		var model []ScaTypeCollection
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return &model, nil, nil
	case http.StatusNotFound: // scan was not triggered with SCA type or SCA scan didn't start yet
		logger.PrintIfVerbose("SCA types for enrichment not found")
		return nil, nil, nil
	default:
		return nil, nil, errors.Errorf(respStatusCode, resp.StatusCode)
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

// GetScanSummariesByScanIDS will no longer be used because it does not support --filters flag
func (r *ResultsHTTPWrapper) GetScanSummariesByScanIDS(params map[string]string) (
	*ScanSummariesModel,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)

	resp, err := SendPrivateHTTPRequestWithQueryParams(http.MethodGet, r.scanSummaryPath, params, http.NoBody, clientTimeout)
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
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedTogetScanSummaries)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := ScanSummariesModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseScanSummaries)
		}

		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf(respStatusCode, resp.StatusCode)
	}
}

func DefaultMapValue(params map[string]string, key, value string) {
	if _, ok := params[key]; !ok {
		params[key] = value
	}
}
