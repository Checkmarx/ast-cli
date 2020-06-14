package wrappers

import healthcheckApi "github.com/checkmarxDev/healthcheck/api/rest/v1"

func run() (*HealthStatus, error) {
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
	return run()
}

func (h *HealthCheckMockWrapper) RunDBCheck() (*HealthStatus, error) {
	return run()
}

func (h *HealthCheckMockWrapper) RunSomeCheck() (*HealthStatus, error) {
	return run()
}
