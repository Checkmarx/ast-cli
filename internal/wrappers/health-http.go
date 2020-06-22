package wrappers

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	healthcheckApi "github.com/checkmarxDev/healthcheck/api/rest/v1"

	errors "github.com/pkg/errors"
)

type healthCheckHTTPWrapper struct {
	WebAppHealthcheckPath       string
	DBHealthcheckPath           string
	MessageQueueHealthcheckPath string
	ObjectStoreHealthcheckPath  string
	InMemoryDBHealthcheckPath   string
	LoggingHealthcheckPath      string
}

func parseHealthcheckResponse(body io.ReadCloser) (*HealthStatus, error) {
	status := &HealthStatus{}
	if err := json.NewDecoder(body).Decode(status); err != nil {
		return nil, errors.Wrapf(err, "Failed to parse healthcheck response")
	}

	return status, nil
}

func runHealthCheckRequest(path string,
	parser func(body io.ReadCloser) (*HealthStatus, error)) (*HealthStatus, error) {
	resp, err := SendHTTPRequest(http.MethodGet, path, nil, false)
	if err != nil {
		return nil, errors.Wrapf(err, "Http request %v failed", GetURL(path))
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return &HealthStatus{
			&healthcheckApi.HealthcheckModel{
				Success: false,
				Message: fmt.Sprintf("Http request %v responded with status code %v and body %v",
					resp.Request.URL, resp.StatusCode, func() string {
						if body != nil {
							return string(body)
						}

						return ""
					}()),
			},
		}, nil
	}

	return parser(resp.Body)
}

func NewHealthCheckHTTPWrapper(astWebAppPath, healthDBPath, healthcheckNatsPath,
	healthcheckMinioPath, healthCheckRedisPath, healthcheckLoggingPath string) HealthCheckWrapper {
	return &healthCheckHTTPWrapper{
		astWebAppPath,
		healthDBPath,
		healthcheckNatsPath,
		healthcheckMinioPath,
		healthCheckRedisPath,
		healthcheckLoggingPath,
	}
}

func (h *healthCheckHTTPWrapper) RunWebAppCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.WebAppHealthcheckPath, func(body io.ReadCloser) (*HealthStatus, error) {
		return &HealthStatus{
			&healthcheckApi.HealthcheckModel{
				Success: true,
				Message: "",
			},
		}, nil
	})
}

func (h *healthCheckHTTPWrapper) RunDBCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.DBHealthcheckPath, parseHealthcheckResponse)
}

func (h *healthCheckHTTPWrapper) RunMessageQueueCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.MessageQueueHealthcheckPath, parseHealthcheckResponse)
}

func (h *healthCheckHTTPWrapper) RunObjectStoreCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.ObjectStoreHealthcheckPath, parseHealthcheckResponse)
}

func (h *healthCheckHTTPWrapper) RunInMemoryDBCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.InMemoryDBHealthcheckPath, parseHealthcheckResponse)
}

func (h *healthCheckHTTPWrapper) RunLoggingCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.LoggingHealthcheckPath, parseHealthcheckResponse)
}
