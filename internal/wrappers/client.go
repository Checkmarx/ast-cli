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
	APIKeyDecodeErrorFormat = "Token decoding error: %s"
	tryPrintOffset          = 2
	retryLimitPrintOffset   = 1
	MissingURI              = "When using client-id and client-secret please provide base-uri or base-auth-uri"
	MissingTenant           = "Failed to authenticate - please provide tenant"
	jwtError                = "Error retrieving URL from jwt token"
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

func setAgentName(req *http.Request) {
	agentStr := viper.GetString(commonParams.AgentNameKey) + "/" + commonParams.Version
	req.Header.Set("User-Agent", agentStr)
}

func GetClient(timeout uint) *http.Client {
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
		enrichWithOath2Credentials(req, accessToken)
	}

	req = addReqMonitor(req)
	var resp *http.Response
	resp, err = request(client, req, bodyPrint)
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

func SendHTTPRequestPasswordAuth(method string, body io.Reader, timeout uint, username, password, adminClientID, adminClientSecret string) (*http.Response, error) {
	u, err := getAuthURI()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, u, body)
	client := GetClient(timeout)
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

func addTenantAuthURI(baseAuthURI string) (string, error) {
	authPath := BaseAuthURLPrefix
	tenant := viper.GetString(commonParams.TenantKey)

	if tenant == "" {
		return "", errors.Errorf(MissingTenant)
	}

	authPath = strings.Replace(authPath, "organization", strings.ToLower(tenant), 1)

	return fmt.Sprintf("%s/%s", strings.Trim(baseAuthURI, "/"), authPath), nil
}

