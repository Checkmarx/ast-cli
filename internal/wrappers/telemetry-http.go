package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type TelemetryHTTPWrapper struct {
	path string
}

func NewHTTPTelemetryAIWrapper(path string) *TelemetryHTTPWrapper {
	return &TelemetryHTTPWrapper{
		path: path,
	}
}

func (r *TelemetryHTTPWrapper) SendAIDataToLog(data *DataForAITelemetry) error {
	if data == nil {
		return errors.New("telemetry data is nil")
	}
	logger.PrintfIfVerbose("Telemetry githubUrl=%q engine=%s scanType=%s type=%s subType=%s totalCount=%d",
		data.GithubURL, data.Engine, data.ScanType, data.Type, data.SubType, data.TotalCount)
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	fn := func() (*http.Response, error) {
		return SendHTTPRequest(http.MethodPost, fmt.Sprint(r.path, "/log"), bytes.NewBuffer(jsonBytes), true, clientTimeout)
	}
	resp, err := retryHTTPRequest(fn, retryAttempts, retryDelay)
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		return errors.Errorf("Failed to log the data, status code: %s", resp.Status)
	case http.StatusNotFound:
		return errors.Errorf("Telemetry AI endpoint not found")
	}
	return nil
}
