package wrappers

import healthcheckApi "github.com/checkmarxDev/healthcheck/api/rest/v1"

type HealthCheckMockWrapper struct {
}

func (h *HealthCheckMockWrapper) RunWebAppCheck() (*HealthStatus, error) {
	return &HealthStatus{
		&healthcheckApi.HealthcheckModel{
			Success: true,
			Message: "",
		},
	}, nil
}

func (h *HealthCheckMockWrapper) RunDBCheck() (*HealthStatus, error) {
	return &HealthStatus{
		&healthcheckApi.HealthcheckModel{
			Success: true,
			Message: "",
		},
	}, nil
}
