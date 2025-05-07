package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"

	"github.com/pkg/errors"
)

type JsonReportsPayload struct {
	ReportName string `json:"reportName" validate:"required"`
	ReportType string `json:"reportType" validate:"required"`
	FileFormat string `json:"fileFormat" validate:"required"`
	Data       struct {
		ScanID     string   `json:"scanId" validate:"required"`
		ProjectID  string   `json:"projectId" validate:"required"`
		BranchName string   `json:"branchName" validate:"required"`
		Host       string   `json:"host"`
		Sections   []string `json:"sections"`
		Scanners   []string `json:"scanners"`
		Email      []string `json:"email"`
	} `json:"data"`
}

type JsonReportsResponse struct {
	ReportID string `json:"reportId"`
}
type JsonPollingResponse struct {
	ReportID string `json:"reportId"`
	Status   string `json:"status"`
	URL      string `json:"url"`
}
type JsonHTTPWrapper struct {
	path string
}

const JsonDownloadTimeout = 420

func NewResultsJsonReportsHTTPWrapper(path string) ResultsJsonWrapper {
	return &JsonHTTPWrapper{
		path: path,
	}
}

func (r *JsonHTTPWrapper) GenerateJsonReport(payload *JsonReportsPayload) (*JsonReportsResponse, *WebError, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	params, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Failed to parse request body")
	}

	resp, err := SendHTTPRequest(http.MethodPost, r.path, bytes.NewBuffer(params), true, clientTimeout)

	if err != nil {
		return nil, nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusAccepted:
		model := JsonReportsResponse{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to parse response body")
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (r *JsonHTTPWrapper) CheckJsonReportStatus(reportID string) (*JsonPollingResponse, *WebError, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	path := fmt.Sprintf("%s/%s", r.path, reportID)
	params := map[string]string{"returnUrl": "true"}
	resp, err := SendPrivateHTTPRequestWithQueryParams(http.MethodGet, path, params, nil, clientTimeout)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		model := JsonPollingResponse{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to parse response body")
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (r *JsonHTTPWrapper) DownloadJsonReport(url, targetFile string, useAccessToken bool) error {
	clientTimeout := uint(JsonDownloadTimeout)
	var resp *http.Response
	var err error
	if useAccessToken {
		customURL := fmt.Sprintf("%s/%s/download", r.path, url)
		resp, err = SendHTTPRequest(http.MethodGet, customURL, http.NoBody, true, clientTimeout)
	} else {
		resp, err = SendHTTPRequestByFullURL(http.MethodGet, url, http.NoBody, useAccessToken, clientTimeout, "", true)
	}

	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("response status code %d", resp.StatusCode)
	}

	file, err := os.Create(targetFile)
	if err != nil {
		return errors.Wrapf(err, "Failed to create file %s", targetFile)
	}
	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return errors.Wrapf(err, "Failed to write file %s", targetFile)
	}
	defer file.Close()

	log.Printf("Downloaded file: %s - %d bytes\n", targetFile, size)
	return nil
}
