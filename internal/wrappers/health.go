package wrappers

type HealthcheckWrapper interface {
	CheckWebAppIsUp() error
	CheckDatabaseHealth() error
}
