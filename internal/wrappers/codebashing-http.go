package wrappers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/utils"
	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	failedToParseCodeBashing    = "Failed to parse list results"
	failedGettingCodeBashingURL = "Authentication failed, not able to retrieve codebashing base link"
	limit                       = "limit"
	noCodebashingLinkAvailable  = "No codebashing link available"
	licenseNotFoundExitCode     = 3
	lessonNotFoundExitCode      = 4
)

type CodeBashingHTTPWrapper struct {
	path string
}

func NewCodeBashingHTTPWrapper(path string) *CodeBashingHTTPWrapper {
	return &CodeBashingHTTPWrapper{
		path: path,
	}
}

func (r *CodeBashingHTTPWrapper) GetCodeBashingLinks(params map[string]string, codeBashingURL string) (
	*[]CodeBashingCollection,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	params[limit] = limitValue
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, r.path, params, nil, clientTimeout)
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
		errorModel := GetWebError(decoder)
		return nil, &errorModel, nil
	case http.StatusOK:
		var decoded []CodeBashingCollection
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseCodeBashing)
		}
		err = json.Unmarshal(body, &decoded)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseCodeBashing)
		}
		/* Only check for position 0 because at the time we are only sending
		   one queryName and getting as output one codebashing link. But it is
		   possible to easily change it and be able to get multiple codebashing
		   links
		*/
		if decoded[0].Path == "" {
			return nil, nil, NewAstError(lessonNotFoundExitCode, errors.Errorf(noCodebashingLinkAvailable))
		}

		decoded[0].Path = fmt.Sprintf("%s%s", codeBashingURL, decoded[0].Path)
		decoded[0].Path, err = utils.CleanURL(decoded[0].Path)
		if err != nil {
			return nil, nil, NewAstError(lessonNotFoundExitCode, errors.Errorf(noCodebashingLinkAvailable))
		}
		return &decoded, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (r *CodeBashingHTTPWrapper) GetCodeBashingURL(field string) (string, error) {
	accessToken, err := GetAccessToken()
	if err != nil {
		return "", errors.Errorf(failedGettingCodeBashingURL)
	}
	token, _, err := new(jwt.Parser).ParseUnverified(accessToken, jwt.MapClaims{})
	if err != nil {
		return "", NewAstError(licenseNotFoundExitCode, errors.Errorf(failedGettingCodeBashingURL))
	}
	var url = ""
	if claims, ok := token.Claims.(jwt.MapClaims); ok && claims[field] != nil {
		url = claims[field].(string)
	}

	if url == "" {
		return "", NewAstError(licenseNotFoundExitCode, errors.Errorf(failedGettingCodeBashingURL))
	}

	return url, nil
}

func (*CodeBashingHTTPWrapper) BuildCodeBashingParams(apiParams []CodeBashingParamsCollection) (map[string]string, error) {
	// Marshall entire object to string
	params := make(map[string]string)
	viewJSON, err := json.Marshal(apiParams)
	if err != nil {
		return nil, err
	}
	params["results"] = string(viewJSON)
	return params, nil
}
