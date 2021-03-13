package wrappers

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
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
	NoTimeout             = 0
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

type credentialsCache map[uint64]*string

const failedToAuth = "Failed to authenticate - please provide an %s"

func getClient(timeout uint) *http.Client {
	insecure := viper.GetBool("insecure")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
		Proxy:           http.ProxyFromEnvironment,
	}
	return &http.Client{Transport: tr, Timeout: time.Duration(timeout) * time.Second}
}

func SendHTTPRequest(method, path string, body io.Reader, auth bool, timeout uint) (*http.Response, error) {
	u := GetURL(path)
	return SendHTTPRequestByFullURL(method, u, body, auth, timeout)
}

func SendHTTPRequestByFullURL(method, fullURL string, body io.Reader, auth bool, timeout uint) (*http.Response, error) {
	client := getClient(timeout)
	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, err
	}

	if auth {
		req, err = enrichWithOath2Credentials(req)
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
	client := getClient(timeout)
	u := GetURL(path)
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("content-type", "application/json")
	req, err = enrichWithPasswordCredentials(req, username, password, adminClientID, adminClientSecret)
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
	return fmt.Sprintf("%s/%s", viper.GetString(commonParams.BaseURIKey), path)
}

func GetAuthURL(path string) string {
	if viper.GetString(commonParams.BaseIAMURIKey) != "" {
		return fmt.Sprintf("%s/%s", viper.GetString(commonParams.BaseIAMURIKey), path)
	}
	return ""
}

func SendHTTPRequestWithQueryParams(method, path string, params map[string]string,
	body io.Reader, timeout uint) (*http.Response, error) {
	client := getClient(timeout)
	u := GetURL(path)
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	req, err = enrichWithOath2Credentials(req)
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

func getAuthURI() (string, error) {
	authPath := viper.GetString(commonParams.AstAuthenticationPathConfigKey)
	if authPath == "" {
		return "", errors.Errorf(fmt.Sprintf(failedToAuth, "authentication path"))
	}
	authURI := GetAuthURL(authPath)
	if authURI == "" {
		authURI = GetURL(authPath)
	}
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

func enrichWithOath2Credentials(request *http.Request) (*http.Request, error) {
	authURI, err := getAuthURI()
	if err != nil {
		return nil, err
	}

	accessKeyID := viper.GetString(commonParams.AccessKeyIDConfigKey)
	accessKeySecret := viper.GetString(commonParams.AccessKeySecretConfigKey)

	if accessKeyID == "" {
		return nil, errors.Errorf(fmt.Sprintf(failedToAuth, "access key ID"))
	} else if accessKeySecret == "" {
		return nil, errors.Errorf(fmt.Sprintf(failedToAuth, "access key secret"))
	}

	accessToken, err := getClientCredentials(accessKeyID, accessKeySecret, authURI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to authenticate")
	}

	request.Header.Add("Authorization", *accessToken)
	return request, nil
}

func enrichWithPasswordCredentials(request *http.Request, username, password,
	adminClientID, adminClientSecret string) (*http.Request, error) {
	authURI, err := getAuthURI()
	if err != nil {
		return nil, err
	}

	accessToken, err := getNewToken(getPasswordCredentialsPayload(username, password, adminClientID, adminClientSecret), authURI)
	if err != nil {
		return nil, errors.Wrap(errors.Wrap(err, "failed to get access token from auth server"),
			"failed to authenticate")
	}

	request.Header.Add("Authorization", "Bearer "+*accessToken)
	return request, nil
}

func getClientCredentials(accessKeyID, accessKeySecret, authURI string) (*string, error) {
	credentialsFilePath := viper.GetString(commonParams.CredentialsFilePathKey)
	tokenExpirySeconds := viper.GetInt(commonParams.TokenExpirySecondsKey)
	var accessToken *string
	var err error
	hash, errHash := hash(accessKeyID + accessKeySecret + authURI)
	if errHash != nil {
		fmt.Printf("failed to create hash of credentials %s\n", err)
	} else {
		accessToken, err = getClientCredentialsFromCache(credentialsFilePath, tokenExpirySeconds, hash)
		if err != nil {
			fmt.Printf("failed to get credentials from file %s %s\n", credentialsFilePath, err)
		}
	}

	if accessToken == nil {
		accessToken, err = getNewToken(getCredentialsPayload(accessKeyID, accessKeySecret), authURI)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get access token from auth server")
		}

		if errHash == nil {
			err = writeCredentialsToCache(credentialsFilePath, hash, accessToken)
			if err != nil {
				fmt.Printf("failed to write credentials to file %s %s\n", credentialsFilePath, err)
			}
		}
	}

	return accessToken, nil
}

// Try to load access token from file, if not expired
func getClientCredentialsFromCache(credentialsFilePath string, tokenExpirySeconds int, credentialsHash uint64) (*string, error) {
	if info, err := os.Stat(credentialsFilePath); err == nil {
		// Credentials file exists. Check for access token validity
		modifiedAt := info.ModTime()
		expired := time.Since(modifiedAt) > time.Duration(tokenExpirySeconds-expiryGraceSeconds)*time.Second
		if !expired {
			b, err := ioutil.ReadFile(credentialsFilePath)
			if err != nil {
				return nil, err
			}

			var credentials credentialsCache
			err = json.Unmarshal(b, &credentials)
			if err != nil {
				return nil, err
			}

			return credentials[credentialsHash], nil
		}
	}

	return nil, nil
}

func writeCredentialsToCache(credentialsFilePath string, credentialsHash uint64, accessToken *string) error {
	cred := credentialsCache{
		credentialsHash: accessToken,
	}
	accessTokenData, err := json.Marshal(cred)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(credentialsFilePath, accessTokenData, 0644)
}

func getNewToken(credentialsPayload, authServerURI string) (*string, error) {
	payload := strings.NewReader(credentialsPayload)
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

	return &credentialsInfo.AccessToken, nil
}

func getCredentialsPayload(accessKeyID, accessKeySecret string) string {
	return fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s", accessKeyID, accessKeySecret)
}

func getPasswordCredentialsPayload(username, password, adminClientID, adminClientSecret string) string {
	return fmt.Sprintf("scope=openid&grant_type=password&username=%s&password=%s"+
		"&client_id=%s&client_secret=%s", username, password, adminClientID, adminClientSecret)
}

func hash(s string) (uint64, error) {
	h := fnv.New64()
	_, err := h.Write([]byte(s))
	return h.Sum64(), err
}
