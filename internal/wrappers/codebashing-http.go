package wrappers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	failedToParseCodeBashing    = "Failed to parse list results"
	failedGettingCodeBashingURL = "Authentication failed, not able to retrieve codebashing base link"
	tenThousand                 = "10000"
	limit                       = "limit"
	codeBashingKey              = "cb-url"
)

type CodeBashingHTTPWrapper struct {
	path string
}

func NewCodeBashingHTTPWrapper(path string) CodeBashingWrapper {
	return &CodeBashingHTTPWrapper{
		path: path,
	}
}

func (r *CodeBashingHTTPWrapper) GetCodeBashingLinks(params map[string]string) (
	*[]CodeBashingCollection,
	*WebError,
	error,
) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	params[limit] = tenThousand
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, r.path, params, nil, clientTimeout)
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
			return nil, nil, errors.Wrapf(err, failedToParseCodeBashing)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		url, err := getCodeBashingURL(codeBashingKey)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedGettingCodeBashingURL)
		}
		var decoded []CodeBashingCollection
		body, err := ioutil.ReadAll(resp.Body)
		err = json.Unmarshal(body, &decoded)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseCodeBashing)
		}
		decoded[0].Path = *url + decoded[0].Path
		return &decoded, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func getCodeBashingURL(field string) (*string, error) {
	authURI, err := getAuthURI()
	if err != nil {
		return nil, err
	}
	accessKeyID := viper.GetString(commonParams.AccessKeyIDConfigKey)
	accessKeySecret := viper.GetString(commonParams.AccessKeySecretConfigKey)
	astAPIKey := viper.GetString(commonParams.AstAPIKey)
	if accessKeyID == "" && astAPIKey == "" {
		return nil, errors.Errorf(fmt.Sprintf(FailedToAuth, "access key ID"))
	} else if accessKeySecret == "" && astAPIKey == "" {
		return nil, errors.Errorf(fmt.Sprintf(FailedToAuth, "access key secret"))
	}
	accessToken, err := getClientCredentials(accessKeyID, accessKeySecret, astAPIKey, authURI)
	if err != nil {
		return nil, err
	}
	token, _, err := new(jwt.Parser).ParseUnverified(*accessToken, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}
	var url = ""
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		url = claims[field].(string)
	} else {
		return nil, err
	}
	return &url, nil
}
