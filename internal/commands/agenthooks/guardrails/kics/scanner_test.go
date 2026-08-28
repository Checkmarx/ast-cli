//go:build !integration

package kics

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/iacrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
)

const enginePodman = "podman"

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
	mockFunc := func(path string) ([]iacrealtime.IacRealtimeResult, error) {
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
	_, _ = s.scan("")
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

	mockFunc := func(path string) ([]iacrealtime.IacRealtimeResult, error) {
		return mockResults, nil
	}

	s := NewScannerWithFunc(mockFunc)
	results, err := s.scan("/some/path")

	assert.NoError(t, err)
	assert.Equal(t, mockResults, results)
}

// ── resolveEngine ────────────────────────────────────────────────────────

// The guardrail has no --engine flag, so it must reach the container-free engine by default;
// otherwise the agent hook would still require Docker on every developer machine.
func TestResolveEngine_DefaultsToEmbedded(t *testing.T) {
	t.Setenv(params.HooksContainerEngineEnv, "")
	assert.Equal(t, util.KicsEngineEmbedded, resolveEngine())
}

func TestResolveEngine_EnvOverrideSelectsContainerEngine(t *testing.T) {
	t.Setenv(params.HooksContainerEngineEnv, enginePodman)
	assert.Equal(t, enginePodman, resolveEngine())
}

func TestResolveEngine_EnvOverrideAcceptsArbitraryEngine(t *testing.T) {
	t.Setenv(params.HooksContainerEngineEnv, "nerdctl")
	assert.Equal(t, "nerdctl", resolveEngine())
}

// The embedded engine needs no binary on PATH, so an empty PATH must not change the choice.
func TestResolveEngine_EmbeddedDoesNotDependOnPath(t *testing.T) {
	t.Setenv(params.HooksContainerEngineEnv, "")
	t.Setenv("PATH", t.TempDir())
	assert.Equal(t, util.KicsEngineEmbedded, resolveEngine())
}
