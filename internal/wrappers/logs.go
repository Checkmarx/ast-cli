package wrappers

type LogsWrapper interface {
	GetLog(scanID, scanType string) (string, error)
}
