//go:build !integration

package cx

import (
	"os"
	"path/filepath"
	"testing"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"

	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/scannerconfig"
)

// PLUGIN_ROOT values as the hosts set them (AST-186850 PoC; Windows paths, any user name).
const (
	rootVSCodeFromCLIInstall = `C:\Users\u\AppData\Roaming\Code\agentPlugins\file-c-3A-Users-u-copilot-installed-plugins-cx-devassist-marketplace-cx-devassist\1a15407167c`
	rootVSCodeFromMarketplace = `C:\Users\u\AppData\Roaming\Code\agentPlugins\file-c-3A-Users-u-vscode-agent-plugins-marketplace-repositories-github-com-Checkmarx-cx-agentic-ai\1a1542d091b`
	rootCLI                   = `C:\Users\u\.copilot\installed-plugins\cx-devassist-marketplace\cx-devassist`
)

// useHost sets the hook environment the host would inject.
func useHost(t *testing.T, pluginRoot, integrationID string) {
	t.Helper()
	t.Setenv("PLUGIN_ROOT", pluginRoot)
	t.Setenv("GITHUB_COPILOT_INTEGRATION_ID", integrationID)
}

// useIDEConfig points the user's home at a temp dir holding ~/.checkmarx/cx-onboarding-ide.properties
// (and optionally cx-onboarding.properties); an empty body means no file at all.
func useIDEConfig(t *testing.T, ideBody, cliBody string) {
	t.Helper()
	dir := t.TempDir()
	t.Cleanup(setHomeDir(dir))
	cxDir := filepath.Join(dir, ".checkmarx")
	if err := os.MkdirAll(cxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"cx-onboarding-ide.properties": ideBody, "cx-onboarding.properties": cliBody} {
		if body == "" {
			continue
		}
		if err := os.WriteFile(filepath.Join(cxDir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPluginHost(t *testing.T) {
	cases := []struct {
		name, root, integrationID, want string
	}{
		{"VS Code copy of a Copilot CLI install", rootVSCodeFromCLIInstall, "", hostVSCode},
		{"VS Code marketplace install", rootVSCodeFromMarketplace, "", hostVSCode},
		{"Linux VS Code", "/home/u/.config/Code/agentPlugins/x/1a", "", hostVSCode},
		{"macOS VS Code Insiders", "/Users/u/Library/Application Support/Code - Insiders/agentPlugins/x/1a", "", hostVSCode},
		{"terminal Copilot CLI", rootCLI, "", hostCLI},
		{"terminal path wins over the integration id", rootCLI, vscodeIntegrationID, hostCLI},
		{"terminal path containing agentPlugins by chance", `C:\Users\agentPlugins\.copilot\installed-plugins\m\cx-devassist`, "", hostCLI},
		{"terminal path under an agentPlugins folder", `/home/u/agentPlugins/.copilot/installed-plugins/m/cx-devassist`, "", hostCLI},
		{"unrecognised root + integration id", `D:\other\plugin`, vscodeIntegrationID, hostVSCode},
		{"no root + integration id", "", vscodeIntegrationID, hostVSCode},
		{"nothing set", "", "", hostUnknown},
		{"other integration id", "", "copilot-developer-cli", hostUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useHost(t, tc.root, tc.integrationID)
			if got := pluginHost(); got != tc.want {
				t.Fatalf("pluginHost() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCxBeforeFileEdit_SelfHealingOff(t *testing.T) {
	const secret = "ghp_self_healing_test_secret_value"
	const off = "cx_schema_version=1\ncx_self_healing_enabled=false\n"
	cases := []struct {
		name      string
		agent     agenthooks.AgentID
		root      string
		ide       string
		wantBlock bool
	}{
		{"VS Code + OFF allows without scanning", agenthooks.AgentCopilotCLI, rootVSCodeFromMarketplace, off, false},
		{"VS Code + ON scans", agenthooks.AgentCopilotCLI, rootVSCodeFromCLIInstall, "cx_self_healing_enabled=true\n", true},
		{"VS Code + no IDE file scans (default ON)", agenthooks.AgentCopilotCLI, rootVSCodeFromCLIInstall, "", true},
		{"VS Code + unknown schema scans", agenthooks.AgentCopilotCLI, rootVSCodeFromCLIInstall, "cx_schema_version=2\ncx_self_healing_enabled=false\n", true},
		{"terminal ignores the IDE file", agenthooks.AgentCopilotCLI, rootCLI, off, true},
		{"other agents ignore it", agenthooks.AgentWindsurf, rootVSCodeFromCLIInstall, off, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetHookGlobals(t)
			useIDEConfig(t, tc.ide, "")
			useHost(t, tc.root, "")
			secretsScanner = stubSecretsScanner(secret)
			v := cxBeforeFileEdit(secretsEdit(secret, tc.agent))
			if v.Permit == tc.wantBlock {
				t.Fatalf("Permit=%v, wantBlock=%v", v.Permit, tc.wantBlock)
			}
		})
	}
}

func TestScannerOn_VSCodeUsesIDEFile(t *testing.T) {
	const ideSecretsOff = "cx_schema_version=1\ncx_secret_scanner_enabled=false\n"
	const cliSecretsOff = "cx_secret_scanner_enabled=false\n"
	cases := []struct {
		name     string
		root     string
		ide, cli string
		want     bool
	}{
		{"VS Code follows the IDE file", rootVSCodeFromCLIInstall, ideSecretsOff, "", false},
		{"VS Code IDE file missing key is on, even if the terminal file is off", rootVSCodeFromCLIInstall, "cx_schema_version=1\n", cliSecretsOff, true},
		{"VS Code without a valid IDE file falls back to the terminal file", rootVSCodeFromCLIInstall, "", cliSecretsOff, false},
		{"terminal never reads the IDE file", rootCLI, ideSecretsOff, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useIDEConfig(t, tc.ide, tc.cli)
			useHost(t, tc.root, "")
			if got := scannerOn(agenthooks.AgentCopilotCLI, scannerconfig.Secrets); got != tc.want {
				t.Fatalf("scannerOn = %v, want %v", got, tc.want)
			}
		})
	}
}
