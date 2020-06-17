package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	uploads "github.com/checkmarxDev/uploads/api/rest/v1"
	"github.com/pkg/errors"
)

const (
	httpClientTimeout = 5
)

type UploadsHTTPWrapper struct {
	url string
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
	defer file.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	wd, _ := os.Getwd()
	fmt.Printf("Input file full path is  %s\n", filepath.Join(wd, sourcesFile))

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, errors.Errorf("Failed to read file %s: %s", sourcesFile, err.Error())
	}

	var req *http.Request
	req, err = http.NewRequest(http.MethodPut, *preSignedURL, bytes.NewReader(fileBytes))
	if err != nil {
		return nil, errors.Errorf("Requesting error model failed - %s", err.Error())
	}

	var client = &http.Client{
		Timeout: time.Second * time.Duration(httpClientTimeout),
	}
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return nil, errors.Errorf("Invoking HTTP request failed - %s", err.Error())
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return preSignedURL, nil
	default:
		return nil, errors.Errorf("Unknown response status code %d", resp.StatusCode)
	}
}

func (u *UploadsHTTPWrapper) getPresignedURLForUploading() (*string, error) {
	resp, err := SendHTTPRequest(http.MethodPost, u.url, nil, true)
	if err != nil {
		return nil, errors.Errorf("Invoking HTTP request to get pre-signed URL failed - %s", err.Error())
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusBadRequest:
		errorModel := uploads.ErrorModel{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, errors.Errorf("Parsing error model failed - %s", err.Error())
		}
		return nil, errors.Errorf("%d - %s", errorModel.Code, errorModel.Message)

	case http.StatusOK:
		model := uploads.UploadModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, errors.Errorf("Parsing upload model failed - %s", err.Error())
		}
		return &model.URL, nil

	default:
		return nil, errors.Errorf("Unknown response status code %d", resp.StatusCode)
	}
}

func NewUploadsHTTPWrapper(url string) UploadsWrapper {
	return &UploadsHTTPWrapper{
		url: url,
	}
}
