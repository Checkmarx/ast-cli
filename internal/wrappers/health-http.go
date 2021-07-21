package wrappers

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

type healthCheckHTTPWrapper struct {
	WebAppHealthcheckPath       string
	KeycloakHealthcheckPath     string
	DBHealthcheckPath           string
	MessageQueueHealthcheckPath string
	ObjectStoreHealthcheckPath  string
	InMemoryDBHealthcheckPath   string
	LoggingHealthcheckPath      string
	ScanFlowHealthcheckPath     string
	SastEnginesHealthcheckPath  string
}

const scanFlowTimeoutSecs uint = 110

func parseHealthcheckResponse(body io.ReadCloser) (*HealthStatus, error) {
	status := &HealthStatus{}
	if err := json.NewDecoder(body).Decode(status); err != nil {
		return nil, errors.Wrapf(err, "Failed to parse healthcheck response")
	}

	return status, nil
}

func runHealthCheckRequest(path string, timeout uint, parser func(body io.ReadCloser) (*HealthStatus, error),
) (*HealthStatus, error) {
	resp, err := SendHTTPRequest(http.MethodGet, path, nil, false, timeout)
	if err != nil {
		return nil, errors.Wrapf(err, "Http request %v failed", GetURL(path))
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		return nil, errors.Errorf("Http request %v responded with status code %v and body %v",
			resp.Request.URL, resp.StatusCode, func() string {
				if err != nil {
					return ""
				}

				return string(body)
			}())
	}

	return parser(resp.Body)
}

func NewHealthCheckHTTPWrapper(
	astWebAppPath,
	astKeycloakWebAppPath,
	healthDBPath,
	healthcheckNatsPath,
	healthcheckMinioPath,
	healthCheckRedisPath,
	healthcheckLoggingPath,
	healthcheckScanFlowPath,
	healthcheckSastEnginesPath string,
) HealthCheckWrapper {
	return &healthCheckHTTPWrapper{
		astWebAppPath,
		astKeycloakWebAppPath,
		healthDBPath,
		healthcheckNatsPath,
		healthcheckMinioPath,
		healthCheckRedisPath,
		healthcheckLoggingPath,
		healthcheckScanFlowPath,
		healthcheckSastEnginesPath,
	}
}

func (h *healthCheckHTTPWrapper) RunWebAppCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.WebAppHealthcheckPath, DefaultTimeoutSeconds, func(body io.ReadCloser) (*HealthStatus, error) {
		return NewHealthStatus("Got OK", true), nil
	})
}

func (h *healthCheckHTTPWrapper) RunKeycloakWebAppCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.KeycloakHealthcheckPath, DefaultTimeoutSeconds, func(body io.ReadCloser) (*HealthStatus, error) {
		return NewHealthStatus("Got OK", true), nil
	})
}

func (h *healthCheckHTTPWrapper) RunDBCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.DBHealthcheckPath, DefaultTimeoutSeconds, parseHealthcheckResponse)
}

func (h *healthCheckHTTPWrapper) RunMessageQueueCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.MessageQueueHealthcheckPath, DefaultTimeoutSeconds, parseHealthcheckResponse)
}

func (h *healthCheckHTTPWrapper) RunObjectStoreCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.ObjectStoreHealthcheckPath, DefaultTimeoutSeconds, parseHealthcheckResponse)
}

func (h *healthCheckHTTPWrapper) RunInMemoryDBCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.InMemoryDBHealthcheckPath, DefaultTimeoutSeconds, parseHealthcheckResponse)
}

func (h *healthCheckHTTPWrapper) RunLoggingCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.LoggingHealthcheckPath, DefaultTimeoutSeconds, parseHealthcheckResponse)
}

func (h *healthCheckHTTPWrapper) RunScanFlowCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.ScanFlowHealthcheckPath, scanFlowTimeoutSecs, parseHealthcheckResponse)
}

func (h *healthCheckHTTPWrapper) RunSastEnginesCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.SastEnginesHealthcheckPath, DefaultTimeoutSeconds, parseHealthcheckResponse)
}
