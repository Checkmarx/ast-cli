package wrappers

type HealthMockWrapper struct {
}

func (h *HealthMockWrapper) RunWebAppCheck() error {
	return nil
}
