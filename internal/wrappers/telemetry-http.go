package wrappers

import (
	"bytes"
	"encoding/json"
	"net/http"

	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const agentHookTelemetryTimeoutSeconds = 1

type TelemetryHTTPWrapper struct {
	path       string
	agentHooks bool
}

func NewHTTPTelemetryAIWrapper(path string) *TelemetryHTTPWrapper {
	return &TelemetryHTTPWrapper{
		path: path,
	}
}

// ForAgentHooks returns a copy with a short timeout and no retries.
func (r *TelemetryHTTPWrapper) ForAgentHooks() *TelemetryHTTPWrapper {
	return &TelemetryHTTPWrapper{path: r.path, agentHooks: true}
}

func (r *TelemetryHTTPWrapper) SendAIDataToLog(data *DataForAITelemetry) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	path := r.path + "/log"
	var resp *http.Response
	if r.agentHooks {
		resp, err = sendHookTelemetry(path, jsonBytes)
	} else {
		clientTimeout := viper.GetUint(commonParams.ClientTimeoutKey)
		fn := func() (*http.Response, error) {
			return SendHTTPRequestWithJSONContentType(http.MethodPost, path, bytes.NewReader(jsonBytes), true, clientTimeout)
		}
		resp, err = retryHTTPRequest(fn, retryAttempts, retryDelay)
	}
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	if r.agentHooks && (resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices) {
		return errors.Errorf("failed to log hook telemetry, status code: %s", resp.Status)
	}
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusInternalServerError:
		return errors.Errorf("Failed to log the data, status code: %s", resp.Status)
	case http.StatusNotFound:
		return errors.Errorf("Telemetry AI endpoint not found")
	}
	return nil
}

// sendHookTelemetry uses the access token cached by the hook's license check.
// Refreshing a missing token or retrying a slow endpoint would delay the verdict.
func sendHookTelemetry(path string, payload []byte) (*http.Response, error) {
	token := getClientCredentialsFromCache(viper.GetInt(commonParams.TokenExpirySecondsKey))
	if token == "" {
		return nil, errors.New("no cached access token for hook telemetry")
	}
	url, err := GetURL(path, token)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	setAgentNameAndOrigin(req, false)
	req.Header.Set("Content-Type", jsonContentType)
	enrichWithOath2Credentials(req, token, bearerFormat)
	client := GetClient(agentHookTelemetryTimeoutSeconds)
	defer client.CloseIdleConnections()
	return client.Do(req)
}
