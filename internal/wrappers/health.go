package wrappers

import (
	healthcheckApi "github.com/checkmarxDev/healthcheck/pkg/api/rest/v1"
)

type HealthCheckWrapper interface {
	RunMessageQueueCheck() (*HealthStatus, error)
	RunWebAppCheck() (*HealthStatus, error)
	RunKeycloakWebAppCheck() (*HealthStatus, error)
	RunDBCheck() (*HealthStatus, error)
	RunObjectStoreCheck() (*HealthStatus, error)
	RunInMemoryDBCheck() (*HealthStatus, error)
	RunLoggingCheck() (*HealthStatus, error)
	GetAstRole() (string, error)
}

type HealthStatus struct {
	*healthcheckApi.HealthcheckModel
}

func NewHealthStatus(success bool, errs ...string) *HealthStatus {
	return &HealthStatus{
		&healthcheckApi.HealthcheckModel{
			Success: success,
			Errors:  errs,
		},
	}
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
