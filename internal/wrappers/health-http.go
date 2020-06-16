package wrappers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	healthcheckApi "github.com/checkmarxDev/healthcheck/api/rest/v1"

	errors "github.com/pkg/errors"
)

// TODO add healthcheck between XXhealthcheckURL
type healthCheckHTTPWrapper struct {
	webAppURL string
	dBURL     string
	natsURL   string
	minioURL  string
	redisURL  string
}

func runHealthCheck(healthcheckURL string) (*HealthStatus, error) {
	resp, err := SendHTTPRequest(http.MethodGet, healthcheckURL, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Http request %v failed", healthcheckURL)
	}

	defer resp.Body.Close()
	status := &HealthStatus{}
	err = json.NewDecoder(resp.Body).Decode(status)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to parse healthcheck response")
	}

	return status, nil
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
	resp, err := SendHTTPRequest(http.MethodGet, h.webAppURL, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Http request %v failed", h.webAppURL)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return &HealthStatus{
			&healthcheckApi.HealthcheckModel{
				Success: false,
				Message: fmt.Sprintf("Http request %v responded with status code %v and body %v",
					h.webAppURL, resp.StatusCode, func() string {
						if body != nil {
							return string(body)
						}

						return ""
					}()),
			},
		}, nil
	}

	return &HealthStatus{
		&healthcheckApi.HealthcheckModel{
			Success: true,
			Message: "",
		},
	}, nil
}

func (h *healthCheckHTTPWrapper) RunDBCheck() (*HealthStatus, error) {
	return runHealthCheck(h.dBURL)
}

func (h *healthCheckHTTPWrapper) RunNatsCheck() (*HealthStatus, error) {
	return runHealthCheck(h.natsURL)
}

func (h *healthCheckHTTPWrapper) RunMinioCheck() (*HealthStatus, error) {
	return runHealthCheck(h.minioURL)
}

func (h *healthCheckHTTPWrapper) RunRedisCheck() (*HealthStatus, error) {
	return runHealthCheck(h.redisURL)
}
