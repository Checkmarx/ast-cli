package wrappers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

func getClientCredentials(authServerURI, accessKeyID, accessKeySecret string) (*ClientCredentialsInfo, error) {
	payload := strings.NewReader(getCredentialsPayload(accessKeyID, accessKeySecret))
	req, err := http.NewRequest("POST", authServerURI, payload)
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

func GetRequestWithCredentials(request *http.Request) (*http.Request, error) {
	authHost := viper.GetString("AST_AUTHENTICATION_URI")
	accessKeyID := viper.GetString("AST_ACCESS_KEY_ID")
	accessKeySecret := viper.GetString("AST_ACCESS_KEY_SECRET")

	credentialsInfo, err := getClientCredentials(authHost, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Authorization", credentialsInfo.AccessToken)
	return request, nil
}
