package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"

	"github.com/pkg/errors"
)

type ExportRequestPayload struct {
	ScanID           string           `json:"ScanID"`
	FileFormat       string           `json:"FileFormat"`
	ExportParameters ExportParameters `json:"ExportParameters,omitempty"`
}

type ExportParameters struct {
	HideDevAndTestDependencies bool `json:"hideDevAndTestDependencies"`
	ShowOnlyEffectiveLicenses  bool `json:"showOnlyEffectiveLicenses"`
	ExcludePackages            bool `json:"excludePackages"`
	ExcludeLicenses            bool `json:"excludeLicenses"`
	ExcludeVulnerabilities     bool `json:"excludeVulnerabilities"`
	ExcludePolicies            bool `json:"excludePolicies"`
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

const (
	retryInterval   = 5 * time.Second
	timeout         = 2 * time.Minute
	errorTimeoutMsg = "operation timed out: failed to retrieve export report"
)

func (e *ExportHTTPWrapper) InitiateExportRequest(payload *ExportRequestPayload) (*ExportResponse, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	params, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse request body")
	}
	path := fmt.Sprintf("%s/%s", e.path, "requests")

	endTime := time.Now().Add(timeout)
	retryCount := 0

	for {
		logger.PrintfIfVerbose("Sending export request, attempt %d", retryCount+1)
		resp, err := SendHTTPRequestWithJSONContentType(http.MethodPost, path, bytes.NewBuffer(params), true, clientTimeout)
		if err != nil {
			return nil, err
		}

		switch resp.StatusCode {
		case http.StatusAccepted:
			logger.PrintIfVerbose("Received 202 Accepted response from server.")
			model := ExportResponse{}
			decoder := json.NewDecoder(resp.Body)
			err = decoder.Decode(&model)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse response body")
			}
			_ = resp.Body.Close()
			return &model, nil
		case http.StatusBadRequest:
			if time.Now().After(endTime) {
				log.Printf("Timeout reached after %d attempts. Last response status code: %d", retryCount+1, resp.StatusCode)
				_ = resp.Body.Close()
				return nil, errors.Errorf("failed to initiate export request - response status code %d", resp.StatusCode)
			}
			retryCount++
			logger.PrintfIfVerbose("Received 400 Bad Request. Retrying in %v... (attempt %d/%d)", retryInterval, retryCount, maxRetries)
			time.Sleep(retryInterval)
		default:
			logger.PrintfIfVerbose("Received unexpected status code %d", resp.StatusCode)
			_ = resp.Body.Close()
			return nil, errors.Errorf("response status code %d", resp.StatusCode)
		}
	}
}

func (e *ExportHTTPWrapper) GetExportReportStatus(reportID string) (*ExportPollingResponse, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	path := fmt.Sprintf("%s/%s", e.path, "requests")
	params := map[string]string{"returnUrl": "true", "exportId": reportID}

	start := time.Now()

	for {
		if time.Since(start) > timeout {
			return nil, errors.New(errorTimeoutMsg)
		}

		resp, err := SendPrivateHTTPRequestWithQueryParams(http.MethodGet, path, params, nil, clientTimeout)
		if err != nil {
			time.Sleep(retryInterval)
			continue
		}

		decoder := json.NewDecoder(resp.Body)

		switch resp.StatusCode {
		case http.StatusOK:
			model := ExportPollingResponse{}
			if err = decoder.Decode(&model); err != nil {
				_ = resp.Body.Close()
				return nil, errors.Wrapf(err, "failed to parse response body")
			}
			return &model, nil
		case http.StatusNotFound:
			_ = resp.Body.Close()
			time.Sleep(retryInterval)
		default:
			_ = resp.Body.Close()
			return nil, errors.Errorf("response status code %d", resp.StatusCode)
		}
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

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return errors.Wrapf(err, "Failed to write file %s", targetFile)
	}

	log.Printf("Downloaded file: %s - %d bytes\n", targetFile, size)
	return nil
}

func (e *ExportHTTPWrapper) GetScaPackageCollectionExport(fileURL string) (*ScaPackageCollectionExport, error) {
	const bomPrefix = "\xef\xbb\xbf"

	accessToken, err := GetAccessToken()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get access token")
	}

	start := time.Now()
	var resp *http.Response

	for {
		if time.Since(start) > timeout {
			return nil, errors.New(errorTimeoutMsg)
		}

		resp, err = SendHTTPRequestByFullURL(http.MethodGet, fileURL, http.NoBody, true, viper.GetUint(commonParams.ClientTimeoutKey), accessToken, true)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		_ = resp.Body.Close()
		time.Sleep(retryInterval)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}
	body = bytes.TrimPrefix(body, []byte(bomPrefix))

	scaPackageCollection := &ScaPackageCollectionExport{}
	if err = json.Unmarshal(body, scaPackageCollection); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}

	logger.PrintIfVerbose("Retrieved SCA package collection export successfully")
	return scaPackageCollection, nil
}
