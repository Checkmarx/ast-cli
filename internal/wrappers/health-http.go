package wrappers

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	errors "github.com/pkg/errors"
)

type healthCheckHTTPWrapper struct {
	WebAppHealthCheckPath       string
	KeycloakHealthCheckPath     string
	DBHealthcheckPath           string
	MessageQueueHealthcheckPath string
	ObjectStoreHealthcheckPath  string
	InMemoryDBHealthcheckPath   string
	LoggingHealthcheckPath      string
	GetAstRolePath              string
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
		return NewHealthStatus(
			false,
			fmt.Sprintf("Http request %v responded with status code %v and body %v",
				resp.Request.URL, resp.StatusCode, func() string {
					if err != nil {
						return ""
					}

					return string(body)
				}()),
		), nil
	}

	return parser(resp.Body)
}

func NewHealthCheckHTTPWrapper(astWebAppPath, astKeycloakWebAppPath, healthDBPath, healthcheckNatsPath,
	healthcheckMinioPath, healthCheckRedisPath, healthcheckLoggingPath, getAstRolePath string) HealthCheckWrapper {
	return &healthCheckHTTPWrapper{
		astWebAppPath,
		astKeycloakWebAppPath,
		healthDBPath,
		healthcheckNatsPath,
		healthcheckMinioPath,
		healthCheckRedisPath,
		healthcheckLoggingPath,
		getAstRolePath,
	}
}

func (h *healthCheckHTTPWrapper) RunWebAppCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.WebAppHealthCheckPath, func(body io.ReadCloser) (*HealthStatus, error) {
		return NewHealthStatus(true), nil
	})
}

func (h *healthCheckHTTPWrapper) RunKeycloakWebAppCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.KeycloakHealthCheckPath, func(body io.ReadCloser) (*HealthStatus, error) {
		return NewHealthStatus(true), nil
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

func (h *healthCheckHTTPWrapper) GetAstRole() (string, error) {
	resp, err := SendHTTPRequest(http.MethodGet, h.GetAstRolePath, nil, false)
	if err != nil {
		return "", errors.Wrapf(err, "Http request %v failed", GetURL(h.GetAstRolePath))
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("Http request %v responded with status code %v and body %v",
			resp.Request.URL, resp.StatusCode, func() string {
				if err != nil {
					return ""
				}

				return string(body)
			}())
	}

	if err != nil {
		return "", errors.Wrapf(err, "Cannot read response body")
	}

	return string(body), nil
}
