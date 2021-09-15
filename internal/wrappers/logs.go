package wrappers

type LogsWrapper interface {
	GetLog(scanId, scanType string) (string, error)
}
