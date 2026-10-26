//go:build !integration

package hostenv

import (
	"os"
	"path/filepath"
	"testing"
)

// PLUGIN_ROOT values as the hosts set them (AST-186850 PoC; Windows paths, any user name).
const (
	rootVSCodeFromCLIInstall  = `C:\Users\u\AppData\Roaming\Code\agentPlugins\file-c-3A-Users-u-copilot-installed-plugins-cx-devassist-marketplace-cx-devassist\1a15407167c`
	rootVSCodeFromMarketplace = `C:\Users\u\AppData\Roaming\Code\agentPlugins\file-c-3A-Users-u-vscode-agent-plugins-marketplace-repositories-github-com-Checkmarx-cx-agentic-ai\1a1542d091b`
	rootCLI                   = `C:\Users\u\.copilot\installed-plugins\cx-devassist-marketplace\cx-devassist`
)

func useHost(t *testing.T, pluginRoot, integrationID string) {
	t.Helper()
	t.Setenv("PLUGIN_ROOT", pluginRoot)
	t.Setenv("GITHUB_COPILOT_INTEGRATION_ID", integrationID)
}

// useIDEFile points the user's home at a temp dir holding ~/.checkmarx/cx-onboarding-ide.properties
// with the given body (no file when empty).
func useIDEFile(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	if body == "" {
		return
	}
	cxDir := filepath.Join(dir, ".checkmarx")
	if err := os.MkdirAll(cxDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cxDir, "cx-onboarding-ide.properties"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPluginHost(t *testing.T) {
	cases := []struct {
		name, root, integrationID, want string
	}{
		{"VS Code copy of a Copilot CLI install", rootVSCodeFromCLIInstall, "", HostVSCode},
		{"VS Code marketplace install", rootVSCodeFromMarketplace, "", HostVSCode},
		{"Linux VS Code", "/home/u/.config/Code/agentPlugins/x/1a", "", HostVSCode},
		{"macOS VS Code Insiders", "/Users/u/Library/Application Support/Code - Insiders/agentPlugins/x/1a", "", HostVSCode},
		{"terminal Copilot CLI", rootCLI, "", HostCLI},
		{"terminal path wins over the integration id", rootCLI, VSCodeIntegrationID, HostCLI},
		{"Windsurf copy is not VS Code", `C:\Users\u\AppData\Roaming\Windsurf\agentPlugins\x\1a`, VSCodeIntegrationID, HostOther},
		{"Cursor copy is not VS Code", "/Users/u/Library/Application Support/Cursor/agentPlugins/x/1a", "", HostOther},
		{"Kiro copy is not VS Code", "/home/u/.config/Kiro/agentPlugins/x/1a", "", HostOther},
		{"terminal path containing agentPlugins by chance", `C:\Users\agentPlugins\.copilot\installed-plugins\m\cx-devassist`, "", HostCLI},
		{"terminal path under an agentPlugins folder", `/home/u/agentPlugins/.copilot/installed-plugins/m/cx-devassist`, "", HostCLI},
		{"other plugin folder (JetBrains, Visual Studio, ...) is not VS Code", `D:\other\plugin`, VSCodeIntegrationID, HostOther},
		{"no root + integration id", "", VSCodeIntegrationID, HostVSCode},
		{"nothing set", "", "", HostUnknown},
		{"other integration id", "", "copilot-developer-cli", HostUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useHost(t, tc.root, tc.integrationID)
			if got := PluginHost(); got != tc.want {
				t.Fatalf("PluginHost() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTelemetryAgent(t *testing.T) {
	const oneAssist = "cx_schema_version=1\ncx_ide_extension=ast-results\n"
	const devAssist = "cx_schema_version=1\ncx_ide_extension=cx-dev-assist\n"
	cases := []struct {
		name, agent, root, ide, want string
	}{
		{"terminal Copilot", "Copilot", rootCLI, oneAssist, "Copilot-cli"},
		{"VS Code + Checkmarx One", "Copilot", rootVSCodeFromCLIInstall, oneAssist, "Copilot-VSCode-OneAssist"},
		{"VS Code + Developer Assist", "Copilot", rootVSCodeFromMarketplace, devAssist, "Copilot-VSCode-DevAssist"},
		{"VS Code, no IDE file", "Copilot", rootVSCodeFromCLIInstall, "", "Copilot-VSCode"},
		{"VS Code, unknown extension id", "Copilot", rootVSCodeFromCLIInstall, "cx_ide_extension=other\n", "Copilot-VSCode"},
		{"VS Code, invalid IDE file", "Copilot", rootVSCodeFromCLIInstall, "cx_schema_version=2\ncx_ide_extension=ast-results\n", "Copilot-VSCode"},
		{"VS Code fork stays -cli", "Copilot", `C:\Users\u\AppData\Roaming\Windsurf\agentPlugins\x\1a`, oneAssist, "Copilot-cli"},
		{"no host signal stays -cli", "Copilot", "", oneAssist, "Copilot-cli"},
		{"other agents unchanged in VS Code", "Claude", rootVSCodeFromCLIInstall, oneAssist, "Claude-cli"},
		{"other agents unchanged", "Cursor", "", "", "Cursor-cli"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useIDEFile(t, tc.ide)
			useHost(t, tc.root, "")
			if got := TelemetryAgent(tc.agent); got != tc.want {
				t.Fatalf("TelemetryAgent(%q) = %q, want %q", tc.agent, got, tc.want)
			}
		})
	}
}
