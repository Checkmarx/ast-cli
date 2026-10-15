//go:build !integration

package cx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"

	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails/kics"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails/secrets"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/iacrealtime"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/secretsrealtime"
)

// useScannerConfig points the user's home at a temp dir holding ~/.checkmarx/cx-onboarding.properties
// with the given body (no file at all when body is empty).
func useScannerConfig(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	restore := setHomeDir(dir)
	t.Cleanup(restore)
	if body == "" {
		return
	}
	cxDir := filepath.Join(dir, ".checkmarx")
	if err := os.MkdirAll(cxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cxDir, "cx-onboarding.properties"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func secretsEdit(secret string, agent agenthooks.AgentID) agenthooks.FileEditEvent {
	return agenthooks.FileEditEvent{
		Agent:     agent,
		SessionID: "switch-sess",
		FilePath:  "config.env",
		WorkDir:   "",
		Changes:   []agenthooks.FileDiff{{Before: "", After: "TOKEN=" + secret}},
	}
}

func stubSecretsScanner(secret string) *secrets.Scanner {
	return secrets.NewScannerWithFunc(func(_, content, _ string) ([]secretsrealtime.SecretsRealtimeResult, error) {
		if strings.Contains(content, secret) {
			return []secretsrealtime.SecretsRealtimeResult{{
				Title: "github-pat", Description: "token", SecretValue: secret, Severity: "Critical",
				Locations: []realtimeengine.Location{{Line: 1}},
			}}, nil
		}
		return nil, nil
	})
}

func TestCxBeforeFileEdit_SecretsSwitch(t *testing.T) {
	const secret = "ghp_switch_test_secret_value"
	cases := []struct {
		name      string
		config    string
		wantBlock bool
	}{
		{"no config file is enabled", "", true},
		{"explicit true is enabled", "cx_secret_scanner_enabled=true\n", true},
		{"typo stays enabled", "cx_secret_scanner_enabled=flase\n", true},
		{"other scanner off does not affect secrets", "cx_asca_scanner_enabled=false\ncx_kics_scanner_enabled=false\n", true},
		{"secrets off is skipped", "cx_secret_scanner_enabled=false\n", false},
		{"secrets off (CRLF, caps)", "cx_secret_scanner_enabled=OFF\r\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetHookGlobals(t)
			useScannerConfig(t, tc.config)
			secretsScanner = stubSecretsScanner(secret)
			v := cxBeforeFileEdit(secretsEdit(secret, agenthooks.AgentCopilotCLI))
			if v.Permit == tc.wantBlock {
				t.Fatalf("Permit=%v, wantBlock=%v", v.Permit, tc.wantBlock)
			}
		})
	}
}

func kicsFinding(severity string) *kics.Scanner {
	return kics.NewScannerWithFunc(func(string, string) ([]iacrealtime.IacRealtimeResult, error) {
		return []iacrealtime.IacRealtimeResult{{
			Title: "Privileged Container", SimilarityID: "sim1", Severity: severity,
			Description: "Container runs as privileged", Locations: []realtimeengine.Location{{Line: 5}},
		}}, nil
	})
}

func kicsEdit(agent agenthooks.AgentID) agenthooks.FileEditEvent {
	return agenthooks.FileEditEvent{
		Agent: agent, SessionID: "thr-sess", FilePath: "/project/Dockerfile",
		Changes: []agenthooks.FileDiff{{Before: "", After: "FROM ubuntu\nUSER root\n"}},
	}
}

func TestCxBeforeFileEdit_SeverityThreshold(t *testing.T) {
	cases := []struct {
		name      string
		config    string
		severity  string
		wantBlock bool
	}{
		{"no config blocks low", "", "LOW", true},
		{"blank threshold blocks low", "cx_severity_threshold=\n", "LOW", true},
		{"invalid threshold blocks low", "cx_severity_threshold=hihg\n", "LOW", true},
		{"high threshold lets low through", "cx_severity_threshold=high\n", "LOW", false},
		{"high threshold lets medium through", "cx_severity_threshold=high\n", "MEDIUM", false},
		{"high threshold blocks high", "cx_severity_threshold=high\n", "HIGH", true},
		{"high threshold blocks critical", "cx_severity_threshold=high\n", "CRITICAL", true},
		{"unknown severity is never relaxed", "cx_severity_threshold=critical\n", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetHookGlobals(t)
			useScannerConfig(t, tc.config)
			kicsScanner = kicsFinding(tc.severity)
			if v := cxBeforeFileEdit(kicsEdit(agenthooks.AgentCopilotCLI)); v.Permit == tc.wantBlock {
				t.Fatalf("Permit=%v, wantBlock=%v", v.Permit, tc.wantBlock)
			}
		})
	}
}

