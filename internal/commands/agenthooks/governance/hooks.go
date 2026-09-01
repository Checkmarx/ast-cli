package governance

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/Checkmarx/ast-cx-hooks/claude"
	"github.com/Checkmarx/ast-cx-hooks/cursor"
)

// safeIDPattern allows only alphanumeric, dash, and underscore — characters a
// standards-compliant UUID or opaque token would contain. Prevents a crafted
// ToolUseID from traversing outside the pending-alerts directory.
var safeIDPattern = regexp.MustCompile(`[^a-zA-Z0-9_\-]`)

// sanitizeID strips any characters outside the safe set. Returns "unknown" if
// nothing safe remains.
func sanitizeID(id string) string {
	safe := safeIDPattern.ReplaceAllString(id, "")
	if safe == "" {
		return "unknown"
	}
	return safe
}

// ============================================================
// Claude governance routes
// ============================================================

// RegisterGovernanceHooks wires all Claude governance route handlers.
// Routes use the "gov-claude-" prefix to avoid colliding with security
// guardrail routes (claude-pre-tool-use, etc.) that coexist in the same binary.
func RegisterGovernanceHooks(cfg GovernanceConfig) {
	agenthooks.AddRoute("gov-claude-session-start", func() {
		agenthooks.Process(func(ev claude.SessionStartEvent) claude.SessionStartResult {
			return handleClaudeSessionStart(ev, cfg)
		})
	})
	agenthooks.AddRoute("gov-claude-pre-tool-use", func() {
		agenthooks.Process(func(ev claude.PreToolUseEvent) claude.PreToolUseResult {
			return handleClaudePreToolUse(ev, cfg)
		})
	})
	agenthooks.AddRoute("gov-claude-post-tool-use", func() {
		agenthooks.Process(func(ev claude.PostToolUseEvent) claude.PostToolUseResult {
			return handleClaudePostToolUse(ev, cfg)
		})
	})
	agenthooks.AddRoute("gov-claude-prompt-submit", func() {
		agenthooks.Process(func(ev claude.UserPromptSubmitEvent) claude.UserPromptSubmitResult {
			return handleClaudePromptSubmit(ev, cfg)
		})
	})
	// UserPromptExpansion fires when a slash command skill is invoked (e.g. /db-status).
	// This is the correct interception point for skills — the Skill PreToolUse fires only
	// for tool-type skills; prompt-type skills only fire UserPromptExpansion.
	agenthooks.AddRoute("gov-claude-prompt-expansion", func() {
		agenthooks.Process(func(ev claude.UserPromptExpansionEvent) claude.UserPromptExpansionResult {
			return handleClaudePromptExpansion(ev, cfg)
		})
	})
	// SessionEnd fires when the session is actually closed — NOT Stop, which
	// fires after every single Claude response (end of turn). Using Stop here
	// incorrectly marks the session as ended after the first prompt reply.
	agenthooks.AddRoute("gov-claude-session-end", func() {
		agenthooks.Process(func(ev claude.SessionEndEvent) claude.SessionEndResult {
			return handleClaudeSessionEnd(ev, cfg)
		})
	})
}

// ============================================================
// Cursor governance routes
// ============================================================

// RegisterGovernanceCursorHooks wires all Cursor governance route handlers.
// Routes use the "gov-cursor-" prefix for the same reason as Claude routes.
func RegisterGovernanceCursorHooks(cfg GovernanceConfig) {
	agenthooks.AddRoute("gov-cursor-session-start", func() {
		agenthooks.Process(func(ev cursor.SessionStartEvent) cursor.SessionStartResult {
			return handleCursorSessionStart(ev, cfg)
		})
	})
	agenthooks.AddRoute("gov-cursor-before-shell", func() {
		agenthooks.Process(func(ev cursor.ShellPreEvent) cursor.PermissionResult {
			return handleCursorShell(ev, cfg)
		})
	})
	agenthooks.AddRoute("gov-cursor-before-mcp", func() {
		agenthooks.Process(func(ev cursor.MCPPreEvent) cursor.PermissionResult {
			return handleCursorMCP(ev, cfg)
		})
	})
	agenthooks.AddRoute("gov-cursor-before-submit-prompt", func() {
		agenthooks.Process(func(ev cursor.PromptPreEvent) cursor.PromptPreResult {
			return handleCursorPrompt(ev, cfg)
		})
	})
	agenthooks.AddRoute("gov-cursor-session-end", func() {
		agenthooks.Process(func(ev cursor.SessionEndEvent) cursor.StopResult {
			return handleCursorSessionEnd(ev, cfg)
		})
	})
}

