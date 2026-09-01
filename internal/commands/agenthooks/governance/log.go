package governance

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

const logRetention = 30 * 24 * time.Hour

// SessionLogEntry is a single line in a PII-free local session log.
// Fields that could identify a person (userId, machineId, agentId, email,
// prompt content, web query text) are deliberately excluded.
type SessionLogEntry struct {
	Timestamp     time.Time `json:"ts"`
	EventID       string    `json:"eventId"`
	SessionID     string    `json:"sessionId"`
	Kind          string    `json:"kind"`           // "session.start" | "tool.pre" | "prompt.submit" | "session.end"
	AgentType     string    `json:"agentType"`      // "claude-code" | "cursor"
	Category      string    `json:"category"`       // "filesystem" | "shell" | "mcp" | "web_search" | "prompt" | ""
	ToolName      string    `json:"toolName"`       // tool name or ""
	Action        string    `json:"action"`         // "ALLOW" | "ALERT" | "BLOCK" | ""
	PolicyName    string    `json:"policyName"`     // policy that fired, "" for ALLOW
	PolicyVersion int       `json:"policyVersion"`  // 0 for ALLOW
	MatchedOn     string    `json:"matchedOn"`      // "path" | "command" | "mcp_tool" | "" — never prompt/query
	MatchedValue  string    `json:"matchedValue"`   // only for path/command/mcp_tool
}

// WriteSessionLog appends a PII-free entry to the local session log for this event.
// Called synchronously after every governance decision — local disk append, ~microseconds.
func WriteSessionLog(sessionID string, ev RuntimeEvent, d Decision) {
	entry := SessionLogEntry{
		Timestamp:     ev.Timestamp,
		EventID:       ev.EventID,
		SessionID:     sessionID,
		Kind:          ev.Kind,
		AgentType:     ev.Agent.Type,
		Category:      ev.Tool.Category,
		ToolName:      ev.Tool.Name,
		Action:        d.Action,
		PolicyName:    d.PolicyName,
		PolicyVersion: d.PolicyVersion,
		MatchedOn:     safeMatchedOn(ev, d),
		MatchedValue:  safeMatchedValue(ev, d),
	}
	appendToLog(sessionLogPath(sessionID, ev.Timestamp), entry)
}

// safeMatchedOn returns the matchedOn field only when the field is non-identifying.
// Prompt and query match targets are omitted from local logs.
func safeMatchedOn(ev RuntimeEvent, d Decision) string {
	if ev.Kind == "prompt.submit" {
		return ""
	}
	if d.MatchedOn == "query" {
		return ""
	}
	return d.MatchedOn
}

// safeMatchedValue returns the matched value only for non-PII match targets.
// Path, command, and mcp_tool values are safe; prompt content and query text are not.
func safeMatchedValue(ev RuntimeEvent, d Decision) string {
	if ev.Kind == "prompt.submit" {
		return ""
	}
	if d.MatchedOn == "query" || d.MatchedOn == "prompt" {
		return ""
	}
	return d.MatchedValue
}

// sessionLogPath constructs the log file path for a given session and event timestamp.
// Format: ~/.checkmarx/governance/logs/session-<YYYY-MM-DD>-<sessionId>.log
// The date prefix makes retention purge efficient — no file parsing needed.
func sessionLogPath(sessionID string, t time.Time) string {
	date := t.UTC().Format("2006-01-02")
	filename := fmt.Sprintf("session-%s-%s.log", date, sessionID)
	return filepath.Join(logDir(), filename)
}

// appendToLog appends a SessionLogEntry as a single NDJSON line to path.
// The file is created if it does not exist. Failures are logged but not propagated.
func appendToLog(path string, entry SessionLogEntry) {
	ensureDir(filepath.Dir(path))
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("governance: log marshal error: %v", err)
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		log.Printf("governance: log open error: %v", err)
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s\n", data)
}

// PurgeStaleLogs deletes session log files older than maxAge.
// Called once per session start to enforce the 30-day retention policy.
// Identified by modification time to avoid reading file contents.
func PurgeStaleLogs() {
	pattern := filepath.Join(logDir(), "session-*.log")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-logRetention)
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(f)
		}
	}
}
