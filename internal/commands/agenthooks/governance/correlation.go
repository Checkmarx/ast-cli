package governance

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)


// Correlation tracks third-party hook enforcement via file-based state.
//
// Flow:
//   PreToolUse (our verdict=ALLOW) → writes .correlation/<id>.pre.json
//   PostToolUse                    → writes .correlation/<id>.post (empty marker)
//   Next hook call                 → sweepOrphanedPre() finds .pre.json with no .post after 5s
//                                  → emits tool.third_party_enforcement events
//
// Also handles Mechanism B: cx-guardrails WARN delivered via
// .hook-verdicts/<tool_use_id>.json written by the guardrail hook.

const correlationWindow = 5 * time.Second

// preHookState is persisted by PreToolUse when our verdict is ALLOW.
type preHookState struct {
	ToolUseID  string    `json:"toolUseId"`
	ToolName   string    `json:"toolName"`
	ToolInput  string    `json:"toolInput"`
	SessionID  string    `json:"sessionId"`
	Timestamp  time.Time `json:"timestamp"`
}

// hookVerdict is written by cx-guardrails (Mechanism B) to .hook-verdicts/<id>.json.
type hookVerdict struct {
	Source  string `json:"source"`  // "cx-guardrails"
	Verdict string `json:"verdict"` // "warn" | "block"
	Reason  string `json:"reason"`
	Event   string `json:"event"` // "PreToolUse:Bash"
}

// writePreHookState records that our PreToolUse verdict was ALLOW for this tool use.
// Called only when our own policy returned ALLOW — if we block it, there's no ambiguity.
func writePreHookState(toolUseID string, state preHookState) {
	dir := correlationDir()
	ensureDir(dir)
	data, err := json.Marshal(state)
	if err != nil {
		return
	}
	safeID := sanitizeID(toolUseID)
	_ = os.WriteFile(filepath.Join(dir, safeID+".pre.json"), data, 0o600)
}

// markPostToolUseSeen writes an empty marker so the sweeper knows PostToolUse fired.
func markPostToolUseSeen(toolUseID string) {
	dir := correlationDir()
	ensureDir(dir)
	safeID := sanitizeID(toolUseID)
	_ = os.WriteFile(filepath.Join(dir, safeID+".post"), []byte{}, 0o600)
}

// readAndDeleteHookVerdict reads the cx-guardrails verdict file (Mechanism B).
// Returns nil when no verdict file exists for this tool use.
// Uses atomic rename-then-read to prevent TOCTOU between concurrent PostToolUse processes.
func readAndDeleteHookVerdict(toolUseID string) *hookVerdict {
	safeID := sanitizeID(toolUseID)
	path := filepath.Join(hookVerdictsDir(), safeID+".json")
	tmp := path + ".read"
	if err := os.Rename(path, tmp); err != nil {
		return nil
	}
	data, err := os.ReadFile(tmp)
	_ = os.Remove(tmp)
	if err != nil {
		return nil
	}
	var v hookVerdict
	if err := json.Unmarshal(data, &v); err != nil {
		return nil
	}
	return &v
}

// sweepOrphanedPre scans the correlation directory for .pre.json files that have
// no corresponding .post file and are older than correlationWindow.
// For each orphan, it emits a tool.third_party_enforcement event and deletes the file.
// Call at the start of any hook handler to catch blocks from the previous tool call.
func sweepOrphanedPre(cfg GovernanceConfig) {
	dir := correlationDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	sessionHooks := loadSessionHooks(cfg)

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".pre.json") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) < correlationWindow {
			continue // too recent — PostToolUse may still fire
		}

		prePath := filepath.Join(dir, entry.Name())
		safeID := strings.TrimSuffix(entry.Name(), ".pre.json")
		postPath := filepath.Join(dir, safeID+".post")

		// If .post exists: tool ran, not blocked — clean up both files.
		if _, err := os.Stat(postPath); err == nil {
			_ = os.Remove(prePath)
			_ = os.Remove(postPath)
			continue
		}

		// No .post → tool was blocked by another hook.
		data, err := os.ReadFile(prePath)
		if err != nil {
			_ = os.Remove(prePath)
			continue
		}
		var state preHookState
		if err := json.Unmarshal(data, &state); err != nil {
			_ = os.Remove(prePath)
			continue
		}
		_ = os.Remove(prePath)

		emitThirdPartyBlock(state, sessionHooks, cfg)
	}
}

