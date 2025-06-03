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

type JSONReportsPayload struct {
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

type JSONReportsResponse struct {
	ReportID string `json:"reportId"`
}
type JSONPollingResponse struct {
	ReportID string `json:"reportId"`
	Status   string `json:"status"`
	URL      string `json:"url"`
}
type JSONHTTPWrapper struct {
	path string
}

const JSONDownloadTimeout = 420

func NewResultsJSONReportsHTTPWrapper(path string) ResultsJSONWrapper {
	return &JSONHTTPWrapper{
		path: path,
	}
}

func (r *JSONHTTPWrapper) GenerateJSONReport(payload *JSONReportsPayload) (*JSONReportsResponse, *WebError, error) {
	clientTimeOut := viper.GetUint(commonParams.ClientTimeoutKey)
	param, err1 := json.Marshal(payload)
	if err1 != nil {
		return nil, nil, errors.Wrapf(err1, "Failed to parse request body")
	}

	response, err1 := SendHTTPRequest(http.MethodPost, r.path, bytes.NewBuffer(param), true, clientTimeOut)

	if err1 != nil {
		return nil, nil, err1
	}

	defer func() {
		_ = response.Body.Close()
	}()

	decoderJSON := json.NewDecoder(response.Body)

	switch response.StatusCode {
	case http.StatusAccepted:
		modelJSON := JSONReportsResponse{}
		err1 = decoderJSON.Decode(&modelJSON)
		if err1 != nil {
			return nil, nil, errors.Wrapf(err1, "failed to parse response body")
		}
		return &modelJSON, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", response.StatusCode)
	}
}

func (r *JSONHTTPWrapper) CheckJSONReportStatus(reportID string) (*JSONPollingResponse, *WebError, error) {
	clientTimeOut := viper.GetUint(commonParams.ClientTimeoutKey)
	apiPath := fmt.Sprintf("%s/%s", r.path, reportID)
	param := map[string]string{"returnUrl": "true"}
	response, err1 := SendPrivateHTTPRequestWithQueryParams(http.MethodGet, apiPath, param, nil, clientTimeOut)
	if err1 != nil {
		return nil, nil, err1
	}

	defer func() {
		_ = response.Body.Close()
	}()

	decoderJSON := json.NewDecoder(response.Body)

	switch response.StatusCode {
	case http.StatusOK:
		modelJSON := JSONPollingResponse{}
		err1 = decoderJSON.Decode(&modelJSON)
		if err1 != nil {
			return nil, nil, errors.Wrapf(err1, "failed to parse response body")
		}
		return &modelJSON, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", response.StatusCode)
	}
}

func (r *JSONHTTPWrapper) DownloadJSONReport(url, targetedFile string, useAccessToken bool) error {
	clientTimeOut := uint(JSONDownloadTimeout)
	var response *http.Response
	var err1 error
	if useAccessToken {
		customURL := fmt.Sprintf("%s/%s/download", r.path, url)
		response, err1 = SendHTTPRequest(http.MethodGet, customURL, http.NoBody, true, clientTimeOut)
	} else {
		response, err1 = SendHTTPRequestByFullURL(http.MethodGet, url, http.NoBody, useAccessToken, clientTimeOut, "", true)
	}

	if err1 != nil {
		return err1
	}

	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return errors.Errorf("response status code %d", response.StatusCode)
	}

	file1, err1 := os.Create(targetedFile)
	if err1 != nil {
		return errors.Wrapf(err1, "Failed to create file %s", targetedFile)
	}
	fileSize, err1 := io.Copy(file1, response.Body)
	if err1 != nil {
		return errors.Wrapf(err1, "Failed to write file %s", targetedFile)
	}
	defer file1.Close()

	log.Printf("Downloaded file: %s - %d bytes\n", targetedFile, fileSize)
	return nil
}
