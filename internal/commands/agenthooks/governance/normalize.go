package governance

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ToolInfo classifies a single tool invocation for policy evaluation.
type ToolInfo struct {
	Category string   `json:"category"`          // "filesystem" | "shell" | "mcp" | "web_search" | "other"
	Name     string   `json:"name"`              // tool name shown in logs and events
	Command  string   `json:"command,omitempty"` // shell: full command string
	Paths    []string `json:"paths,omitempty"`   // filesystem: accessed file paths
	Query    string   `json:"query,omitempty"`   // web_search: query or URL
	RawName  string   `json:"rawName,omitempty"` // original tool_name as sent by the agent
	MCP      *MCPInfo `json:"mcp,omitempty"`     // populated for category = "mcp"
}

// MCPInfo identifies an MCP server and the tool within it.
type MCPInfo struct {
	Server string `json:"server"` // MCP server name (first segment after "mcp__")
	Name   string `json:"name"`   // tool name within the server
}

// AgentInfo identifies the AI coding agent that fired the hook.
type AgentInfo struct {
	Type string `json:"type"` // "claude-code" | "cursor"
	ID   string `json:"id"`   // governance agentId (SHA256)
}

// PromptInfo carries prompt metadata.
// When PII is detected, RedactedText holds the masked prompt; raw content is never stored.
type PromptInfo struct {
	Length       int            `json:"length"` // prompt length in bytes
	PIIDetected  bool           `json:"piiDetected,omitempty"`
	PromptText   string         `json:"promptText,omitempty"` // redacted prompt, set only when PII found
	PIIMatches   []PIIMatchMeta `json:"piiMatches,omitempty"`
}

// RuntimeEvent is the normalized in-process view of a governance hook event.
// Built once per hook invocation; carries all data needed for evaluation and logging.
type RuntimeEvent struct {
	EventID   string
	Timestamp time.Time
	Kind      string // "session.start" | "tool.pre" | "tool.post" | "prompt.submit" | "session.end"
	SessionID string
	MachineID string
	UserID    string // OS username — used for agentId derivation, never written to local logs
	Agent     AgentInfo
	Tool      ToolInfo
	Prompt    PromptInfo
	IDE       IDEInfo
	ModelInfo *ModelDetectionResult
}

// ActivityEvent is the serialized form written to the audit spool and sent to the backend.
type ActivityEvent struct {
	EventID   string      `json:"eventId"`
	Timestamp time.Time   `json:"timestamp"`
	Kind      string      `json:"kind"`
	SessionID string      `json:"sessionId"`
	MachineID string      `json:"machineId"`
	AgentID   string      `json:"agentId"`
	UserID    string      `json:"userId"`
	Agent     AgentInfo   `json:"agent"`
	Tool      ToolInfo    `json:"tool,omitempty"`
	Prompt    *PromptInfo `json:"prompt,omitempty"`
	Decision  DecisionInfo `json:"decision"`
	IDE       IDEInfo     `json:"ide,omitempty"`
	ModelInfo *ModelDetectionResult `json:"modelInfo,omitempty"`
	// Third-party enforcement fields — set when another hook's action is detected.
	ThirdPartyTool           string `json:"thirdPartyTool,omitempty"`
	ThirdPartyAction         string `json:"thirdPartyAction,omitempty"`           // "block" | "warn"
	AttributionConfidence    string `json:"attributionConfidence,omitempty"`      // "definitive" | "candidate"
}

// DecisionInfo is the policy evaluation result serialized into an ActivityEvent.
type DecisionInfo struct {
	Action        string `json:"action"`                  // "ALLOW" | "ALERT" | "BLOCK"
	PolicyID      string `json:"policyId,omitempty"`
	PolicyName    string `json:"policyName,omitempty"`
	PolicyVersion int    `json:"policyVersion,omitempty"`
	PackVersion   int    `json:"packVersion,omitempty"`
	MatchedOn     string `json:"matchedOn,omitempty"`     // "path" | "command" | "mcp_tool" | "query" | "prompt"
	MatchedValue  string `json:"matchedValue,omitempty"`
	AgentMessage  string `json:"agentMessage,omitempty"`
	LatencyMs     int64  `json:"latencyMs,omitempty"`
}

