// Package models contains all shared domain types for the governance API.
// These mirror the database schema (§5) and the ActivityEvent wire format sent by cx.
package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ============================================================
// Policy & Versions
// ============================================================

type Policy struct {
	PolicyID         uuid.UUID  `json:"policyId"         db:"policy_id"`
	PolicyName       string     `json:"policyName"       db:"policy_name"`
	Description      string     `json:"description"      db:"description"`
	CurrentVersionID *uuid.UUID `json:"currentVersionId" db:"current_version_id"`
	Status           string     `json:"status"           db:"status"`
	CreatedAt        time.Time  `json:"createdAt"        db:"created_at"`
	UpdatedAt        time.Time  `json:"updatedAt"        db:"updated_at"`
}

type PolicyVersion struct {
	PolicyVersionID uuid.UUID       `json:"policyVersionId"  db:"policy_version_id"`
	PolicyID        uuid.UUID       `json:"policyId"         db:"policy_id"`
	VersionNumber   int             `json:"versionNumber"    db:"version_number"`
	Definition      json.RawMessage `json:"definition"       db:"definition"`
	EnforcementMode string          `json:"enforcementMode"  db:"enforcement_mode"`
	CreatedBy       *uuid.UUID      `json:"createdBy"        db:"created_by"`
	CreatedAt       time.Time       `json:"createdAt"        db:"created_at"`
	ChangeSummary   string          `json:"changeSummary"    db:"change_summary"`
}

