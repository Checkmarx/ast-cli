package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"

	"github.com/pkg/errors"
)

type ExportRequestPayload struct {
	ScanID     string `json:"ScanID"`
	FileFormat string `json:"FileFormat"`
}

type ExportResponse struct {
	ExportID string `json:"exportId"`
}

type ExportPollingResponse struct {
	ExportID     string `json:"exportId"`
	ExportStatus string `json:"exportStatus"`
	FileURL      string `json:"fileUrl"`
	ErrorMessage string `json:"errorMessage"`
}
type ExportHTTPWrapper struct {
	path string
}

func NewExportHTTPWrapper(path string) ExportWrapper {
	return &ExportHTTPWrapper{
		path: path,
	}
}

func (e *ExportHTTPWrapper) InitiateExportRequest(payload *ExportRequestPayload) (*ExportResponse, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	params, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to parse request body")
	}
	path := fmt.Sprintf("%s/%s", e.path, "requests")
	resp, err := SendHTTPRequestWithJSONContentType(http.MethodPost, path, bytes.NewBuffer(params), true, clientTimeout)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	switch resp.StatusCode {
	case http.StatusAccepted:
		model := ExportResponse{}
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

func (e *ExportHTTPWrapper) GetExportReportStatus(reportID string) (*ExportPollingResponse, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	path := fmt.Sprintf("%s/%s", e.path, "requests")
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
		model := ExportPollingResponse{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse response body")
		}
		return &model, nil
	default:
		return nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}

func (e *ExportHTTPWrapper) DownloadExportReport(reportID, targetFile string) error {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	customURL := fmt.Sprintf("%s/%s/%s/%s", e.path, "requests", reportID, "download")
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
	defer file.Close()
	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return errors.Wrapf(err, "Failed to write file %s", targetFile)
	}

	log.Printf("Downloaded file: %s - %d bytes\n", targetFile, size)
	return nil
}

func (e *ExportHTTPWrapper) GetScaPackageCollectionExport(fileURL string) (*ScaPackageCollectionExport, error) {
	accessToken, err := GetAccessToken()
	if err != nil {
		return nil, err
	}
	resp, err := SendHTTPRequestByFullURL(http.MethodGet, fileURL, http.NoBody, true, viper.GetUint(commonParams.ClientTimeoutKey), accessToken, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var scaPackageCollection ScaPackageCollectionExport
	if err := json.NewDecoder(resp.Body).Decode(&scaPackageCollection); err != nil {
		return nil, err
	}

	logger.PrintIfVerbose("Retrieved SCA package collection export successfully")

	return &scaPackageCollection, nil
}
