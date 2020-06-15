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

type HealthChecker struct {
	Name    string
	Checker func() (*HealthStatus, error)
	roles   map[string]bool
}

func (h *HealthChecker) AllowRoles(roles ...string) *HealthChecker {
	h.roles = make(map[string]bool, len(roles))
	for _, r := range roles {
		h.roles[r] = true
	}

	return h
}

func (h *HealthChecker) HasRole(role string) bool {
	return h.roles[role]
}

type HealthCheckWrapper interface {
	RunNatsCheck() (*HealthStatus, error)
	RunWebAppCheck() (*HealthStatus, error)
	RunDBCheck() (*HealthStatus, error)
	RunMinioCheck() (*HealthStatus, error)
	RunRedisCheck() (*HealthStatus, error)
}