// GovernancePack is the policy-pack.json format delivered to cx and stored in
// policy_versions.definition (architecture §5.9).
type GovernancePack struct {
	PackVersion int          `json:"packVersion"`
	Policies    []PackPolicy `json:"policies"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

type PackPolicy struct {
	PolicyID      string `json:"policyId"`
	PolicyName    string `json:"policyName"`
	PolicyVersion int    `json:"policyVersion"`
	Type          string `json:"type"`
	Action        string `json:"action"`
	MatchTarget   string `json:"matchTarget"`
	Pattern       string `json:"pattern"`
	AgentMessage  string `json:"agentMessage,omitempty"`
}

// ============================================================
// Developers & Agents
// ============================================================

type Developer struct {
	DeveloperID uuid.UUID `json:"developerId"  db:"developer_id"`
	Email       string    `json:"email"        db:"email"`
	DisplayName string    `json:"displayName"  db:"display_name"`
	Username    string    `json:"username"     db:"username"`
	Team        string    `json:"team"         db:"team"`
	Department  string    `json:"department"   db:"department"`
	Role        string    `json:"role"         db:"role"`
	Status      string    `json:"status"       db:"status"`
	FirstSeenAt time.Time `json:"firstSeenAt"  db:"first_seen_at"`
	LastSeenAt  time.Time `json:"lastSeenAt"   db:"last_seen_at"`
}

type AICodingAgent struct {
	AgentID     uuid.UUID `json:"agentId"      db:"agent_id"`
	AgentName   string    `json:"agentName"    db:"agent_name"`
	Vendor      string    `json:"vendor"       db:"vendor"`
	AgentType   string    `json:"agentType"    db:"agent_type"`
	Version     string    `json:"version"      db:"version"`
	Environment string    `json:"environment"  db:"environment"`
	Status      string    `json:"status"       db:"status"`
	FirstSeenAt time.Time `json:"firstSeenAt"  db:"first_seen_at"`
	LastSeenAt  time.Time `json:"lastSeenAt"   db:"last_seen_at"`
}

type DeveloperAgent struct {
	DeveloperAgentID  uuid.UUID `json:"developerAgentId"  db:"developer_agent_id"`
	DeveloperID       uuid.UUID `json:"developerId"       db:"developer_id"`
	AgentProductID    uuid.UUID `json:"agentProductId"    db:"agent_product_id"`
	MachineID         string    `json:"machineId"         db:"machine_id"`
	GovernanceAgentID string    `json:"governanceAgentId" db:"governance_agent_id"`
	FirstSeenAt       time.Time `json:"firstSeenAt"       db:"first_seen_at"`
	LastSeenAt        time.Time `json:"lastSeenAt"        db:"last_seen_at"`
}

// ============================================================
// Sessions
// ============================================================

type AISession struct {
	SessionID        uuid.UUID  `json:"sessionId"       db:"session_id"`
	ClaudeSessionID  string     `json:"claudeSessionId" db:"claude_session_id"`
	DeveloperID      uuid.UUID  `json:"developerId"     db:"developer_id"`
	AgentID          uuid.UUID  `json:"agentId"         db:"agent_id"`
	DeveloperAgentID *uuid.UUID `json:"developerAgentId" db:"developer_agent_id"`
	ModelID          *uuid.UUID `json:"modelId"         db:"model_id"`
	StartedAt        time.Time  `json:"startedAt"       db:"started_at"`
	EndedAt          *time.Time `json:"endedAt"         db:"ended_at"`
	Duration         *int       `json:"duration"        db:"duration"`
	RequestCount     int        `json:"requestCount"    db:"request_count"`
	ToolCallCount    int        `json:"toolCallCount"   db:"tool_call_count"`
	Status           string     `json:"status"          db:"status"`
}

// ============================================================
// MCP Servers
// ============================================================

type MCPServer struct {
	MCPServerID   uuid.UUID `json:"mcpServerId"   db:"mcp_server_id"`
	ServerName    string    `json:"serverName"    db:"server_name"`
	ServerURL     string    `json:"serverUrl"     db:"server_url"`
	Vendor        string    `json:"vendor"        db:"vendor"`
	ServerVersion string    `json:"serverVersion" db:"server_version"`
	Status        string    `json:"status"        db:"status"`
	FirstSeenAt   time.Time `json:"firstSeenAt"   db:"first_seen_at"`
	LastSeenAt    time.Time `json:"lastSeenAt"    db:"last_seen_at"`
}

// ============================================================
// Policy Actions
// ============================================================

type PolicyAction struct {
	PolicyActionID   uuid.UUID      `json:"policyActionId"  db:"policy_action_id"`
	PolicyID         *uuid.UUID     `json:"policyId"        db:"policy_id"`
	PolicyVersionID  *uuid.UUID     `json:"policyVersionId" db:"policy_version_id"`
	SessionID        uuid.UUID      `json:"sessionId"       db:"session_id"`
	DeveloperID      uuid.UUID      `json:"developerId"     db:"developer_id"`
	AgentID          uuid.UUID      `json:"agentId"         db:"agent_id"`
	MCPServerID      *uuid.UUID     `json:"mcpServerId"     db:"mcp_server_id"`
	ActionType       string         `json:"actionType"      db:"action_type"`
	ViolationType    string         `json:"violationType"   db:"violation_type"`
	Severity         string         `json:"severity"        db:"severity"`
	ResourceType     string         `json:"resourceType"    db:"resource_type"`
	ResourceID       string         `json:"resourceId"      db:"resource_id"`
	Reason           string         `json:"reason"          db:"reason"`
	MatchedRule      string         `json:"matchedRule"     db:"matched_rule"`
	OccurredAt       time.Time      `json:"occurredAt"      db:"occurred_at"`
	ResolutionStatus string         `json:"resolutionStatus" db:"resolution_status"`
	ResolvedAt       *time.Time     `json:"resolvedAt"      db:"resolved_at"`
	Metadata         map[string]any `json:"metadata"   db:"metadata"`
}

// ============================================================
// Audit Events
// ============================================================

type AuditEvent struct {
	AuditEventID uuid.UUID       `json:"auditEventId" db:"audit_event_id"`
	ActorID      *uuid.UUID      `json:"actorId"      db:"actor_id"`
	ActorType    string          `json:"actorType"    db:"actor_type"`
	EventType    string          `json:"eventType"    db:"event_type"`
	ResourceType string          `json:"resourceType" db:"resource_type"`
	ResourceID   *uuid.UUID      `json:"resourceId"   db:"resource_id"`
	OldValue     json.RawMessage `json:"oldValue"     db:"old_value"`
	NewValue     json.RawMessage `json:"newValue"     db:"new_value"`
	Timestamp    time.Time       `json:"timestamp"    db:"timestamp"`
	Metadata     map[string]any  `json:"metadata"     db:"metadata"`
}

// Audit event type constants (architecture §5.1)
const (
	AuditPolicyCreated        = "POLICY_CREATED"
	AuditPolicyVersionCreated = "POLICY_VERSION_CREATED"
	AuditPolicyActivated      = "POLICY_ACTIVATED"
	AuditPolicyDeactivated    = "POLICY_DEACTIVATED"
	AuditPolicyRolledBack     = "POLICY_ROLLED_BACK"
	AuditPolicyUpdated        = "POLICY_UPDATED"
	AuditDeveloperAdded       = "DEVELOPER_ADDED"
	AuditDeveloperDisabled    = "DEVELOPER_DISABLED"
	AuditAgentRegistered      = "AGENT_REGISTERED"
	AuditAgentDisabled        = "AGENT_DISABLED"
)

// ============================================================
// ActivityEvent wire format (sent by cx spool flusher)
// ============================================================

// ActivityEvent is the inbound payload from the cx governance hooks spool.
// Maps to database columns per §5.6.
type ActivityEvent struct {
	EventID   string           `json:"eventId"`
	Timestamp time.Time        `json:"timestamp"`
	Kind      string           `json:"kind"` // session.start | tool.pre | tool.post | prompt.submit | session.end
	SessionID string           `json:"sessionId"`
	MachineID string           `json:"machineId"`
	AgentID   string           `json:"agentId"` // governance_agent_id (SHA256 hex)
	UserID    string           `json:"userId"`  // OS username
	Agent     ActivityAgent    `json:"agent"`
	Tool      ActivityTool     `json:"tool"`
	Prompt    *ActivityPrompt  `json:"prompt,omitempty"`
	Decision  ActivityDecision `json:"decision"`
}

type ActivityAgent struct {
	Type string `json:"type"` // "claude-code" | "cursor"
	ID   string `json:"id"`
}

type ActivityTool struct {
	Category string   `json:"category"`
	Name     string   `json:"name"`
	Command  string   `json:"command,omitempty"`
	Paths    []string `json:"paths,omitempty"`
	Query    string   `json:"query,omitempty"`
	RawName  string   `json:"rawName,omitempty"`
	MCP      *MCPRef  `json:"mcp,omitempty"`
}

type MCPRef struct {
	Server string `json:"server"`
	Name   string `json:"name"`
}

type ActivityPrompt struct {
	Length int `json:"length"`
}

type ActivityDecision struct {
	Action        string `json:"action"`
	PolicyID      string `json:"policyId,omitempty"`
	PolicyName    string `json:"policyName,omitempty"`
	PolicyVersion int    `json:"policyVersion,omitempty"`
	PackVersion   int    `json:"packVersion,omitempty"`
	MatchedOn     string `json:"matchedOn,omitempty"`
	MatchedValue  string `json:"matchedValue,omitempty"`
	AgentMessage  string `json:"agentMessage,omitempty"`
	LatencyMs     int64  `json:"latencyMs,omitempty"`
}

// SeverityFromAction derives the severity level from the action type (§5.5).
func SeverityFromAction(action string) string {
	switch action {
	case "BLOCK":
		return "high"
	case "ALERT":
		return "medium"
	default:
		return "low"
	}
}

// ============================================================
// Request/Response types
// ============================================================

type CreatePolicyRequest struct {
	PolicyName      string          `json:"policyName"`
	Description     string          `json:"description"`
	Definition      json.RawMessage `json:"definition"`
	EnforcementMode string          `json:"enforcementMode"`
	ChangeSummary   string          `json:"changeSummary"`
}

type ActivatePolicyRequest struct {
	PolicyVersionID uuid.UUID `json:"policyVersionId"`
}

type ActivatePolicyResponse struct {
	PolicyID      uuid.UUID `json:"policyId"`
	PolicyVersion int       `json:"policyVersion"`
	PackVersion   int       `json:"packVersion"`
}

type IngestAuditRequest struct {
	Events []ActivityEvent `json:"events"`
}

type RegisterAgentRequest struct {
	AgentID   string `json:"agentId"`
	AgentType string `json:"agentType"`
	MachineID string `json:"machineId"`
	UserID    string `json:"userId"`
}

type SessionFilter struct {
	DeveloperID string
	MachineID   string
	AgentName   string
	Status      string
	From        *time.Time
	To          *time.Time
	Limit       int
	Offset      int
}

type ViolationFilter struct {
	DeveloperID string
	MachineID   string
	AgentName   string
	PolicyID    string
	ActionType  string
	Severity    string
	From        *time.Time
	To          *time.Time
	Limit       int
	Offset      int
}
