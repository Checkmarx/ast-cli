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

type PdfReportsPayload struct {
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

type PdfReportsResponse struct {
	ReportID string `json:"reportId"`
}
type PdfPollingResponse struct {
	ReportID string `json:"reportId"`
	Status   string `json:"status"`
	URL      string `json:"url"`
}
type PdfHTTPWrapper struct {
	path string
}

const downloadTimeout = 420

func NewResultsPdfReportsHTTPWrapper(path string) ResultsPdfWrapper {
	return &PdfHTTPWrapper{
		path: path,
	}
}

func (r *PdfHTTPWrapper) GeneratePdfReport(payload *PdfReportsPayload) (*PdfReportsResponse, *WebError, error) {
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
		model := PdfReportsResponse{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to parse response body")
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (r *PdfHTTPWrapper) CheckPdfReportStatus(reportID string) (*PdfPollingResponse, *WebError, error) {
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
		model := PdfPollingResponse{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to parse response body")
		}
		return &model, nil, nil
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (r *PdfHTTPWrapper) DownloadPdfReport(reportID, targetFile string) error {
	clientTimeout := uint(downloadTimeout)
	customURL := fmt.Sprintf("%s/%s/download", r.path, reportID)
	resp, err := SendHTTPRequest(http.MethodGet, customURL, http.NoBody, true, clientTimeout)
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
