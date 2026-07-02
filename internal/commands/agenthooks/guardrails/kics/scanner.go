package kics

import (
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/iacrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

// Scanner runs IaC realtime scans on behalf of the KICS guardrail. It holds
// the wrappers needed to construct an IacRealtimeService per call. Tests
// substitute scan via NewScannerWithFunc.
type Scanner struct {
	jwt  wrappers.JWTWrapper
	ff   wrappers.FeatureFlagsWrapper
	scan func(path string) ([]iacrealtime.IacRealtimeResult, error)
}

// NewScanner returns a Scanner backed by the given wrappers.
func NewScanner(jwt wrappers.JWTWrapper, ff wrappers.FeatureFlagsWrapper) *Scanner {
	s := &Scanner{jwt: jwt, ff: ff}
	s.scan = s.runRealScan
	return s
}

// NewScannerWithFunc returns a Scanner whose scan call is replaced with f.
// For unit tests only.
func NewScannerWithFunc(f func(path string) ([]iacrealtime.IacRealtimeResult, error)) *Scanner {
	return &Scanner{scan: f}
}

func (s *Scanner) runRealScan(path string) ([]iacrealtime.IacRealtimeResult, error) {
	svc := iacrealtime.NewIacRealtimeService(s.jwt, s.ff, iacrealtime.NewContainerManager())
	return svc.RunIacRealtimeScan(path, "", existingIgnoreFilePath())
}
