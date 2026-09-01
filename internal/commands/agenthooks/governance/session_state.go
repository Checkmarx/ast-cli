package governance

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// sessionHookEntry describes a single hook entry from the agent settings file.
type sessionHookEntry struct {
	Event   string `json:"event"`
	Matcher string `json:"matcher"`
	Command string `json:"command"`
	Vendor  string `json:"vendor"` // inferred from command, e.g. "cx-guardrails"
}

// sessionState is persisted at session start and read by the correlation sweeper.
type sessionState struct {
	SessionID   string             `json:"sessionId"`
	ThirdPartyHooks []sessionHookEntry `json:"thirdPartyHooks"`
}

// saveSessionState writes the session state to disk for the correlation sweeper.
func saveSessionState(sessionID string, hooks []sessionHookEntry) {
	dir := sessionStateDir()
	ensureDir(dir)
	state := sessionState{SessionID: sessionID, ThirdPartyHooks: hooks}
	data, err := json.Marshal(state)
	if err != nil {
		return
	}
	path := filepath.Join(dir, sanitizeID(sessionID)+".json")
	_ = os.WriteFile(path, data, 0o600)
}

// loadSessionHooks reads the session state for the most recent session.
// Used by the correlation sweeper to attribute third-party blocks.
func loadSessionHooks(cfg GovernanceConfig) []sessionHookEntry {
	_ = cfg // reserved for future per-agent state paths
	dir := sessionStateDir()
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return nil
	}
	// Use the most recently modified session state file.
	var latestPath string
	var latestTime int64
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().UnixNano() > latestTime {
			latestTime = info.ModTime().UnixNano()
			latestPath = filepath.Join(dir, e.Name())
		}
	}
	if latestPath == "" {
		return nil
	}
	data, err := os.ReadFile(latestPath)
	if err != nil {
		return nil
	}
	var state sessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil
	}
	return state.ThirdPartyHooks
}

// readInstalledHooks reads ~/.claude/settings.json and returns all non-governance hook entries.
// Claude settings.json stores hooks as an object keyed by event name:
//
//	{ "hooks": { "PreToolUse": [{matcher, hooks:[{command}]}], "PostToolUse": [...] } }
func readInstalledHooks() []sessionHookEntry {
	settingsPath := claudeSettingsPath()
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil
	}

	// Parse the object format: hooks is a map from event name → array of matcher entries.
	var raw struct {
		Hooks map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Printf("governance: settings.json parse error: %v", err)
		return nil
	}

	var result []sessionHookEntry
	for eventName, entries := range raw.Hooks {
		for _, entry := range entries {
			for _, h := range entry.Hooks {
				cmd := h.Command
				if cmd == "" {
					continue
				}
				// Skip our own governance routes.
				if strings.Contains(cmd, "gov-claude-") || strings.Contains(cmd, "gov-cursor-") {
					continue
				}
				result = append(result, sessionHookEntry{
					Event:   eventName,
					Matcher: entry.Matcher,
					Command: cmd,
					Vendor:  inferVendor(cmd),
				})
			}
		}
	}
	return result
}

// inferVendor guesses the vendor from the hook command string.
func inferVendor(cmd string) string {
	lower := strings.ToLower(cmd)
	switch {
	case strings.Contains(lower, "claude-pre-tool-use") || strings.Contains(lower, "claude-post-tool-use"):
		return "cx-guardrails"
	case strings.Contains(lower, "checkmarx") || strings.Contains(lower, "cx hooks"):
		return "cx-guardrails"
	case strings.Contains(lower, "aegis"):
		return "aegis"
	default:
		return ""
	}
}

// claudeSettingsPath returns the path to ~/.claude/settings.json.
func claudeSettingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(home, ".claude", "settings.json")
	}
	return filepath.Join(home, ".claude", "settings.json")
}
