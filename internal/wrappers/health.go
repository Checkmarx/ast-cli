package wrappers

import (
	"fmt"

	healthcheckApi "github.com/checkmarxDev/healthcheck/api/rest/v1"
)

type HealthCheckWrapper interface {
	RunMessageQueueCheck() (*HealthStatus, error)
	RunWebAppCheck() (*HealthStatus, error)
	RunDBCheck() (*HealthStatus, error)
	RunObjectStoreCheck() (*HealthStatus, error)
	RunInMemoryDBCheck() (*HealthStatus, error)
}

type HealthStatus struct {
	*healthcheckApi.HealthcheckModel
}

func (h *HealthStatus) String() string {
	if h.Success {
		return "Success"
	}

	return fmt.Sprintf("Failure, due to %v", h.Message)
}

type HealthCheck struct {
	Name    string
	Handler func() (*HealthStatus, error)
	roles   map[string]bool
}

func NewHealthCheck(name string, handler func() (*HealthStatus, error), roles []string) *HealthCheck {
	h := &HealthCheck{name, handler, make(map[string]bool, len(roles))}
	for _, r := range roles {
		h.roles[r] = true
	}

	return h
}

func (h *HealthCheck) HasRole(role string) bool {
	return h.roles[role]
}
