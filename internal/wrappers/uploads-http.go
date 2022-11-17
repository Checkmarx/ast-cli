package wrappers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type UploadModel struct {
	URL string `json:"url"`
}

type UploadsHTTPWrapper struct {
	path string
}

func (u *UploadsHTTPWrapper) UploadFile(sourcesFile string) (*string, error) {
	preSignedURL, err := u.getPresignedURLForUploading()
	if err != nil {
		return nil, errors.Errorf("Failed creating pre-signed URL - %s", err.Error())
	}

	file, err := os.Open(sourcesFile)
	if err != nil {
		return nil, errors.Errorf("Failed to open file %s: %s", sourcesFile, err.Error())
	}
	// Close the file later
	defer func() {
		_ = file.Close()
	}()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, errors.Errorf("Failed to read file %s: %s", sourcesFile, err.Error())
	}
	accessToken, err := GetAccessToken()
	if err != nil {
		return nil, err
	}
	resp, err := SendHTTPRequestByFullURL(http.MethodPut, *preSignedURL, bytes.NewReader(fileBytes), true, NoTimeout, accessToken)
	if err != nil {
		return nil, errors.Errorf("Invoking HTTP request to upload file failed - %s", err.Error())
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		return preSignedURL, nil
	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (u *UploadsHTTPWrapper) getPresignedURLForUploading() (*string, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	resp, err := SendHTTPRequest(http.MethodPost, u.path, nil, true, clientTimeout)
	if err != nil {
		return nil, errors.Errorf("invoking HTTP request to get pre-signed URL failed - %s", err.Error())
	}

	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest:
		errorModel := ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, errors.Errorf("Parsing error model failed - %s", err.Error())
		}
		return nil, errors.Errorf("%d - %s", errorModel.Code, errorModel.Message)

	case http.StatusOK:
		model := UploadModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, errors.Errorf("Parsing upload model failed - %s", err.Error())
		}
		return &model.URL, nil

	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func NewUploadsHTTPWrapper(path string) UploadsWrapper {
	return &UploadsHTTPWrapper{
		path: path,
	}
}
