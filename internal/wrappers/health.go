package wrappers

type HealthWrapper interface {
	RunWebAppCheck() error
}
