package wrappers

type HealthCheckMockWrapper struct {
}

func (h *HealthCheckMockWrapper) RunWebAppCheck() (*HealthStatus, error) {
	return &HealthStatus{Success: true, Message: ""}, nil
}
