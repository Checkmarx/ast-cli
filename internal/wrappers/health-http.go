package wrappers

type HealthCheckHTTPWrapper struct {
}

func NewHTTPHealthCheckWrapper() HealthCheckWrapper {
	return &HealthCheckHTTPWrapper{}
}

func (h *HealthCheckHTTPWrapper) RunWebAppCheck() error {
	return nil
}
