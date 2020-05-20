package wrappers

type MockHealthcheckWrapper struct {
}

func NewMockHealthcheckWrapper(astWebAppURL string) HealthcheckWrapper {
	return &MockHealthcheckWrapper{}
}

func (m *MockHealthcheckWrapper) CheckWebAppIsUp() error {
	return nil
}

func (s *MockHealthcheckWrapper) CheckDatabaseHealth() error {
	return nil
}