// ============================================================
// Claude handlers
// ============================================================

func handleClaudeSessionStart(ev claude.SessionStartEvent, cfg GovernanceConfig) claude.SessionStartResult {
	SyncOnSessionStart(cfg.ServerURL, cfg.Token)

	// Inventory third-party hooks from ~/.claude/settings.json and persist for
	// the correlation sweeper to use when attributing third-party blocks.
	thirdPartyHooks := readInstalledHooks()
	saveSessionState(ev.SessionID, thirdPartyHooks)

	// Detect Claude model being used. ev.Model is populated by Claude Code
	// itself in the SessionStart payload and reflects the actual running
	// model (including /model overrides), so it takes priority.
	modelResult := DetectModel(DetectionContext{
		UserID:    CurrentUserName(),
		AgentID:   AgentID(cfg.AgentType),
		DBClient:  nil,
		HookModel: ev.Model,
	})

	// Write session.start BEFORE flushing so this session is included in the flush.
	event := buildSessionEvent(ev.SessionID, "session.start", cfg)
	event.IDE = DetectIDE()
	event.ModelInfo = &modelResult
	WriteSessionLog(ev.SessionID, event, Decision{Action: ""})
	Write(event.WithDecision(Decision{Action: ""}))

	// Log model detection result
	if modelResult.Confidence < 0.80 {
		log.Printf("[WARN] Model detection: low confidence for session %s: model=%s source=%s confidence=%.2f errors=%v",
			ev.SessionID, modelResult.Model, modelResult.Source, modelResult.Confidence, modelResult.Errors)
	}

	// Emit hook inventory event so the backend can populate session_hooks table.
	if len(thirdPartyHooks) > 0 {
		type hookPayload struct {
			SessionID string             `json:"sessionId"`
			Hooks     []sessionHookEntry `json:"hooks"`
		}
		payload, err := json.Marshal(hookPayload{SessionID: ev.SessionID, Hooks: thirdPartyHooks})
		if err == nil {
			inventoryEvent := ActivityEvent{
				EventID:   newEventID(),
				Timestamp: time.Now().UTC(),
				Kind:      "session.hook_inventory",
				SessionID: ev.SessionID,
				MachineID: MachineID(),
				AgentID:   AgentID(cfg.AgentType),
				UserID:    CurrentUserName(),
				Agent:     buildAgentInfo(cfg),
				Decision: DecisionInfo{
					Action:       "",
					AgentMessage: string(payload),
				},
			}
			Write(inventoryEvent)
		}
	}

	FlushOnce(cfg.ServerURL, cfg.Token, flushTimeout)

	go EnsureResolved(cfg)
	go RegisterAgentWithBackend(cfg.AgentType, cfg)
	go PurgeStaleLogs()

	return claude.AcknowledgeSession()
}

