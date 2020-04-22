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

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"

	"github.com/spf13/viper"
)

const (
	expiryGraceSeconds = 10
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

func getClient() *http.Client {
	insecure := viper.GetBool("insecure")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	return &http.Client{Transport: tr}
}

func SendHTTPRequest(method, url string, body io.Reader) (*http.Response, error) {
	client := getClient()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
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

func SendHTTPRequestWithQueryParams(method, url string, params map[string]string,
	body io.Reader) (*http.Response, error) {
	client := getClient()
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
	authHost := viper.GetString(commonParams.AstAuthenticationURIConfigKey)
	accessKeyID := viper.GetString(commonParams.AccessKeyIDConfigKey)
	accessKeySecret := viper.GetString(commonParams.AccessKeySecretConfigKey)

	accessToken, err := getClientCredentials(authHost, accessKeyID, accessKeySecret)
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
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
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
