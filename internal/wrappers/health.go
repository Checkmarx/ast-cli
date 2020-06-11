package wrappers

import (
	"fmt"

	healthcheckApi "github.com/checkmarxDev/healthcheck/api/rest/v1"
)

type HealthStatus struct {
	*healthcheckApi.HealthcheckModel
}

func (h *HealthStatus) String() string {
	if h.Success {
		return "Success"
	}

	return fmt.Sprintf("Failure, due to %v", h.Message)
}

type HealthChecker func() (*HealthStatus, error)

type HealthCheckWrapper interface {
	RunWebAppCheck() (*HealthStatus, error)
	RunDBCheck() (*HealthStatus, error)
}