func handleClaudePreToolUse(ev claude.PreToolUseEvent, cfg GovernanceConfig) claude.PreToolUseResult {
	// Sweep for third-party blocks from the previous tool call before processing this one.
	sweepOrphanedPre(cfg)

	start := time.Now()
	pack := Load()
	if pack == nil {
		return claude.ApproveToolUse() // fail-open
	}

	event := RuntimeEvent{
		EventID:   newEventID(),
		Timestamp: time.Now().UTC(),
		Kind:      "tool.pre",
		SessionID: ev.SessionID,
		MachineID: MachineID(),
		UserID:    CurrentUserName(),
		Agent:     buildAgentInfo(cfg),
		Tool:      normalizeTool(ev.ToolName, ev.ToolInput),
	}
	decision := pack.Evaluate(event)
	decision.LatencyMs = time.Since(start).Milliseconds()

	Write(event.WithDecision(decision))
	WriteSessionLog(ev.SessionID, event, decision)

	FlushOnce(cfg.ServerURL, cfg.Token, time.Second)
	go CheckAndSyncVersion(cfg.ServerURL, cfg.Token)

	switch decision.Action {
	case "BLOCK":
		return claude.DenyToolUse(decision.Message())
	case "ALERT":
		savePendingAlert(ev.ToolUseID, pendingAlertPayload{
			Decision:  decision,
			EventID:   event.EventID,
			ToolUseID: ev.ToolUseID,
		})
		return claude.ApproveToolUse()
	default:
		// Our verdict is ALLOW — record pre-hook state so the correlation sweeper can
		// detect if another hook blocked this tool call (PostToolUse won't fire if so).
		writePreHookState(ev.ToolUseID, preHookState{
			ToolUseID: ev.ToolUseID,
			ToolName:  ev.ToolName,
			SessionID: ev.SessionID,
			Timestamp: time.Now().UTC(),
		})
		return claude.ApproveToolUse()
	}
}

func handleClaudePostToolUse(ev claude.PostToolUseEvent, cfg GovernanceConfig) claude.PostToolUseResult {
	// Mark that PostToolUse fired for this tool use — used by correlation sweeper
	// to confirm the tool actually ran (i.e., no other hook blocked it).
	markPostToolUseSeen(ev.ToolUseID)

	// Mechanism B: read cx-guardrails WARN verdict written to .hook-verdicts/<id>.json.
	// The guardrail hook writes this file when it issues a WARN/ALERT on a tool call.
	if verdict := readAndDeleteHookVerdict(ev.ToolUseID); verdict != nil {
		// verdict.Event is "PreToolUse:Bash" or "PreToolUse:Write" — extract the blocked tool.
		blockedTool := ev.ToolName // fall back to the tool name from the PostToolUse event
		if verdict.Event != "" {
			if idx := len("PreToolUse:"); len(verdict.Event) > idx {
				blockedTool = verdict.Event[idx:]
			}
		}
		hookName := verdict.Source
		if hookName == "" {
			hookName = "cx-guardrails" // Mechanism B is always cx-guardrails
		}
		agentMsg := verdict.Reason
		if agentMsg == "" {
			agentMsg = hookName + " " + verdict.Verdict + "ed " + blockedTool
		}
		thirdPartyEvent := ActivityEvent{
			EventID:   newEventID(),
			Timestamp: time.Now().UTC(),
			Kind:      "tool.third_party_enforcement",
			SessionID: ev.SessionID,
			MachineID: MachineID(),
			AgentID:   AgentID(cfg.AgentType),
			UserID:    CurrentUserName(),
			Agent:     AgentInfo{Type: cfg.AgentType, ID: AgentID(cfg.AgentType)},
			Tool: ToolInfo{
				Category: "third_party",
				Name:     blockedTool,
				RawName:  blockedTool,
			},
			ThirdPartyTool:        hookName,
			ThirdPartyAction:      verdict.Verdict,
			AttributionConfidence: "definitive",
			Decision: DecisionInfo{
				// Map verdict to action: block→BLOCK, anything else (warn)→ALERT
				Action:       verdictToAction(verdict.Verdict),
				AgentMessage: agentMsg,
			},
		}
		Write(thirdPartyEvent)
		FlushOnce(cfg.ServerURL, cfg.Token, time.Second)
	}

	// Deliver our own pending ALERT message if one was saved in PreToolUse.
	alert := popPendingAlert(ev.ToolUseID)
	if alert == nil {
		return claude.AcknowledgeToolUse()
	}
	return claude.AddToolContext("⚠ " + alert.Decision.Message())
}

