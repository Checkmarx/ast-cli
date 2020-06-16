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
	WebAppHealthcheckURL       string
	DBHealthcheckURL           string
	MessageQueueHealthcheckURL string
	ObjectStoreHealthcheckURL  string
	InMemoryDBHealthcheckURL   string
}

func parseHealthcheckResponse(body io.ReadCloser) (*HealthStatus, error) {
	status := &HealthStatus{}
	if err := json.NewDecoder(body).Decode(status); err != nil {
		return nil, errors.Wrapf(err, "Failed to parse healthcheck response")
	}

	return status, nil
}

func runHealthCheckRequest(url string,
	parser func(body io.ReadCloser) (*HealthStatus, error)) (*HealthStatus, error) {
	resp, err := SendHTTPRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Http request %v failed", url)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return &HealthStatus{
			&healthcheckApi.HealthcheckModel{
				Success: false,
				Message: fmt.Sprintf("Http request %v responded with status code %v and body %v",
					url, resp.StatusCode, func() string {
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

func NewHealthCheckHTTPWrapper(astWebAppURL, healthDBURL, healthcheckNatsURL,
	healthcheckMinioURL, healthCheckRedisURL string) HealthCheckWrapper {
	return &healthCheckHTTPWrapper{
		astWebAppURL,
		healthDBURL,
		healthcheckNatsURL,
		healthcheckMinioURL,
		healthCheckRedisURL,
	}
}

func (h *healthCheckHTTPWrapper) RunWebAppCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.WebAppHealthcheckURL, func(body io.ReadCloser) (*HealthStatus, error) {
		return &HealthStatus{
			&healthcheckApi.HealthcheckModel{
				Success: true,
				Message: "",
			},
		}, nil
	})
}

func (h *healthCheckHTTPWrapper) RunDBCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.DBHealthcheckURL, parseHealthcheckResponse)
}

func (h *healthCheckHTTPWrapper) RunMessageQueueCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.MessageQueueHealthcheckURL, parseHealthcheckResponse)
}

func (h *healthCheckHTTPWrapper) RunObjectStoreCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.ObjectStoreHealthcheckURL, parseHealthcheckResponse)
}

func (h *healthCheckHTTPWrapper) RunInMemoryDBCheck() (*HealthStatus, error) {
	return runHealthCheckRequest(h.InMemoryDBHealthcheckURL, parseHealthcheckResponse)
}
