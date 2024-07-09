package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type ExportHTTPWrapper struct {
	ExportPath string
}

func NewExportHTTPWrapper(exportPath string) ExportWrapper {
	return &ExportHTTPWrapper{
		ExportPath: exportPath,
	}
}

func (e *ExportHTTPWrapper) InitiateExportRequest(scanID, format string) (string, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	payload := RequestPayload{
		ScanID:     scanID,
		FileFormat: format,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	resp, err := SendHTTPRequestWithJSONContentType(http.MethodPost, e.ExportPath, bytes.NewReader(body), true, clientTimeout)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var exportResp ExportResponse
	if err := json.NewDecoder(resp.Body).Decode(&exportResp); err != nil {
		return "", err
	}

	logger.PrintIfVerbose(fmt.Sprintf("Initiated export request. Response: %+v", exportResp))

	return exportResp.ExportID, nil
}

func (e *ExportHTTPWrapper) CheckExportStatus(exportID string) (string, error) {
	params := map[string]string{"exportId": exportID}
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	startTime := time.Now()

	var resp *http.Response
	var err error
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()

	for {
		if time.Since(startTime) >= time.Duration(clientTimeout)*time.Minute {
			return "", fmt.Errorf("timeout waiting for export status")
		}

		resp, err = SendHTTPRequestWithQueryParams(http.MethodGet, e.ExportPath, params, nil, clientTimeout)
		if err != nil {
			return "", err
		}

		var statusResp ExportStatusResponse
		if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
			return "", err
		}

		logger.Printf("Export status: %s", statusResp.ExportStatus)

		if statusResp.ExportStatus == "Completed" {
			return statusResp.FileURL, nil
		}

		if statusResp.ErrorMessage != "" {
			return "", fmt.Errorf("export error: %s", statusResp.ErrorMessage)
		}

		resp.Body.Close()
		time.Sleep(1 * time.Second)
	}
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

func (e *ExportHTTPWrapper) DownloadExportReport(reportURL, targetFile string) error {
	accessToken, err := GetAccessToken()
	if err != nil {
		return err
	}
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	resp, err := SendHTTPRequestByFullURL(http.MethodGet, reportURL, http.NoBody, true, clientTimeout, accessToken, true)
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

	logger.Printf("Downloaded file: %s - %d bytes\n", targetFile, size)
	return nil
}
