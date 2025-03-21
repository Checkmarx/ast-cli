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

var CachedAccessToken string
var CachedAccessTime time.Time
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
			logger.PrintIfVerbose("Bad Gateway (502), retrying")
		} else if resp.StatusCode == http.StatusUnauthorized {
			logger.PrintIfVerbose("Unauthorized request (401), refreshing token")
			_, _ = configureClientCredentialsAndGetNewToken()
		} else {
			return resp, nil
		}
		_ = resp.Body.Close()
		time.Sleep(baseDelayInMilliSec * (1 << attempt))
	}
	return resp, nil
}

func setAgentNameAndOrigin(req *http.Request) {
	agentStr := viper.GetString(commonParams.AgentNameKey) + "/" + commonParams.Version
	req.Header.Set("User-Agent", agentStr)

	originStr := viper.GetString(commonParams.OriginKey)
	req.Header.Set("Cx-Origin", originStr)
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
	setAgentNameAndOrigin(req)
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
	setAgentNameAndOrigin(req)
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
	setAgentNameAndOrigin(req)
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
	setAgentNameAndOrigin(req)
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
	accessToken = "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJlMXd2YTJWRmlKakZYLWlUaVpQRmJfc0U0SEhaZ3RTV2hkZE5NR0sxNzRzIn0.eyJleHAiOjE3NDI1NjkxNDcsImlhdCI6MTc0MjU2NzM0NywiYXV0aF90aW1lIjoxNzQyNTY3MzQ1LCJqdGkiOiJkNjJhM2UxNS1jNTZhLTRhOGItYjUzNy00MDE5MjlhYmI0MGMiLCJpc3MiOiJodHRwczovL2lhbS1kZXYuZGV2LmN4YXN0Lm5ldC9hdXRoL3JlYWxtcy9kZXZfdGVuYW50IiwiYXVkIjpbInJlYWxtLW1hbmFnZW1lbnQiLCJhY2NvdW50Il0sInN1YiI6ImY2Mzc0ZTQwLTNkNTYtNGMyZS1iMGY3LTliNWUxZTllYzRhYiIsInR5cCI6IkJlYXJlciIsImF6cCI6ImFzdC1hcHAiLCJzaWQiOiIxNGRjNjk0Yi00ZjJjLTQzNGYtYWQ3ZS00ODNmNDU2OGJlMGQiLCJhbGxvd2VkLW9yaWdpbnMiOlsiaHR0cDovL2xvY2FsaG9zdDo4MDg0IiwiKiIsIi8qIiwiaHR0cHM6Ly9zYXN0LnB0LmN4ZGV2b3BzLmNvbS8qIl0sInJlc291cmNlX2FjY2VzcyI6eyJyZWFsbS1tYW5hZ2VtZW50Ijp7InJvbGVzIjpbInZpZXctcmVhbG0iLCJ2aWV3LWlkZW50aXR5LXByb3ZpZGVycyIsIm1hbmFnZS1pZGVudGl0eS1wcm92aWRlcnMiLCJpbXBlcnNvbmF0aW9uIiwicmVhbG0tYWRtaW4iLCJjcmVhdGUtY2xpZW50IiwibWFuYWdlLXVzZXJzIiwidW1hX3Byb3RlY3Rpb24iLCJxdWVyeS1yZWFsbXMiLCJtYW5hZ2Uta2V5cyIsInZpZXctYXV0aG9yaXphdGlvbiIsInF1ZXJ5LWNsaWVudHMiLCJxdWVyeS11c2VycyIsIm1hbmFnZS1ldmVudHMiLCJtYW5hZ2UtcmVhbG0iLCJ2aWV3LWV2ZW50cyIsInZpZXctdXNlcnMiLCJ2aWV3LWNsaWVudHMiLCJtYW5hZ2UtYXV0aG9yaXphdGlvbiIsIm1hbmFnZS1ncm91cHMiLCJtYW5hZ2UtY2xpZW50cyIsInF1ZXJ5LWdyb3VwcyJdfSwiYWNjb3VudCI6eyJyb2xlcyI6WyJtYW5hZ2UtYWNjb3VudCIsIm1hbmFnZS1hY2NvdW50LWxpbmtzIiwidmlldy1wcm9maWxlIl19fSwic2NvcGUiOiJvcGVuaWQgaWFtLWFwaSBncm91cHMgYXN0LWFwaSBwcm9maWxlIHJvbGVzIGVtYWlsIiwidGVuYW50X2lkIjoiaWFtLWRldi5kZXYuY3hhc3QubmV0Ojo3NWQ0ZmUzNS05NjVhLTQ1MDYtYjIyNi1lMDE1NWVjODRjMzQiLCJ0ZW5hbnRfbmFtZSI6ImRldl90ZW5hbnQiLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwicm9sZXMiOlsibWFuYWdlLXVzZXJzIiwibWFuYWdlLWtleXMiLCJ1bWFfYXV0aG9yaXphdGlvbiIsIm1hbmFnZS1ncm91cHMiLCJkZWZhdWx0LXJvbGVzLTc1ZDRmZTM1LTk2NWEtNDUwNi1iMjI2LWUwMTU1ZWM4NGMzNCIsInVzZXIiLCJtYW5hZ2UtY2xpZW50cyIsImlhbS1hZG1pbiJdLCJldWxhLWFjY2VwdGVkIjp0cnVlLCJncm91cHMiOltdLCJncm91cHNOYW1lcyI6W10sImNiLXVybCI6Imh0dHBzOi8vY2hlY2ttYXJ4LmNvZGViYXNoaW5nLmNvbSIsInByZWZlcnJlZF91c2VybmFtZSI6Im9yZ19hZG1pbiIsImdpdmVuX25hbWUiOiJvcmciLCJhc3QtYmFzZS11cmwiOiJodHRwczovL2FzdC1tYXN0ZXItY29tcG9uZW50cy5kZXYuY3hhc3QubmV0Iiwic2YtaWQiOiIwMDEzejAwMDAyTHpjdkFBQVIiLCJyb2xlc19hc3QiOlsiY3JlYXRlLXByb2plY3QiLCJhbmFseXRpY3Mtc2Nhbi1kYXNoYm9hcmQtdmlldyIsImRlbGV0ZS1hcHBsaWNhdGlvbiIsInVwZGF0ZS1wcm9qZWN0LWlmLWluLWdyb3VwIiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1zdGF0ZTEiLCJkYXN0LWRlbGV0ZS1zY2FuIiwiY3JlYXRlLXdlYmhvb2siLCJhbmFseXRpY3MtZXhlY3V0aXZlLW92ZXJ2aWV3LXZpZXciLCJ2aWV3LXByb2plY3RzLWlmLWluLWdyb3VwIiwidXBkYXRlLXNjYW4iLCJ1cGRhdGUtcmVzdWx0LXNldmVyaXR5IiwibWFuYWdlLXNhc3QtZGVmYXVsdC1jb25maWdzIiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1jdXN0b20tc3RhdGUtNTY3MzQ5MzYzNTM0Nzk3OTc0NyIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtY3VzdG9tLXN0YXRlLTc2NjYxNTA5NDcxMjI2NjE4ODciLCJ1cGRhdGUtcmVzdWx0LXN0YXRlLWN1c3RvbS1zdGF0ZS0xMDQ1ODY5NjMwNDAwNDI3OTEyIiwib3Blbi1mZWF0dXJlLXJlcXVlc3QiLCJ2aWV3LXBvbGljeS1tYW5hZ2VtZW50IiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1jdXN0b20tc3RhdGUtNzA4MTE0ODA5MDkxODQ1MjcyNCIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtbm90LWV4cGxvaXRhYmxlIiwidmlldy1xdWVyaWVzIiwibWFuYWdlLXdlYmhvb2siLCJ1cGRhdGUtc2NoZWR1bGUtc2NhbiIsImNyZWF0ZS1hcHBsaWNhdGlvbiIsInF1ZXJpZXMtZWRpdG9yIiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1oZWxsbyIsInVwZGF0ZS1xdWVyeSIsInVwZGF0ZS1yZXN1bHQtbm90LWV4cGxvaXRhYmxlIiwidXBkYXRlLXByb2plY3QiLCJ2aWV3LWVuZ2luZXMiLCJjcmVhdGUtcXVlcnkiLCJ2aWV3LWNvZGViYXNoaW5nIiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1hYWFhYSIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtbXl0ZXN0MiIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtY3VzdG9tLXN0YXRlLTI1ODU1MTY2NDQyMDc0NDQzMiIsImRhc3QtY3JlYXRlLWVudmlyb25tZW50IiwiYWRkLW5vdGVzIiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1saWF2IiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1jdXN0b20tc3RhdGUtNTc5MjczNDkzMDQ1MTg1NTg5MiIsImFzc2lnbi10by1hcHBsaWNhdGlvbiIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtY3VzdG9tLXN0YXRlLTk1Mzg1NTU5NDQwMDQ5NjQ2MSIsIm9wZW4tc3VwcG9ydC10aWNrZXQiLCJ2aWV3LWFjY2VzcyIsInVwZGF0ZS1zY2FuLWlmLWluLWdyb3VwIiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1ub3QtZXhwbG9pdGFibGUtaWYtaW4tZ3JvdXAiLCJzYXN0LW1pZ3JhdGlvbiIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtY3VzdG9tLXN0YXRlLTQzMzM2MDMzNDA4NDU0NTA4NiIsInZpZXctcnVudGltZS1jbG91ZCIsInVwZGF0ZS1wcm9qZWN0LXBhcmFtcyIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtY3VzdG9tLXN0YXRlLTE3ODI2NTY5MDY2ODk1NzI4MTkiLCJjcmVhdGUtcHJlc2V0IiwiZGFzdC1leHRlcm5hbC1zY2FucyIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtY3VzdG9tLXN0YXRlLTExMTU1MzkzMjMwMTYzNzg5ODQiLCJ1cGRhdGUtcmVzdWx0LXN0YXRlLWN1c3RvbS1zdGF0ZS0yODI4NDI1Mjg3Njk1NjY5MTY4IiwiZGVsZXRlLWxpbmtzIiwibWFuYWdlLXByb2plY3QiLCJ1cGRhdGUtcmVzdWx0LXN0YXRlLWNzMTIiLCJ1cGRhdGUtcmVzdWx0LXN0YXRlLWNzMTEiLCJ2aWV3LXNjYW5zIiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1jdXN0b20tc3RhdGUtNTIzOTU0MDE0NjM0NjM4Nzc5NiIsIlZpZXcgcXVlcmllcyIsImNyZWF0ZS1yZXN1bHQtY3VzdG9tLXN0YXRlIiwiYXN0LXNjYW5uZXIiLCJ1cGRhdGUtcmVzdWx0LXN0YXRlLXJvdGVtZGRkZGQiLCJ1cGRhdGUtcmVzdWx0LXNldmVyaXR5LWlmLWluLWdyb3VwIiwic3RhcnQtZGF0YS1yZXRlbnRpb24iLCJ1cGRhdGUtcmVzdWx0LXN0YXRlLWFhYWFhYSIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtcm90ZW1kMSIsInZpZXctcmlzay1tYW5hZ2VtZW50LWRhc2hib2FyZCIsInZpZXctcHJvamVjdC1wYXJhbXMiLCJkZWxldGUtcG9vbCIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtY3VzdG9tLXN0YXRlLTQ3MDEwMDU3MTU4NjU2NzE0MDAiLCJ1cGRhdGUtcmVzdWx0LXN0YXRlLWNzMSIsImNyZWF0ZS1zY2hlZHVsZS1zY2FuIiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1jczIiLCJ1cGRhdGUtcmVzdWx0LXN0YXRlLWN1c3RvbS1zdGF0ZS0yNDYwMTk0NDYyODc1NDI5ODI3IiwiZGVsZXRlLXByb2plY3QtaWYtaW4tZ3JvdXAiLCJ1cGRhdGUtc2NhLWxpY2Vuc2Utc3RhdGUiLCJkYXN0LWNhbmNlbC1zY2FuIiwidXBkYXRlLXNjYS1saWNlbnNlLXByb3BlcnRpZXMtaWYtaW4tZ3JvdXAiLCJkZWxldGUtcnVudGltZS1jbG91ZCIsImNyZWF0ZS1mZWVkYmFja2FwcCIsInVwZGF0ZS1yZXN1bHQtaWYtaW4tZ3JvdXAiLCJ2aWV3LXJlc3VsdHMiLCJkZWxldGUtZmVlZGJhY2thcHAiLCJydWlwX2NyZWF0ZV9wcmVzZXQiLCJ1cGRhdGUtcG9saWN5LW1hbmFnZW1lbnQiLCJ2aWV3LXJpc2stbWFuYWdlbWVudC10YWIiLCJkYXN0LXVwZGF0ZS1yZXN1bHRzIiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1jdXN0b20tc3RhdGUtODkxNDI3ODE3MzIwMTc3ODkzMSIsImFuYWx5dGljcy12dWxuZXJhYmlsaXR5LWRhc2hib2FyZC12aWV3IiwidXBkYXRlLXNjYS1saWNlbnNlLXByb3BlcnRpZXMiLCJ2aWV3LWFwcGxpY2F0aW9ucyIsInVwZGF0ZS1wYWNrYWdlLXN0YXRlLXNub296ZSIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtcHJvcG9zZS1ub3QtZXhwbG9pdGFibGUiLCJkYXN0LXVwZGF0ZS1yZXN1bHQtc3RhdGUtcHJvcG9zZS1ub3QtZXhwbG9pdGFibGUiLCJ1cGRhdGUtcmVzdWx0cyIsImRhc3QtZGVsZXRlLWVudmlyb25tZW50Iiwidmlldy1zY2Fucy1pZi1pbi1ncm91cCIsImRhc3QtaGlnaC1sZXZlbC11cGRhdGUtcmVzdWx0LXN0YXRlcyIsInZpZXctY29udHJpYnV0b3JzIiwidmlldy1wcm9qZWN0LXBhcmFtcy1pZi1pbi1ncm91cCIsImFzdC1hZG1pbiIsIm1hbmFnZS1hcHBsaWNhdGlvbiIsInVwZGF0ZS13ZWJob29rIiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1saWF2MiIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtY3MyMjIiLCJ1cGRhdGUtcmVzdWx0LXN0YXRlLWNzMTAxIiwiZGVsZXRlLXdlYmhvb2siLCJkZWxldGUtc2Nhbi1pZi1pbi1ncm91cCIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtY3MxMDAiLCJkYXN0LWFkbWluIiwiZGVsZXRlLXByb2plY3QiLCJ1cGRhdGUtbGlua3MiLCJkYXN0LXVwZGF0ZS1yZXN1bHQtc3RhdGUtbm90LWV4cGxvaXRhYmxlIiwidmlldy1saW5rcyIsIm1hbmFnZS1yZXBvcnRzIiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1jdXN0b20tc3RhdGUtNjYyMzM1NzIzMDA1MzY4NzQzMSIsInVwZGF0ZS1zY2EtbGljZW5zZS1zdGF0ZS1pZi1pbi1ncm91cCIsInVwZGF0ZS1ydW50aW1lLWNsb3VkIiwiZGFzdC11cGRhdGUtc2NhbiIsImRlbGV0ZS1zY2FuIiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1jdXN0b20tc3RhdGUtNTg2ODkxNDcxMjE4NzkxNzQ3OSIsImltcG9ydC1maW5kaW5ncy1leHRlcm5hbC1wbGF0Zm9ybXMiLCJ2aWV3LXByZXNldCIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtY3MxMjEiLCJ1cGRhdGUtcmVzdWx0LXN0YXRlcy1pZi1pbi1ncm91cCIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtbWljaGFsdGVzdCIsImFib3J0LWRhdGEtcmV0ZW50aW9uIiwidmlldy13ZWJob29rcyIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtbWljaGFsdGVzdDExMTEiLCJkYXN0LWFkZC1ub3RlcyIsInZpZXctcG9vbHMiLCJ1cGRhdGUtcmVzdWx0LXN0YXRlLWN1c3RvbS1zdGF0ZS0xMTc3ODEzMzE3NDAyNjgzNzAxIiwidXBkYXRlLWFwcGxpY2F0aW9uIiwiZGFzdC11cGRhdGUtcmVzdWx0LXNldmVyaXR5IiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1yb3RlbWRkIiwidmlldy1mZWVkYmFja2FwcCIsImNyZWF0ZS1wb2xpY3ktbWFuYWdlbWVudCIsInZpZXctY25hcyIsIm1hbmFnZS1saW5rcyIsInZpZXctdGVuYW50LXBhcmFtcyIsImNyZWF0ZS1zY2FuIiwidmlldy1wcm9qZWN0cyIsImRlbGV0ZS1yZXN1bHQtY3VzdG9tLXN0YXRlIiwidmlldy1zY2hlZHVsZS1zY2FucyIsInVwZGF0ZS1yZXN1bHQtYWxsLXN0YXRlcyIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtY3VzdG9tLXN0YXRlLTM1NDAwMTU0MDU2MzA3MTg4Iiwidmlldy1hdWRpdC10cmFpbCIsInZpZXctcmlzay1tYW5hZ2VtZW50IiwiY3JlYXRlLWxpbmtzIiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1jczIyIiwiYWRkLXBhY2thZ2UiLCJkZWxldGUtcXVlcnkiLCJ1cGRhdGUtZmVlZGJhY2thcHAiLCJ1cGRhdGUtcmVzdWx0LXN0YXRlLXByb3Bvc2Utbm90LWV4cGxvaXRhYmxlLWlmLWluLWdyb3VwIiwiY3JlYXRlLXBvb2wiLCJ1cGRhdGUtdGVuYW50LXBhcmFtcyIsImRvd25sb2FkLXNvdXJjZS1jb2RlIiwidXBkYXRlLXByb2plY3QtcGFyYW1zLWlmLWluLWdyb3VwIiwiZGFzdC12aWV3LWVudmlyb25tZW50cyIsInVwZGF0ZS1yZXN1bHQtbm90LWV4cGxvaXRhYmxlLWlmLWluLWdyb3VwIiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1mYWZlIiwidXBkYXRlLXBhY2thZ2Utc3RhdGUtbXV0ZSIsInVwZGF0ZS1wb29sIiwiYWNjZXNzLWlhbSIsImRlbGV0ZS1zY2hlZHVsZS1zY2FuIiwidmlldy1kYXRhLXJldGVudGlvbiIsInVwZGF0ZS1yaXNrLW1hbmFnZW1lbnQiLCJjcmVhdGUtcnVudGltZS1jbG91ZCIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtcm90ZW1kIiwidXBkYXRlLXJlc3VsdCIsInVwZGF0ZS1yZXN1bHQtc3RhdGVzIiwidXBkYXRlLXByZXNldCIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtY3VzdG9tLXN0YXRlLTYyMzczNDk3NDM1MzMwMzI0NTMiLCJkYXN0LWNyZWF0ZS1zY2FuIiwiZGVsZXRlLXBvbGljeS1tYW5hZ2VtZW50IiwibWFuYWdlLWNuYXMiLCJ1cGRhdGUtYWNjZXNzIiwidXBkYXRlLXBhY2thZ2Utc3RhdGUtc25vb3plLWlmLWluLWdyb3VwIiwibWFuYWdlLXBvbGljeS1tYW5hZ2VtZW50IiwiYXN0LXZpZXdlciIsIm1hbmFnZS1mZWVkYmFja2FwcCIsIm1hbmFnZS1kYXRhLXJldGVudGlvbiIsImRhc3QtdXBkYXRlLXJlc3VsdC1zdGF0ZXMiLCJkZWxldGUtcHJlc2V0IiwidXBkYXRlLXBhY2thZ2Utc3RhdGUtbXV0ZS1pZi1pbi1ncm91cCIsInVwZGF0ZS1sb2NrZWQtc2NhbnMiLCJ2aWV3LWxpY2Vuc2UiLCJjcmVhdGUtc2Nhbi1pZi1pbi1ncm91cCIsInVwZGF0ZS1yZXN1bHQtc3RhdGUtY3VzdG9tLXN0YXRlLTIzNDk3NzgwNzQ3Mzc0MjM5OTMiLCJvcmRlci1zZXJ2aWNlcyIsImFuYWx5dGljcy1yZXBvcnRzLWFkbWluIiwiZGFzdC11cGRhdGUtZW52aXJvbm1lbnQiLCJ1cGRhdGUtcmVzdWx0LXN0YXRlLWN1c3RvbS1zdGF0ZS0zIiwidmlldy1yZXN1bHRzLWlmLWluLWdyb3VwIiwidXBkYXRlLXJlc3VsdC1zdGF0ZS1yb3RlbWQtc3RhdGUiLCJtYW5hZ2UtYWNjZXNzIl0sImlzLXNlcnZpY2UtdXNlciI6IiIsIm5hbWUiOiJvcmcgYWRtaW4iLCJ0ZW5hbnQtdHlwZSI6IkludGVybmFsIiwiYXN0LWxpY2Vuc2UiOnsiSUQiOjE1ODE0LCJUZW5hbnRJRCI6Ijc1ZDRmZTM1LTk2NWEtNDUwNi1iMjI2LWUwMTU1ZWM4NGMzNCIsIklzQWN0aXZlIjp0cnVlLCJQYWNrYWdlSUQiOjIxOSwiTGljZW5zZURhdGEiOnsiYWN0aXZhdGlvbkRhdGUiOjE2MjYwMDEwMzEyMjgsImFsbG93ZWRFbmdpbmVzIjpbIktJQ1MiLCJFbnRlcnByaXNlIFNlY3JldHMiLCJDb2RlYmFzaGluZyIsIlNBU1QiLCJBUEkgU2VjdXJpdHkiLCJTQ0EiLCJBcHBsaWNhdGlvbiBSaXNrIE1hbmFnZW1lbnQiLCJDb250YWluZXJzIiwiREFTVCIsIlNDUyIsIkNsb3VkIEluc2lnaHRzIiwiTWFsaWNpb3VzIFBhY2thZ2VzIl0sImFwaVNlY3VyaXR5RW5hYmxlZCI6dHJ1ZSwiY29kZUJhc2hpbmdFbmFibGVkIjp0cnVlLCJjb2RlQmFzaGluZ1VybCI6Imh0dHBzOi8vY2hlY2ttYXJ4LmNvZGViYXNoaW5nLmNvbSIsImNvZGVCYXNoaW5nVXNlcnNDb3VudCI6MSwiY3VzdG9tTWF4Q29uY3VycmVudFNjYW5zRW5hYmxlZCI6dHJ1ZSwiZGFzdEVuYWJsZWQiOnRydWUsImV4cGlyYXRpb25EYXRlIjoxNzUyMTQ1MDMxMDAwLCJmZWF0dXJlcyI6WyJTU08iXSwibGFzdENvbW1lbnRMaW1pdCI6OTAsIm1heENvbmN1cnJlbnRTY2FucyI6MjAsIm1heFF1ZXVlZFNjYW5zIjoxMDAwLCJzY3NFbmFibGVkIjp0cnVlLCJzZXJ2aWNlVHlwZSI6IlN0YW5kYXJkIiwic2VydmljZXMiOlsiMCBBcHBzZWMgSGVscGRlc2sgQXNzaXN0YW5jZSIsIjAgT3B0aW1pemF0aW9uIFNlcnZpY2UgT3JkZXIiXSwidW5saW1pdGVkUHJvamVjdHMiOnRydWUsInVzZXJzQ291bnQiOjUwfSwiUGFja2FnZU5hbWUiOiJDeE9uZSBQcm9mZXNzaW9uYWwgTkcifSwic2VydmljZV91c2Vyc19lbmFibGVkIjp0cnVlLCJmYW1pbHlfbmFtZSI6ImFkbWluIiwidGVuYW50IjoiZGV2X3RlbmFudCIsImVtYWlsIjoiYWxleHJAY2hlY2ttYXJ4LmNvbSJ9.LBfuJQ5CIu07QwCrL_QOk5KUf0tZFInmyG9ffKncdpGFIJDoAckTWFnHerJ-MNYy8YW6xAnRl3F1P5v3hYY9amR23oc2VdbDns18818CrI-izibRmJPe_chpODzKN1KxSmmVhUOC6TGHthmQFS1jr5qF190LxERdOK-Q6G-8fDPpKRYk7fwYEW-ijJjukLnduulbfMZJBSoHh_7sOueSqQk9IHbu0uvOx_pgTg-EC2jgGTIizwBJXuUo38XCSIM-e_x_uJ9Tr1L59i11-q9mVuGHnfhApL928Wp1Ubny6uyUqCoLTkfR_dTO44qPrqNetwi2bHBLI8u93fJuIEi80w"
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

	expired := time.Since(CachedAccessTime) > time.Duration(tokenExpirySeconds-expiryGraceSeconds)*time.Second
	if !expired {
		logger.PrintIfVerbose("Using cached API access token!")
		return CachedAccessToken
	}
	logger.PrintIfVerbose("API access token not found in cache!")
	return ""
}

func writeCredentialsToCache(accessToken string) {
	credentialsMutex.Lock()
	defer credentialsMutex.Unlock()

	logger.PrintIfVerbose("Storing API access token to cache.")
	viper.Set(commonParams.AstToken, accessToken)
	CachedAccessToken = accessToken
	CachedAccessTime = time.Now()
}

func getNewToken(credentialsPayload, authServerURI string) (string, error) {
	payload := strings.NewReader(credentialsPayload)
	req, err := http.NewRequest(http.MethodPost, authServerURI, payload)
	setAgentNameAndOrigin(req)
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
