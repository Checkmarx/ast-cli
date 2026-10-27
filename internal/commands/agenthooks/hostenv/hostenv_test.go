//go:build !integration

package hostenv

import (
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

func TestPluginHost(t *testing.T) {
	cases := []struct {
		name, root, integrationID, want string
	}{
		{"VS Code copy of a Copilot CLI install", rootVSCodeFromCLIInstall, "", HostVSCode},
		{"VS Code marketplace install", rootVSCodeFromMarketplace, "", HostVSCode},
		{"Linux VS Code", "/home/u/.config/Code/agentPlugins/x/1a", "", HostVSCode},
		{"macOS VS Code Insiders", "/Users/u/Library/Application Support/Code - Insiders/agentPlugins/x/1a", "", HostVSCode},
		{"terminal Copilot CLI", rootCLI, "", HostCLI},
		{"terminal path with the VS Code integration id", rootCLI, "vscode-chat", HostCLI},
		{"Windsurf copy is not VS Code", `C:\Users\u\AppData\Roaming\Windsurf\agentPlugins\x\1a`, "vscode-chat", HostOther},
		{"Cursor copy is not VS Code", "/Users/u/Library/Application Support/Cursor/agentPlugins/x/1a", "", HostOther},
		{"Kiro copy is not VS Code", "/home/u/.config/Kiro/agentPlugins/x/1a", "", HostOther},
		{"terminal path containing agentPlugins by chance", `C:\Users\agentPlugins\.copilot\installed-plugins\m\cx-devassist`, "", HostCLI},
		{"terminal path under an agentPlugins folder", `/home/u/agentPlugins/.copilot/installed-plugins/m/cx-devassist`, "", HostCLI},
		{"other plugin folder (JetBrains, Visual Studio, ...) is not VS Code", `D:\other\plugin`, "vscode-chat", HostOther},
		{"integration id alone is not VS Code", "", "vscode-chat", HostUnknown},
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
	cases := []struct {
		name, agent, root, want string
	}{
		{"terminal Copilot", "Copilot", rootCLI, "Copilot-cli"},
		{"VS Code copy of a Copilot CLI install", "Copilot", rootVSCodeFromCLIInstall, "Copilot-VSCode"},
		{"VS Code marketplace install", "Copilot", rootVSCodeFromMarketplace, "Copilot-VSCode"},
		{"VS Code fork stays -cli", "Copilot", `C:\Users\u\AppData\Roaming\Windsurf\agentPlugins\x\1a`, "Copilot-cli"},
		{"no host signal stays -cli", "Copilot", "", "Copilot-cli"},
		{"other agents unchanged in VS Code", "Claude", rootVSCodeFromCLIInstall, "Claude-cli"},
		{"other agents unchanged", "Cursor", "", "Cursor-cli"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useHost(t, tc.root, "")
			if got := TelemetryAgent(tc.agent); got != tc.want {
				t.Fatalf("TelemetryAgent(%q) = %q, want %q", tc.agent, got, tc.want)
			}
		})
	}
}
