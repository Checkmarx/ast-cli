package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"

	"github.com/pkg/errors"
)

const (
	failedToParseGetResults = "Failed to parse list results"
)

type ResultsHTTPWrapper struct {
	path           string
	scaPackagePath string
}

func NewHTTPResultsWrapper(path, scaPackagePath string) ResultsWrapper {
	return &ResultsHTTPWrapper{
		path:           path,
		scaPackagePath: scaPackagePath,
	}
}

func (r *ResultsHTTPWrapper) GetAllResultsByScanID(params map[string]string) (
	*ScanResultsCollection,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	// AST has a limit of 10000 results, this makes it get all of them
	params["limit"] = limitValue
	resp, err := SendPrivateHTTPRequestWithQueryParams(http.MethodGet, r.path, params, nil, clientTimeout)
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
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
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
		nil,
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
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

type GraphQLResponse struct {
	Data   interface{} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func (r *ResultsHTTPWrapper) GetAllResultsPackageByScanIDPoc() (
	*[]ScaPackageCollection,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	query := `query ( $scanId: UUID!, $isExploitablePathEnabled: Boolean!,$packageId: String) { package ( scanId: $scanId, isExploitablePathEnabled: $isExploitablePathEnabled, packageId: $packageId) { packageId, locations,dependencyPaths:dependencyPath{name,version,isResolved,isDevelopment,vulnerabilityRiskLevel}} }`
	variables := map[string]interface{}{
		"scanId" :"68a2321f-b409-425b-b7cd-a04f9aab8373",
		"skip": 0,
		"take":10,
		"isExploitablePathEnabled":true,
		"packageId":"Npm-debug-2.6.9",
	}
	body := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}
	jsonBody, err := json.Marshal(body)

	if err != nil {
		fmt.Printf("Error marshaling the graphQL query  %v", err)
	}
	resp, err := SendPrivateHTTPRequest(
		http.MethodPost,
		"api/sca/graphql/graphql",
		bytes.NewBuffer(jsonBody),
		clientTimeout,
		true,
	)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	decoder := json.NewDecoder(resp.Body)
	fmt.Printf("resp status %v\n",resp.StatusCode)
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		var model GraphQLResponse
		err = decoder.Decode(&model)
		println("ok")
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		fmt.Printf("Response from graphQL : %v\n",model)
		return nil, nil, nil
	case http.StatusNotFound: // scan was not triggered with SCA type or SCA scan didn't start yet
		logger.PrintIfVerbose("SCA packages for enrichment not found")
		return nil, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
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
		nil,
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
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
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
