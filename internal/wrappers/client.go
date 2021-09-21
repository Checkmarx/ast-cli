package wrappers

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/checkmarxDev/ast-cli/internal/wrappers/ntlm"

	"github.com/spf13/viper"
)

const (
	expiryGraceSeconds = 10
	NoTimeout          = 0
	ntlmProxyToken     = "ntlm"
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

const failedToAuth = "Failed to authenticate - please provide an %s"

var cachedAccessToken string
var cachedAccessTime time.Time

func setAgentName(req *http.Request) {
	agentStr := viper.GetString(commonParams.AgentNameKey) + "/" + commonParams.Version
	req.Header.Set("User-Agent", agentStr)
}

func getClient(timeout uint) *http.Client {
	proxyTypeStr := viper.GetString(commonParams.ProxyTypeKey)
	proxyStr := viper.GetString(commonParams.ProxyKey)
	if proxyTypeStr == ntlmProxyToken {
		return ntmlProxyClient(timeout, proxyStr)
	}
	return basicProxyClient(timeout, proxyStr)
}

func basicProxyClient(timeout uint, proxyStr string) *http.Client {
	insecure := viper.GetBool("insecure")
	u, _ := url.Parse(proxyStr)
	var tr *http.Transport
	if len(proxyStr) > 0 {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
			Proxy:           http.ProxyURL(u),
		}
	} else {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
		}
	}
	return &http.Client{Transport: tr, Timeout: time.Duration(timeout) * time.Second}
}

func ntmlProxyClient(timeout uint, proxyStr string) *http.Client {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	u, _ := url.Parse(proxyStr)
	domainStr := viper.GetString(commonParams.ProxyDomainKey)
	proxyUser := u.User.Username()
	proxyPass, _ := u.User.Password()
	ntlmDialContext := ntlm.NewNTLMProxyDialContext(dialer, u, proxyUser, proxyPass, domainStr, nil)
	return &http.Client{
		Transport: &http.Transport{
			Proxy:       nil,
			DialContext: ntlmDialContext,
		},
		Timeout: time.Duration(timeout) * time.Second}
}

func SendHTTPRequest(method, path string, body io.Reader, auth bool, timeout uint) (*http.Response, error) {
	u := GetURL(path)
	return SendHTTPRequestByFullURL(method, u, body, auth, timeout)
}

