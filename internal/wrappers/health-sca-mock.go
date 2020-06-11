package wrappers

type MockScaHealthCheckWrapper struct {
}

func (h *MockScaHealthCheckWrapper) Run() (*HealthStatus, error) {
	return &HealthStatus{Success: true, Message: ""}, nil
}
