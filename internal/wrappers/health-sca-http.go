package wrappers

type HTTPScaHealthCheckWrapper struct {
}

func (h *HTTPScaHealthCheckWrapper) Run() (*HealthStatus, error) {
	return &HealthStatus{Message: "", Success: true}, nil
}
