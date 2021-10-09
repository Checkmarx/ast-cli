package wrappers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"
	"github.com/checkmarxDev/ast-cli/internal/wrappers/ntlm"

	"github.com/spf13/viper"
)

const (
	expiryGraceSeconds = 10
	NoTimeout          = 0
	ntlmProxyToken     = "ntlm"
	checkmarxURLError  = "Could not reach provided Checkmarx server"
	DebugFlag          = "debug"
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

func PrintIfVerbose(msg string) {
	if viper.GetBool(commonParams.DebugFlag) {
		if utf8.Valid([]byte(msg)) {
			log.Print(msg)
		} else {
			PrintIfVerbose("Request contains binary data and cannot be printed!")
		}
	}
}

func convertReqBodyToString(body io.Reader) (string, io.Reader) {
	var bodyStr string
	if body != nil {
		b, err := ioutil.ReadAll(body)
		if err != nil {
			panic(err)
		}
		body = bytes.NewBuffer(b)
		bodyStr = string(b)
	}
	return bodyStr, body
}

func setAgentName(req *http.Request) {
	agentStr := viper.GetString(commonParams.AgentNameKey) + "/" + commonParams.Version
	PrintIfVerbose("Using Agent Name: " + agentStr)
	fmt.Println(req)
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
		PrintIfVerbose("Creating HTTP Client with Proxy: " + proxyStr)
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
			Proxy:           http.ProxyURL(u),
		}
	} else {
		PrintIfVerbose("Creating HTTP Client.")
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
	PrintIfVerbose("Creating HTTP client using NTLM Proxy using: " + proxyStr)
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
	bodyStr, body := convertReqBodyToString(body)
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
	PrintIfVerbose("Sending API request to: " + fullURL)
	if len(bodyStr) > 0 {
		PrintIfVerbose(bodyStr)
	}
	req = addReqMonitor(req)
	var resp *http.Response
	resp, err = client.Do(req)
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
				log.Print("Starting connection: " + hostPort)
			},
			DNSStart: func(dnsInfo httptrace.DNSStartInfo) {
				log.Print("DNS looking up host information for: " + dnsInfo.Host)
			},
			DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
				log.Print(fmt.Sprintf("DNS found host address(s): %+v\n", dnsInfo.Addrs))
			},
			TLSHandshakeStart: func() {
				log.Print("Started TLS Handshake")
			},
			TLSHandshakeDone: func(c tls.ConnectionState, err error) {
				if err == nil {
					log.Print("Completed TLS handshake")
				} else {
					log.Print("Error completing TLS handshake")
				}
			},
			GotFirstResponseByte: func() {
				endTime := time.Now().UnixNano() / int64(time.Millisecond)
				diff := endTime - startTime
				log.Printf("Connected completed in: %d (ms)", diff)
			},
		}
		return req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	}
	return req
}

func SendHTTPRequestPasswordAuth(method, path string, body io.Reader, timeout uint,
	username, password, adminClientID, adminClientSecret string) (*http.Response, error) {
	bodyStr, body := convertReqBodyToString(body)
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
	PrintIfVerbose("Requesting Password Auth with Auth URL: " + u)
	if len(bodyStr) > 0 {
		PrintIfVerbose(bodyStr)
	}
	req = addReqMonitor(req)
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func GetURL(path string) string {
	cleanURL := strings.TrimSpace(viper.GetString(commonParams.BaseURIKey))
	cleanURL = strings.Trim(cleanURL, "/")
	return fmt.Sprintf("%s/%s", cleanURL, path)
}

func GetAuthURL(path string) string {
	if viper.GetString(commonParams.BaseAuthURIKey) != "" {
		cleanURL := strings.TrimSpace(viper.GetString(commonParams.BaseAuthURIKey))
		cleanURL = strings.Trim(cleanURL, "/")
		PrintIfVerbose("Auth URL is: " + cleanURL + path)
		return fmt.Sprintf("%s/%s", cleanURL, path)
	}
	PrintIfVerbose("Auth URL is: " + path)
	return GetURL(path)
}

func SendHTTPRequestWithQueryParams(method, path string, params map[string]string,
	body io.Reader, timeout uint) (*http.Response, error) {
	bodyStr, body := convertReqBodyToString(body)
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
	PrintIfVerbose("Sending API request to: " + u)
	if len(bodyStr) > 0 {
		PrintIfVerbose(bodyStr)
	}
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return resp, errors.Errorf("%s %s \n", checkmarxURLError, req.URL)
	}
	if resp.StatusCode == http.StatusForbidden {
		return resp, errors.Errorf("%s", "Provided credentials do not have permissions for this command")
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
		authURI = GetURL("/" + strings.TrimLeft(authURI, "/"))
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
		return err
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
	PrintIfVerbose("Fetching API access token.")
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
	PrintIfVerbose("Checking cache for API access token.")
	expired := time.Since(cachedAccessTime) > time.Duration(tokenExpirySeconds-expiryGraceSeconds)*time.Second
	if !expired {
		PrintIfVerbose("Using cached API access token!")
		return &cachedAccessToken
	}
	PrintIfVerbose("API access token not found in cache!")
	return nil
}

func writeCredentialsToCache(accessToken *string) {
	PrintIfVerbose("Storing API access token to cache.")
	cachedAccessToken = *accessToken
	cachedAccessTime = time.Now()
}

func sanitizeCredentials(credentialsPayload string) string {
	sanitized := credentialsPayload
	if strings.Contains(credentialsPayload, "grant_type=client_credentials") {
		strs := strings.Split(credentialsPayload, "client_secret")
		sanitized = strs[0] + "client_secret=XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
	}
	if strings.Contains(credentialsPayload, "grant_type=refresh_token") {
		sanitized = "client_secret=XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
	}
	return sanitized
}

func getNewToken(credentialsPayload, authServerURI string) (*string, error) {
	PrintIfVerbose("Getting new API access token from: " + authServerURI)
	PrintIfVerbose(sanitizeCredentials(credentialsPayload))
	payload := strings.NewReader(credentialsPayload)
	req, err := http.NewRequest(http.MethodPost, authServerURI, payload)
	setAgentName(req)
	if err != nil {
		return nil, err
	}
	req = addReqMonitor(req)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Errorf("%s %s", checkmarxURLError, GetAuthURL(""))
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

	PrintIfVerbose("Successfully retreived API token.")
	return &credentialsInfo.AccessToken, nil
}

func getCredentialsPayload(accessKeyID, accessKeySecret string) string {
	PrintIfVerbose("Using Client ID and secret credentials.")
	return fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s", accessKeyID, accessKeySecret)
}

func getAPIKeyPayload(astToken string) string {
	PrintIfVerbose("Using API key credentials.")
	return fmt.Sprintf("grant_type=refresh_token&client_id=ast-app&refresh_token=%s", astToken)
}

func getPasswordCredentialsPayload(username, password, adminClientID, adminClientSecret string) string {
	PrintIfVerbose("Using username and password credentials.")
	return fmt.Sprintf("scope=openid&grant_type=password&username=%s&password=%s"+
		"&client_id=%s&client_secret=%s", username, password, adminClientID, adminClientSecret)
}
