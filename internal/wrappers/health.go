package wrappers

type HealthCheckWrapper interface {
	RunWebAppCheck() error
}