func SendHTTPRequestByFullURL(method, fullURL string, body io.Reader, auth bool, timeout uint) (*http.Response, error) {
	req, err := http.NewRequest(method, fullURL, body)
	client := getClient(timeout)
	setAgentName(req)
	if err != nil {
		return nil, err
	}
	if auth {
		err = enrichWithOath2Credentials(req)
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

func SendHTTPRequestPasswordAuth(method, path string, body io.Reader, timeout uint,
	username, password, adminClientID, adminClientSecret string) (*http.Response, error) {
	u := GetAuthURL(path)
	req, err := http.NewRequest(method, u, body)
	client := getClient(timeout)
	setAgentName(req)
	if err != nil {
		return nil, err
	}
	req.Header.Add("content-type", "application/json")
	err = enrichWithPasswordCredentials(req, username, password, adminClientID, adminClientSecret)
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

func GetURL(path string) string {
	cleanURL := strings.Trim(viper.GetString(commonParams.BaseURIKey), "/")
	return fmt.Sprintf("%s/%s", cleanURL, path)
}

func GetAuthURL(path string) string {
	if viper.GetString(commonParams.BaseAuthURIKey) != "" {
		cleanURL := strings.Trim(viper.GetString(commonParams.BaseAuthURIKey), "/")
		return fmt.Sprintf("%s/%s", cleanURL, path)
	}
	return GetURL(path)
}

func SendHTTPRequestWithQueryParams(method, path string, params map[string]string,
	body io.Reader, timeout uint) (*http.Response, error) {
	u := GetURL(path)
	req, err := http.NewRequest(method, u, body)
	client := getClient(timeout)
	setAgentName(req)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	err = enrichWithOath2Credentials(req)
	if err != nil {
		return nil, err
	}
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func getAuthURI() (string, error) {
	authPath := viper.GetString(commonParams.AstAuthenticationPathConfigKey)
	tenant := viper.GetString(commonParams.TenantKey)
	authPath = strings.Replace(authPath, "organization", strings.ToLower(tenant), 1)
	if authPath == "" {
		return "", errors.Errorf(fmt.Sprintf(failedToAuth, "authentication path"))
	}
	authURI := GetAuthURL(authPath)
	authURL, err := url.Parse(authURI)
	if err != nil {
		return "", errors.Wrap(err, "authentication URI is not in a correct format")
	}

	if authURL.Scheme == "" && authURL.Host == "" {
		baseURI := viper.GetString(commonParams.BaseURIKey)
		authURI = baseURI + "/" + strings.TrimLeft(authURI, "/")
	}

	return authURI, nil
}

func enrichWithOath2Credentials(request *http.Request) error {
	authURI, err := getAuthURI()
	if err != nil {
		return err
	}

	accessKeyID := viper.GetString(commonParams.AccessKeyIDConfigKey)
	accessKeySecret := viper.GetString(commonParams.AccessKeySecretConfigKey)
	astAPIKey := viper.GetString(commonParams.AstAPIKey)

	if accessKeyID == "" && astAPIKey == "" {
		return errors.Errorf(fmt.Sprintf(failedToAuth, "access key ID"))
	} else if accessKeySecret == "" && astAPIKey == "" {
		return errors.Errorf(fmt.Sprintf(failedToAuth, "access key secret"))
	} else if astAPIKey == "" && accessKeyID == "" && accessKeySecret == "" {
		return errors.Errorf(fmt.Sprintf(failedToAuth, "access API Key"))
	}

	accessToken, err := getClientCredentials(accessKeyID, accessKeySecret, astAPIKey, authURI)
	if err != nil {
		return errors.Wrap(err, "failed to authenticate")
	}

	request.Header.Add("Authorization", *accessToken)
	return nil
}

func enrichWithPasswordCredentials(request *http.Request, username, password,
	adminClientID, adminClientSecret string) error {
	authURI, err := getAuthURI()
	if err != nil {
		return err
	}

	accessToken, err := getNewToken(getPasswordCredentialsPayload(username, password, adminClientID, adminClientSecret), authURI)
	if err != nil {
		return errors.Wrap(errors.Wrap(err, "failed to get access token from auth server"),
			"failed to authenticate")
	}

	request.Header.Add("Authorization", "Bearer "+*accessToken)
	return nil
}

func getClientCredentials(accessKeyID, accessKeySecret, astAPKey, authURI string) (*string, error) {
	tokenExpirySeconds := viper.GetInt(commonParams.TokenExpirySecondsKey)
	var accessToken *string
	var err error
	accessToken = getClientCredentialsFromCache(tokenExpirySeconds)

	if accessToken == nil {
		// If the token is present the default to that.
		if astAPKey != "" {
			accessToken, err = getNewToken(getAPIKeyPayload(astAPKey), authURI)
		} else {
			accessToken, err = getNewToken(getCredentialsPayload(accessKeyID, accessKeySecret), authURI)
		}

		if err != nil {
			return nil, errors.Errorf("%s", err)
		}

		writeCredentialsToCache(accessToken)
	}

	return accessToken, nil
}

func getClientCredentialsFromCache(tokenExpirySeconds int) *string {
	expired := time.Since(cachedAccessTime) > time.Duration(tokenExpirySeconds-expiryGraceSeconds)*time.Second
	if !expired {
		return &cachedAccessToken
	}
	return nil
}

func writeCredentialsToCache(accessToken *string) {
	cachedAccessToken = *accessToken
	cachedAccessTime = time.Now()
}

func getNewToken(credentialsPayload, authServerURI string) (*string, error) {
	payload := strings.NewReader(credentialsPayload)
	req, err := http.NewRequest(http.MethodPost, authServerURI, payload)
	setAgentName(req)
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
		return nil, errors.Errorf("%v %s %s", res.StatusCode, credentialsErr.Error, credentialsErr.Description)
	}

	defer res.Body.Close()

	credentialsInfo := ClientCredentialsInfo{}
	err = json.Unmarshal(body, &credentialsInfo)
	if err != nil {
		return nil, err
	}

	return &credentialsInfo.AccessToken, nil
}

func getCredentialsPayload(accessKeyID, accessKeySecret string) string {
	return fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s", accessKeyID, accessKeySecret)
}

func getAPIKeyPayload(astToken string) string {
	return fmt.Sprintf("grant_type=refresh_token&client_id=ast-app&refresh_token=%s", astToken)
}

func getPasswordCredentialsPayload(username, password, adminClientID, adminClientSecret string) string {
	return fmt.Sprintf("scope=openid&grant_type=password&username=%s&password=%s"+
		"&client_id=%s&client_secret=%s", username, password, adminClientID, adminClientSecret)
}
