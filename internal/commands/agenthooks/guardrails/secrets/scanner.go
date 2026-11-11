package secrets

import (
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/secretsrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

// Scanner runs secret realtime scans on behalf of the file-edit guardrail.
// Tests substitute scan via NewScannerWithFunc.
type Scanner struct {
	jwt  wrappers.JWTWrapper
	scan func(sourcePath, content, ignoreFilePath string) ([]secretsrealtime.SecretsRealtimeResult, error)
}

// NewScanner returns a Scanner backed by the secrets realtime engine.
func NewScanner(jwt wrappers.JWTWrapper) *Scanner {
	s := &Scanner{jwt: jwt}
	s.scan = s.runRealScan
	return s
}

// NewScannerWithFunc returns a Scanner whose scan call is replaced with f.
// For unit tests only.
func NewScannerWithFunc(f func(sourcePath, content, ignoreFilePath string) ([]secretsrealtime.SecretsRealtimeResult, error)) *Scanner {
	return &Scanner{scan: f}
}

func (s *Scanner) runRealScan(sourcePath, content, ignoreFilePath string) ([]secretsrealtime.SecretsRealtimeResult, error) {
	svc := secretsrealtime.NewSecretsRealtimeService(s.jwt)
	return svc.ScanContent(sourcePath, content, ignoreFilePath)
}