func enrichWithOath2Credentials(request *http.Request, accessToken string) {
	request.Header.Add("Authorization", "Bearer "+accessToken)
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

func GetAccessToken() (string, error) {
	authURI, err := getAuthURI()
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

	request.Header.Add("Authorization", "Bearer "+accessToken)
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
	//accessToken = "eyJleHAiOjE2ODU5NTU0MDEsImlhdCI6MTY4NTk1MzYwMSwianRpIjoiNDU5ZjhkYTQtMWEyZS00NjJlLTk0NjgtMThmNTk1OTA1YjI0IiwiaXNzIjoiaHR0cHM6Ly9pYW0tZGV2LmRldi5jeGFzdC5uZXQvYXV0aC9yZWFsbXMvYXBpc2VjLXRlbmFudCIsImF1ZCI6WyJyZWFsbS1tYW5hZ2VtZW50IiwiYWNjb3VudCJdLCJzdWIiOiI0Y2QzZDg2Yi00NGNjLTQzMzYtOTNhYS05MTZkYjM2YjhkOTAiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhc3QtYXBwIiwic2Vzc2lvbl9zdGF0ZSI6IjcwNzYxZTU0LTc2ZGQtNGFmYS1iYTg1LWNlMWRkYzcxNjY2NyIsImFsbG93ZWQtb3JpZ2lucyI6WyIvKiIsImh0dHBzOi8vKiJdLCJyZXNvdXJjZV9hY2Nlc3MiOnsicmVhbG0tbWFuYWdlbWVudCI6eyJyb2xlcyI6WyJ2aWV3LXJlYWxtIiwidmlldy11c2VycyIsIm1hbmFnZS1rZXlzIiwidmlldy1jbGllbnRzIiwidmlldy1hdXRob3JpemF0aW9uIiwicXVlcnktY2xpZW50cyIsInF1ZXJ5LWdyb3VwcyIsInF1ZXJ5LXVzZXJzIl19LCJhY2NvdW50Ijp7InJvbGVzIjpbIm1hbmFnZS1hY2NvdW50IiwibWFuYWdlLWFjY291bnQtbGlua3MiLCJ2aWV3LXByb2ZpbGUiXX19LCJzY29wZSI6ImlhbS1hcGkgZ3JvdXBzIGFzdC1hcGkgcm9sZXMgZW1haWwgcHJvZmlsZSBvZmZsaW5lX2FjY2VzcyIsInNpZCI6IjcwNzYxZTU0LTc2ZGQtNGFmYS1iYTg1LWNlMWRkYzcxNjY2NyIsInRlbmFudF9pZCI6ImlhbS1kZXYuZGV2LmN4YXN0Lm5ldDo6NWI3OTJmOWMtNDI5MS00YjQ1LWE4NTktYTQ4MjMxNjUyNThiIiwidGVuYW50X25hbWUiOiJhcGlzZWMtdGVuYW50IiwiZW1haWxfdmVyaWZpZWQiOnRydWUsInJvbGVzIjpbImRlZmF1bHQtcm9sZXMtYXBpc2VjLXRlbmFudCIsIm9mZmxpbmVfYWNjZXNzIiwibWFuYWdlLWtleXMiLCJ1bWFfYXV0aG9yaXphdGlvbiIsInVzZXIiXSwiZXVsYS1hY2NlcHRlZCI6dHJ1ZSwiZ3JvdXBzIjpbIjAyZmM4MDZhLTdmNmItNGE4NC1iOGUxLWE1NDMyMTUwMGNjNCIsIjgyMmY1YWE3LWMzNTItNGNmMi1iMWU2LTY4OTQwM2I1Mzk4YSIsIjViYzRlYWE4LTMwYTItNDY5MC1iNjgyLTMyMjQ4NjZmYjZiMiJdLCJncm91cHNOYW1lcyI6WyJDaGVja21hcngvYXBpc2VjLWRldmVsb3BlcnMiLCJuZXctZ3JvdXAtdGVzdCIsIkNoZWNrbWFyeCJdLCJjYi11cmwiOiIiLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJhcGlzZWMtYXV0b21hdGlvbiIsImdpdmVuX25hbWUiOiJBcGlzZWMiLCJhc3QtYmFzZS11cmwiOiJodHRwczovL3BsYXlncm91bmQtbWljcm8tZW5naW5lcy5hcGlzZWMuY3hhc3QubmV0Iiwic2YtaWQiOiIwMDEzejAwMDAyTHpjdkFBQVIiLCJyb2xlc19hc3QiOlsiY3JlYXRlLXByb2plY3QiLCJ2aWV3LXByb2plY3RzIiwiZGVsZXRlLWFwcGxpY2F0aW9uIiwiZGVsZXRlLXdlYmhvb2siLCJkYXN0LWFkbWluIiwiZGFzdC1kZWxldGUtc2NhbiIsImNyZWF0ZS13ZWJob29rIiwiYXN0LXNjYW5uZXIiLCJ1cGRhdGUtc2NhbiIsImRlbGV0ZS1xdWVyeSIsInVwZGF0ZS1mZWVkYmFja2FwcCIsImRlbGV0ZS1wcm9qZWN0Iiwidmlldy1wcm9qZWN0LXBhcmFtcyIsImRlbGV0ZS1wb29sIiwiY3JlYXRlLXBvb2wiLCJvcGVuLWZlYXR1cmUtcmVxdWVzdCIsImFzdC1yaXNrLW1hbmFnZXIiLCJ1cGRhdGUtdGVuYW50LXBhcmFtcyIsInZpZXctcXVlcmllcyIsIm1hbmFnZS13ZWJob29rIiwiZGFzdC1jYW5jZWwtc2NhbiIsImNyZWF0ZS1mZWVkYmFja2FwcCIsInVwZGF0ZS1wb29sIiwiYWNjZXNzLWlhbSIsImNyZWF0ZS1hcHBsaWNhdGlvbiIsInF1ZXJpZXMtZWRpdG9yIiwidmlldy1yZXN1bHRzIiwidXBkYXRlLXF1ZXJ5IiwiZGVsZXRlLWZlZWRiYWNrYXBwIiwidXBkYXRlLXJlc3VsdC1ub3QtZXhwbG9pdGFibGUiLCJkYXN0LXVwZGF0ZS1zY2FuIiwidXBkYXRlLXByb2plY3QiLCJ2aWV3LWVuZ2luZXMiLCJkZWxldGUtc2NhbiIsImNyZWF0ZS1xdWVyeSIsInVwZGF0ZS1yZXN1bHQiLCJkYXN0LWNyZWF0ZS1lbnZpcm9ubWVudCIsInZpZXctcHJlc2V0IiwidXBkYXRlLXByZXNldCIsImRhc3QtdXBkYXRlLXJlc3VsdHMiLCJkYXN0LWNyZWF0ZS1zY2FuIiwidXBkYXRlLWFjY2VzcyIsInZpZXctd2ViaG9va3MiLCJvcGVuLXN1cHBvcnQtdGlja2V0IiwiYXN0LXZpZXdlciIsInZpZXctYWNjZXNzIiwibWFuYWdlLWZlZWRiYWNrYXBwIiwidmlldy1wb29scyIsImEiLCJkZWxldGUtcHJlc2V0IiwiaWdub3JlIHJpc2siLCJ2aWV3LWFwcGxpY2F0aW9ucyIsInVwZGF0ZS1hcHBsaWNhdGlvbiIsInZpZXctbGljZW5zZSIsInVwZGF0ZS1wcm9qZWN0LXBhcmFtcyIsImNyZWF0ZS1wcmVzZXQiLCJkYXN0LWV4dGVybmFsLXNjYW5zIiwidmlldy1mZWVkYmFja2FwcCIsIm9yZGVyLXNlcnZpY2VzIiwiZGFzdC11cGRhdGUtZW52aXJvbm1lbnQiLCJkYXN0LWRlbGV0ZS1lbnZpcm9ubWVudCIsInZpZXctdGVuYW50LXBhcmFtcyIsIm1hbmFnZS1wcm9qZWN0Iiwidmlldy1zY2FucyIsImFzdC1hZG1pbiIsImNyZWF0ZS1zY2FuIiwibWFuYWdlLWFjY2VzcyIsInVwZGF0ZS13ZWJob29rIiwibWFuYWdlLWFwcGxpY2F0aW9uIl0sIm5hbWUiOiJBcGlzZWMgQXV0b21hdGlvbiIsInRlbmFudC10eXBlIjoiQ3VzdG9tZXIiLCJhc3QtbGljZW5zZSI6eyJJRCI6MzkwNywiVGVuYW50SUQiOiI1Yjc5MmY5Yy00MjkxLTRiNDUtYTg1OS1hNDgyMzE2NTI1OGIiLCJJc0FjdGl2ZSI6dHJ1ZSwiUGFja2FnZUlEIjozNCwiTGljZW5zZURhdGEiOnsiYWN0aXZhdGlvbkRhdGUiOjE2MzgxODQ5OTEzNDEsImFsbG93ZWRFbmdpbmVzIjpbIlNBU1QiLCJTQ0EiLCJLSUNTIiwiQ29udGFpbmVycyIsIkZ1c2lvbiIsIkFQSSBTZWN1cml0eSJdLCJhcGlTZWN1cml0eUVuYWJsZWQiOnRydWUsImNvZGVCYXNoaW5nRW5hYmxlZCI6ZmFsc2UsImNvZGVCYXNoaW5nVXJsIjoiIiwiY29kZUJhc2hpbmdVc2Vyc0NvdW50IjoxMCwiY3VzdG9tTWF4Q29uY3VycmVudFNjYW5zRW5hYmxlZCI6ZmFsc2UsImRhc3RFbmFibGVkIjpmYWxzZSwiZXhwaXJhdGlvbkRhdGUiOjE3MzU2NDQxODAwMDAsImZlYXR1cmVzIjpbIlNTTyJdLCJtYXhDb25jdXJyZW50U2NhbnMiOjEsInNjc0VuYWJsZWQiOmZhbHNlLCJzZXJ2aWNlVHlwZSI6IlN0YW5kYXJkIiwic2VydmljZXMiOlsiMCBPcHRpbWl6YXRpb24gU2VydmljZSBPcmRlciIsIjAgQXBwc2VjIEhlbHBkZXNrIEFzc2lzdGFuY2UiXSwidW5saW1pdGVkUHJvamVjdHMiOnRydWUsInVzZXJzQ291bnQiOjUwfSwiUGFja2FnZU5hbWUiOiJQcm9mZXNzaW9uYWwifSwiZmFtaWx5X25hbWUiOiJBdXRvbWF0aW9uIiwidGVuYW50IjoiYXBpc2VjLXRlbmFudCIsImVtYWlsIjoibGV2dnZhQGdtYWlsLmNvbSIsImFsZyI6IkhTMjU2In0.e30.DTbf6HUBCbFAqaGphLn8P_WNZA1zEBoZ_7lZPnbWluw"
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
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	client := GetClient(clientTimeout)

	res, err := doPrivateRequest(client, req)
	if err != nil {
		authURL, _ := getAuthURI()
		return "", errors.Errorf("%s %s", checkmarxURLError, authURL)
	}
	if res.StatusCode == http.StatusBadRequest {
		return "", errors.Errorf("%v %s \n", res.StatusCode, "Provided credentials are invalid")
	}
	if res.StatusCode == http.StatusNotFound {
		return "", errors.Errorf("%v %s \n", res.StatusCode, "Provided Tenant Name is invalid")
	}
	if res.StatusCode == http.StatusUnauthorized {
		return "", errors.Errorf("%v %s \n", res.StatusCode, "Provided credentials are invalid")
	}

	body, _ := ioutil.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		credentialsErr := ClientCredentialsError{}
		err = json.Unmarshal(body, &credentialsErr)

		if err != nil {
			return "", err
		}

		return "", errors.Errorf("%v %s %s", res.StatusCode, credentialsErr.Error, credentialsErr.Description)
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
		if err != nil {
			logger.PrintIfVerbose(err.Error())
		}
		if resp != nil && err == nil {
			logger.PrintResponse(resp, responseBody)
			return resp, nil
		}
		logger.PrintIfVerbose(fmt.Sprintf("Request failed in attempt %d", try+tryPrintOffset))
		time.Sleep(time.Duration(retryWaitTimeSeconds) * time.Second)
	}
	return nil, err
}

func getAuthURI() (string, error) {
	var authURI string
	var err error
	override := viper.GetBool(commonParams.ApikeyOverrideFlag)

	apiKey := viper.GetString(commonParams.AstAPIKey)
	if len(apiKey) > 0 {
		logger.PrintIfVerbose("Base Auth URI - Extract from API KEY")
		authURI, err = extractFromTokenClaims(apiKey, audienceClaimKey)
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
		cleanURL, err = extractFromTokenClaims(accessToken, baseURLKey)
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

func extractFromTokenClaims(accessToken, claim string) (string, error) {
	var value string
	token, _, err := new(jwt.Parser).ParseUnverified(accessToken, jwt.MapClaims{})
	if err != nil {
		return "", errors.Errorf(APIKeyDecodeErrorFormat, err)
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && claims[claim] != nil {
		value = strings.TrimSpace(claims[claim].(string))
	} else {
		return "", errors.Errorf(jwtError)
	}
	return value, nil
}
