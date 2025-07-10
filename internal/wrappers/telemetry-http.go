package wrappers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

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

func (r *TelemetryHTTPWrapper) SendDataToLog(data DataForAITelemetry) error {
	clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Println("try to send data to Telemetry AI log")

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
		return errors.Errorf("Failed to scan packages, status code: %s", resp.Status)
	case http.StatusNotFound:
		return errors.Errorf("Telemetry AI endpoint not found")
	}
	return nil
}
