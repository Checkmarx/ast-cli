package wrappers

import (
	"encoding/json"
	"fmt"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	failedToParseScanSummary = "Failed to parse scan-summary response"
)

// ScanSummaryHTTPWrapper is a struct that implements the ScanSummaryWrapper interface for retrieving scan summaries via HTTP requests.
type ScanSummaryHTTPWrapper struct {
	path string
}

// NewHTTPScanSummaryWrapper creates a new instance of ScanSummaryHTTPWrapper with the provided path.
func NewHTTPScanSummaryWrapper(path string) ScanSummaryWrapper {
	return &ScanSummaryHTTPWrapper{
		path: path,
	}
}
// GetScanSummaryByScanID retrieves the scan summary for a given scan ID by making an HTTP GET request to the configured path.
func (s *ScanSummaryHTTPWrapper) GetScanSummaryByScanID(scanID string) (*ScanSummariesModel, *WebError, error) {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	// Construct the path with query parameter
	path := fmt.Sprintf("%s?scan-ids=%s", s.path, scanID)
	resp, err := SendHTTPRequest(http.MethodGet, path, http.NoBody, true, clientTimeout)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	decoder := json.NewDecoder(resp.Body)
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		errorModel := WebError{}
		err = decoder.Decode(&errorModel)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseScanSummary)
		}
		return nil, &errorModel, nil
	case http.StatusOK:
		model := ScanSummariesModel{}
		err = decoder.Decode(&model)
		if err != nil {
			return nil, nil, errors.Wrapf(err, failedToParseScanSummary)
		}
		return &model, nil, nil
	case http.StatusNotFound:
		return nil, nil, errors.Errorf("scan summary not found for scan ID: %s", scanID)
	default:
		return nil, nil, errors.Errorf("response status code %d", resp.StatusCode)
	}
}
