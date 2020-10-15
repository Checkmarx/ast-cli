package wrappers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

type AuthHTTPWrapper struct {
	path string
}

func NewAuthHTTPWrapper(path string) AuthWrapper {
	return &AuthHTTPWrapper{
		path: path,
	}
}

const failedToParseCreateClientResult = "failed to parse create client result"

func (a *AuthHTTPWrapper) CreateOauth2Client(client *Oath2Client, username, password,
	adminClientID, adminClientSecret string) (*ErrorMsg, error) {
	jsonBytes, err := json.Marshal(client)
	if err != nil {
		return nil, err
	}

	res, err := SendHTTPRequestPasswordAuth(http.MethodPost, a.path, bytes.NewBuffer(jsonBytes), DefaultTimeoutSeconds,
		username, password, adminClientID, adminClientSecret)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusBadRequest:
		decoder := json.NewDecoder(res.Body)
		errorMsg := ErrorMsg{}
		err = decoder.Decode(&errorMsg)
		if err != nil {
			return nil, errors.Wrap(err, failedToParseCreateClientResult)
		}

		return &errorMsg, nil
	case http.StatusOK:
		return nil, nil
	default:
		b, err := ioutil.ReadAll(res.Body)
		return nil, errors.Errorf("response status code %d body %s", res.StatusCode, func() string {
			if err != nil {
				return ""
			}

			return string(b)
		}())
	}
}
