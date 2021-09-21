package mock

type LogsMockWrapper struct {
}

func (l *LogsMockWrapper) GetLog(_, _ string) (string, error) {
	return "logs", nil
}
