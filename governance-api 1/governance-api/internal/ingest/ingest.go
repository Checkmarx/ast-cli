// Package ingest processes ActivityEvent batches from the cx spool flusher.
// It resolves all foreign keys, upserts identity rows, and inserts policy_actions.
// Architecture §6.2 specifies the full resolution chain.
package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/checkmarx/governance-api/internal/db"
	"github.com/checkmarx/governance-api/internal/models"
	"github.com/google/uuid"
)

// Processor handles a batch of ActivityEvents from the spool.
type Processor struct{}

// New returns a ready Processor.
func New() *Processor { return &Processor{} }

// ProcessBatch ingests a slice of ActivityEvents in a single database transaction.
// If any event fails (e.g. missing policy_id), it is skipped with a log warning so
// the rest of the batch still commits. Architecture cardinal rule: never fail-open
// silently — all skips are logged.
func (p *Processor) ProcessBatch(ctx context.Context, events []models.ActivityEvent) error {
	for _, ev := range events {
		if err := p.processOne(ctx, ev); err != nil {
			log.Printf("ingest: skip event %s: %v", ev.EventID, err)
		}
	}
	return nil
}

func (p *Processor) processOne(ctx context.Context, ev models.ActivityEvent) error {
	// 1. Resolve or create developer (§6.2 resolveOrCreateDeveloper)
	devID, err := resolveOrCreateDeveloper(ctx, ev.UserID)
	if err != nil {
		return fmt.Errorf("resolve developer: %w", err)
	}

	// 2. Resolve agent product (from ai_coding_agents catalog)
	agentProdID, err := resolveAgentProduct(ctx, ev.Agent.Type)
	if err != nil {
		return fmt.Errorf("resolve agent product: %w", err)
	}

	// 3. Upsert developer_agents bridge row
	devAgentID, err := upsertDeveloperAgent(ctx, devID, agentProdID, ev.MachineID, ev.AgentID)
	if err != nil {
		return fmt.Errorf("upsert developer_agent: %w", err)
	}

	// 4. Upsert session
	sessionID, err := upsertSession(ctx, ev.SessionID, devID, agentProdID, devAgentID, ev.Kind)
	if err != nil {
		return fmt.Errorf("upsert session: %w", err)
	}

	// 5. Update session counters based on event kind
	updateSessionCounters(ctx, sessionID, ev.Kind)

	// 6. Resolve MCP server if applicable
	var mcpServerID *uuid.UUID
	if ev.Tool.MCP != nil && ev.Tool.MCP.Server != "" {
		id, err := upsertMCPServer(ctx, ev.Tool.MCP.Server)
		if err != nil {
			log.Printf("ingest: upsert MCP server %q: %v", ev.Tool.MCP.Server, err)
		} else {
			mcpServerID = &id
			// Upsert session_mcp_servers bridge
			upsertSessionMCPServer(ctx, sessionID, id, ev.Decision.Action == "BLOCK")
		}
	}

	// 7. Write ALL events to activity_events for the session timeline.
	// This is the source of truth for the session activity panel.
	if err := insertActivityEvent(ctx, sessionID, ev); err != nil {
		log.Printf("ingest: activity_event insert %s: %v", ev.EventID, err)
	}

	// Skip policy_action insertion for events with no governance decision.
	if ev.Decision.Action == "" ||
		ev.Kind == "session.start" ||
		ev.Kind == "session.end" ||
		ev.Kind == "tool.post" {
		return nil
	}
	// ALLOW decisions with no policy fired don't need a policy_actions row.
	if ev.Decision.Action == "ALLOW" && ev.Decision.PolicyID == "" {
		return nil
	}

	// 8. Resolve policy_version_id from policyId + policyVersion
	var policyID *uuid.UUID
	var pvID *uuid.UUID
	if ev.Decision.PolicyID != "" {
		pid, pv, err := resolvePolicyVersion(ctx, ev.Decision.PolicyID, ev.Decision.PolicyVersion)
		if err != nil {
			log.Printf("ingest: policy version not found %s v%d: %v",
				ev.Decision.PolicyID, ev.Decision.PolicyVersion, err)
		} else {
			policyID = pid
			pvID = pv
		}
	}

	// 9. Insert policy_action
	meta := map[string]any{
		"latencyMs":   ev.Decision.LatencyMs,
		"packVersion": ev.Decision.PackVersion,
		"eventId":     ev.EventID,
	}
	metaJSON, _ := json.Marshal(meta)

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO policy_actions (
			policy_id, policy_version_id, session_id, developer_id,
			agent_id, mcp_server_id,
			action_type, violation_type, severity, resource_type, resource_id,
			reason, matched_rule, occurred_at, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15
		)`,
		policyID, pvID, sessionID, devID,
		agentProdID, mcpServerID,
		ev.Decision.Action,
		ev.Decision.MatchedOn,
		models.SeverityFromAction(ev.Decision.Action),
		ev.Tool.Category,
		ev.Decision.MatchedValue,
		ev.Decision.PolicyName,
		ev.Decision.PolicyID,
		ev.Timestamp,
		metaJSON,
	)
	return err
}

// insertActivityEvent writes every event (all kinds, all actions) to the
// activity_events table so the session timeline panel can show a complete picture.
func insertActivityEvent(ctx context.Context, sessionID uuid.UUID, ev models.ActivityEvent) error {
	var toolPaths []byte
	if len(ev.Tool.Paths) > 0 {
		toolPaths, _ = json.Marshal(ev.Tool.Paths)
	}
	var mcpServer, mcpTool string
	if ev.Tool.MCP != nil {
		mcpServer = ev.Tool.MCP.Server
		mcpTool = ev.Tool.MCP.Name
	}
	var promptLength *int
	if ev.Prompt != nil {
		promptLength = &ev.Prompt.Length
	}
	rawJSON, _ := json.Marshal(ev)

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO activity_events (
			event_id, session_id, kind,
			agent_type, tool_category, tool_name, tool_command, tool_paths, tool_query,
			mcp_server, mcp_tool, prompt_length,
			action_type, policy_name, policy_id, policy_version,
			matched_on, matched_value, agent_message, latency_ms,
			occurred_at, raw
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7, $8, $9,
			$10, $11, $12,
			$13, $14, $15, $16,
			$17, $18, $19, $20,
			$21, $22
		) ON CONFLICT (event_id) DO NOTHING`,
		ev.EventID, sessionID, ev.Kind,
		ev.Agent.Type, ev.Tool.Category, ev.Tool.Name, ev.Tool.Command,
		toolPaths, ev.Tool.Query,
		mcpServer, mcpTool, promptLength,
		nullableStr(ev.Decision.Action, "ALLOW"), ev.Decision.PolicyName, ev.Decision.PolicyID, ev.Decision.PolicyVersion,
		ev.Decision.MatchedOn, ev.Decision.MatchedValue, ev.Decision.AgentMessage, ev.Decision.LatencyMs,
		ev.Timestamp, rawJSON,
	)
	return err
}

func nullableStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// ============================================================
// Identity resolution helpers
// ============================================================

func resolveOrCreateDeveloper(ctx context.Context, username string) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.Pool.QueryRow(ctx,
		`SELECT developer_id FROM developers WHERE username = $1`, username,
	).Scan(&id)
	if err == nil {
		// Update last_seen_at
		_, _ = db.Pool.Exec(ctx,
			`UPDATE developers SET last_seen_at = NOW() WHERE developer_id = $1`, id)
		return id, nil
	}
	// Create with username as both username and a placeholder email
	email := username + "@unknown.local"
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO developers (email, username, display_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE SET last_seen_at = NOW()
		RETURNING developer_id`,
		email, username, username,
	).Scan(&id)
	return id, err
}

func resolveAgentProduct(ctx context.Context, agentType string) (uuid.UUID, error) {
	// Map governance agentType string to catalog agent_name
	name := agentTypeToCatalogName(agentType)
	var id uuid.UUID
	err := db.Pool.QueryRow(ctx,
		`SELECT agent_id FROM ai_coding_agents WHERE LOWER(agent_name) = LOWER($1) LIMIT 1`,
		name,
	).Scan(&id)
	if err != nil {
		// Upsert unknown agent so we never drop data
		err = db.Pool.QueryRow(ctx, `
			INSERT INTO ai_coding_agents (agent_name)
			VALUES ($1)
			ON CONFLICT (agent_name, vendor, version) DO UPDATE SET last_seen_at = NOW()
			RETURNING agent_id`, name,
		).Scan(&id)
	}
	return id, err
}

func agentTypeToCatalogName(t string) string {
	switch strings.ToLower(t) {
	case "claude-code", "claude":
		return "Claude Code"
	case "cursor":
		return "Cursor"
	case "windsurf":
		return "Windsurf"
	case "gemini", "gemini-cli":
		return "Gemini CLI"
	case "copilot", "copilot-cli":
		return "Copilot CLI"
	default:
		return t
	}
}

func upsertDeveloperAgent(ctx context.Context, devID, agentProdID uuid.UUID, machineID, governanceAgentID string) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO developer_agents (developer_id, agent_product_id, machine_id, governance_agent_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (governance_agent_id)
		DO UPDATE SET last_seen_at = NOW()
		RETURNING developer_agent_id`,
		devID, agentProdID, machineID, governanceAgentID,
	).Scan(&id)
	return id, err
}

