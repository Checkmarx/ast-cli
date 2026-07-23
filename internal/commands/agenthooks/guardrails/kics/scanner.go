package kics

import (
	"os"
	"os/exec"

	"github.com/checkmarx/ast-cli/internal/params"
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

// defaultContainerEngine mirrors the "docker" default of the --engine flag on
// the manual `cx scan iac-realtime` command (internal/commands/scan.go), used
// when neither an override nor auto-detection finds a usable engine.
const defaultContainerEngine = "docker"

// resolveContainerEngine picks the container engine name to pass to
// RunIacRealtimeScan. The guardrail is invoked as `cx hooks <route>` with only
// stdin JSON (no --engine flag like the manual `cx scan iac-realtime`
// command), so it resolves the engine itself:
//  1. HooksContainerEngineEnv, if set — lets a Podman/Colima-only user (or the
//     agent plugin's own hook environment) override the choice explicitly.
//  2. Auto-detect via PATH lookup: try "docker" then "podman", first one found wins.
//  3. defaultContainerEngine, if neither resolves — preserves prior behavior
//     and existing error messaging when no engine is installed at all.
func resolveContainerEngine() string {
	if engine := os.Getenv(params.HooksContainerEngineEnv); engine != "" {
		return engine
	}
	for _, engine := range []string{"docker", "podman"} {
		if _, err := exec.LookPath(engine); err == nil {
			return engine
		}
	}
	return defaultContainerEngine
}

func (s *Scanner) runRealScan(path string) ([]iacrealtime.IacRealtimeResult, error) {
	svc := iacrealtime.NewIacRealtimeService(s.jwt, s.ff, iacrealtime.NewContainerManager())
	return svc.RunIacRealtimeScan(path, resolveContainerEngine(), existingIgnoreFilePath())
}
