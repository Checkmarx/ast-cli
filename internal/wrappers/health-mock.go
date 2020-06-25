package wrappers

import (
	"github.com/checkmarxDev/ast-cli/internal/params"
	healthcheckApi "github.com/checkmarxDev/healthcheck/api/rest/v1"
)

func mockRun() (*HealthStatus, error) {
	return &HealthStatus{
		&healthcheckApi.HealthcheckModel{
			Success: true,
			Message: "",
		},
	}, nil
}

type HealthCheckMockWrapper struct {
}

func (h *HealthCheckMockWrapper) RunWebAppCheck() (*HealthStatus, error) {
	return mockRun()
}

func (h *HealthCheckMockWrapper) RunKeycloakWebAppCheck() (*HealthStatus, error) {
	return mockRun()
}

func (h *HealthCheckMockWrapper) RunDBCheck() (*HealthStatus, error) {
	return mockRun()
}

func (h *HealthCheckMockWrapper) RunMessageQueueCheck() (*HealthStatus, error) {
	return mockRun()
}

func (h *HealthCheckMockWrapper) RunObjectStoreCheck() (*HealthStatus, error) {
	return mockRun()
}

func (h *HealthCheckMockWrapper) RunInMemoryDBCheck() (*HealthStatus, error) {
	return mockRun()
}

func (h *HealthCheckMockWrapper) RunLoggingCheck() (*HealthStatus, error) {
	return mockRun()
}

func (h *HealthCheckMockWrapper) GetAstRole() (string, error) {
	return params.ScaAgent, nil
}
