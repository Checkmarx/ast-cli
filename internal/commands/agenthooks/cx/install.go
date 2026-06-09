package cx

import (
	"github.com/CheckmarxDev/ast-cx-hooks/install"
)

// Route is a single hook dispatch route exposed as a hidden `cx hooks <Use>`
// subcommand. The Use string is what the agent invokes; Short is the cobra
// short description.
type Route struct {
	Use   string
	Short string
}

// Agent describes one supported AI coding agent: how to install its hook
// config and which dispatch routes its hooks invoke. The Routes list must
// match the route names the corresponding Install function writes into the
// agent's config file — the canonical installers live in ast-cx-hooks's
// install package; here we just thread the cx-CLI route prefix through.
type Agent struct {
	ID          string
	DisplayName string
	ConfigPath  string
	Install     func(home string, cmdFor install.CmdForFunc) error
	Routes      []Route
}

// Agents is the single source of truth for supported AI coding agents.
// Adding a new agent is one entry here plus one installer in ast-cx-hooks.
var Agents = []Agent{
	{
		ID:          "claude",
		DisplayName: "Claude Code",
		ConfigPath:  "~/.claude/settings.json",
		Install:     install.InstallClaude,
		Routes: []Route{
			{"claude-stop", "Claude Code agent finished"},
			{"claude-pre-tool-use", "Gate Claude Code tool use"},
			{"claude-pre-file-write", "Gate Claude Code file write"},
			{"claude-user-prompt-submit", "Gate Claude Code prompt"},
		},
	},
	{
		ID:          "cursor",
		DisplayName: "Cursor",
		ConfigPath:  "~/.cursor/hooks.json",
		Install:     install.InstallCursor,
		Routes: []Route{
			{"cursor-stop", "Cursor agent finished"},
			{"cursor-before-shell", "Gate Cursor shell execution"},
			{"cursor-before-mcp", "Gate Cursor MCP execution"},
			{"cursor-before-file-read", "Gate Cursor file read"},
			{"cursor-after-file-edit", "React to Cursor file edit"},
			{"cursor-before-submit-prompt", "Gate Cursor prompt"},
		},
	},
	{
		ID:          "windsurf",
		DisplayName: "Windsurf",
		ConfigPath:  "~/.codeium/windsurf/hooks.json",
		Install:     install.InstallWindsurf,
		Routes: []Route{
			{"windsurf-pre-run-command", "Gate Windsurf shell execution"},
			{"windsurf-pre-mcp-tool-use", "Gate Windsurf MCP execution"},
			{"windsurf-pre-user-prompt", "Gate Windsurf prompt"},
			{"windsurf-pre-write-code", "Gate Windsurf file write"},
			{"windsurf-post-cascade-response", "Windsurf agent finished"},
		},
	},
	{
		ID:          "droid",
		DisplayName: "Factory Droid",
		ConfigPath:  "~/.factory/settings.json",
		Install:     install.InstallDroid,
		Routes: []Route{
			{"droid-stop", "Factory Droid agent finished"},
			{"droid-pre-tool-use", "Gate Factory Droid tool use"},
			{"droid-pre-file-write", "Gate Factory Droid file write"},
			{"droid-user-prompt-submit", "Gate Factory Droid prompt"},
		},
	},
	{
		ID:          "gemini",
		DisplayName: "Gemini CLI",
		ConfigPath:  "~/.gemini/settings.json",
		Install:     install.InstallGemini,
		Routes: []Route{
			{"gemini-before-agent", "Gemini CLI agent starting"},
			{"gemini-before-tool", "Gate Gemini CLI tool execution"},
			{"gemini-before-file-tool", "Gate Gemini CLI file write"},
			{"gemini-after-agent", "Gemini CLI agent finished"},
		},
	},
}

// FindAgent returns the Agent with the given ID, or nil if not found.
func FindAgent(id string) *Agent {
	for i := range Agents {
		if Agents[i].ID == id {
			return &Agents[i]
		}
	}
	return nil
}

// CxCmdFor returns a CmdForFunc that maps each route to the shell command
// "<cx> hooks <route>" — the form ast-cli uses when wiring its own routes
// into agent config files.
func CxCmdFor(cxPath string) install.CmdForFunc {
	return func(route string) string {
		return install.FormatCommand(cxPath, "hooks", route)
	}
}
