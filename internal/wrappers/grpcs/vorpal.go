package grpcs

import vorpalScan "github.com/checkmarx/ast-cli/internal/wrappers/grpcs/protos/vorpal/scans"

type VorpalWrapper interface {
	Scan(fileName, sourceCode string) (*vorpalScan.ScanResult, error)
	HealthCheck() error
	ShutDown() error
}
