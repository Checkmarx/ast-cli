package wrappers

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"

	"github.com/spf13/viper"
)

const (
	expiryGraceSeconds    = 10
	DefaultTimeoutSeconds = 5
)

type ClientCredentialsInfo struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	SessionState     string `json:"session_state"`
	Scope            string `json:"scope"`
}

type ClientCredentialsError struct {
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

func getClient(timeout uint) *http.Client {
	insecure := viper.GetBool("insecure")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	return &http.Client{Transport: tr, Timeout: time.Duration(timeout) * time.Second}
}

func SendHTTPRequest(method, path string, body io.Reader, auth bool, timeout uint) (*http.Response, error) {
	client := getClient(timeout)
	url := GetURL(path)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if auth {
		req, err = enrichWithCredentials(req)
		if err != nil {
			return nil, err
		}
	}

	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func GetURL(path string) string {
	return fmt.Sprintf("%s/%s", viper.GetString(commonParams.BaseURIKey), path)
}

func SendHTTPRequestWithQueryParams(method, path string, params map[string]string,
	body io.Reader) (*http.Response, error) {
	client := getClient(DefaultTimeoutSeconds)
	url := GetURL(path)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	req, err = enrichWithCredentials(req)
	if err != nil {
		return nil, err
	}
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func enrichWithCredentials(request *http.Request) (*http.Request, error) {
	authURI := viper.GetString(commonParams.AstAuthenticationURIConfigKey)
	accessKeyID := viper.GetString(commonParams.AccessKeyIDConfigKey)
	accessKeySecret := viper.GetString(commonParams.AccessKeySecretConfigKey)
	failedToAuth := "Failed to authenticate - please provide an %s"

	if authURI == "" {
		return nil, errors.Errorf(fmt.Sprintf(failedToAuth, "authentication URI"))
	} else if accessKeyID == "" {
		return nil, errors.Errorf(fmt.Sprintf(failedToAuth, "access key ID"))
	} else if accessKeySecret == "" {
		return nil, errors.Errorf(fmt.Sprintf(failedToAuth, "access key secret"))
	}

	baseURI := viper.GetString(commonParams.BaseURIKey)
	authURI = strings.ReplaceAll(authURI, fmt.Sprintf("${%s}", commonParams.BaseURIKey), baseURI)

	accessToken, err := getClientCredentials(authURI, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Authorization", *accessToken)
	return request, nil
}

func getClientCredentials(authServerURI, accessKeyID, accessKeySecret string) (*string, error) {
	// Try to load access token from file, if not expired
	credentialsFilePath := viper.GetString(commonParams.CredentialsFilePathKey)
	tokenExpirySeconds := viper.GetInt(commonParams.TokenExpirySecondsKey)

	if info, err := os.Stat(credentialsFilePath); err == nil {
		// Credentials file exists. Check for access token validity
		modifiedAt := info.ModTime()
		expired := time.Since(modifiedAt) > time.Duration(tokenExpirySeconds-expiryGraceSeconds)*time.Second
		if !expired {
			b, err := ioutil.ReadFile(credentialsFilePath)
			if err != nil {
				return nil, err
			}
			accessToken := string(b)
			if accessToken == "" {
				return getNewToken(accessKeyID, accessKeySecret, authServerURI, credentialsFilePath)
			}
			return &accessToken, nil
		}
	}

	// Here the file can either not exist, exist and failed opening or the token has expired.
	// We don't care. Create a new token.
	return getNewToken(accessKeyID, accessKeySecret, authServerURI, credentialsFilePath)
}

func getNewToken(accessKeyID, accessKeySecret, authServerURI, credentialsFilePath string) (*string, error) {
	payload := strings.NewReader(getCredentialsPayload(accessKeyID, accessKeySecret))
	req, err := http.NewRequest(http.MethodPost, authServerURI, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	body, _ := ioutil.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		credentialsErr := ClientCredentialsError{}
		err = json.Unmarshal(body, &credentialsErr)
		if err != nil {
			return nil, err
		}
		return nil, errors.Errorf("%s: %s", credentialsErr.Error, credentialsErr.Description)
	}

	defer res.Body.Close()
	credentialsInfo := ClientCredentialsInfo{}
	err = json.Unmarshal(body, &credentialsInfo)
	if err != nil {
		return nil, err
	}

	// We have a new access token. Save it to file
	accessToken := credentialsInfo.AccessToken
	accessTokenData := []byte(accessToken)
	_ = ioutil.WriteFile(credentialsFilePath, accessTokenData, 0644)

	return &accessToken, nil
}

func getCredentialsPayload(accessKeyID, accessKeySecret string) string {
	return fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s", accessKeyID, accessKeySecret)
}
