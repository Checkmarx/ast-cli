package governance

import (
	"encoding/json"
	"os"
	"strings"
)

// IDEInfo describes the IDE or runtime environment that launched this session.
type IDEInfo struct {
	Name          string `json:"name"`                    // "vscode" | "jetbrains" | "cursor" | "desktop" | "terminal"
	Version       string `json:"version,omitempty"`       // IDE version when detectable
	WorkspacePath string `json:"workspacePath,omitempty"` // project root (PWD at session start)
}

// DetectIDE inspects environment variables to determine the IDE or runtime that
// launched Claude Code. Detection is ordered from most to least specific.
func DetectIDE() IDEInfo {
	workspace, _ := os.Getwd()

	// 1. Claude Code desktop app — sets CLAUDE_CODE_ENTRYPOINT when running as GUI app.
	if ep := os.Getenv("CLAUDE_CODE_ENTRYPOINT"); ep != "" {
		switch strings.ToLower(ep) {
		case "desktop", "app":
			return IDEInfo{Name: "desktop", WorkspacePath: workspace}
		case "vscode", "vscode-extension":
			return IDEInfo{Name: "vscode", Version: vscodeVersion(), WorkspacePath: workspace}
		case "jetbrains", "jetbrains-extension":
			return IDEInfo{Name: "jetbrains", WorkspacePath: workspace}
		}
	}

	// 2. VS Code — VSCODE_PID is set by the VS Code extension host process.
	if os.Getenv("VSCODE_PID") != "" || os.Getenv("VSCODE_IPC_HOOK") != "" || os.Getenv("VSCODE_IPC_HOOK_CLI") != "" {
		return IDEInfo{Name: "vscode", Version: vscodeVersion(), WorkspacePath: workspace}
	}

	// 3. TERM_PROGRAM — set by VS Code, Cursor, and some other terminals.
	switch strings.ToLower(os.Getenv("TERM_PROGRAM")) {
	case "vscode":
		return IDEInfo{Name: "vscode", Version: vscodeVersion(), WorkspacePath: workspace}
	case "cursor":
		return IDEInfo{Name: "cursor", WorkspacePath: workspace}
	}

	// 4. JetBrains IDEs — JediTerm is the built-in terminal emulator for all JetBrains products.
	if strings.Contains(os.Getenv("TERMINAL_EMULATOR"), "JetBrains") ||
		os.Getenv("INTELLIJ_ENVIRONMENT_READER") != "" ||
		os.Getenv("JETBRAINS_REMOTE_DEV_PROJECT_DATA_DIRS") != "" {
		return IDEInfo{Name: "jetbrains", WorkspacePath: workspace}
	}

	// 5. Windsurf (Codeium)
	if strings.ToLower(os.Getenv("TERM_PROGRAM")) == "windsurf" ||
		os.Getenv("WINDSURF_PID") != "" {
		return IDEInfo{Name: "windsurf", WorkspacePath: workspace}
	}

	// 6. Claude Code web (claude.ai/code) — no reliable env var; leave as terminal.

	return IDEInfo{Name: "terminal", WorkspacePath: workspace}
}

// vscodeVersion extracts the VS Code version from VSCODE_NLS_CONFIG env var.
// The var is a JSON blob: {"_languagePackVersion":"1.86.0",...}.
// Falls back to TERM_PROGRAM_VERSION if available.
func vscodeVersion() string {
	if v := os.Getenv("TERM_PROGRAM_VERSION"); v != "" {
		return v
	}
	raw := os.Getenv("VSCODE_NLS_CONFIG")
	if raw == "" {
		return ""
	}
	var cfg map[string]any
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return ""
	}
	for _, key := range []string{"_languagePackVersion", "languagePackVersion", "_version"} {
		if v, ok := cfg[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}
