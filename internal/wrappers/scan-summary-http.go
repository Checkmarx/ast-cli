package wrappers

import (
	"encoding/json"
	"fmt"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"net/http"
)

const (
	failedToParseScanSummary = "Failed to parse scan-summary response"
)

type ScanSummaryHTTPWrapper struct {
	path string
}

func NewHTTPScanSummaryWrapper(path string) ScanSummaryWrapper {
	return &ScanSummaryHTTPWrapper{
		path: path,
	}
}
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
