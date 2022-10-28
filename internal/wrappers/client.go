package wrappers

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"

	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/ntlm"
)

const (
	expiryGraceSeconds      = 10
	NoTimeout               = 0
	ntlmProxyToken          = "ntlm"
	checkmarxURLError       = "Could not reach provided Checkmarx server"
	APIKeyDecodeErrorFormat = "Invalid api key: token decoding error: %s"
	tryPrintOffset          = 2
	retryLimitPrintOffset   = 1
	MissingURI              = "When using client-id and client-secret please provide base-uri or base-auth-uri"
	MissingTenant           = "Failed to authenticate - please provide tenant when using base-auth-uri"
	jwtError                = "Error retreiving URL from jwt token"
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

const FailedToAuth = "Failed to authenticate - please provide an %s"
const BaseAuthURLSuffix = "protocol/openid-connect/token"
const BaseAuthURLTenantSuffix = "auth/realms"
const baseURLKey = "ast-base-url"

const audienceClaimKey = "aud"

var cachedAccessToken string
var cachedAccessTime time.Time

func setAgentName(req *http.Request) {
	agentStr := viper.GetString(commonParams.AgentNameKey) + "/" + commonParams.Version
	req.Header.Set("User-Agent", agentStr)
}

func getClient(timeout uint) *http.Client {
	proxyTypeStr := viper.GetString(commonParams.ProxyTypeKey)
	proxyStr := viper.GetString(commonParams.ProxyKey)

	var client *http.Client
	if proxyTypeStr == ntlmProxyToken {
		client = ntmlProxyClient(timeout, proxyStr)
	} else {
		client = basicProxyClient(timeout, proxyStr)
	}

	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 1 {
			return fmt.Errorf("too many redirects")
		}
		if len(via) != 0 && req.Response.StatusCode == http.StatusMovedPermanently {
			for attr, val := range via[0].Header {
				if _, ok := req.Header[attr]; !ok {
					req.Header[attr] = val
				}
			}
		}

		return nil
	}

	return client
}

func basicProxyClient(timeout uint, proxyStr string) *http.Client {
	insecure := viper.GetBool("insecure")
	u, _ := url.Parse(proxyStr)
	var tr *http.Transport
	if len(proxyStr) > 0 {
		logger.PrintIfVerbose("Creating HTTP Client with Proxy: " + proxyStr)
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
			Proxy:           http.ProxyURL(u),
		}
	} else {
		logger.PrintIfVerbose("Creating HTTP Client.")
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
	logger.PrintIfVerbose("Creating HTTP client using NTLM Proxy using: " + proxyStr)
	ntlmDialContext := ntlm.NewNTLMProxyDialContext(dialer, u, proxyUser, proxyPass, domainStr, nil)
	return &http.Client{
		Transport: &http.Transport{
			Proxy:       nil,
			DialContext: ntlmDialContext,
		},
		Timeout: time.Duration(timeout) * time.Second,
	}
}

func SendHTTPRequest(method, path string, body io.Reader, auth bool, timeout uint) (*http.Response, error) {
	accessToken, err := GetAccessToken()
	if err != nil {
		return nil, err
	}
	u, err := GetURL(path, accessToken)
	if err != nil {
		return nil, err
	}
	return SendHTTPRequestByFullURL(method, u, body, auth, timeout, accessToken)
}