// emitThirdPartyBlock spools a tool.third_party_enforcement event for an inferred block.
func emitThirdPartyBlock(state preHookState, sessionHooks []sessionHookEntry, cfg GovernanceConfig) {
	candidates := filterPreToolHooks(sessionHooks)
	confidence := "candidate"
	hookName := ""

	switch len(candidates) {
	case 0:
		// No session hook inventory — can't attribute; leave name empty.
	case 1:
		confidence = "definitive"
		hookName = hookDisplayName(candidates[0])
	default:
		// Multiple candidates — join them all.
		var names []string
		for _, c := range candidates {
			names = append(names, hookDisplayName(c))
		}
		hookName = strings.Join(names, ", ")
	}

	// Carry the blocked Claude tool (e.g., "Bash", "Write") so the UI can show
	// what was stopped, independent of who stopped it.
	blockedTool := state.ToolName

	agentMsg := "Third-party hook blocked " + blockedTool
	if hookName != "" {
		agentMsg = hookName + " blocked " + blockedTool
	}

	// Enhanced attribution: if no hooks found but devAssist is likely (security-related blocks),
	// infer cx-devassist as the blocker
	thirdPartyTool := hookName
	attributionConfidence := confidence
	if thirdPartyTool == "" && blockedTool == "Write" {
		// Write blocks are typically from security tools like devAssist
		thirdPartyTool = "cx-devassist"
		attributionConfidence = "candidate"
	} else if thirdPartyTool == "" {
		thirdPartyTool = "unknown-hook"
		attributionConfidence = "candidate"
	}

	event := ActivityEvent{
		EventID:   newEventID(),
		Timestamp: time.Now().UTC(),
		Kind:      "tool.third_party_enforcement",
		SessionID: state.SessionID,
		MachineID: MachineID(),
		AgentID:   AgentID(cfg.AgentType),
		UserID:    CurrentUserName(),
		Agent:     AgentInfo{Type: cfg.AgentType, ID: AgentID(cfg.AgentType)},
		// What was blocked — the Claude tool that was stopped.
		Tool: ToolInfo{
			Category: "third_party",
			Name:     blockedTool,
			RawName:  blockedTool,
		},
		ThirdPartyTool:        thirdPartyTool,
		ThirdPartyAction:      "block",
		AttributionConfidence: attributionConfidence,
		Decision: DecisionInfo{
			Action:       "BLOCK",
			AgentMessage: agentMsg,
		},
	}

	log.Printf("governance: third-party block detected for tool_use %s hook=%q blocked=%q (confidence=%s)",
		state.ToolUseID, hookName, blockedTool, confidence)
	Write(event)
	FlushOnce(cfg.ServerURL, cfg.Token, time.Second)
}

// hookDisplayName returns the most human-readable name for a session hook entry.
// Priority: Vendor > binary name from Command > route name from Command args > raw Command.
func hookDisplayName(h sessionHookEntry) string {
	if h.Vendor != "" && h.Vendor != "unknown" {
		return h.Vendor
	}
	if h.Command == "" {
		return "unknown hook"
	}
	// Extract binary name from the command path.
	parts := strings.Fields(h.Command)
	binary := filepath.Base(parts[0])
	binary = strings.TrimSuffix(binary, ".exe")

	// If the binary is "cx", append the route name (last arg) for specificity.
	// e.g., "cx hooks claude-pre-tool-use" → "cx (claude-pre-tool-use)"
	if (binary == "cx" || binary == "cx.exe") && len(parts) > 1 {
		route := parts[len(parts)-1]
		return binary + " (" + route + ")"
	}
	return binary
}

// filterPreToolHooks returns only PreToolUse hooks from the session inventory.
func filterPreToolHooks(hooks []sessionHookEntry) []sessionHookEntry {
	var out []sessionHookEntry
	for _, h := range hooks {
		if strings.Contains(strings.ToLower(h.Event), "pretooluse") {
			out = append(out, h)
		}
	}
	return out
}
