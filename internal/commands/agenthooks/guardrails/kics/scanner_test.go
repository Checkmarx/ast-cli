//go:build !integration

package kics

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/iacrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

// ── NewScanner ──────────────────────────────────────────────────────────────

func TestNewScanner_ReturnsValidScanner(t *testing.T) {
	jwt := &mock.JWTMockWrapper{}
	ff := &mock.FeatureFlagsMockWrapper{}

	s := NewScanner(jwt, ff)
	if s == nil {
		t.Fatal("expected non-nil scanner")
	}
	if s.scan == nil {
		t.Fatal("expected scan function to be set")
	}
}

func TestNewScanner_HoldsWrappers(t *testing.T) {
	jwt := &mock.JWTMockWrapper{}
	ff := &mock.FeatureFlagsMockWrapper{}

	s := NewScanner(jwt, ff)
	assert.NotNil(t, s)
	// Verify wrappers are internally stored
	assert.NotNil(t, s.scan)
}

// ── NewScannerWithFunc ──────────────────────────────────────────────────────

func TestNewScannerWithFunc_UsesMockFunction(t *testing.T) {
	called := false
	mockFunc := func(path, ignoreFilePath string) ([]iacrealtime.IacRealtimeResult, error) {
		called = true
		return []iacrealtime.IacRealtimeResult{}, nil
	}

	s := NewScannerWithFunc(mockFunc)
	if s == nil {
		t.Fatal("expected non-nil scanner")
	}
	if s.scan == nil {
		t.Fatal("expected scan function to be set")
	}

	// Verify the mock function is called
	_, _ = s.scan("", "")
	if !called {
		t.Fatal("expected mock function to be called")
	}
}

func TestNewScannerWithFunc_MockReturnsResults(t *testing.T) {
	mockResults := []iacrealtime.IacRealtimeResult{
		{
			SimilarityID: "test-id",
			Title:        "Test Finding",
			Severity:     "HIGH",
		},
	}

	mockFunc := func(path, ignoreFilePath string) ([]iacrealtime.IacRealtimeResult, error) {
		return mockResults, nil
	}

	s := NewScannerWithFunc(mockFunc)
	results, err := s.scan("/some/path", "")

	assert.NoError(t, err)
	assert.Equal(t, mockResults, results)
}

// ── resolveContainerEngine ───────────────────────────────────────────────────

func TestResolveContainerEngine_EnvOverrideWins(t *testing.T) {
	t.Setenv(params.HooksContainerEngineEnv, enginePodman)
	if got := resolveContainerEngine(); got != enginePodman {
		t.Errorf("expected env override %q, got %q", enginePodman, got)
	}
}

func TestResolveContainerEngine_EnvOverrideArbitraryValue(t *testing.T) {
	t.Setenv(params.HooksContainerEngineEnv, "nerdctl")
	if got := resolveContainerEngine(); got != "nerdctl" {
		t.Errorf("expected env override %q, got %q", "nerdctl", got)
	}
}

func TestResolveContainerEngine_FallsBackToDefaultWhenNothingResolves(t *testing.T) {
	t.Setenv(params.HooksContainerEngineEnv, "")
	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)

	if got := resolveContainerEngine(); got != defaultContainerEngine {
		t.Errorf("expected fallback default %q, got %q", defaultContainerEngine, got)
	}
}

func TestResolveContainerEngine_DefaultContainerEngineConstant(t *testing.T) {
	assert.Equal(t, "docker", defaultContainerEngine)
}

func TestResolveContainerEngine_EmptyEnvFallsBack(t *testing.T) {
	t.Setenv(params.HooksContainerEngineEnv, "")
	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)

	got := resolveContainerEngine()
	assert.Equal(t, defaultContainerEngine, got)
}

// ── fallbackEngineFor ────────────────────────────────────────────────────────

func stubEngineReady(t *testing.T, ready string) {
	t.Helper()
	orig := engineReady
	t.Cleanup(func() { engineReady = orig })
	engineReady = func(engine string) bool { return engine == ready }
}

// A stopped Docker must hand the scan to a live Podman — otherwise the scan
// errors, the guardrail fails open, and vulnerable IaC ships unflagged.
func TestFallbackEngineFor_StoppedDockerFallsBackToPodman(t *testing.T) {
	t.Setenv(params.HooksContainerEngineEnv, "")
	stubEngineReady(t, enginePodman)

	assert.Equal(t, enginePodman, fallbackEngineFor(engineDocker))
}

func TestFallbackEngineFor_StoppedPodmanFallsBackToDocker(t *testing.T) {
	t.Setenv(params.HooksContainerEngineEnv, "")
	stubEngineReady(t, engineDocker)

	assert.Equal(t, engineDocker, fallbackEngineFor(enginePodman))
}

func TestFallbackEngineFor_NoRetryWhenOtherEngineIsAlsoDown(t *testing.T) {
	t.Setenv(params.HooksContainerEngineEnv, "")
	stubEngineReady(t, "")

	assert.Equal(t, "", fallbackEngineFor(engineDocker))
}

// An explicit override is the user's choice; silently switching engines under
// them would be worse than the error they asked for.
func TestFallbackEngineFor_NoRetryWhenEngineExplicitlyOverridden(t *testing.T) {
	t.Setenv(params.HooksContainerEngineEnv, engineDocker)
	stubEngineReady(t, enginePodman)

	assert.Equal(t, "", fallbackEngineFor(engineDocker))
}
