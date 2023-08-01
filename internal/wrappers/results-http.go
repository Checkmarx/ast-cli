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

func (r *ResultsHTTPWrapper) GetResultsWithDevByScanID(scanID string, hideDevDependencies bool,take int ,skip int) (
	*VulnerabilitiesRisks,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	// Build the query field for the graphQL request
	query := `query ($where: VulnerabilityModelFilterInput, $take: Int!, $skip: Int!, $order: [VulnerabilitiesSort!], $scanId: UUID!, $isExploitablePathEnabled: Boolean!) { vulnerabilitiesRisksByScanId (where: $where, take: $take, skip: $skip, order: $order, scanId: $scanId, isExploitablePathEnabled: $isExploitablePathEnabled) { totalCount, items { state, isIgnored, cve, cwe, description, packageId, severity, type, published, vulnerabilityFixResolutionText, score, violatedPolicies, isExploitable, isKevDataExists, isExploitDbDataExists, relation, cweInfo { title }, packageInfo { name, packageRepository, version }, exploitablePath { methodMatch { fullName, line, namespace, shortName, sourceFile }, methodSourceCall { fullName, line, namespace, shortName, sourceFile } }, vulnerablePackagePath { id, isDevelopment, isResolved, name, version, vulnerabilityRiskLevel }, references { comment, type, url }, cvss2 { attackComplexity, attackVector, authentication, availability, availabilityRequirement, baseScore, collateralDamagePotential, confidentiality, confidentialityRequirement, exploitCodeMaturity, integrityImpact, integrityRequirement, remediationLevel, reportConfidence, targetDistribution }, cvss3 { attackComplexity, attackVector, availability, availabilityRequirement, baseScore, confidentiality, confidentialityRequirement, exploitCodeMaturity, integrity, integrityRequirement, privilegesRequired, remediationLevel, reportConfidence, scope, userInteraction } } } }`
	// Create a map to represent the "where" object to use in variables
	where := make(map[string]interface{})
	// Case the hide dev dependencies flag is being used
	if hideDevDependencies {
		and := make(map[string]interface{})
		and["isDev"] = map[string]bool{"eq": false}
		and["isTest"] = map[string]bool{"eq": false}
		where["and"] = and
	} else { // otherwise, send the where object as nil
		where = nil
	}
	// build the variables field for the graphQL request
	variables := map[string]interface{}{
		"where":                    where,
		"skip":                     skip,
		"take":                     take,
		"order":                    map[string]string{"score": "DESC"},
		"isExploitablePathEnabled": true,
		"scanId":                   scanID,
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
		"api/sca/graphql/graphql", // TO DO: Add this to constants
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
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		var model GraphQLVulnerabilityRisks
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetResults)
		}
		return &model.Data, nil, nil
	case http.StatusNotFound:
		logger.PrintIfVerbose("SCA results for dev dependencies filter not found")
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
