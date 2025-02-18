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
		if resp.StatusCode != http.StatusBadGateway {
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
	authURI, err := GetAuthURI()
	if err != nil {
		return "", err
	}
	tokenExpirySeconds := viper.GetInt(commonParams.TokenExpirySecondsKey)
	accessToken := getClientCredentialsFromCache(tokenExpirySeconds)
	accessKeyID := viper.GetString(commonParams.AccessKeyIDConfigKey)
	accessKeySecret := viper.GetString(commonParams.AccessKeySecretConfigKey)
	astAPIKey := viper.GetString(commonParams.AstAPIKey)
	if accessKeyID == "" && astAPIKey == "" {
		return "", errors.Errorf(fmt.Sprintf(FailedToAuth, "access key ID"))
	} else if accessKeySecret == "" && astAPIKey == "" {
		return "", errors.Errorf(fmt.Sprintf(FailedToAuth, "access key secret"))
	}
	if accessToken == "" {
		accessToken, err = getClientCredentials(accessKeyID, accessKeySecret, astAPIKey, authURI)
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

func getClientCredentials(accessKeyID, accessKeySecret, astAPKey, authURI string) (string, error) {
	logger.PrintIfVerbose("Fetching API access token.")
	tokenExpirySeconds := viper.GetInt(commonParams.TokenExpirySecondsKey)

	var err error
	accessToken := getClientCredentialsFromCache(tokenExpirySeconds)

	if accessToken == "" {
		// If the token is present the default to that.
		if astAPKey != "" {
			accessToken, err = getNewToken(getAPIKeyPayload(astAPKey), authURI)
		} else {
			accessToken, err = getNewToken(getCredentialsPayload(accessKeyID, accessKeySecret), authURI)
		}

		if err != nil {
			return "", errors.Errorf("%s", err)
		}

		writeCredentialsToCache(accessToken)
	}

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

	return fmt.Sprintf("grant_type=refresh_tokenpackage wrappers\n\nimport (\n\t\"bytes\"\n\t\"crypto/tls\"\n\t\"encoding/json\"\n\t\"fmt\"\n\t\"io\"\n\t\"io/ioutil\"\n\t\"net\"\n\t\"net/http\"\n\t\"net/http/httptrace\"\n\t\"net/url\"\n\t\"strings\"\n\t\"sync\"\n\t\"time\"\n\n\tapplicationErrors \"github.com/checkmarx/ast-cli/internal/constants/errors\"\n\t\"github.com/checkmarx/ast-cli/internal/logger\"\n\t\"github.com/golang-jwt/jwt/v5\"\n\n\t\"github.com/pkg/errors\"\n\t\"github.com/spf13/viper\"\n\n\tcommonParams \"github.com/checkmarx/ast-cli/internal/params\"\n\t\"github.com/checkmarx/ast-cli/internal/wrappers/ntlm\"\n)\n\nconst (\n\texpiryGraceSeconds      = 10\n\tNoTimeout               = 0\n\tntlmProxyToken          = \"ntlm\"\n\tcheckmarxURLError       = \"Could not reach provided Checkmarx server\"\n\tinvalidCredentialsError = \"Provided credentials are invalid\"\n\tAPIKeyDecodeErrorFormat = \"Token decoding error: %s\"\n\ttryPrintOffset          = 2\n\tretryLimitPrintOffset   = 1\n\tMissingURI              = \"When using client-id and client-secret please provide base-uri or base-auth-uri\"\n\tMissingTenant           = \"Failed to authenticate - please provide tenant\"\n\tjwtError                = \"Error retrieving %s from jwt token\"\n\tbasicFormat             = \"Basic %s\"\n\tbearerFormat            = \"Bearer %s\"\n\tcontentTypeHeader       = \"Content-Type\"\n\tformURLContentType      = \"application/x-www-form-urlencoded\"\n\tjsonContentType         = \"application/json\"\n)\n\nvar (\n\tcredentialsMutex sync.Mutex\n)\n\ntype ClientCredentialsInfo struct {\n\tAccessToken      string `json:\"access_token\"`\n\tExpiresIn        int    `json:\"expires_in\"`\n\tRefreshExpiresIn int    `json:\"refresh_expires_in\"`\n\tRefreshToken     string `json:\"refresh_token\"`\n\tTokenType        string `json:\"token_type\"`\n\tSessionState     string `json:\"session_state\"`\n\tScope            string `json:\"scope\"`\n}\n\ntype ClientCredentialsError struct {\n\tError       string `json:\"error\"`\n\tDescription string `json:\"error_description\"`\n}\n\nconst FailedToAuth = \"Failed to authenticate - please provide an %s\"\nconst BaseAuthURLSuffix = \"protocol/openid-connect/token\"\nconst BaseAuthURLPrefix = \"auth/realms/organization\"\nconst baseURLKey = \"ast-base-url\"\n\nconst audienceClaimKey = \"aud\"\n\nvar cachedAccessToken string\nvar cachedAccessTime time.Time\nvar Domains = make(map[string]struct{})\n\nfunc retryHTTPRequest(requestFunc func() (*http.Response, error), retries int, baseDelayInMilliSec time.Duration) (*http.Response, error) {\n\n\tvar resp *http.Response\n\tvar err error\n\n\tfor attempt := 0; attempt < retries; attempt++ {\n\t\tresp, err = requestFunc()\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t\tif resp.StatusCode != http.StatusBadGateway {\n\t\t\treturn resp, nil\n\t\t}\n\t\t_ = resp.Body.Close()\n\t\ttime.Sleep(baseDelayInMilliSec * (1 << attempt))\n\t}\n\treturn resp, nil\n}\n\nfunc setAgentName(req *http.Request) {\n\tagentStr := viper.GetString(commonParams.AgentNameKey) + \"/\" + commonParams.Version\n\treq.Header.Set(\"User-Agent\", agentStr)\n}\n\nfunc GetClient(timeout uint) *http.Client {\n\tproxyTypeStr := viper.GetString(commonParams.ProxyTypeKey)\n\tproxyStr := viper.GetString(commonParams.ProxyKey)\n\tignoreProxy := viper.GetBool(commonParams.IgnoreProxyKey)\n\n\tvar client *http.Client\n\tif ignoreProxy {\n\t\tclient = basicProxyClient(timeout, \"\")\n\t} else if proxyTypeStr == ntlmProxyToken {\n\t\tclient = ntmlProxyClient(timeout, proxyStr)\n\t} else {\n\t\tclient = basicProxyClient(timeout, proxyStr)\n\t}\n\n\tclient.CheckRedirect = func(req *http.Request, via []*http.Request) error {\n\t\tif len(via) > 1 {\n\t\t\treturn fmt.Errorf(\"too many redirects\")\n\t\t}\n\t\tif len(via) != 0 && req.Response.StatusCode == http.StatusMovedPermanently {\n\t\t\tfor attr, val := range via[0].Header {\n\t\t\t\tif _, ok := req.Header[attr]; !ok {\n\t\t\t\t\treq.Header[attr] = val\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\n\t\treturn nil\n\t}\n\n\treturn client\n}\n\nfunc basicProxyClient(timeout uint, proxyStr string) *http.Client {\n\tinsecure := viper.GetBool(\"insecure\")\n\tu, _ := url.Parse(proxyStr)\n\tvar tr *http.Transport\n\tif len(proxyStr) > 0 {\n\t\tlogger.PrintIfVerbose(\"Creating HTTP Client with Proxy: \" + proxyStr)\n\t\ttr = &http.Transport{\n\t\t\tTLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},\n\t\t\tProxy:           http.ProxyURL(u),\n\t\t}\n\t} else {\n\t\tlogger.PrintIfVerbose(\"Creating HTTP Client.\")\n\t\ttr = &http.Transport{\n\t\t\tTLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},\n\t\t}\n\t}\n\treturn &http.Client{Transport: tr, Timeout: time.Duration(timeout) * time.Second}\n}\n\nfunc ntmlProxyClient(timeout uint, proxyStr string) *http.Client {\n\tdialer := &net.Dialer{\n\t\tTimeout:   30 * time.Second,\n\t\tKeepAlive: 30 * time.Second,\n\t}\n\tu, _ := url.Parse(proxyStr)\n\tdomainStr := viper.GetString(commonParams.ProxyDomainKey)\n\tproxyUser := u.User.Username()\n\tproxyPass, _ := u.User.Password()\n\tlogger.PrintIfVerbose(\"Creating HTTP client using NTLM Proxy using: \" + proxyStr)\n\tntlmDialContext := ntlm.NewNTLMProxyDialContext(dialer, u, proxyUser, proxyPass, domainStr, nil)\n\treturn &http.Client{\n\t\tTransport: &http.Transport{\n\t\t\tProxy:       nil,\n\t\t\tDialContext: ntlmDialContext,\n\t\t},\n\t\tTimeout: time.Duration(timeout) * time.Second,\n\t}\n}\n\nfunc getURLAndAccessToken(path string) (urlFromPath, accessToken string, err error) {\n\taccessToken, err = GetAccessToken()\n\tif err != nil {\n\t\treturn \"\", \"\", err\n\t}\n\turlFromPath, err = GetURL(path, accessToken)\n\tif err != nil {\n\t\treturn \"\", \"\", err\n\t}\n\treturn\n}\n\nfunc SendHTTPRequest(method, path string, body io.Reader, auth bool, timeout uint) (*http.Response, error) {\n\tu, accessToken, err := getURLAndAccessToken(path)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treturn SendHTTPRequestByFullURL(method, u, body, auth, timeout, accessToken, true)\n}\n\nfunc SendPrivateHTTPRequest(method, path string, body io.Reader, timeout uint, auth bool) (*http.Response, error) {\n\tu, accessToken, err := getURLAndAccessToken(path)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treturn SendHTTPRequestByFullURL(method, u, body, auth, timeout, accessToken, false)\n}\n\nfunc SendHTTPRequestByFullURL(\n\tmethod, fullURL string,\n\tbody io.Reader,\n\tauth bool,\n\ttimeout uint,\n\taccessToken string,\n\tbodyPrint bool,\n) (*http.Response, error) {\n\treturn SendHTTPRequestByFullURLContentLength(method, fullURL, body, -1, auth, timeout, accessToken, bodyPrint)\n}\n\nfunc SendHTTPRequestByFullURLContentLength(\n\tmethod, fullURL string,\n\tbody io.Reader,\n\tcontentLength int64,\n\tauth bool,\n\ttimeout uint,\n\taccessToken string,\n\tbodyPrint bool,\n) (*http.Response, error) {\n\treq, err := http.NewRequest(method, fullURL, body)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tif contentLength >= 0 {\n\t\treq.ContentLength = contentLength\n\t}\n\tclient := GetClient(timeout)\n\tsetAgentName(req)\n\tif auth {\n\t\tenrichWithOath2Credentials(req, accessToken, bearerFormat)\n\t}\n\n\treq = addReqMonitor(req)\n\n\treturn request(client, req, bodyPrint)\n}\n\nfunc addReqMonitor(req *http.Request) *http.Request {\n\tstartTime := time.Now().UnixNano() / int64(time.Millisecond)\n\tif viper.GetBool(commonParams.DebugFlag) {\n\t\ttrace := &httptrace.ClientTrace{\n\t\t\tGetConn: func(hostPort string) {\n\t\t\t\tstartTime = time.Now().UnixNano() / int64(time.Millisecond)\n\t\t\t\tlogger.Print(\"Starting connection: \" + hostPort)\n\t\t\t},\n\t\t\tDNSStart: func(dnsInfo httptrace.DNSStartInfo) {\n\t\t\t\tlogger.Print(\"DNS looking up host information for: \" + dnsInfo.Host)\n\t\t\t},\n\t\t\tDNSDone: func(dnsInfo httptrace.DNSDoneInfo) {\n\t\t\t\tlogger.Printf(\"DNS found host address(s): %+v\\n\", dnsInfo.Addrs)\n\t\t\t},\n\t\t\tTLSHandshakeStart: func() {\n\t\t\t\tlogger.Print(\"Started TLS Handshake\")\n\t\t\t},\n\t\t\tTLSHandshakeDone: func(c tls.ConnectionState, err error) {\n\t\t\t\tif err == nil {\n\t\t\t\t\tlogger.Print(\"Completed TLS handshake\")\n\t\t\t\t} else {\n\t\t\t\t\tlogger.Printf(\"%s, %s\", \"Error completing TLS handshake\", err)\n\t\t\t\t}\n\t\t\t},\n\t\t\tGotFirstResponseByte: func() {\n\t\t\t\tendTime := time.Now().UnixNano() / int64(time.Millisecond)\n\t\t\t\tdiff := endTime - startTime\n\t\t\t\tlogger.Printf(\"Connected completed in: %d (ms)\", diff)\n\t\t\t},\n\t\t}\n\t\treturn req.WithContext(httptrace.WithClientTrace(req.Context(), trace))\n\t}\n\treturn req\n}\n\nfunc SendHTTPRequestPasswordAuth(method string, body io.Reader, timeout uint, username, password, adminClientID, adminClientSecret string) (*http.Response, error) {\n\tu, err := GetAuthURI()\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treq, err := http.NewRequest(method, u, body)\n\tclient := GetClient(timeout)\n\tsetAgentName(req)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treq.Header.Add(contentTypeHeader, jsonContentType)\n\terr = enrichWithPasswordCredentials(req, username, password, adminClientID, adminClientSecret)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treq = addReqMonitor(req)\n\treturn doRequest(client, req)\n}\n\nfunc SendPrivateHTTPRequestWithQueryParams(\n\tmethod, path string, params map[string]string,\n\tbody io.Reader, timeout uint,\n) (*http.Response, error) {\n\treturn HTTPRequestWithQueryParams(method, path, params, body, timeout, false)\n}\n\nfunc SendHTTPRequestWithQueryParams(\n\tmethod, path string, params map[string]string,\n\tbody io.Reader, timeout uint,\n) (*http.Response, error) {\n\treturn HTTPRequestWithQueryParams(method, path, params, body, timeout, true)\n}\n\nfunc HTTPRequestWithQueryParams(\n\tmethod, path string, params map[string]string,\n\tbody io.Reader, timeout uint, printBody bool,\n) (*http.Response, error) {\n\tu, accessToken, err := getURLAndAccessToken(path)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treq, err := http.NewRequest(method, u, body)\n\tclient := GetClient(timeout)\n\tsetAgentName(req)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tq := req.URL.Query()\n\tfor k, v := range params {\n\t\tq.Add(k, v)\n\t}\n\treq.URL.RawQuery = q.Encode()\n\tenrichWithOath2Credentials(req, accessToken, bearerFormat)\n\tvar resp *http.Response\n\tresp, err = request(client, req, printBody)\n\tif err != nil {\n\t\treturn resp, errors.Errorf(\"%s %s \\n\", checkmarxURLError, req.URL.RequestURI())\n\t}\n\tif resp.StatusCode == http.StatusForbidden {\n\t\treturn resp, errors.Errorf(\"%s\", \"Provided credentials do not have permissions for this command\")\n\t}\n\treturn resp, nil\n}\n\nfunc addTenantAuthURI(baseAuthURI string) (string, error) {\n\tauthPath := BaseAuthURLPrefix\n\ttenant := viper.GetString(commonParams.TenantKey)\n\n\tif tenant == \"\" {\n\t\treturn \"\", errors.Errorf(MissingTenant)\n\t}\n\n\tauthPath = strings.Replace(authPath, \"organization\", strings.ToLower(tenant), 1)\n\n\treturn fmt.Sprintf(\"%s/%s\", strings.Trim(baseAuthURI, \"/\"), authPath), nil\n}\n\nfunc enrichWithOath2Credentials(request *http.Request, accessToken, authFormat string) {\n\trequest.Header.Add(AuthorizationHeader, fmt.Sprintf(authFormat, accessToken))\n}\n\nfunc SendHTTPRequestWithJSONContentType(method, path string, body io.Reader, auth bool, timeout uint) (\n\t*http.Response,\n\terror,\n) {\n\tfullURL, accessToken, err := getURLAndAccessToken(path)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\treq, err := http.NewRequest(method, fullURL, body)\n\tclient := GetClient(timeout)\n\tsetAgentName(req)\n\treq.Header.Add(\"Content-Type\", jsonContentType)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tif auth {\n\t\tenrichWithOath2Credentials(req, accessToken, bearerFormat)\n\t}\n\n\treq = addReqMonitor(req)\n\treturn doRequest(client, req)\n}\n\nfunc GetWithQueryParams(client *http.Client, urlAddress, token, authFormat string, queryParams map[string]string) (*http.Response, error) {\n\treq, err := http.NewRequest(http.MethodGet, urlAddress, http.NoBody)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tlogger.PrintRequest(req)\n\treturn GetWithQueryParamsAndCustomRequest(client, req, urlAddress, token, authFormat, queryParams)\n}\n\n// GetWithQueryParamsAndCustomRequest used when we need to add custom headers to the request\nfunc GetWithQueryParamsAndCustomRequest(client *http.Client, customReq *http.Request, urlAddress, token, authFormat string, queryParams map[string]string) (*http.Response, error) {\n\tif len(token) > 0 {\n\t\tenrichWithOath2Credentials(customReq, token, authFormat)\n\t}\n\tq := customReq.URL.Query()\n\tfor k, v := range queryParams {\n\t\tq.Add(k, v)\n\t}\n\tcustomReq.URL.RawQuery = q.Encode()\n\tcustomReq = addReqMonitor(customReq)\n\treturn request(client, customReq, true)\n}\nfunc GetAccessToken() (string, error) {\n\tauthURI, err := GetAuthURI()\n\tif err != nil {\n\t\treturn \"\", err\n\t}\n\ttokenExpirySeconds := viper.GetInt(commonParams.TokenExpirySecondsKey)\n\taccessToken := getClientCredentialsFromCache(tokenExpirySeconds)\n\taccessKeyID := viper.GetString(commonParams.AccessKeyIDConfigKey)\n\taccessKeySecret := viper.GetString(commonParams.AccessKeySecretConfigKey)\n\tastAPIKey := viper.GetString(commonParams.AstAPIKey)\n\tif accessKeyID == \"\" && astAPIKey == \"\" {\n\t\treturn \"\", errors.Errorf(fmt.Sprintf(FailedToAuth, \"access key ID\"))\n\t} else if accessKeySecret == \"\" && astAPIKey == \"\" {\n\t\treturn \"\", errors.Errorf(fmt.Sprintf(FailedToAuth, \"access key secret\"))\n\t}\n\tif accessToken == \"\" {\n\t\taccessToken, err = getClientCredentials(accessKeyID, accessKeySecret, astAPIKey, authURI)\n\t\tif err != nil {\n\t\t\treturn \"\", err\n\t\t}\n\t}\n\treturn accessToken, nil\n}\n\nfunc enrichWithPasswordCredentials(\n\trequest *http.Request, username, password,\n\tadminClientID, adminClientSecret string,\n) error {\n\tauthURI, err := GetAuthURI()\n\tif err != nil {\n\t\treturn err\n\t}\n\n\taccessToken, err := getNewToken(\n\t\tgetPasswordCredentialsPayload(username, password, adminClientID, adminClientSecret),\n\t\tauthURI,\n\t)\n\tif err != nil {\n\t\treturn errors.Wrap(\n\t\t\terrors.Wrap(err, \"failed to get access token from auth server\"),\n\t\t\t\"failed to authenticate\",\n\t\t)\n\t}\n\tenrichWithOath2Credentials(request, accessToken, bearerFormat)\n\treturn nil\n}\n\nfunc getClientCredentials(accessKeyID, accessKeySecret, astAPKey, authURI string) (string, error) {\n\tlogger.PrintIfVerbose(\"Fetching API access token.\")\n\ttokenExpirySeconds := viper.GetInt(commonParams.TokenExpirySecondsKey)\n\n\tvar err error\n\taccessToken := getClientCredentialsFromCache(tokenExpirySeconds)\n\n\tif accessToken == \"\" {\n\t\t// If the token is present the default to that.\n\t\tif astAPKey != \"\" {\n\t\t\taccessToken, err = getNewToken(getAPIKeyPayload(astAPKey), authURI)\n\t\t} else {\n\t\t\taccessToken, err = getNewToken(getCredentialsPayload(accessKeyID, accessKeySecret), authURI)\n\t\t}\n\n\t\tif err != nil {\n\t\t\treturn \"\", errors.Errorf(\"%s\", err)\n\t\t}\n\n\t\twriteCredentialsToCache(accessToken)\n\t}\n\n\treturn accessToken, nil\n}\n\nfunc getClientCredentialsFromCache(tokenExpirySeconds int) string {\n\tlogger.PrintIfVerbose(\"Checking cache for API access token.\")\n\n\texpired := time.Since(cachedAccessTime) > time.Duration(tokenExpirySeconds-expiryGraceSeconds)*time.Second\n\tif !expired {\n\t\tlogger.PrintIfVerbose(\"Using cached API access token!\")\n\t\treturn cachedAccessToken\n\t}\n\tlogger.PrintIfVerbose(\"API access token not found in cache!\")\n\treturn \"\"\n}\n\nfunc writeCredentialsToCache(accessToken string) {\n\tcredentialsMutex.Lock()\n\tdefer credentialsMutex.Unlock()\n\n\tlogger.PrintIfVerbose(\"Storing API access token to cache.\")\n\tviper.Set(commonParams.AstToken, accessToken)\n\tcachedAccessToken = accessToken\n\tcachedAccessTime = time.Now()\n}\n\nfunc getNewToken(credentialsPayload, authServerURI string) (string, error) {\n\tpayload := strings.NewReader(credentialsPayload)\n\treq, err := http.NewRequest(http.MethodPost, authServerURI, payload)\n\tsetAgentName(req)\n\tif err != nil {\n\t\treturn \"\", err\n\t}\n\treq = addReqMonitor(req)\n\treq.Header.Add(contentTypeHeader, formURLContentType)\n\tclientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)\n\tclient := GetClient(clientTimeout)\n\n\tres, err := doPrivateRequest(client, req)\n\tif err != nil {\n\t\tauthURL, _ := GetAuthURI()\n\t\treturn \"\", errors.Errorf(\"%s %s\", checkmarxURLError, authURL)\n\t}\n\tif res.StatusCode == http.StatusBadRequest {\n\t\treturn \"\", errors.Errorf(\"%d %s \\n\", res.StatusCode, invalidCredentialsError)\n\t}\n\tif res.StatusCode == http.StatusNotFound {\n\t\treturn \"\", errors.Errorf(\"%d %s \\n\", res.StatusCode, \"Provided Tenant Name is invalid\")\n\t}\n\tif res.StatusCode == http.StatusUnauthorized {\n\t\treturn \"\", errors.Errorf(\"%d %s \\n\", res.StatusCode, invalidCredentialsError)\n\t}\n\n\tbody, _ := ioutil.ReadAll(res.Body)\n\tif res.StatusCode != http.StatusOK {\n\t\tcredentialsErr := ClientCredentialsError{}\n\t\terr = json.Unmarshal(body, &credentialsErr)\n\n\t\tif err != nil {\n\t\t\treturn \"\", err\n\t\t}\n\n\t\treturn \"\", errors.Errorf(\"%d %s %s\", res.StatusCode, credentialsErr.Error, credentialsErr.Description)\n\t}\n\n\tdefer func() {\n\t\t_ = res.Body.Close()\n\t}()\n\n\tcredentialsInfo := ClientCredentialsInfo{}\n\terr = json.Unmarshal(body, &credentialsInfo)\n\tif err != nil {\n\t\treturn \"\", err\n\t}\n\n\tlogger.PrintIfVerbose(\"Successfully retrieved API token.\")\n\treturn credentialsInfo.AccessToken, nil\n}\n\nfunc getCredentialsPayload(accessKeyID, accessKeySecret string) string {\n\tlogger.PrintIfVerbose(\"Using Client ID and secret credentials.\")\n\t// escape possible characters such as +,%, etc...\n\tclientID := url.QueryEscape(accessKeyID)\n\tclientSecret := url.QueryEscape(accessKeySecret)\n\treturn fmt.Sprintf(\"grant_type=client_credentials&client_id=%s&client_secret=%s\", clientID, clientSecret)\n}\n\nfunc getAPIKeyPayload(astToken string) string {\n\tlogger.PrintIfVerbose(\"Using API key credentials.\")\n\t\n\tclientID, err := extractAZPFromToken(astToken)\n\tif err != nil {\n\t\tlogger.PrintIfVerbose(\"Failed to extract azp from token, using default client_id\")\n\t\tclientID = \"ast-app\"\n\t}\n\t\n\treturn fmt.Sprintf(\"grant_type=refresh_token&client_id=%s&refresh_token=%s\", clientID, astToken)\n}\n\nfunc getPasswordCredentialsPayload(username, password, adminClientID, adminClientSecret string) string {\n\tlogger.PrintIfVerbose(\"Using username and password credentials.\")\n\t// escape possible characters such as +,%, etc...\n\tencodedUsername := url.QueryEscape(username)\n\tencodedAdminClientID := url.QueryEscape(adminClientID)\n\tencodedPassword := url.QueryEscape(password)\n\tencodedAdminClientSecret := url.QueryEscape(adminClientSecret)\n\treturn fmt.Sprintf(\n\t\t\"scope=openid&grant_type=password&username=%s&password=%s\"+\n\t\t\t\"&client_id=%s&client_secret=%s\", encodedUsername, encodedPassword, encodedAdminClientID, encodedAdminClientSecret,\n\t)\n}\n\nfunc doPrivateRequest(client *http.Client, req *http.Request) (*http.Response, error) {\n\treturn request(client, req, false)\n}\n\nfunc doRequest(client *http.Client, req *http.Request) (*http.Response, error) {\n\treturn request(client, req, true)\n}\n\nfunc request(client *http.Client, req *http.Request, responseBody bool) (*http.Response, error) {\n\tvar err error\n\tvar resp *http.Response\n\tvar body []byte\n\tretryLimit := int(viper.GetUint(commonParams.RetryFlag))\n\tretryWaitTimeSeconds := viper.GetUint(commonParams.RetryDelayFlag)\n\tlogger.PrintRequest(req)\n\tif req.Body != nil {\n\t\tbody, err = ioutil.ReadAll(req.Body)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t}\n\t// try starts at -1 as we always do at least one request, retryLimit can be 0\n\tfor try := -1; try < retryLimit; try++ {\n\t\tif body != nil {\n\t\t\t_ = req.Body.Close()\n\t\t\treq.Body = ioutil.NopCloser(bytes.NewBuffer(body))\n\t\t}\n\t\tlogger.PrintIfVerbose(\n\t\t\tfmt.Sprintf(\n\t\t\t\t\"Request attempt %d in %d\",\n\t\t\t\ttry+tryPrintOffset, retryLimit+retryLimitPrintOffset,\n\t\t\t),\n\t\t)\n\t\tresp, err = client.Do(req)\n\t\tDomains = AppendIfNotExists(Domains, req.URL.Host)\n\t\tif err != nil {\n\t\t\tlogger.PrintIfVerbose(err.Error())\n\t\t}\n\t\tif resp != nil && err == nil {\n\t\t\tif hasRedirectStatusCode(resp) {\n\t\t\t\treq, err = handleRedirect(resp, req, body)\n\t\t\t\tcontinue\n\t\t\t}\n\t\t\tlogger.PrintResponse(resp, responseBody)\n\t\t\treturn resp, nil\n\t\t}\n\t\tlogger.PrintIfVerbose(fmt.Sprintf(\"Request failed in attempt %d\", try+tryPrintOffset))\n\t\ttime.Sleep(time.Duration(retryWaitTimeSeconds) * time.Second)\n\t}\n\treturn nil, err\n}\n\nfunc handleRedirect(resp *http.Response, req *http.Request, body []byte) (*http.Request, error) {\n\tredirectURL := resp.Header.Get(\"Location\")\n\tif redirectURL == \"\" {\n\t\treturn nil, fmt.Errorf(applicationErrors.RedirectURLNotFound)\n\t}\n\n\tmethod := GetHTTPMethod(req)\n\tif method == \"\" {\n\t\treturn nil, fmt.Errorf(applicationErrors.HTTPMethodNotFound)\n\t}\n\n\tnewReq, err := recreateRequest(req, method, redirectURL, body)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\treturn newReq, nil\n}\n\nfunc recreateRequest(oldReq *http.Request, method, redirectURL string, body []byte) (*http.Request, error) {\n\tnewReq, err := http.NewRequest(method, redirectURL, io.NopCloser(bytes.NewBuffer(body)))\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\tfor key, values := range oldReq.Header {\n\t\tfor _, value := range values {\n\t\t\tnewReq.Header.Add(key, value)\n\t\t}\n\t}\n\n\treturn newReq, nil\n}\n\nfunc GetHTTPMethod(req *http.Request) string {\n\tswitch req.Method {\n\tcase http.MethodGet:\n\t\treturn http.MethodGet\n\tcase http.MethodPost:\n\t\treturn http.MethodPost\n\tcase http.MethodPut:\n\t\treturn http.MethodPut\n\tcase http.MethodDelete:\n\t\treturn http.MethodDelete\n\tcase http.MethodOptions:\n\t\treturn http.MethodOptions\n\tcase http.MethodPatch:\n\t\treturn http.MethodPatch\n\tdefault:\n\t\treturn \"\"\n\t}\n}\n\nfunc hasRedirectStatusCode(resp *http.Response) bool {\n\treturn resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusMovedPermanently\n}\n\nfunc GetAuthURI() (string, error) {\n\tvar authURI string\n\tvar err error\n\toverride := viper.GetBool(commonParams.ApikeyOverrideFlag)\n\n\tapiKey := viper.GetString(commonParams.AstAPIKey)\n\tif len(apiKey) > 0 {\n\t\tlogger.PrintIfVerbose(\"Base Auth URI - Extract from API KEY\")\n\t\tauthURI, err = ExtractFromTokenClaims(apiKey, audienceClaimKey)\n\t\tif err != nil {\n\t\t\treturn \"\", err\n\t\t}\n\t}\n\n\tif authURI == \"\" || override {\n\t\tlogger.PrintIfVerbose(\"Base Auth URI - Extract from Base Auth URI flag\")\n\t\tauthURI = strings.TrimSpace(viper.GetString(commonParams.BaseAuthURIKey))\n\n\t\tif authURI != \"\" {\n\t\t\tauthURI, err = addTenantAuthURI(authURI)\n\t\t\tif err != nil {\n\t\t\t\treturn \"\", err\n\t\t\t}\n\t\t}\n\t}\n\n\tif authURI == \"\" {\n\t\tlogger.PrintIfVerbose(\"Base Auth URI - Extract from Base URI\")\n\t\tauthURI, err = GetURL(\"\", \"\")\n\t\tif err != nil {\n\t\t\treturn \"\", err\n\t\t}\n\n\t\tif authURI != \"\" {\n\t\t\tauthURI, err = addTenantAuthURI(authURI)\n\t\t\tif err != nil {\n\t\t\t\treturn \"\", err\n\t\t\t}\n\t\t}\n\t}\n\n\tif err != nil {\n\t\treturn \"\", err\n\t}\n\n\tauthURI = strings.Trim(authURI, \"/\")\n\tlogger.PrintIfVerbose(fmt.Sprintf(\"Base Auth URI - %s \", authURI))\n\treturn fmt.Sprintf(\"%s/%s\", authURI, BaseAuthURLSuffix), nil\n}\n\nfunc GetURL(path, accessToken string) (string, error) {\n\tvar err error\n\tvar cleanURL string\n\toverride := viper.GetBool(commonParams.ApikeyOverrideFlag)\n\tif accessToken != \"\" {\n\t\tlogger.PrintIfVerbose(\"Base URI - Extract from JWT token\")\n\t\tcleanURL, err = ExtractFromTokenClaims(accessToken, baseURLKey)\n\t\tif err != nil {\n\t\t\treturn \"\", err\n\t\t}\n\t}\n\n\tif cleanURL == \"\" || override {\n\t\tlogger.PrintIfVerbose(\"Base URI - Extract from Base URI flag\")\n\t\tcleanURL = strings.TrimSpace(viper.GetString(commonParams.BaseURIKey))\n\t}\n\n\tif cleanURL == \"\" {\n\t\treturn \"\", errors.Errorf(MissingURI)\n\t}\n\n\tcleanURL = strings.Trim(cleanURL, \"/\")\n\tlogger.PrintIfVerbose(fmt.Sprintf(\"Base URI - %s \", cleanURL))\n\n\treturn fmt.Sprintf(\"%s/%s\", cleanURL, path), nil\n}\n\nfunc ExtractFromTokenClaims(accessToken, claim string) (string, error) {\n\tvar value string\n\n\tparser := jwt.NewParser(jwt.WithoutClaimsValidation())\n\n\ttoken, _, err := parser.ParseUnverified(accessToken, jwt.MapClaims{})\n\tif err != nil {\n\t\treturn \"\", errors.Errorf(APIKeyDecodeErrorFormat, err)\n\t}\n\n\tif claims, ok := token.Claims.(jwt.MapClaims); ok && claims[claim] != nil {\n\t\tvalue = strings.TrimSpace(claims[claim].(string))\n\t} else {\n\t\treturn \"\", errors.Errorf(jwtError, claim)\n\t}\n\n\treturn value, nil\n}\n\nfunc AppendIfNotExists(domainsMap map[string]struct{}, newDomain string) map[string]struct{} {\n\tif _, exists := domainsMap[newDomain]; !exists {\n\t\tdomainsMap[newDomain] = struct{}{}\n\t}\n\treturn domainsMap\n}\n\nfunc extractAZPFromToken(astToken string) (string, error) {\n\tconst azpClaim = \"azp\"\n\tazp, err := ExtractFromTokenClaims(astToken, azpClaim)\n\tif err != nil {\n\t\treturn \"ast-app\", nil // default value in case of error\n\t}\n\treturn azp, nil\n}\n&client_id=%s&refresh_token=%s", clientID, astToken)
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