func TestCxBeforeFileEdit_SeverityThreshold_Secrets(t *testing.T) {
	const secret = "ghp_threshold_test_secret_value"
	cases := []struct {
		name      string
		config    string
		severity  string
		wantBlock bool
	}{
		{"no config blocks low", "", "Low", true},
		{"high threshold lets medium through", "cx_severity_threshold=high\n", "Medium", false},
		{"high threshold blocks high", "cx_severity_threshold=high\n", "High", true},
		{"critical threshold lets low through", "cx_severity_threshold=critical\n", "Low", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetHookGlobals(t)
			useScannerConfig(t, tc.config)
			secretsScanner = secrets.NewScannerWithFunc(func(_, content, _ string) ([]secretsrealtime.SecretsRealtimeResult, error) {
				if strings.Contains(content, secret) {
					return []secretsrealtime.SecretsRealtimeResult{{
						Title: "generic-key", Description: "key", SecretValue: secret, Severity: tc.severity,
						Locations: []realtimeengine.Location{{Line: 1}},
					}}, nil
				}
				return nil, nil
			})
			if v := cxBeforeFileEdit(secretsEdit(secret, agenthooks.AgentCopilotCLI)); v.Permit == tc.wantBlock {
				t.Fatalf("Permit=%v, wantBlock=%v", v.Permit, tc.wantBlock)
			}
		})
	}
}

func TestCxBeforeFileEdit_SeverityThreshold_SecretsMixed(t *testing.T) {
	resetHookGlobals(t)
	useScannerConfig(t, "cx_severity_threshold=high\n")
	secretsScanner = secrets.NewScannerWithFunc(func(_, content, _ string) ([]secretsrealtime.SecretsRealtimeResult, error) {
		return []secretsrealtime.SecretsRealtimeResult{
			{Title: "high-rule", Description: "d", SecretValue: "hv", Severity: "High", Locations: []realtimeengine.Location{{Line: 1}}},
			{Title: "medium-rule", Description: "d", SecretValue: "mv", Severity: "Medium", Locations: []realtimeengine.Location{{Line: 2}}},
		}, nil
	})
	v := cxBeforeFileEdit(secretsEdit("x", agenthooks.AgentCopilotCLI))
	if v.Permit {
		t.Fatal("the high secret must block")
	}
	if blob := fmt.Sprintf("%+v", v); !strings.Contains(blob, "high-rule") || strings.Contains(blob, "medium-rule") {
		t.Fatalf("deny must list the high finding and omit the medium one: %s", blob)
	}
}

// The switches and secret detection are features of every agent except Claude, Cursor, Gemini and Codex: those four skip secret
// detection and, with the shared properties file present, keeps ASCA/KICS/SCA blocking at any severity.
func TestSwitches_IgnoredForOtherAgents(t *testing.T) {
	const secret = "ghp_other_agent_secret_value"
	others := []agenthooks.AgentID{
		agenthooks.AgentClaude, agenthooks.AgentCursor, agenthooks.AgentGemini, agenthooks.AgentCodex,
	}
	for _, agent := range others {
		t.Run(agentToString(agent)+"/secrets are skipped", func(t *testing.T) {
			resetHookGlobals(t)
			useScannerConfig(t, "")
			secretsScanner = stubSecretsScanner(secret)
			if v := cxBeforeFileEdit(secretsEdit(secret, agent)); !v.Permit {
				t.Fatal("secret detection must not run for this agent")
			}
		})
		t.Run(agentToString(agent)+"/kics switch off still blocks", func(t *testing.T) {
			resetHookGlobals(t)
			useScannerConfig(t, "cx_kics_scanner_enabled=false\n")
			kicsScanner = kicsFinding("HIGH")
			if v := cxBeforeFileEdit(kicsEdit(agent)); v.Permit {
				t.Fatal("kics switch must not apply to this agent")
			}
		})
		t.Run(agentToString(agent)+"/threshold does not relax", func(t *testing.T) {
			resetHookGlobals(t)
			useScannerConfig(t, "cx_severity_threshold=critical\n")
			kicsScanner = kicsFinding("LOW")
			if v := cxBeforeFileEdit(kicsEdit(agent)); v.Permit {
				t.Fatal("severity threshold must not apply to this agent")
			}
		})
	}
}

func TestSwitchesApply_ExcludesFourAgents(t *testing.T) {
	for agent, want := range map[agenthooks.AgentID]bool{
		agenthooks.AgentCopilotCLI: true, agenthooks.AgentCopilot: true,
		agenthooks.AgentDroid: true, agenthooks.AgentWindsurf: true,
		agenthooks.AgentClaude: false, agenthooks.AgentCursor: false, agenthooks.AgentGemini: false,
		agenthooks.AgentCodex: false,
	} {
		if got := switchesApply(agent); got != want {
			t.Errorf("%s: got %v, want %v", agentToString(agent), got, want)
		}
	}
}

func TestSecretsApply_ExcludesFourAgents(t *testing.T) {
	for agent, want := range map[agenthooks.AgentID]bool{
		agenthooks.AgentCopilotCLI: true, agenthooks.AgentCopilot: true,
		agenthooks.AgentDroid: true, agenthooks.AgentWindsurf: true,
		agenthooks.AgentClaude: false, agenthooks.AgentCursor: false,
		agenthooks.AgentGemini: false, agenthooks.AgentCodex: false,
	} {
		if got := secretsApply(agent); got != want {
			t.Errorf("%s: got %v, want %v", agentToString(agent), got, want)
		}
	}
}
