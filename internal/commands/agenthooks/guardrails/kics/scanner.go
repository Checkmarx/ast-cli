package kics

import (
	"os"
	"os/exec"

	"github.com/checkmarx/ast-cli/internal/logger"
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
	scan func(path, ignoreFilePath string) ([]iacrealtime.IacRealtimeResult, error)

	// engine that last completed a scan. A delta edit scans twice (proposed and
	// original content), so without this the second scan repeats the whole
	// selection — including a failed engine's image-inspect and pull. One hook
	// invocation is one event on one goroutine, so no lock is needed. A value
	// that goes stale (the engine stopped since) costs one failed scan and is
	// then corrected by the fallback below.
	engine string
}

// NewScanner returns a Scanner backed by the given wrappers.
func NewScanner(jwt wrappers.JWTWrapper, ff wrappers.FeatureFlagsWrapper) *Scanner {
	s := &Scanner{jwt: jwt, ff: ff}
	s.scan = s.runRealScan
	return s
}

// NewScannerWithFunc returns a Scanner whose scan call is replaced with f.
// For unit tests only.
func NewScannerWithFunc(f func(path, ignoreFilePath string) ([]iacrealtime.IacRealtimeResult, error)) *Scanner {
	return &Scanner{scan: f}
}

// Container engine names.
const (
	engineDocker = "docker"
	enginePodman = "podman"
)

// defaultContainerEngine mirrors the "docker" default of the --engine flag on
// the manual `cx scan iac-realtime` command (internal/commands/scan.go), used
// when neither an override nor auto-detection finds a usable engine.
const defaultContainerEngine = engineDocker

// resolveContainerEngine picks the container engine to try first. The guardrail
// is invoked as `cx hooks <route>` with only stdin JSON (no --engine flag like
// the manual `cx scan iac-realtime` command), so it resolves the engine itself:
//  1. HooksContainerEngineEnv, if set — lets a Podman/Colima-only user (or the
//     agent plugin's own hook environment) override the choice explicitly.
//  2. PATH lookup: try "docker" then "podman", first one found wins.
//  3. defaultContainerEngine, if neither resolves — preserves prior behavior
//     and existing error messaging when no engine is installed at all.
//
// Deliberately a PATH lookup and nothing more: it runs before every IaC scan,
// and a daemon probe here would add a round-trip to edits that scan fine.
// Whether the daemon is actually up is settled by fallbackEngineFor, which is
// reached only after a scan has already failed.
func resolveContainerEngine() string {
	if engine := engineOverride(); engine != "" {
		return engine
	}
	for _, engine := range []string{engineDocker, enginePodman} {
		if _, err := exec.LookPath(engine); err == nil {
			return engine
		}
	}
	return defaultContainerEngine
}

// fallbackEngineFor returns the engine to retry with after a scan failed on
// `tried`, or "" when there is nothing worth retrying. A resolvable binary is
// not proof of a usable engine — Docker Desktop and the Podman machine can be
// installed but stopped, which is what made the guardrail fail open and let
// vulnerable IaC through. Reached only after a failure, so the daemon probe it
// costs never lands on a working scan.
func fallbackEngineFor(tried string) string {
	if engineOverride() != "" {
		return "" // explicit user choice — do not second-guess it
	}
	other := enginePodman
	if tried == enginePodman {
		other = engineDocker
	}
	if !engineReady(other) {
		return ""
	}
	return other
}

// engineOverride is the user's explicit engine choice, if any. Read in one place
// so resolveContainerEngine and fallbackEngineFor cannot disagree about it.
func engineOverride() string {
	return os.Getenv(params.HooksContainerEngineEnv)
}

// engineReady is iacrealtime.IsEngineRunning; replaced in tests.
var engineReady = iacrealtime.IsEngineRunning

func (s *Scanner) runRealScan(path, ignoreFilePath string) ([]iacrealtime.IacRealtimeResult, error) {
	svc := iacrealtime.NewIacRealtimeService(s.jwt, s.ff, iacrealtime.NewContainerManager())

	engine := s.engine
	if engine == "" {
		engine = resolveContainerEngine()
	}
	results, err := svc.RunIacRealtimeScan(path, engine, ignoreFilePath)
	if err == nil {
		s.engine = engine
		return results, nil
	}

	other := fallbackEngineFor(engine)
	if other == "" {
		return results, err
	}

	logger.PrintfIfVerbose("kics guardrail: %s scan failed (%v); retrying with %s", engine, err, other)
	results, err = svc.RunIacRealtimeScan(path, other, ignoreFilePath)
	if err == nil {
		s.engine = other
	}
	return results, err
}
