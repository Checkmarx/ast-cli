package wrappers

import (
	"encoding/json"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"

	"github.com/pkg/errors"
)

const (
	failedToParseGetPolicies = "Failed to parse policy"
	failedToGetPolicies      = "Failed to evaluate policy"
)

type PolicyHTTPWrapper struct {
	path string
}

func NewHTTPPolicyWrapper(path string) PolicyWrapper {
	return &PolicyHTTPWrapper{
		path: path,
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
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
