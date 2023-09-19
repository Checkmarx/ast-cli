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
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	// AST has a limit of 10000 results, this makes it get all of them
	params[limit] = limitValue
	resp, err := SendPrivateHTTPRequestWithQueryParams(http.MethodGet, r.resultsPath, params, http.NoBody, clientTimeout)
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
		model := ScanResultsCollection{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}

		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf(respStatusCode, resp.StatusCode)
	}
}

func (r *ResultsHTTPWrapper) GetAllResultsPackageByScanID(params map[string]string) (
	*[]ScaPackageCollection,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	// AST has a limit of 10000 results, this makes it get all of them
	params["limit"] = limitValue
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
	params["limit"] = limitValue
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

	defer resp.Body.Close()

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

	defer resp.Body.Close()

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