func SendHTTPRequestByFullURL(method, fullURL string, body io.Reader, auth bool, timeout uint, accessToken *string) (*http.Response, error) {
	req, err := http.NewRequest(method, fullURL, body)
	client := getClient(timeout)
	setAgentName(req)
	if err != nil {
		return nil, err
	}
	if auth {
		enrichWithOath2Credentials(req, accessToken)
	}

	req = addReqMonitor(req)
	var resp *http.Response
	resp, err = doRequest(client, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func addReqMonitor(req *http.Request) *http.Request {
	startTime := time.Now().UnixNano() / int64(time.Millisecond)
	if viper.GetBool(commonParams.DebugFlag) {
		trace := &httptrace.ClientTrace{
			GetConn: func(hostPort string) {
				startTime = time.Now().UnixNano() / int64(time.Millisecond)
				logger.Print("Starting connection: " + hostPort)
			},
			DNSStart: func(dnsInfo httptrace.DNSStartInfo) {
				logger.Print("DNS looking up host information for: " + dnsInfo.Host)
			},
			DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
				logger.Printf("DNS found host address(s): %+v\n", dnsInfo.Addrs)
			},
			TLSHandshakeStart: func() {
				logger.Print("Started TLS Handshake")
			},
			TLSHandshakeDone: func(c tls.ConnectionState, err error) {
				if err == nil {
					logger.Print("Completed TLS handshake")
				} else {
					logger.Printf("%s, %s", "Error completing TLS handshake", err)
				}
			},
			GotFirstResponseByte: func() {
				endTime := time.Now().UnixNano() / int64(time.Millisecond)
				diff := endTime - startTime
				logger.Printf("Connected completed in: %d (ms)", diff)
			},
		}
		return req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	}
	return req
}

func SendHTTPRequestPasswordAuth(
	method, path string, body io.Reader, timeout uint,
	username, password, adminClientID, adminClientSecret string,
) (*http.Response, error) {
	u, err := GetAuthURL(path)
	if err != nil {
		return nil, err
	}
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

	req = addReqMonitor(req)
	resp, err = doRequest(client, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func GetCleanURL(path string) string {
	cleanURL := strings.TrimSpace(viper.GetString(commonParams.BaseURIKey))
	cleanURL = strings.Trim(cleanURL, "/")
	return fmt.Sprintf("%s/%s", cleanURL, path)
}

func GetURL(path string, accessToken *string) (string, error) {
	var err error
	var cleanURL string
	if accessToken != nil {
		cleanURL, err = extractBaseURLFromToken(accessToken)
		if err != nil {
			return "", err
		}
	}

	if cleanURL == "" {
		cleanURL = strings.TrimSpace(viper.GetString(commonParams.BaseURIKey))
	}

	if cleanURL == "" {
		return "", errors.Errorf(MissingURI)
	}

	cleanURL = strings.Trim(cleanURL, "/")
	return fmt.Sprintf("%s/%s", cleanURL, path), nil
}

func GetAuthURL(path string) (string, error) {
	var authURL string
	var err error
	cleanURL := strings.TrimSpace(viper.GetString(commonParams.BaseAuthURIKey))
	// case we use base-auth-uri flag
	if cleanURL != "" {
		authURL = fmt.Sprintf("%s/%s", strings.Trim(cleanURL, "/"), path)
		// case we don't use base-auth-uri flag, we try to get the base-uri instead
	} else {
		authURL, err = GetURL(path, nil)
		if err != nil {
			return "", err
		}
	}
	logger.PrintIfVerbose("Auth URL is: " + authURL)
	return authURL, nil
}

func SendPrivateHTTPRequestWithQueryParams(
	method, path string, params map[string]string,
	body io.Reader, timeout uint,
) (*http.Response, error) {
	return HTTPRequestWithQueryParams(method, path, params, body, timeout, false)
}

func SendHTTPRequestWithQueryParams(
	method, path string, params map[string]string,
	body io.Reader, timeout uint,
) (*http.Response, error) {
	return HTTPRequestWithQueryParams(method, path, params, body, timeout, true)
}

func HTTPRequestWithQueryParams(
	method, path string, params map[string]string,
	body io.Reader, timeout uint, printBody bool,
) (*http.Response, error) {
	accessToken, err := GetAccessToken()
	if err != nil {
		return nil, err
	}
	u, err := GetURL(path, accessToken)
	if err != nil {
		return nil, err
	}
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
	enrichWithOath2Credentials(req, accessToken)
	var resp *http.Response
	resp, err = request(client, req, printBody)
	if err != nil {
		return resp, errors.Errorf("%s %s \n", checkmarxURLError, req.URL.RequestURI())
	}
	if resp.StatusCode == http.StatusForbidden {
		return resp, errors.Errorf("%s", "Provided credentials do not have permissions for this command")
	}
	return resp, nil
}

func getClaimsFromToken(tokenString string) (*jwt.Token, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, err
	}
	return token, err
}

func getAuthURI() (string, error) {
	var authURI string
	apiKey := viper.GetString(commonParams.AstAPIKey)
	var err error
	if len(apiKey) > 0 {
		logger.PrintIfVerbose("Using API Key to extract Auth URI")
		authURI, err = extractAuthURIFromAPIKey(apiKey)
	} else {
		logger.PrintIfVerbose("Using configuration and parameters to prepare Auth URI")
		authURI, err = extractAuthURIFromConfig()
	}
	if err != nil {
		return "", err
	}

	authURL, err := url.Parse(authURI)
	if err != nil {
		return "", errors.Wrap(err, "authentication URI is not in a correct format")
	}

	if authURL.Scheme == "" && authURL.Host == "" {
		authURI, err = GetURL("/"+strings.TrimLeft(authURI, "/"), nil)
		if err != nil {
			return "", err
		}
	}

	return authURI, nil
}

func extractBaseURLFromToken(accessToken *string) (string, error) {
	var baseURL string
	token, _, err := new(jwt.Parser).ParseUnverified(*accessToken, jwt.MapClaims{})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && claims[baseURLKey] != nil {
		baseURL = strings.TrimSpace(claims[baseURLKey].(string))
	} else {
		return "", errors.Errorf(jwtError)
	}
	return baseURL, nil
}

func extractAuthURIFromConfig() (string, error) {
	authPath := viper.GetString(commonParams.AstAuthenticationPathConfigKey)
	tenant := viper.GetString(commonParams.TenantKey)
	authPath = strings.Replace(authPath, "organization", strings.ToLower(tenant), 1)
	if authPath == "" {
		return "", errors.Errorf(fmt.Sprintf(FailedToAuth, "authentication path"))
	}
	authURI, err := GetAuthURL(authPath)
	if err != nil {
		return "", err
	}
	return authURI, nil
}

func extractAuthURIFromAPIKey(key string) (string, error) {
	token, err := getClaimsFromToken(key)
	if err != nil {
		return "", errors.Errorf(fmt.Sprintf(APIKeyDecodeErrorFormat, err.Error()))
	}

	claims := token.Claims.(jwt.MapClaims)
	authURI := claims[audienceClaimKey].(string)

	if authURI == "" {
		authURI = strings.TrimSpace(viper.GetString(commonParams.BaseAuthURIKey))
		tenant := viper.GetString(commonParams.TenantKey)
		if tenant == "" {
			return "", errors.Errorf(MissingTenant)
		}
		authURI = fmt.Sprintf("%s/%s/%s/%s", authURI, BaseAuthURLTenantSuffix, tenant, BaseAuthURLSuffix)
	} else {
		authURI = fmt.Sprintf("%s/%s", authURI, BaseAuthURLSuffix)
	}

	return authURI, nil
}

func enrichWithOath2Credentials(request *http.Request, accessToken *string) {
	request.Header.Add("Authorization", "Bearer "+*accessToken)
}

func SendHTTPRequestWithJSONContentType(method, path string, body io.Reader, auth bool, timeout uint) (
	*http.Response,
	error,
) {
	accessToken, err := GetAccessToken()
	if err != nil {
		return nil, err
	}
	fullURL, err := GetURL(path, accessToken)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, fullURL, body)
	client := getClient(timeout)
	setAgentName(req)
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}
	if auth {
		enrichWithOath2Credentials(req, accessToken)
	}

	req = addReqMonitor(req)
	var resp *http.Response
	resp, err = doRequest(client, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func GetAccessToken() (*string, error) {
	authURI, err := getAuthURI()
	if err != nil {
		return nil, err
	}
	tokenExpirySeconds := viper.GetInt(commonParams.TokenExpirySecondsKey)
	accessToken := getClientCredentialsFromCache(tokenExpirySeconds)
	accessKeyID := viper.GetString(commonParams.AccessKeyIDConfigKey)
	accessKeySecret := viper.GetString(commonParams.AccessKeySecretConfigKey)
	astAPIKey := viper.GetString(commonParams.AstAPIKey)
	if accessKeyID == "" && astAPIKey == "" {
		return nil, errors.Errorf(fmt.Sprintf(FailedToAuth, "access key ID"))
	} else if accessKeySecret == "" && astAPIKey == "" {
		return nil, errors.Errorf(fmt.Sprintf(FailedToAuth, "access key secret"))
	}
	if accessToken == nil {
		accessToken, err = getClientCredentials(accessKeyID, accessKeySecret, astAPIKey, authURI)
		if err != nil {
			return nil, err
		}
	}
	return accessToken, nil
}

func enrichWithPasswordCredentials(
	request *http.Request, username, password,
	adminClientID, adminClientSecret string,
) error {
	authURI, err := getAuthURI()
	if err != nil {
		return err
	}

	accessToken, err := getNewToken(
		getPasswordCredentialsPayload(username, password, adminClientID, adminClientSecret),
		authURI,
	)
	if err != nil {
		return errors.Wrap(
			errors.Wrap(err, "failed to get access token from auth server"),
			"failed to authenticate",
		)
	}

	request.Header.Add("Authorization", "Bearer "+*accessToken)
	return nil
}

func getClientCredentials(accessKeyID, accessKeySecret, astAPKey, authURI string) (*string, error) {
	logger.PrintIfVerbose("Fetching API access token.")
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
	logger.PrintIfVerbose("Checking cache for API access token.")
	expired := time.Since(cachedAccessTime) > time.Duration(tokenExpirySeconds-expiryGraceSeconds)*time.Second
	if !expired {
		logger.PrintIfVerbose("Using cached API access token!")
		return &cachedAccessToken
	}
	logger.PrintIfVerbose("API access token not found in cache!")
	return nil
}

func writeCredentialsToCache(accessToken *string) {
	logger.PrintIfVerbose("Storing API access token to cache.")
	viper.Set(commonParams.AstToken, *accessToken)
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
	req = addReqMonitor(req)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	client := getClient(clientTimeout)

	res, err := doPrivateRequest(client, req)
	if err != nil {
		authURL, _ := GetAuthURL("")
		return nil, errors.Errorf("%s %s", checkmarxURLError, authURL)
	}
	if res.StatusCode == http.StatusBadRequest {
		return nil, errors.Errorf("%v %s \n", res.StatusCode, "Provided credentials are invalid")
	}
	if res.StatusCode == http.StatusNotFound {
		return nil, errors.Errorf("%v %s \n", res.StatusCode, "Provided Tenant Name is invalid")
	}
	if res.StatusCode == http.StatusUnauthorized {
		return nil, errors.Errorf("%v %s \n", res.StatusCode, "Provided credentials are invalid")
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

	defer func() {
		_ = res.Body.Close()
	}()

	credentialsInfo := ClientCredentialsInfo{}
	err = json.Unmarshal(body, &credentialsInfo)
	if err != nil {
		return nil, err
	}

	logger.PrintIfVerbose("Successfully retrieved API token.")
	return &credentialsInfo.AccessToken, nil
}

func getCredentialsPayload(accessKeyID, accessKeySecret string) string {
	logger.PrintIfVerbose("Using Client ID and secret credentials.")
	return fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s", accessKeyID, accessKeySecret)
}

func getAPIKeyPayload(astToken string) string {
	logger.PrintIfVerbose("Using API key credentials.")
	return fmt.Sprintf("grant_type=refresh_token&client_id=ast-app&refresh_token=%s", astToken)
}

func getPasswordCredentialsPayload(username, password, adminClientID, adminClientSecret string) string {
	logger.PrintIfVerbose("Using username and password credentials.")
	return fmt.Sprintf(
		"scope=openid&grant_type=password&username=%s&password=%s"+
			"&client_id=%s&client_secret=%s", username, password, adminClientID, adminClientSecret,
	)
}

func doPrivateRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	return request(client, req, false)
}

func doRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	return request(client, req, true)
}

func request(client *http.Client, req *http.Request, responseBody bool) (*http.Response, error) {
	var err error
	var resp *http.Response
	retryLimit := int(viper.GetUint(commonParams.RetryFlag))
	retryWaitTimeSeconds := viper.GetUint(commonParams.RetryDelayFlag)
	// try starts at -1 as we always do at least one request, retryLimit can be 0
	logger.PrintRequest(req)
	for try := -1; try < retryLimit; try++ {
		logger.PrintIfVerbose(
			fmt.Sprintf(
				"Request attempt %d in %d",
				try+tryPrintOffset, retryLimit+retryLimitPrintOffset,
			),
		)
		resp, err = client.Do(req)
		if resp != nil && err == nil {
			logger.PrintResponse(resp, responseBody)
			return resp, nil
		}
		logger.PrintIfVerbose(fmt.Sprintf("Request failed in attempt %d", try+tryPrintOffset))
		time.Sleep(time.Duration(retryWaitTimeSeconds) * time.Second)
	}
	return nil, err
}
