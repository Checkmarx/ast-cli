package wrappers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	errors "github.com/pkg/errors"
)

type HealthCheckHTTPWrapper struct {
	webAppURL string
	DBURL     string
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
		return nil, errors.Wrapf(err, "Failed to parse helthcheck response")
	}

	return status, nil
}

func NewHTTPHealthCheckWrapper(astWebAppURL, healthDBUrl string) HealthCheckWrapper {
	return &HealthCheckHTTPWrapper{webAppURL: astWebAppURL, DBURL: healthDBUrl}
}

func (h *HealthCheckHTTPWrapper) RunWebAppCheck() (*HealthStatus, error) {
	resp, err := SendHTTPRequest(http.MethodGet, h.webAppURL, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Http request %v failed", h.webAppURL)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return &HealthStatus{
			Success: false,
			Message: fmt.Sprintf("Http request %v responded with status code %v and body %v",
				h.webAppURL, resp.StatusCode, func() string {
					if body != nil {
						return string(body)
					}

					return ""
				}()),
		}, nil
	}

	return &HealthStatus{Success: true, Message: ""}, nil
}

func (h *HealthCheckHTTPWrapper) RunDBCheck() (*HealthStatus, error) {
	return runHealthCheck(h.DBURL)
}