// WithDecision attaches a Decision to a RuntimeEvent and returns a spoolable ActivityEvent.
func (e RuntimeEvent) WithDecision(d Decision) ActivityEvent {
	var prompt *PromptInfo
	if e.Kind == "prompt.submit" {
		p := e.Prompt
		prompt = &p
	}
	return ActivityEvent{
		EventID:   e.EventID,
		Timestamp: e.Timestamp,
		Kind:      e.Kind,
		SessionID: e.SessionID,
		MachineID: e.MachineID,
		AgentID:   AgentID(e.Agent.Type),
		UserID:    e.UserID,
		Agent:     e.Agent,
		Tool:      e.Tool,
		Prompt:    prompt,
		IDE:       e.IDE,
		ModelInfo: e.ModelInfo,
		Decision: DecisionInfo{
			Action:        d.Action,
			PolicyID:      d.PolicyID,
			PolicyName:    d.PolicyName,
			PolicyVersion: d.PolicyVersion,
			PackVersion:   d.PackVersion,
			MatchedOn:     d.MatchedOn,
			MatchedValue:  d.MatchedValue,
			AgentMessage:  d.AgentMessage,
			LatencyMs:     d.LatencyMs,
		},
	}
}

// newEventID generates a unique event identifier for the audit trail.
func newEventID() string {
	return "evt_" + uuid.New().String()
}

// buildAgentInfo constructs an AgentInfo from the governance config.
func buildAgentInfo(cfg GovernanceConfig) AgentInfo {
	return AgentInfo{
		Type: cfg.AgentType,
		ID:   AgentID(cfg.AgentType),
	}
}

// normalizeTool maps a Claude tool_name + raw JSON input to a ToolInfo.
func normalizeTool(toolName string, rawInput json.RawMessage) ToolInfo {
	switch {
	case isFilesystemTool(toolName):
		return ToolInfo{
			Category: "filesystem",
			Name:     toolName,
			RawName:  toolName,
			Paths:    extractPaths(rawInput),
		}
	case toolName == "Bash":
		return ToolInfo{
			Category: "shell",
			Name:     "Bash",
			RawName:  "Bash",
			Command:  extractCommand(rawInput),
		}
	case strings.HasPrefix(toolName, "mcp__"):
		server, tool := parseMCPName(toolName)
		return ToolInfo{
			Category: "mcp",
			Name:     tool,
			RawName:  toolName,
			MCP:      &MCPInfo{Server: server, Name: tool},
		}
	case toolName == "WebSearch" || toolName == "WebFetch":
		return ToolInfo{
			Category: "web_search",
			Name:     toolName,
			RawName:  toolName,
			Query:    extractQuery(rawInput),
		}
	case toolName == "Skill":
		skillName := extractSkillName(rawInput)
		if skillName == "" {
			skillName = "unknown"
		}
		return ToolInfo{
			Category: "skill",
			Name:     skillName,
			RawName:  toolName,
		}
	default:
		return ToolInfo{Category: "other", Name: toolName, RawName: toolName}
	}
}

// isFilesystemTool returns true for Claude's built-in file operation tools.
func isFilesystemTool(name string) bool {
	switch name {
	case "Read", "Write", "Edit", "MultiEdit", "Glob", "Grep", "LS":
		return true
	}
	return false
}

// parseMCPName splits "mcp__<server>__<tool>" into (server, tool).
func parseMCPName(name string) (server, tool string) {
	without := strings.TrimPrefix(name, "mcp__")
	parts := strings.SplitN(without, "__", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return without, without
}

func extractCommand(raw json.RawMessage) string {
	var input struct {
		Command string `json:"command"`
	}
	_ = json.Unmarshal(raw, &input)
	return input.Command
}

func extractPaths(raw json.RawMessage) []string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	for _, key := range []string{"file_path", "path", "pattern", "directory"} {
		if v, ok := m[key]; ok {
			var s string
			if err := json.Unmarshal(v, &s); err == nil && s != "" {
				return []string{s}
			}
		}
	}
	if v, ok := m["paths"]; ok {
		var paths []string
		if err := json.Unmarshal(v, &paths); err == nil {
			return paths
		}
	}
	return nil
}

// extractSkillName extracts the skill name from Skill tool input.
// Tries multiple field names to handle different Claude Code versions:
// {"skill":"name"}, {"subcommand":"name"}, {"name":"name"}, {"command_name":"name"}
func extractSkillName(raw json.RawMessage) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	for _, key := range []string{"skill", "subcommand", "name", "command_name"} {
		if v, ok := m[key]; ok {
			var s string
			if err := json.Unmarshal(v, &s); err == nil && s != "" {
				return s
			}
		}
	}
	return ""
}

func extractQuery(raw json.RawMessage) string {
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	for _, key := range []string{"query", "url"} {
		if v, ok := m[key]; ok && v != "" {
			return v
		}
	}
	return ""
}
