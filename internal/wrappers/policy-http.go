package wrappers

import (
	"encoding/json"
	"net/http"

	featureFlagsConstants "github.com/checkmarx/ast-cli/internal/constants/feature-flags"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"

	"github.com/pkg/errors"
)

const (
	failedToParseGetPolicies = "Failed to parse policy"
	failedToGetPolicies      = "Failed to evaluate policy"
)

type PolicyHTTPWrapper struct {
	path               string
	featureFlagWrapper FeatureFlagsWrapper
}

func NewHTTPPolicyWrapper(path string, featureFlagsWrapper FeatureFlagsWrapper) PolicyWrapper {
	return &PolicyHTTPWrapper{
		path:               path,
		featureFlagWrapper: featureFlagsWrapper,
	}
}

func (r *PolicyHTTPWrapper) EvaluatePolicy(params map[string]string) (
	*PolicyResponseModel,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
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
			return nil, nil, errors.Wrapf(err, failedToGetPolicies)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := PolicyResponseModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseGetPolicies)
		}
		return &model, nil, nil
	case http.StatusUnauthorized:
		// decoding the error response
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToGetPolicies)
		}
		// policyManagement checks if AM1 Feature flag is ON,if yes policy permissions are checked otherwise skipped
		amPhase1Enabled, _ := r.featureFlagWrapper.GetSpecificFlag(featureFlagsConstants.AccessManagementEnabled)
		if amPhase1Enabled != nil && amPhase1Enabled.Status {
			logger.Printf("This user doesn’t have the necessary permissions to view the Policy Management evaluation for this scan.")
			return nil, &errorModel, nil
		}
		fallthrough

	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
