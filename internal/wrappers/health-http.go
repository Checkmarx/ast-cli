package wrappers

type HealthHTTPWrapper struct {
}

func NewHTTPHealthWrapper() HealthWrapper {
	return &HealthHTTPWrapper{}
}

func (h *HealthHTTPWrapper) RunWebAppCheck() error {
	return nil
}