func handleClaudePromptSubmit(ev claude.UserPromptSubmitEvent, cfg GovernanceConfig) claude.UserPromptSubmitResult {
	// Sweep for third-party blocks from any preceding tool call.
	sweepOrphanedPre(cfg)

	start := time.Now()
	pack := Load()
	if pack == nil {
		return claude.ApprovePrompt()
	}

	promptInfo := PromptInfo{Length: len(ev.Prompt)}

	// PII scan — only when policy is configured and enabled.
	var piiDecision string
	if pack.PIIPolicy != nil && pack.PIIPolicy.Enabled {
		scanner := NewPIIScanner(pack.PIIPolicy)
		result := scanner.Scan(ev.Prompt)
		if result.Found {
			promptInfo.PIIDetected = true
			promptInfo.PromptText = result.RedactedText // masked, never raw
			promptInfo.PIIMatches = result.Matches
			piiDecision = result.Decision // "ALERT" or "BLOCK"
		}
	}

	event := RuntimeEvent{
		EventID:   newEventID(),
		Timestamp: time.Now().UTC(),
		Kind:      "prompt.submit",
		SessionID: ev.SessionID,
		MachineID: MachineID(),
		UserID:    CurrentUserName(),
		Agent:     buildAgentInfo(cfg),
		Tool:      ToolInfo{Category: "prompt"},
		Prompt:    promptInfo,
	}
	decision := pack.Evaluate(event)

	// PII decision takes precedence over policy match when it is more severe.
	if piiDecision == "BLOCK" || (piiDecision == "ALERT" && decision.Action == "ALLOW") {
		decision.Action = piiDecision
		if decision.PolicyName == "" {
			decision.PolicyName = "PII Detection"
		}
		// MatchedOn must be a non-empty violation_type_enum value for the
		// policy_actions insert on the backend to succeed — a pure PII
		// verdict (no policy-pack rule matched) otherwise leaves this "".
		if decision.MatchedOn == "" {
			decision.MatchedOn = "prompt"
		}
	}
	decision.LatencyMs = time.Since(start).Milliseconds()

	Write(event.WithDecision(decision))
	WriteSessionLog(ev.SessionID, event, decision)

	FlushOnce(cfg.ServerURL, cfg.Token, time.Second)
	go CheckAndSyncVersion(cfg.ServerURL, cfg.Token)

	switch decision.Action {
	case "BLOCK":
		return claude.RejectPrompt(decision.Message())
	case "ALERT":
		// AppendToPrompt injects a warning note visible to the model without blocking the prompt.
		return claude.AppendToPrompt("⚠ Governance warning: " + decision.Message())
	default:
		return claude.ApprovePrompt()
	}
}

func handleClaudePromptExpansion(ev claude.UserPromptExpansionEvent, cfg GovernanceConfig) claude.UserPromptExpansionResult {
	// Only capture slash command expansions — not MCP prompt expansions.
	if ev.ExpansionType != "slash_command" || ev.CommandName == "" {
		return claude.UserPromptExpansionResult{}
	}

	pack := Load()
	if pack == nil {
		return claude.UserPromptExpansionResult{} // fail-open
	}

	event := RuntimeEvent{
		EventID:   newEventID(),
		Timestamp: time.Now().UTC(),
		Kind:      "tool.pre",
		SessionID: ev.SessionID,
		MachineID: MachineID(),
		UserID:    CurrentUserName(),
		Agent:     buildAgentInfo(cfg),
		Tool: ToolInfo{
			Category: "skill",
			Name:     ev.CommandName,
			RawName:  "Skill",
		},
	}
	decision := pack.Evaluate(event)
	decision.LatencyMs = 0

	Write(event.WithDecision(decision))
	WriteSessionLog(ev.SessionID, event, decision)
	FlushOnce(cfg.ServerURL, cfg.Token, time.Second)
	go CheckAndSyncVersion(cfg.ServerURL, cfg.Token)

	if decision.Action == "BLOCK" {
		return claude.UserPromptExpansionResult{
			Details: &claude.UserPromptExpansionDetails{
				ExtraContext: "⛔ Governance blocked skill: " + decision.Message(),
			},
		}
	}
	return claude.UserPromptExpansionResult{}
}

