// Package hostenv tells where a GitHub Copilot hook runs — the terminal Copilot CLI or GitHub Copilot
// inside VS Code — and derives the telemetry agent label from it. Shared by the cx hook handlers and the
// guardrail packages so they cannot drift apart. Mirrors the cx-devassist Copilot plugin's
// _in_vscode / cx_in_vscode.
package hostenv

import (
	"os"
	"strings"
)

// Where a Copilot hook runs.
const (
	HostUnknown = ""
	HostCLI     = "cli"
	HostVSCode  = "vscode"
	HostOther   = "other" // a VS Code fork (Cursor, Windsurf, Kiro, Devin, ...): never treated as VS Code
)

// copilotAgent is the agent name the hook handlers use for GitHub Copilot (see cx.agentToString).
const copilotAgent = "Copilot"

// agentVSCode is the telemetry agent label for GitHub Copilot inside VS Code. The terminal keeps "<agent>-cli".
const agentVSCode = "Copilot-VSCode"

// PluginHost reports where this hook runs, from the PLUGIN_ROOT the host injected into the hook's
// environment (cx inherits it from the plugin's hook script). VS Code always runs its own copy of the
// plugin under <VS Code user data>/agentPlugins/ (Windows %APPDATA%\Code\agentPlugins, macOS
// ~/Library/Application Support/Code/agentPlugins, Linux ~/.config/Code/agentPlugins, Insiders
// likewise), whether the plugin was installed by the Copilot CLI or from VS Code's marketplace; the
// terminal Copilot CLI (also when started in VS Code's integrated terminal) runs it from
// ~/.copilot/installed-plugins/. Hashes and user names in the path are ignored. Any other PLUGIN_ROOT
// (a VS Code fork, JetBrains, Visual Studio, ...) is HostOther; no PLUGIN_ROOT is HostUnknown.
func PluginHost() string {
	root := strings.ToLower(strings.ReplaceAll(os.Getenv("PLUGIN_ROOT"), `\`, "/"))
	// The terminal marker is checked first: VS Code's folder names encode the source path with dashes
	// (…-copilot-installed-plugins-…), so they never contain it, while a terminal path could contain
	// "agentPlugins" by chance (a user or folder name). Only VS Code's own user-data folder ("Code",
	// "Code - Insiders") counts: VS Code forks keep their agentPlugins copies under their own folder.
	switch {
	case strings.Contains(root, "/.copilot/installed-plugins/"):
		return HostCLI
	case strings.Contains(root, "/code/agentplugins/"), strings.Contains(root, "/code - insiders/agentplugins/"):
		return HostVSCode
	case strings.TrimSpace(root) != "":
		// Any other plugin folder (a VS Code fork's agentPlugins, JetBrains, Visual Studio, ...) is not VS Code.
		return HostOther
	}
	return HostUnknown
}

// InVSCode reports whether this hook runs inside VS Code.
func InVSCode() bool {
	return PluginHost() == HostVSCode
}

// TelemetryAgent is the telemetry `agent` value for an event from the given agent name: GitHub Copilot
// inside VS Code is "Copilot-VSCode"; everything else keeps "<agent>-cli".
func TelemetryAgent(agent string) string {
	if agent != copilotAgent || !InVSCode() {
		return agent + "-cli"
	}
	return agentVSCode
}
