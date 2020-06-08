package wrappers

type HealthCheckMockWrapper struct {
}

func (h *HealthCheckMockWrapper) RunWebAppCheck() (*HealthStatus, error) {
	return &HealthStatus{Success: true, Message: ""}, nil
}

func (h *HealthCheckMockWrapper) RunDBCheck() (*HealthStatus, error) {
	return &HealthStatus{Success: true, Message: ""}, nil
}
