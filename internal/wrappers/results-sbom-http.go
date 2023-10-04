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

type SbomReportsPayload struct {
	ScanID     string `json:"ScanId"`
	FileFormat string `json:"FileFormat"`
}

type SbomReportsResponse struct {
	ExportID string `json:"exportId"`
}

type SbomPollingResponse struct {
	ExportID     string `json:"exportId"`
	ExportStatus string `json:"exportStatus"`
	FileURL      string `json:"fileUrl"`
}
type SbomHTTPWrapper struct {
	path      string
	proxyPath string
}

func NewResultsSbomReportsHTTPWrapper(path, proxyPath string) ResultsSbomWrapper {
	return &SbomHTTPWrapper{
		path:      path,
		proxyPath: proxyPath,
	}
}

func (r *SbomHTTPWrapper) GenerateSbomReport(payload *SbomReportsPayload) (*SbomReportsResponse, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	params, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to parse request body")
	}
	path := fmt.Sprintf("%s/%s", r.path, "requests")
	resp, err := SendHTTPRequestWithJSONContentType(http.MethodPost, path, bytes.NewBuffer(params), true, clientTimeout)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusAccepted:
		model := SbomReportsResponse{}
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&model)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse response body")
		}
		return &model, nil
	case http.StatusBadRequest:
		return nil, errors.Errorf("SBOM report is currently in beta mode and not available for this tenant type")
	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (r *SbomHTTPWrapper) GenerateSbomReportWithProxy(payload *SbomReportsPayload, targetFile string) (bool, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	path := fmt.Sprintf("%s/%s/%s", r.proxyPath, payload.ScanID, "export")
	params := map[string]string{"format": payload.FileFormat}
	resp, err := SendPrivateHTTPRequestWithQueryParams(http.MethodGet, path, params, nil, clientTimeout)
	if err != nil {
		return true, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return true, errors.Errorf("response status code %d", resp.StatusCode)
	}

	file, err := os.Create(targetFile)
	if err != nil {
		return true, errors.Wrapf(err, "Failed to create file %s", targetFile)
	}
	defer file.Close()
	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return true, errors.Wrapf(err, "Failed to write file %s", targetFile)
	}

	log.Printf("Downloaded file: %s - %d bytes\n", targetFile, size)
	return true, nil
}

func (r *SbomHTTPWrapper) GetSbomReportStatus(reportID string) (*SbomPollingResponse, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	path := fmt.Sprintf("%s/%s", r.path, "requests")
	params := map[string]string{"returnUrl": "true", "exportId": reportID}
	resp, err := SendPrivateHTTPRequestWithQueryParams(http.MethodGet, path, params, nil, clientTimeout)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	decoder := json.NewDecoder(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		model := SbomPollingResponse{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse response body")
		}
		return &model, nil
	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (r *SbomHTTPWrapper) DownloadSbomReport(reportID, targetFile string) error {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	customURL := fmt.Sprintf("%s/%s/%s/%s", r.path, "requests", reportID, "download")
	resp, err := SendHTTPRequest(http.MethodGet, customURL, nil, true, clientTimeout)
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
	defer file.Close()
	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return errors.Wrapf(err, "Failed to write file %s", targetFile)
	}

	log.Printf("Downloaded file: %s - %d bytes\n", targetFile, size)
	return nil
}
