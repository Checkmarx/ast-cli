package wrappers

import (
	"fmt"

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"

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

type HealthChecker struct {
	Name    string
	Checker func() (*HealthStatus, error)
}

type HealthCheckWrapper interface {
	RunWebAppCheck() (*HealthStatus, error)
	RunDBCheck() (*HealthStatus, error)
	RunSomeCheck() (*HealthStatus, error)
}

func NewHealthChecksByRole(h HealthCheckWrapper, role string) []*HealthChecker {
	healthChecks := []*HealthChecker{
		{"DB", h.RunDBCheck},
		{"Web App", h.RunWebAppCheck},
		{"Some Check", h.RunSomeCheck},
	}

	if role == commonParams.ScaAgent {
		return healthChecks[2:]
	}

	return healthChecks
}
