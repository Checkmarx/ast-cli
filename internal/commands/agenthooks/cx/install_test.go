//go:build !integration

package cx

import "testing"

// TestFindAgentCopilot pins the GitHub Copilot CLI agent entry: its config path
// and the curated route set the installer mirrors. The route Use names must match
// the copilot-cli-* routes ast-cx-hooks registers, or `cx hooks agenthooks install
// copilot` would write commands that don't resolve.
func TestFindAgentCopilot(t *testing.T) {
	agent := FindAgent("copilot")
	if agent == nil {
		t.Fatal("FindAgent(\"copilot\") returned nil; Copilot agent not registered")
	}
	if agent.DisplayName != "GitHub Copilot CLI" {
		t.Errorf("DisplayName = %q, want %q", agent.DisplayName, "GitHub Copilot CLI")
	}
	if agent.ConfigPath != "~/.copilot/hooks/agenthooks.json" {
		t.Errorf("ConfigPath = %q, want %q", agent.ConfigPath, "~/.copilot/hooks/agenthooks.json")
	}
	if agent.Install == nil {
		t.Error("Install func is nil")
	}

	wantRoutes := []string{
		"copilot-cli-stop",
		"copilot-cli-pre-tool-use",
		"copilot-cli-pre-file-write",
		"copilot-cli-user-prompt-submit",
	}
	if len(agent.Routes) != len(wantRoutes) {
		t.Fatalf("got %d routes, want %d: %+v", len(agent.Routes), len(wantRoutes), agent.Routes)
	}
	for i, want := range wantRoutes {
		if agent.Routes[i].Use != want {
			t.Errorf("Routes[%d].Use = %q, want %q", i, agent.Routes[i].Use, want)
		}
		if agent.Routes[i].Short == "" {
			t.Errorf("Routes[%d] (%q) has empty Short description", i, want)
		}
	}
}

// TestFindAgentUnknown verifies FindAgent returns nil for an unregistered id.
func TestFindAgentUnknown(t *testing.T) {
	if a := FindAgent("not-a-real-agent"); a != nil {
		t.Errorf("FindAgent of unknown id = %+v, want nil", a)
	}
}
