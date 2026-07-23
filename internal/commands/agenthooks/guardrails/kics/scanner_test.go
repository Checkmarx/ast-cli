//go:build !integration

package kics

import (
	"os"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
)

// ── resolveContainerEngine ───────────────────────────────────────────────────

func TestResolveContainerEngine_EnvOverrideWins(t *testing.T) {
	t.Setenv(params.HooksContainerEngineEnv, "podman")
	if got := resolveContainerEngine(); got != "podman" {
		t.Errorf("expected env override %q, got %q", "podman", got)
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
	// Point PATH somewhere with no docker/podman binaries so auto-detection
	// finds nothing and falls back to the default.
	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)

	if got := resolveContainerEngine(); got != defaultContainerEngine {
		t.Errorf("expected fallback default %q, got %q", defaultContainerEngine, got)
	}
}

func TestResolveContainerEngine_AutoDetectsFromPath(t *testing.T) {
	t.Setenv(params.HooksContainerEngineEnv, "")

	dir := t.TempDir()
	podmanPath := dir + string(os.PathSeparator) + "podman"
	if err := os.WriteFile(podmanPath, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatalf("failed to create fake podman binary: %v", err)
	}
	t.Setenv("PATH", dir)

	if got := resolveContainerEngine(); got != "podman" {
		t.Errorf("expected auto-detected %q, got %q", "podman", got)
	}
}
