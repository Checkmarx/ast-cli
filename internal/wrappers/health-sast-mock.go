package wrappers

type MockSastHealthCheckWrapper struct {
}

func (h *MockSastHealthCheckWrapper) RunWebAppCheck() (*HealthStatus, error) {
	return &HealthStatus{Success: true, Message: ""}, nil
}
