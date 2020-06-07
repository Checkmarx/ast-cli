package wrappers

type HealthCheckMockWrapper struct {
}

func (h *HealthCheckMockWrapper) RunWebAppCheck() error {
	return nil
}
