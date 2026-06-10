package sca

import (
	"os"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

// statusOK and statusUnknown are the only Status values that the SCA hooks
// treat as "clean". Everything else is either Malicious (escalates the deny
// message) or generic Vulnerable. The literal strings come from the upstream
// realtime-scanner API and match the values used in oss-realtime tests.
const (
	statusOK        = "OK"
	statusMalicious = "Malicious"
	statusUnknown   = "Unknown"
)

// Scanner runs oss-realtime scans on behalf of the SCA guardrails. It holds
// the wrappers needed to construct an OssRealtimeService per call. Tests
// substitute scan via NewScannerWithFunc.
type Scanner struct {
	JWT  wrappers.JWTWrapper
	FF   wrappers.FeatureFlagsWrapper
	RT   wrappers.RealtimeScannerWrapper
	scan func(path string) (*ossrealtime.OssPackageResults, error)
}

// NewScanner returns a Scanner backed by the given wrappers. The scan call
// goes through ossrealtime.NewOssRealtimeService.RunOssRealtimeScan.
func NewScanner(jwt wrappers.JWTWrapper, ff wrappers.FeatureFlagsWrapper, rt wrappers.RealtimeScannerWrapper) *Scanner {
	s := &Scanner{JWT: jwt, FF: ff, RT: rt}
	s.scan = s.runRealScan
	return s
}

// NewScannerWithFunc returns a Scanner whose scan call is replaced with f.
// For unit tests only.
func NewScannerWithFunc(f func(path string) (*ossrealtime.OssPackageResults, error)) *Scanner {
	return &Scanner{scan: f}
}

func (s *Scanner) runRealScan(path string) (*ossrealtime.OssPackageResults, error) {
	svc := ossrealtime.NewOssRealtimeService(s.JWT, s.FF, s.RT)
	return svc.RunOssRealtimeScan(path, "")
}

// ScanPackages synthesises a temp manifest from pkgs and scans it. Returns
// (malicious, vulnerable) buckets. On error the buckets are nil and the error
// is propagated — callers fail open by treating errors as "no findings".
func (s *Scanner) ScanPackages(format Format, pkgs []Package) (malicious, vulnerable []ossrealtime.OssPackage, err error) {
	if len(pkgs) == 0 {
		return nil, nil, nil
	}
	normalized := make([]Package, len(pkgs))
	for i, p := range pkgs {
		normalized[i] = Package{Name: p.Name, Version: normalizeSemver(p.Version)}
	}
	dir, err := os.MkdirTemp("", "sca-scan-")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(dir)

	path, err := Synthesize(format, normalized, dir)
	if err != nil {
		return nil, nil, err
	}
	return s.scanAndBucket(path)
}

// ScanFile scans an existing manifest at path (used for `pip install -r ...`
// and for the Cursor post-write audit).
func (s *Scanner) ScanFile(path string) (malicious, vulnerable []ossrealtime.OssPackage, err error) {
	return s.scanAndBucket(path)
}

func (s *Scanner) scanAndBucket(path string) (malicious, vulnerable []ossrealtime.OssPackage, err error) {
	if s == nil || s.scan == nil {
		return nil, nil, nil
	}
	results, err := s.scan(path)
	if err != nil {
		return nil, nil, err
	}
	if results == nil {
		return nil, nil, nil
	}
	for _, p := range results.Packages {
		switch p.Status {
		case statusMalicious:
			malicious = append(malicious, p)
		case statusOK, statusUnknown, "":
			// clean / not classified — ignore
		default:
			vulnerable = append(vulnerable, p)
		}
	}
	return malicious, vulnerable, nil
}
