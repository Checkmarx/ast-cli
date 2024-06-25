package wrappers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type AuthHTTPWrapper struct {
	path string
}

func NewAuthHTTPWrapper() AuthWrapper {
	return &AuthHTTPWrapper{}
}

const failedToParseCreateClientResult = "failed to parse create client result"

func (a *AuthHTTPWrapper) SetPath(newPath string) {
	a.path = newPath
}

func (a *AuthHTTPWrapper) CreateOauth2Client(
	client *Oath2Client, username, password,
	adminClientID, adminClientSecret string,
) (*ErrorMsg, error) {
	clientTimeout := viper.GetUint(params.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(client)
	if err != nil {
		return nil, err
	}
	// Update the auth path, delayed to here because bind not ready in main.go
	createClientPath := viper.GetString(params.CreateOath2ClientPathKey)
	tenant := viper.GetString(params.TenantKey)
	createClientPath = strings.Replace(createClientPath, "organization", tenant, 1)
	a.SetPath(createClientPath)
	// send the request
	res, err := SendHTTPRequestPasswordAuth(http.MethodPost, bytes.NewBuffer(jsonBytes), clientTimeout, username, password, adminClientID, adminClientSecret)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = res.Body.Close()
	}()

	switch res.StatusCode {
	case http.StatusBadRequest:
		decoder := json.NewDecoder(res.Body)
		errorMsg := ErrorMsg{}
		DecodeErrorModel(decoder, &errorMsg)
		return &errorMsg, nil
	case http.StatusOK:
		return nil, nil
	case http.StatusForbidden:
		return nil, errors.Errorf("User does not have permission for roles %v", client.Roles)
	default:
		b, err := ioutil.ReadAll(res.Body)
		return nil, errors.Errorf(
			"response status code %d body %s", res.StatusCode, func() string {
				if err != nil {
					return err.Error()
				}
				return string(b)
			}(),
		)
	}
}

func (a *AuthHTTPWrapper) ValidateLogin() error {
	clientTimeout := viper.GetUint(params.ClientTimeoutKey)
	resp, err := SendHTTPRequestWithQueryParams(http.MethodGet, a.path, map[string]string{}, nil, clientTimeout)
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return errors.Errorf("failed authentication: %d", resp.StatusCode)
	}
}
