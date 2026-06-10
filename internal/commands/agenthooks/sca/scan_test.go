package sca

import (
	"errors"
	"testing"

	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
)

func fakeResults(pkgs ...ossrealtime.OssPackage) func(string) (*ossrealtime.OssPackageResults, error) {
	return func(string) (*ossrealtime.OssPackageResults, error) {
		return &ossrealtime.OssPackageResults{Packages: pkgs}, nil
	}
}

func TestScanner_BucketsByStatus(t *testing.T) {
	s := NewScannerWithFunc(fakeResults(
		ossrealtime.OssPackage{PackageName: "ok", Status: "OK"},
		ossrealtime.OssPackage{PackageName: "bad", Status: "Malicious"},
		ossrealtime.OssPackage{PackageName: "vuln", Status: "Vulnerable"},
		ossrealtime.OssPackage{PackageName: "huh", Status: "Unknown"},
	))
	mal, vuln, err := s.ScanPackages(FormatNpmPackageJson, []Package{{Name: "x"}})
	if err != nil {
		t.Fatalf("ScanPackages: %v", err)
	}
	if len(mal) != 1 || mal[0].PackageName != "bad" {
		t.Errorf("malicious=%v, want [bad]", mal)
	}
	if len(vuln) != 1 || vuln[0].PackageName != "vuln" {
		t.Errorf("vulnerable=%v, want [vuln]", vuln)
	}
}

func TestScanner_AllClean(t *testing.T) {
	s := NewScannerWithFunc(fakeResults(
		ossrealtime.OssPackage{PackageName: "a", Status: "OK"},
		ossrealtime.OssPackage{PackageName: "b", Status: "OK"},
	))
	mal, vuln, err := s.ScanPackages(FormatNpmPackageJson, []Package{{Name: "a"}, {Name: "b"}})
	if err != nil {
		t.Fatalf("ScanPackages: %v", err)
	}
	if len(mal) != 0 || len(vuln) != 0 {
		t.Errorf("expected no findings, got mal=%v vuln=%v", mal, vuln)
	}
}

func TestScanner_UpstreamErrorPropagates(t *testing.T) {
	wantErr := errors.New("boom")
	s := NewScannerWithFunc(func(string) (*ossrealtime.OssPackageResults, error) {
		return nil, wantErr
	})
	_, _, err := s.ScanPackages(FormatNpmPackageJson, []Package{{Name: "x"}})
	if !errors.Is(err, wantErr) {
		t.Errorf("got err %v, want %v", err, wantErr)
	}
}

func TestScanner_EmptyPackagesIsNoop(t *testing.T) {
	called := false
	s := NewScannerWithFunc(func(string) (*ossrealtime.OssPackageResults, error) {
		called = true
		return nil, nil
	})
	mal, vuln, err := s.ScanPackages(FormatNpmPackageJson, nil)
	if err != nil || len(mal) != 0 || len(vuln) != 0 {
		t.Errorf("expected zero results no error, got mal=%v vuln=%v err=%v", mal, vuln, err)
	}
	if called {
		t.Errorf("scan should not be invoked for empty package list")
	}
}

func TestScanner_NilResultsAreSafe(t *testing.T) {
	s := NewScannerWithFunc(func(string) (*ossrealtime.OssPackageResults, error) {
		return nil, nil
	})
	mal, vuln, err := s.ScanPackages(FormatNpmPackageJson, []Package{{Name: "x"}})
	if err != nil || len(mal) != 0 || len(vuln) != 0 {
		t.Errorf("expected zero results no error, got mal=%v vuln=%v err=%v", mal, vuln, err)
	}
}
