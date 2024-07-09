package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
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

func (e *ExportHTTPWrapper) GetExportPackage(scanID string) (*ScaPackageCollectionExport, error) {
	exportID, err := e.initiateExportRequest(scanID)
	if err != nil {
		return nil, err
	}
	logger.PrintIfVerbose(fmt.Sprintf("Initiated export request for scanID %s. ExportID: %s", scanID, exportID))

	fileURL, err := e.checkExportStatus(exportID)
	if err != nil {
		return nil, err
	}
	logger.PrintIfVerbose(fmt.Sprintf("Checked export status. File URL: %s", fileURL))

	scaPackageCollection, err := e.getScaPackageCollectionExport(fileURL)
	if err != nil {
		return nil, err
	}
	logger.PrintIfVerbose("Retrieved SCA package collection from export service successfully")

	return scaPackageCollection, nil
}

func (e *ExportHTTPWrapper) initiateExportRequest(scanID string) (string, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	payload := RequestPayload{
		ScanID:     scanID,
		FileFormat: "ScanReportJson",
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

func (e *ExportHTTPWrapper) checkExportStatus(exportID string) (string, error) {
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
		if time.Since(startTime) >= 5*time.Second {
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

func (e *ExportHTTPWrapper) getScaPackageCollectionExport(fileURL string) (*ScaPackageCollectionExport, error) {
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
