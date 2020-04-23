package wrappers

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/spf13/viper"
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

func SendHTTPRequestWithLimitAndOffset(method, url string, limit, offset uint64, body io.Reader) (*http.Response, error) {
	client := getClient()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	if limit > 0 {
		q.Add(limitQueryParam, strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		q.Add(offsetQueryParam, strconv.FormatUint(offset, 10))
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
	authHost := viper.GetString("ast_authentication_uri")
	accessKeyID := viper.GetString("ast_access_key_id")
	accessKeySecret := viper.GetString("ast_access_key_secret")

	credentialsInfo, err := getClientCredentials(authHost, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Authorization", credentialsInfo.AccessToken)
	return request, nil
}

func getClientCredentials(authServerURI, accessKeyID, accessKeySecret string) (*ClientCredentialsInfo, error) {
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
	info := ClientCredentialsInfo{}
	err = json.Unmarshal(body, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func getCredentialsPayload(accessKeyID, accessKeySecret string) string {
	return fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s", accessKeyID, accessKeySecret)
}