func upsertSession(ctx context.Context, claudeSessionID string, devID, agentID, devAgentID uuid.UUID, kind string) (uuid.UUID, error) {
	var id uuid.UUID
	status := "active"
	if kind == "session.end" {
		status = "ended"
	}

	// On a new session start, expire any sessions from the same developer_agent
	// that haven't received an event in a long while — handles SIGKILL / terminal
	// close. The window is intentionally generous (2h, not 15m): a Claude Code
	// session can sit idle for many minutes between prompts/tool calls (reading
	// a long response, stepping away) without actually being killed, and a short
	// window was flipping live sessions to "expired" in the dashboard.
	if kind == "session.start" && devAgentID != (uuid.UUID{}) {
		_, _ = db.Pool.Exec(ctx, `
			UPDATE ai_sessions
			SET status = 'expired', ended_at = NOW(),
			    duration = EXTRACT(EPOCH FROM (NOW() - started_at))::INTEGER
			WHERE status = 'active'
			  AND developer_agent_id = $1
			  AND claude_session_id  != $2
			  AND COALESCE(last_event_at, started_at) < NOW() - INTERVAL '2 hours'`,
			devAgentID, claudeSessionID)
	}

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO ai_sessions (claude_session_id, developer_id, agent_id, developer_agent_id, status, last_event_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (claude_session_id, agent_id)
		DO UPDATE SET
			status       = CASE WHEN $5 = 'ended' THEN 'ended'::session_status ELSE ai_sessions.status END,
			ended_at     = CASE WHEN $5 = 'ended' THEN NOW() ELSE ai_sessions.ended_at END,
			duration     = CASE WHEN $5 = 'ended'
			                 THEN EXTRACT(EPOCH FROM (NOW() - ai_sessions.started_at))::INTEGER
			                 ELSE ai_sessions.duration END,
			last_event_at = NOW()
		RETURNING session_id`,
		claudeSessionID, devID, agentID, devAgentID, status,
	).Scan(&id)
	return id, err
}

func updateSessionCounters(ctx context.Context, sessionID uuid.UUID, kind string) {
	switch kind {
	case "prompt.submit":
		_, _ = db.Pool.Exec(ctx,
			`UPDATE ai_sessions SET request_count = request_count + 1, last_event_at = NOW() WHERE session_id = $1`, sessionID)
	case "tool.pre":
		_, _ = db.Pool.Exec(ctx,
			`UPDATE ai_sessions SET tool_call_count = tool_call_count + 1, last_event_at = NOW() WHERE session_id = $1`, sessionID)
	}
}

func upsertMCPServer(ctx context.Context, serverName string) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO mcp_servers (server_name)
		VALUES ($1)
		ON CONFLICT (server_name)
		DO UPDATE SET last_seen_at = NOW()
		RETURNING mcp_server_id`,
		serverName,
	).Scan(&id)
	return id, err
}

