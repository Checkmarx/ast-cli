package governance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// toShellPath converts an absolute path to a form that works when Claude Code
// fires hooks through /usr/bin/bash (Git for Windows / MSYS2).
// On Windows: C:\Users\foo\bar.exe → /c/Users/foo/bar.exe
// On other platforms: no change.
func toShellPath(p string) string {
	if runtime.GOOS != "windows" {
		return p
	}
	// Replace backslashes with forward slashes first.
	p = strings.ReplaceAll(p, `\`, "/")
	// Convert drive letter: C:/... → /c/...
	if len(p) >= 2 && p[1] == ':' {
		drive := strings.ToLower(string(p[0]))
		p = "/" + drive + p[2:]
	}
	return p
}

// InstallGovernanceClaude writes governance hook routes into ~/.claude/settings.json.
// It preserves all existing entries — governance hooks are appended, never replacing
// security guardrail hooks or any user-configured hooks.
//
// ALL entries use the {"matcher","hooks":[...]} format — Claude Code requires this
// shape for every hook event, including SessionStart, Stop, and UserPromptSubmit.
// An empty matcher string matches all (equivalent to ".*" for non-tool events).
// The cx binary path is converted to a bash-compatible forward-slash form so that
// Claude Code can invoke it through /usr/bin/bash on Windows (Git Bash / MSYS2).
func InstallGovernanceClaude(home, cxPath string) error {
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	// Quoted so paths containing spaces (e.g. "AI Governance") survive
	// tokenization when Claude Code invokes the command through bash.
	shellCxPath := fmt.Sprintf("%q", toShellPath(cxPath))
	return patchSettingsFile(settingsPath, func(m map[string]any) {
		hooks := ensureSettingsMap(m, "hooks")

		// All events use matcher+hooks format. Empty matcher = match all.
		appendMatcherHookCommand(hooks, "SessionStart", "",
			fmt.Sprintf("%s hooks gov-claude-session-start", shellCxPath))

		appendMatcherHookCommand(hooks, "PreToolUse", ".*",
			fmt.Sprintf("%s hooks gov-claude-pre-tool-use", shellCxPath))

		appendMatcherHookCommand(hooks, "PostToolUse", ".*",
			fmt.Sprintf("%s hooks gov-claude-post-tool-use", shellCxPath))

		appendMatcherHookCommand(hooks, "UserPromptSubmit", "",
			fmt.Sprintf("%s hooks gov-claude-prompt-submit", shellCxPath))

		// UserPromptExpansion fires when a slash command skill is invoked (e.g. /db-status).
		appendMatcherHookCommand(hooks, "UserPromptExpansion", "",
			fmt.Sprintf("%s hooks gov-claude-prompt-expansion", shellCxPath))

		// SessionEnd fires when the Claude Code session is actually closed.
		// Stop fires after every response (end of turn) — do NOT use it for session-end.
		appendMatcherHookCommand(hooks, "SessionEnd", "",
			fmt.Sprintf("%s hooks gov-claude-session-end", shellCxPath))
	})
}

// InstallGovernanceCursor writes governance hook routes into ~/.cursor/hooks.json.
//
// Cursor format: governance uses a top-level "hooks" array ({"event","command"} pairs)
// which coexists with the flat event-key entries written by security guardrails.
// Architecture §8.4 specifies this array format for governance-specific Cursor hooks.
func InstallGovernanceCursor(home, cxPath string) error {
	hookPath := filepath.Join(home, ".cursor", "hooks.json")
	// Quoted so paths containing spaces (e.g. "AI Governance") survive
	// tokenization when Cursor invokes the command through a shell.
	quotedCxPath := fmt.Sprintf("%q", cxPath)
	return patchSettingsFile(hookPath, func(m map[string]any) {
		arr := toAnySlice(m["hooks"])
		govEntries := []struct{ event, cmd string }{
			{"session-start",        fmt.Sprintf("%s hooks gov-cursor-session-start", quotedCxPath)},
			{"before-shell",         fmt.Sprintf("%s hooks gov-cursor-before-shell", quotedCxPath)},
			{"before-mcp",           fmt.Sprintf("%s hooks gov-cursor-before-mcp", quotedCxPath)},
			{"before-submit-prompt", fmt.Sprintf("%s hooks gov-cursor-before-submit-prompt", quotedCxPath)},
			{"session-end",          fmt.Sprintf("%s hooks gov-cursor-session-end", quotedCxPath)},
		}
		for _, e := range govEntries {
			if !containsEventCommand(arr, e.event, e.cmd) {
				arr = append(arr, map[string]any{"event": e.event, "command": e.cmd})
			}
		}
		m["hooks"] = arr
	})
}

// patchSettingsFile reads the settings file (creating it if absent), applies patch,
// and writes it back atomically via a .tmp rename. Aborts without writing if the
// existing file contains invalid JSON, and writes a .bak backup before patching.
func patchSettingsFile(path string, patch func(map[string]any)) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("governance install: create dir: %w", err)
	}
	m := map[string]any{}
	if data, err := os.ReadFile(path); err == nil && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("governance install: refusing to overwrite %s — not valid JSON: %w", path, err)
		}
		if err := os.WriteFile(path+".bak", data, 0o600); err != nil {
			return fmt.Errorf("governance install: writing backup %s.bak: %w", path, err)
		}
	}
	patch(m)
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	// Append newline. Do NOT prepend a BOM — JSON parsers reject it.
	content := append(out, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// appendHookCommand appends {"type":"command","command":cmd} to hooks[eventKey] if
// the command is not already present. Existing entries are preserved.
func appendHookCommand(hooks map[string]any, eventKey, cmd string) {
	arr := toAnySlice(hooks[eventKey])
	if containsCommand(arr, cmd) {
		return
	}
	hooks[eventKey] = append(arr, map[string]any{"type": "command", "command": cmd})
}

// appendMatcherHookCommand appends a nested {"matcher":matcher,"hooks":[{type,command}]}
// entry to hooks[eventKey] if the command is not already present anywhere in the array.
func appendMatcherHookCommand(hooks map[string]any, eventKey, matcher, cmd string) {
	arr := toAnySlice(hooks[eventKey])
	if containsCommand(arr, cmd) {
		return
	}
	hooks[eventKey] = append(arr, map[string]any{
		"matcher": matcher,
		"hooks":   []any{map[string]any{"type": "command", "command": cmd}},
	})
}

// ensureSettingsMap returns the map at key, creating it if absent.
func ensureSettingsMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if sub, ok := v.(map[string]any); ok {
			return sub
		}
	}
	sub := map[string]any{}
	m[key] = sub
	return sub
}

func toAnySlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return nil
}

// containsCommand recursively checks whether cmd is already present in the array.
// Handles both plain {"type":"command","command":cmd} and nested {"matcher":…,"hooks":[…]} forms.
func containsCommand(arr []any, cmd string) bool {
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if m["command"] == cmd {
			return true
		}
		if inner, ok := m["hooks"].([]any); ok && containsCommand(inner, cmd) {
			return true
		}
	}
	return false
}

// containsEventCommand checks if the Cursor-style hooks array already has an entry
// for the given event with the given command.
func containsEventCommand(arr []any, event, cmd string) bool {
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if m["event"] == event && m["command"] == cmd {
			return true
		}
	}
	return false
}