func handleClaudeSessionEnd(ev claude.SessionEndEvent, cfg GovernanceConfig) claude.SessionEndResult {
	event := buildSessionEvent(ev.SessionID, "session.end", cfg)
	WriteSessionLog(ev.SessionID, event, Decision{Action: ""})
	Write(event.WithDecision(Decision{Action: ""}))
	// Flush the spool before the process exits — this is the last chance
	// to drain events from this session without waiting for the next session start.
	FlushOnce(cfg.ServerURL, cfg.Token, flushTimeout)
	return claude.SessionEndResult{}
}

// ============================================================
// Cursor handlers
// ============================================================

func handleCursorSessionStart(ev cursor.SessionStartEvent, cfg GovernanceConfig) cursor.SessionStartResult {
	SyncOnSessionStart(cfg.ServerURL, cfg.Token)

	sessionID := ev.SessionID
	event := buildSessionEvent(sessionID, "session.start", cfg)
	WriteSessionLog(sessionID, event, Decision{Action: ""})
	Write(event.WithDecision(Decision{Action: ""}))

	FlushOnce(cfg.ServerURL, cfg.Token, flushTimeout)

	go EnsureResolved(cfg)
	go RegisterAgentWithBackend(cfg.AgentType, cfg)
	go PurgeStaleLogs()

	return cursor.AcknowledgeSession()
}

func handleCursorShell(ev cursor.ShellPreEvent, cfg GovernanceConfig) cursor.PermissionResult {
	start := time.Now()
	pack := Load()
	if pack == nil {
		return cursor.Permit()
	}

	event := RuntimeEvent{
		EventID:   newEventID(),
		Timestamp: time.Now().UTC(),
		Kind:      "tool.pre",
		SessionID: ev.ConversationID,
		MachineID: MachineID(),
		UserID:    CurrentUserName(),
		Agent:     buildAgentInfo(cfg),
		Tool:      ToolInfo{Category: "shell", Name: "Bash", Command: ev.Command},
	}
	decision := pack.Evaluate(event)
	decision.LatencyMs = time.Since(start).Milliseconds()

	Write(event.WithDecision(decision))
	WriteSessionLog(ev.ConversationID, event, decision)
	FlushOnce(cfg.ServerURL, cfg.Token, time.Second)
	go CheckAndSyncVersion(cfg.ServerURL, cfg.Token)

	if decision.Action == "BLOCK" {
		return cursor.Forbid(decision.Message(), decision.Message())
	}
	return cursor.Permit()
}

func handleCursorMCP(ev cursor.MCPPreEvent, cfg GovernanceConfig) cursor.PermissionResult {
	start := time.Now()
	pack := Load()
	if pack == nil {
		return cursor.Permit()
	}

	event := RuntimeEvent{
		EventID:   newEventID(),
		Timestamp: time.Now().UTC(),
		Kind:      "tool.pre",
		SessionID: ev.ConversationID,
		MachineID: MachineID(),
		UserID:    CurrentUserName(),
		Agent:     buildAgentInfo(cfg),
		Tool:      normalizeCursorMCPTool(ev),
	}
	decision := pack.Evaluate(event)
	decision.LatencyMs = time.Since(start).Milliseconds()

	Write(event.WithDecision(decision))
	WriteSessionLog(ev.ConversationID, event, decision)
	go CheckAndSyncVersion(cfg.ServerURL, cfg.Token)

	if decision.Action == "BLOCK" {
		return cursor.Forbid(decision.Message(), decision.Message())
	}
	return cursor.Permit()
}