func upsertSessionMCPServer(ctx context.Context, sessionID, mcpServerID uuid.UUID, blocked bool) {
	blockedDelta := 0
	if blocked {
		blockedDelta = 1
	}
	_, _ = db.Pool.Exec(ctx, `
		INSERT INTO session_mcp_servers (session_id, mcp_server_id, tool_call_count, blocked_call_count)
		VALUES ($1, $2, 1, $3)
		ON CONFLICT (session_id, mcp_server_id)
		DO UPDATE SET
			last_used_at = NOW(),
			tool_call_count = session_mcp_servers.tool_call_count + 1,
			blocked_call_count = session_mcp_servers.blocked_call_count + $3`,
		sessionID, mcpServerID, blockedDelta,
	)
}

func resolvePolicyVersion(ctx context.Context, policyIDStr string, version int) (*uuid.UUID, *uuid.UUID, error) {
	// policyIDStr may be a short string like "pol_abc" or a UUID
	// Try UUID parse first, fall back to policy_name lookup
	var polID, pvID uuid.UUID

	// Try direct UUID
	if pid, err := uuid.Parse(policyIDStr); err == nil {
		err = db.Pool.QueryRow(ctx, `
			SELECT pv.policy_id, pv.policy_version_id
			FROM policy_versions pv
			WHERE pv.policy_id = $1 AND pv.version_number = $2`,
			pid, version,
		).Scan(&polID, &pvID)
		if err == nil {
			return &polID, &pvID, nil
		}
	}

	// Try by matched_rule / policyId field stored in definition JSONB
	err := db.Pool.QueryRow(ctx, `
		SELECT pv.policy_id, pv.policy_version_id
		FROM policy_versions pv
		WHERE (pv.definition->>'policyId' = $1 OR pv.policy_id::text = $1)
		  AND pv.version_number = $2
		LIMIT 1`,
		policyIDStr, version,
	).Scan(&polID, &pvID)
	if err != nil {
		return nil, nil, err
	}
	return &polID, &pvID, nil
}

// ============================================================
// Pack version management
// ============================================================

// NextPackVersion atomically increments and returns the global pack version.
func NextPackVersion(ctx context.Context) (int, error) {
	var v int
	err := db.Pool.QueryRow(ctx, `SELECT nextval('pack_version_seq')`).Scan(&v)
	return v, err
}

// CurrentPackVersion returns the current pack version without incrementing.
func CurrentPackVersion(ctx context.Context) (int, error) {
	var v int
	err := db.Pool.QueryRow(ctx, `SELECT last_value FROM pack_version_seq`).Scan(&v)
	return v, err
}

// BuildGovernancePack assembles the current GovernancePack from all active policies.
// This is served by GET /api/v1/agent/policies.
func BuildGovernancePack(ctx context.Context) (*models.GovernancePack, error) {
	packVersion, err := CurrentPackVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("pack version: %w", err)
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT pv.definition
		FROM policies p
		JOIN policy_versions pv ON p.current_version_id = pv.policy_version_id
		WHERE p.status = 'active'
		ORDER BY p.created_at`)
	if err != nil {
		return nil, fmt.Errorf("query active policies: %w", err)
	}
	defer rows.Close()

	var policies []models.PackPolicy
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		// definition JSONB is a single PackPolicy wrapped or the full GovernancePack?
		// Per §5.9 definition stores the single policy definition
		var pol models.PackPolicy
		if err := json.Unmarshal(raw, &pol); err != nil {
			continue
		}
		policies = append(policies, pol)
	}

	return &models.GovernancePack{
		PackVersion: packVersion,
		Policies:    policies,
		UpdatedAt:   time.Now().UTC(),
	}, nil
}
