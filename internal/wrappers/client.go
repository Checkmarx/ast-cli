package wrappers

import (
	"bytes"
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
	"sync"
	"time"

	applicationErrors "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/golang-jwt/jwt/v5"

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
	invalidCredentialsError = "Provided credentials are invalid"
	APIKeyDecodeErrorFormat = "Token decoding error: %s"
	tryPrintOffset          = 2
	retryLimitPrintOffset   = 1
	MissingURI              = "When using client-id and client-secret please provide base-uri or base-auth-uri"
	MissingTenant           = "Failed to authenticate - please provide tenant"
	jwtError                = "Error retrieving %s from jwt token"
	basicFormat             = "Basic %s"
	bearerFormat            = "Bearer %s"
	contentTypeHeader       = "Content-Type"
	formURLContentType      = "application/x-www-form-urlencoded"
	jsonContentType         = "application/json"
)

var (
	credentialsMutex sync.Mutex
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
const BaseAuthURLPrefix = "auth/realms/organization"
const baseURLKey = "ast-base-url"

const audienceClaimKey = "aud"

var cachedAccessToken string
var cachedAccessTime time.Time
var Domains = make(map[string]struct{})

func retryHTTPRequest(requestFunc func() (*http.Response, error), retries int, baseDelayInMilliSec time.Duration) (*http.Response, error) {

	var resp *http.Response
	var err error

	for attempt := 0; attempt < retries; attempt++ {
		resp, err = requestFunc()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusBadGateway {
			logger.PrintIfVerbose("Bad Gateway, retrying")
		} else if resp.StatusCode == http.StatusUnauthorized {
			logger.PrintIfVerbose("Unauthorized request, refreshing token")
			_, _ = configureClientCredentialsAndGetNewToken()
		} else {
			return resp, nil
		}
		_ = resp.Body.Close()
		time.Sleep(baseDelayInMilliSec * (1 << attempt))
	}
	return resp, nil
}

func setAgentName(req *http.Request) {
	agentStr := viper.GetString(commonParams.AgentNameKey) + "/" + commonParams.Version
	req.Header.Set("User-Agent", agentStr)
}

func GetClient(timeout uint) *http.Client {
	proxyTypeStr := viper.GetString(commonParams.ProxyTypeKey)
	proxyStr := viper.GetString(commonParams.ProxyKey)
	ignoreProxy := viper.GetBool(commonParams.IgnoreProxyKey)

	var client *http.Client
	if ignoreProxy {
		client = basicProxyClient(timeout, "")
	} else if proxyTypeStr == ntlmProxyToken {
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

func getURLAndAccessToken(path string) (urlFromPath, accessToken string, err error) {
	accessToken, err = GetAccessToken()
	if err != nil {
		return "", "", err
	}
	urlFromPath, err = GetURL(path, accessToken)
	if err != nil {
		return "", "", err
	}
	return
}

func SendHTTPRequest(method, path string, body io.Reader, auth bool, timeout uint) (*http.Response, error) {
	u, accessToken, err := getURLAndAccessToken(path)
	if err != nil {
		return nil, err
	}
	return SendHTTPRequestByFullURL(method, u, body, auth, timeout, accessToken, true)
}

func SendPrivateHTTPRequest(method, path string, body io.Reader, timeout uint, auth bool) (*http.Response, error) {
	u, accessToken, err := getURLAndAccessToken(path)
	if err != nil {
		return nil, err
	}
	return SendHTTPRequestByFullURL(method, u, body, auth, timeout, accessToken, false)
}

func SendHTTPRequestByFullURL(
	method, fullURL string,
	body io.Reader,
	auth bool,
	timeout uint,
	accessToken string,
	bodyPrint bool,
) (*http.Response, error) {
	return SendHTTPRequestByFullURLContentLength(method, fullURL, body, -1, auth, timeout, accessToken, bodyPrint)
}

func SendHTTPRequestByFullURLContentLength(
	method, fullURL string,
	body io.Reader,
	contentLength int64,
	auth bool,
	timeout uint,
	accessToken string,
	bodyPrint bool,
) (*http.Response, error) {
	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, err
	}
	if contentLength >= 0 {
		req.ContentLength = contentLength
	}
	client := GetClient(timeout)
	setAgentName(req)
	if auth {
		enrichWithOath2Credentials(req, accessToken, bearerFormat)
	}

	req = addReqMonitor(req)

	return request(client, req, bodyPrint)
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

func SendHTTPRequestPasswordAuth(method string, body io.Reader, timeout uint, username, password, adminClientID, adminClientSecret string) (*http.Response, error) {
	u, err := GetAuthURI()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, u, body)
	client := GetClient(timeout)
	setAgentName(req)
	if err != nil {
		return nil, err
	}
	req.Header.Add(contentTypeHeader, jsonContentType)
	err = enrichWithPasswordCredentials(req, username, password, adminClientID, adminClientSecret)
	if err != nil {
		return nil, err
	}
	req = addReqMonitor(req)
	return doRequest(client, req)
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
	u, accessToken, err := getURLAndAccessToken(path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, u, body)
	client := GetClient(timeout)
	setAgentName(req)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	enrichWithOath2Credentials(req, accessToken, bearerFormat)
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

func addTenantAuthURI(baseAuthURI string) (string, error) {
	authPath := BaseAuthURLPrefix
	tenant := viper.GetString(commonParams.TenantKey)

	if tenant == "" {
		return "", errors.Errorf(MissingTenant)
	}

	authPath = strings.Replace(authPath, "organization", strings.ToLower(tenant), 1)

	return fmt.Sprintf("%s/%s", strings.Trim(baseAuthURI, "/"), authPath), nil
}

func enrichWithOath2Credentials(request *http.Request, accessToken, authFormat string) {
	request.Header.Add(AuthorizationHeader, fmt.Sprintf(authFormat, accessToken))
}

func SendHTTPRequestWithJSONContentType(method, path string, body io.Reader, auth bool, timeout uint) (
	*http.Response,
	error,
) {
	fullURL, accessToken, err := getURLAndAccessToken(path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, fullURL, body)
	client := GetClient(timeout)
	setAgentName(req)
	req.Header.Add("Content-Type", jsonContentType)
	if err != nil {
		return nil, err
	}
	if auth {
		enrichWithOath2Credentials(req, accessToken, bearerFormat)
	}

	req = addReqMonitor(req)
	return doRequest(client, req)
}

func GetWithQueryParams(client *http.Client, urlAddress, token, authFormat string, queryParams map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, urlAddress, http.NoBody)
	if err != nil {
		return nil, err
	}
	logger.PrintRequest(req)
	return GetWithQueryParamsAndCustomRequest(client, req, urlAddress, token, authFormat, queryParams)
}

// GetWithQueryParamsAndCustomRequest used when we need to add custom headers to the request
func GetWithQueryParamsAndCustomRequest(client *http.Client, customReq *http.Request, urlAddress, token, authFormat string, queryParams map[string]string) (*http.Response, error) {
	if len(token) > 0 {
		enrichWithOath2Credentials(customReq, token, authFormat)
	}
	q := customReq.URL.Query()
	for k, v := range queryParams {
		q.Add(k, v)
	}
	customReq.URL.RawQuery = q.Encode()
	customReq = addReqMonitor(customReq)
	return request(client, customReq, true)
}

func GetAccessToken() (string, error) {
	var err error
	tokenExpirySeconds := viper.GetInt(commonParams.TokenExpirySecondsKey)

	accessToken := getClientCredentialsFromCache(tokenExpirySeconds)

	if accessToken == "" {
		logger.PrintIfVerbose("Fetching API access token.")
		accessToken, err = configureClientCredentialsAndGetNewToken()
		if err != nil {
			return "", err
		}
	}

	return accessToken, nil
}

func enrichWithPasswordCredentials(
	request *http.Request, username, password,
	adminClientID, adminClientSecret string,
) error {
	authURI, err := GetAuthURI()
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
	enrichWithOath2Credentials(request, accessToken, bearerFormat)
	return nil
}

func configureClientCredentialsAndGetNewToken() (string, error) {
	accessKeyID := viper.GetString(commonParams.AccessKeyIDConfigKey)
	accessKeySecret := viper.GetString(commonParams.AccessKeySecretConfigKey)
	astAPIKey := viper.GetString(commonParams.AstAPIKey)
	var accessToken string

	if accessKeyID == "" && astAPIKey == "" {
		return "", errors.Errorf(fmt.Sprintf(FailedToAuth, "access key ID"))
	} else if accessKeySecret == "" && astAPIKey == "" {
		return "", errors.Errorf(fmt.Sprintf(FailedToAuth, "access key secret"))
	}

	authURI, err := GetAuthURI()
	if err != nil {
		return "", err
	}

	if astAPIKey != "" {
		accessToken, err = getNewToken(getAPIKeyPayload(astAPIKey), authURI)
	} else {
		accessToken, err = getNewToken(getCredentialsPayload(accessKeyID, accessKeySecret), authURI)
	}

	if err != nil {
		return "", errors.Errorf("%s", err)
	}

	writeCredentialsToCache(accessToken)

	return accessToken, nil
}

func getClientCredentialsFromCache(tokenExpirySeconds int) string {
	logger.PrintIfVerbose("Checking cache for API access token.")

	expired := time.Since(cachedAccessTime) > time.Duration(tokenExpirySeconds-expiryGraceSeconds)*time.Second
	if !expired {
		logger.PrintIfVerbose("Using cached API access token!")
		return cachedAccessToken
	}
	logger.PrintIfVerbose("API access token not found in cache!")
	return ""
}

func writeCredentialsToCache(accessToken string) {
	credentialsMutex.Lock()
	defer credentialsMutex.Unlock()

	logger.PrintIfVerbose("Storing API access token to cache.")
	viper.Set(commonParams.AstToken, accessToken)
	cachedAccessToken = accessToken
	cachedAccessTime = time.Now()
}

func getNewToken(credentialsPayload, authServerURI string) (string, error) {
	payload := strings.NewReader(credentialsPayload)
	req, err := http.NewRequest(http.MethodPost, authServerURI, payload)
	setAgentName(req)
	if err != nil {
		return "", err
	}
	req = addReqMonitor(req)
	req.Header.Add(contentTypeHeader, formURLContentType)
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	client := GetClient(clientTimeout)

	res, err := doPrivateRequest(client, req)
	if err != nil {
		authURL, _ := GetAuthURI()
		return "", errors.Errorf("%s %s", checkmarxURLError, authURL)
	}
	if res.StatusCode == http.StatusBadRequest {
		return "", errors.Errorf("%d %s \n", res.StatusCode, invalidCredentialsError)
	}
	if res.StatusCode == http.StatusNotFound {
		return "", errors.Errorf("%d %s \n", res.StatusCode, "Provided Tenant Name is invalid")
	}
	if res.StatusCode == http.StatusUnauthorized {
		return "", errors.Errorf("%d %s \n", res.StatusCode, invalidCredentialsError)
	}

	body, _ := ioutil.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		credentialsErr := ClientCredentialsError{}
		err = json.Unmarshal(body, &credentialsErr)

		if err != nil {
			return "", err
		}

		return "", errors.Errorf("%d %s %s", res.StatusCode, credentialsErr.Error, credentialsErr.Description)
	}

	defer func() {
		_ = res.Body.Close()
	}()

	credentialsInfo := ClientCredentialsInfo{}
	err = json.Unmarshal(body, &credentialsInfo)
	if err != nil {
		return "", err
	}

	logger.PrintIfVerbose("Successfully retrieved API token.")
	return credentialsInfo.AccessToken, nil
}

func getCredentialsPayload(accessKeyID, accessKeySecret string) string {
	logger.PrintIfVerbose("Using Client ID and secret credentials.")
	// escape possible characters such as +,%, etc...
	clientID := url.QueryEscape(accessKeyID)
	clientSecret := url.QueryEscape(accessKeySecret)
	return fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s", clientID, clientSecret)
}

func getAPIKeyPayload(astToken string) string {
	logger.PrintIfVerbose("Using API key credentials.")

	clientID, err := extractAZPFromToken(astToken)
	if err != nil {
		logger.PrintIfVerbose("Failed to extract azp from token, using default client_id")
		clientID = "ast-app"
	}

	return fmt.Sprintf("grant_type=refresh_token&client_id=%s&refresh_token=%s", clientID, astToken)
}

func getPasswordCredentialsPayload(username, password, adminClientID, adminClientSecret string) string {
	logger.PrintIfVerbose("Using username and password credentials.")
	// escape possible characters such as +,%, etc...
	encodedUsername := url.QueryEscape(username)
	encodedAdminClientID := url.QueryEscape(adminClientID)
	encodedPassword := url.QueryEscape(password)
	encodedAdminClientSecret := url.QueryEscape(adminClientSecret)
	return fmt.Sprintf(
		"scope=openid&grant_type=password&username=%s&password=%s"+
			"&client_id=%s&client_secret=%s", encodedUsername, encodedPassword, encodedAdminClientID, encodedAdminClientSecret,
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
	var body []byte
	retryLimit := int(viper.GetUint(commonParams.RetryFlag))
	retryWaitTimeSeconds := viper.GetUint(commonParams.RetryDelayFlag)
	logger.PrintRequest(req)
	if req.Body != nil {
		body, err = ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
	}
	// try starts at -1 as we always do at least one request, retryLimit can be 0
	for try := -1; try < retryLimit; try++ {
		if body != nil {
			_ = req.Body.Close()
			req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		}
		logger.PrintIfVerbose(
			fmt.Sprintf(
				"Request attempt %d in %d",
				try+tryPrintOffset, retryLimit+retryLimitPrintOffset,
			),
		)
		resp, err = client.Do(req)
		Domains = AppendIfNotExists(Domains, req.URL.Host)
		if err != nil {
			logger.PrintIfVerbose(err.Error())
		}
		if resp != nil && err == nil {
			if hasRedirectStatusCode(resp) {
				req, err = handleRedirect(resp, req, body)
				continue
			}
			logger.PrintResponse(resp, responseBody)
			return resp, nil
		}
		logger.PrintIfVerbose(fmt.Sprintf("Request failed in attempt %d", try+tryPrintOffset))
		time.Sleep(time.Duration(retryWaitTimeSeconds) * time.Second)
	}
	return nil, err
}

func handleRedirect(resp *http.Response, req *http.Request, body []byte) (*http.Request, error) {
	redirectURL := resp.Header.Get("Location")
	if redirectURL == "" {
		return nil, fmt.Errorf(applicationErrors.RedirectURLNotFound)
	}

	method := GetHTTPMethod(req)
	if method == "" {
		return nil, fmt.Errorf(applicationErrors.HTTPMethodNotFound)
	}

	newReq, err := recreateRequest(req, method, redirectURL, body)
	if err != nil {
		return nil, err
	}

	return newReq, nil
}

func recreateRequest(oldReq *http.Request, method, redirectURL string, body []byte) (*http.Request, error) {
	newReq, err := http.NewRequest(method, redirectURL, io.NopCloser(bytes.NewBuffer(body)))
	if err != nil {
		return nil, err
	}

	for key, values := range oldReq.Header {
		for _, value := range values {
			newReq.Header.Add(key, value)
		}
	}

	return newReq, nil
}

func GetHTTPMethod(req *http.Request) string {
	switch req.Method {
	case http.MethodGet:
		return http.MethodGet
	case http.MethodPost:
		return http.MethodPost
	case http.MethodPut:
		return http.MethodPut
	case http.MethodDelete:
		return http.MethodDelete
	case http.MethodOptions:
		return http.MethodOptions
	case http.MethodPatch:
		return http.MethodPatch
	default:
		return ""
	}
}

func hasRedirectStatusCode(resp *http.Response) bool {
	return resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusMovedPermanently
}

func GetAuthURI() (string, error) {
	var authURI string
	var err error
	override := viper.GetBool(commonParams.ApikeyOverrideFlag)

	apiKey := viper.GetString(commonParams.AstAPIKey)
	if len(apiKey) > 0 {
		logger.PrintIfVerbose("Base Auth URI - Extract from API KEY")
		authURI, err = ExtractFromTokenClaims(apiKey, audienceClaimKey)
		if err != nil {
			return "", err
		}
	}

	if authURI == "" || override {
		logger.PrintIfVerbose("Base Auth URI - Extract from Base Auth URI flag")
		authURI = strings.TrimSpace(viper.GetString(commonParams.BaseAuthURIKey))

		if authURI != "" {
			authURI, err = addTenantAuthURI(authURI)
			if err != nil {
				return "", err
			}
		}
	}

	if authURI == "" {
		logger.PrintIfVerbose("Base Auth URI - Extract from Base URI")
		authURI, err = GetURL("", "")
		if err != nil {
			return "", err
		}

		if authURI != "" {
			authURI, err = addTenantAuthURI(authURI)
			if err != nil {
				return "", err
			}
		}
	}

	if err != nil {
		return "", err
	}

	authURI = strings.Trim(authURI, "/")
	logger.PrintIfVerbose(fmt.Sprintf("Base Auth URI - %s ", authURI))
	return fmt.Sprintf("%s/%s", authURI, BaseAuthURLSuffix), nil
}

func GetURL(path, accessToken string) (string, error) {
	var err error
	var cleanURL string
	override := viper.GetBool(commonParams.ApikeyOverrideFlag)
	if accessToken != "" {
		logger.PrintIfVerbose("Base URI - Extract from JWT token")
		cleanURL, err = ExtractFromTokenClaims(accessToken, baseURLKey)
		if err != nil {
			return "", err
		}
	}

	if cleanURL == "" || override {
		logger.PrintIfVerbose("Base URI - Extract from Base URI flag")
		cleanURL = strings.TrimSpace(viper.GetString(commonParams.BaseURIKey))
	}

	if cleanURL == "" {
		return "", errors.Errorf(MissingURI)
	}

	cleanURL = strings.Trim(cleanURL, "/")
	logger.PrintIfVerbose(fmt.Sprintf("Base URI - %s ", cleanURL))

	return fmt.Sprintf("%s/%s", cleanURL, path), nil
}

func ExtractFromTokenClaims(accessToken, claim string) (string, error) {
	var value string

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())

	token, _, err := parser.ParseUnverified(accessToken, jwt.MapClaims{})
	if err != nil {
		return "", errors.Errorf(APIKeyDecodeErrorFormat, err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && claims[claim] != nil {
		value = strings.TrimSpace(claims[claim].(string))
	} else {
		return "", errors.Errorf(jwtError, claim)
	}

	return value, nil
}

func AppendIfNotExists(domainsMap map[string]struct{}, newDomain string) map[string]struct{} {
	if _, exists := domainsMap[newDomain]; !exists {
		domainsMap[newDomain] = struct{}{}
	}
	return domainsMap
}

func extractAZPFromToken(astToken string) (string, error) {
	const azpClaim = "azp"
	azp, err := ExtractFromTokenClaims(astToken, azpClaim)
	if err != nil {
		return "ast-app", nil // default value in case of error
	}
	return azp, nil
}
