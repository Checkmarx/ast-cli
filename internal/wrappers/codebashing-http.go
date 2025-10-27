package wrappers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/utils"
	"github.com/golang-jwt/jwt/v5"
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

func (r *CodeBashingHTTPWrapper) GetCodeBashingLinks(queryId string, codeBashingURL string) (
	*CodeBashingCollection,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	resp, err := SendHTTPRequestNoBaseCBURL(http.MethodGet, r.path+"/"+queryId, http.NoBody, true, clientTimeout)
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
			return nil, nil, NewAstError(lessonNotFoundExitCode, errors.Wrapf(err, failedToParseCodeBashing))
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		var decoded *CodeBashingCollection
		body, err := io.ReadAll(resp.Body)
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
		if decoded.Path == "" {
			return nil, nil, NewAstError(lessonNotFoundExitCode, errors.Errorf(noCodebashingLinkAvailable))
		}

		decoded.Path = fmt.Sprintf("%s%s", codeBashingURL, decoded.Path)
		decoded.Path, err = utils.CleanURL(decoded.Path)
		if err != nil {
			return nil, nil, NewAstError(lessonNotFoundExitCode, errors.Errorf(noCodebashingLinkAvailable))
		}
		return decoded, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (r *CodeBashingHTTPWrapper) GetCodeBashingURL(field string) (string, error) {
	accessToken, err := GetAccessToken()
	if err != nil {
		return "", errors.Errorf(failedGettingCodeBashingURL)
	}

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())

	token, _, err := parser.ParseUnverified(accessToken, jwt.MapClaims{})
	if err != nil {
		return "", NewAstError(licenseNotFoundExitCode, errors.Errorf(failedGettingCodeBashingURL))
	}

	var url string
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