func handleCursorPrompt(ev cursor.PromptPreEvent, cfg GovernanceConfig) cursor.PromptPreResult {
	start := time.Now()
	pack := Load()
	if pack == nil {
		return cursor.AcceptPrompt()
	}

	event := RuntimeEvent{
		EventID:   newEventID(),
		Timestamp: time.Now().UTC(),
		Kind:      "prompt.submit",
		SessionID: ev.ConversationID,
		MachineID: MachineID(),
		UserID:    CurrentUserName(),
		Agent:     buildAgentInfo(cfg),
		Tool:      ToolInfo{Category: "prompt"},
		Prompt:    PromptInfo{Length: len(ev.Prompt)},
	}
	decision := pack.Evaluate(event)
	decision.LatencyMs = time.Since(start).Milliseconds()

	Write(event.WithDecision(decision))
	WriteSessionLog(ev.ConversationID, event, decision)
	FlushOnce(cfg.ServerURL, cfg.Token, time.Second)
	go CheckAndSyncVersion(cfg.ServerURL, cfg.Token)

	if decision.Action == "BLOCK" {
		return cursor.BlockPrompt(decision.Message())
	}
	return cursor.AcceptPrompt()
}

func handleCursorSessionEnd(ev cursor.SessionEndEvent, cfg GovernanceConfig) cursor.StopResult {
	event := buildSessionEvent(ev.SessionID, "session.end", cfg)
	WriteSessionLog(ev.SessionID, event, Decision{Action: ""})
	Write(event.WithDecision(Decision{Action: ""}))
	FlushOnce(cfg.ServerURL, cfg.Token, flushTimeout)
	return cursor.LetStop()
}

// ============================================================
// Shared helpers
// ============================================================

// buildSessionEvent constructs a RuntimeEvent for session lifecycle events (no tool or prompt data).
func buildSessionEvent(sessionID, kind string, cfg GovernanceConfig) RuntimeEvent {
	return RuntimeEvent{
		EventID:   newEventID(),
		Timestamp: time.Now().UTC(),
		Kind:      kind,
		SessionID: sessionID,
		MachineID: MachineID(),
		UserID:    CurrentUserName(),
		Agent:     buildAgentInfo(cfg),
	}
}

// normalizeCursorMCPTool converts a Cursor MCPPreEvent into a governance ToolInfo.
// Cursor provides the tool name and server URL separately rather than as a combined
// mcp__<server>__<tool> string as Claude does.
func normalizeCursorMCPTool(ev cursor.MCPPreEvent) ToolInfo {
	server := ev.ServerURL
	if server == "" {
		server = ev.Command // stdio-based MCP servers use Command instead of URL
	}
	return ToolInfo{
		Category: "mcp",
		Name:     ev.ToolName,
		RawName:  "mcp__" + server + "__" + ev.ToolName,
		MCP:      &MCPInfo{Server: server, Name: ev.ToolName},
	}
}

// ============================================================
// Pending alert file store
// ============================================================
// PreToolUse and PostToolUse run in separate OS subprocesses, so the pending
// alert is persisted to disk rather than held in memory.

// pendingAlertPayload is the on-disk record bridging PreToolUse to PostToolUse.
type pendingAlertPayload struct {
	Decision  Decision `json:"decision"`
	EventID   string   `json:"eventId"`
	ToolUseID string   `json:"toolUseId"`
}

func savePendingAlert(toolUseID string, payload pendingAlertPayload) {
	dir := pendingAlertsDir()
	ensureDir(dir)
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("governance: pending alert marshal error: %v", err)
		return
	}
	safeID := sanitizeID(toolUseID)
	path := filepath.Join(dir, safeID+".json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		log.Printf("governance: pending alert write error: %v", err)
	}
}

// verdictToAction converts a third-party hook verdict string to a governance action.
func verdictToAction(verdict string) string {
	switch verdict {
	case "block", "BLOCK", "deny", "DENY":
		return "BLOCK"
	default:
		return "ALERT"
	}
}

func popPendingAlert(toolUseID string) *pendingAlertPayload {
	safeID := sanitizeID(toolUseID)
	path := filepath.Join(pendingAlertsDir(), safeID+".json")
	// Atomic rename-then-read prevents TOCTOU race between concurrent PostToolUse subprocesses.
	tmpPath := path + ".read"
	if err := os.Rename(path, tmpPath); err != nil {
		return nil
	}
	data, err := os.ReadFile(tmpPath)
	_ = os.Remove(tmpPath)
	if err != nil {
		return nil
	}
	var payload pendingAlertPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil
	}
	return &payload
}
