package wrappers

func mockRun() (*HealthStatus, error) {
	return NewHealthStatus("mock", true), nil
}

type HealthCheckMockWrapper struct {
}

func (h *HealthCheckMockWrapper) RunWebAppCheck() (*HealthStatus, error) {
	return mockRun()
}

func (h *HealthCheckMockWrapper) RunKeycloakWebAppCheck() (*HealthStatus, error) {
	return mockRun()
}

func (h *HealthCheckMockWrapper) RunDBCheck() (*HealthStatus, error) {
	return mockRun()
}

func (h *HealthCheckMockWrapper) RunMessageQueueCheck() (*HealthStatus, error) {
	return mockRun()
}

func (h *HealthCheckMockWrapper) RunObjectStoreCheck() (*HealthStatus, error) {
	return mockRun()
}

func (h *HealthCheckMockWrapper) RunInMemoryDBCheck() (*HealthStatus, error) {
	return mockRun()
}

func (h *HealthCheckMockWrapper) RunLoggingCheck() (*HealthStatus, error) {
	return mockRun()
}

func (h *HealthCheckMockWrapper) RunScanFlowCheck() (*HealthStatus, error) {
	return mockRun()
}

func (h *HealthCheckMockWrapper) RunSastEnginesCheck() (*HealthStatus, error) {
	return mockRun()
}
