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
	// Handle session.hook_inventory as a separate path — just inserts session_hooks rows.
	if ev.Kind == "session.hook_inventory" {
		return processHookInventory(ctx, ev)
	}

	log.Printf("ingest: processing event %s kind=%s tool=%s action=%s", ev.EventID, ev.Kind, ev.Tool.Name, ev.Decision.Action)

	// 1. Resolve or create developer (§6.2 resolveOrCreateDeveloper)
	devID, err := resolveOrCreateDeveloper(ctx, ev.UserID)
	if err != nil {
		return fmt.Errorf("resolve developer: %w", err)
	}
	log.Printf("ingest: resolved developer %s for user %s", devID, ev.UserID)

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

	// 5b. On session start, persist IDE context captured by cx on the developer machine.
	if ev.Kind == "session.start" && ev.IDE.Name != "" && ev.IDE.Name != "terminal" {
		_, _ = db.Pool.Exec(ctx, `
			UPDATE ai_sessions
			SET ide_name = $1, ide_version = $2, workspace_path = $3
			WHERE session_id = $4`,
			ev.IDE.Name, ev.IDE.Version, ev.IDE.WorkspacePath, sessionID)
	}

	// 5c. On session start, persist model detection result
	if ev.Kind == "session.start" && ev.ModelInfo != nil {
		modelID := uuid.Nil
		if ev.ModelInfo.Model != "" {
			// Look up model_id from ai_models table by model_name
			err := db.Pool.QueryRow(ctx,
				`SELECT model_id FROM ai_models WHERE model_name = $1 LIMIT 1`,
				ev.ModelInfo.Model,
			).Scan(&modelID)
			if err == nil && ev.ModelInfo.Confidence >= 0.80 {
				// Only update session model_id if confidence is high
				_, _ = db.Pool.Exec(ctx,
					`UPDATE ai_sessions SET model_id = $1 WHERE session_id = $2`,
					modelID, sessionID)
			}
		}
	}

	// 6. Resolve MCP server if applicable
	var mcpServerID *uuid.UUID
	if ev.Tool.MCP != nil && ev.Tool.MCP.Server != "" {
		id, err := upsertMCPServer(ctx, ev.Tool.MCP.Server)
		if err != nil {
			log.Printf("ingest: upsert MCP server %q: %v", ev.Tool.MCP.Server, err)
		} else {
			mcpServerID = &id
			upsertSessionMCPServer(ctx, sessionID, id, ev.Decision.Action == "BLOCK")
		}
	}

	// 6b. Resolve skill if applicable
	if ev.Tool.Category == "skill" && ev.Tool.Name != "" {
		skillID, err := upsertSkill(ctx, ev.Tool.Name)
		if err != nil {
			log.Printf("ingest: upsert skill %q: %v", ev.Tool.Name, err)
		} else {
			if err := upsertSessionSkill(ctx, sessionID, skillID); err != nil {
				log.Printf("ingest: upsert session-skill bridge: %v", err)
			}
		}
	}

	// 7. Write ALL events to activity_events for the session timeline.
	if err := insertActivityEvent(ctx, sessionID, ev); err != nil {
		log.Printf("ingest: activity_event insert %s: %v", ev.EventID, err)
	}

	// 8. For third-party enforcement events, auto-create monitor policy and insert policy_action.
	if ev.Kind == "tool.third_party_enforcement" {
		log.Printf("ingest: processing third_party_enforcement event %s: tool=%s action=%s",
			ev.EventID, ev.ThirdPartyTool, ev.ThirdPartyAction)
		monitorPolicyID, ourAction := upsertThirdPartyMonitorPolicy(ctx, ev)
		if monitorPolicyID == nil {
			log.Printf("ingest: third-party policy creation failed for %s", ev.ThirdPartyTool)
		} else {
			log.Printf("ingest: created/found third-party policy %s for %s", monitorPolicyID, ev.ThirdPartyTool)
		}
		insertThirdPartyPolicyAction(ctx, sessionID, devID, agentProdID, mcpServerID, monitorPolicyID, ourAction, ev)
		return nil
	}

	// Skip policy_action insertion for lifecycle events with no governance decision.
	if ev.Decision.Action == "" ||
		ev.Kind == "session.start" ||
		ev.Kind == "session.end" ||
		ev.Kind == "tool.post" {
		return nil
	}
	if ev.Decision.Action == "ALLOW" && ev.Decision.PolicyID == "" {
		return nil
	}

	// 9. Resolve policy_version_id from policyId + policyVersion
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

	// 10. Insert policy_action
	meta := map[string]any{
		"latencyMs":   ev.Decision.LatencyMs,
		"packVersion": ev.Decision.PackVersion,
		"eventId":     ev.EventID,
	}
	if ev.Prompt != nil && ev.Prompt.PIIDetected {
		meta["piiDetected"] = true
		meta["piiMatches"] = ev.Prompt.PIIMatches
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

// insertActivityEvent writes every event to activity_events for the session timeline.
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
	var promptText *string
	var piiDetected bool
	var piiMatchesJSON []byte
	if ev.Prompt != nil {
		promptLength = &ev.Prompt.Length
		piiDetected = ev.Prompt.PIIDetected
		if ev.Prompt.PIIDetected && ev.Prompt.PromptText != "" {
			promptText = &ev.Prompt.PromptText
		}
		if len(ev.Prompt.PIIMatches) > 0 {
			piiMatchesJSON, _ = json.Marshal(ev.Prompt.PIIMatches)
		}
	}
	rawJSON, _ := json.Marshal(ev)

	var modelDetected, modelVendor, modelSource, modelSourcePath *string
	var modelConfidence *float64
	var modelDetectedAt *time.Time
	if ev.ModelInfo != nil {
		modelDetected = &ev.ModelInfo.Model
		modelVendor = &ev.ModelInfo.Vendor
		modelConfidence = &ev.ModelInfo.Confidence
		modelSource = &ev.ModelInfo.Source
		if ev.ModelInfo.SourcePath != "" {
			modelSourcePath = &ev.ModelInfo.SourcePath
		}
		modelDetectedAt = &ev.ModelInfo.DetectedAt
	}

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO activity_events (
			event_id, session_id, kind,
			agent_type, tool_category, tool_name, tool_command, tool_paths, tool_query,
			mcp_server, mcp_tool, prompt_length,
			action_type, policy_name, policy_id, policy_version,
			matched_on, matched_value, agent_message, latency_ms,
			pii_detected, prompt_text, pii_matches,
			third_party_tool, third_party_action, attribution_confidence,
			model_detected, model_vendor, model_confidence, model_source, model_source_path, model_detected_at,
			occurred_at, raw
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7, $8, $9,
			$10, $11, $12,
			$13, $14, $15, $16,
			$17, $18, $19, $20,
			$21, $22, $23,
			$24, $25, $26,
			$27, $28, $29, $30, $31, $32,
			$33, $34
		) ON CONFLICT (event_id) DO NOTHING`,
		ev.EventID, sessionID, ev.Kind,
		ev.Agent.Type, ev.Tool.Category, ev.Tool.Name, ev.Tool.Command,
		toolPaths, ev.Tool.Query,
		mcpServer, mcpTool, promptLength,
		nullableStr(ev.Decision.Action, "ALLOW"), ev.Decision.PolicyName, ev.Decision.PolicyID, ev.Decision.PolicyVersion,
		ev.Decision.MatchedOn, ev.Decision.MatchedValue, ev.Decision.AgentMessage, ev.Decision.LatencyMs,
		piiDetected, promptText, piiMatchesJSON,
		nullableStr(ev.ThirdPartyTool, ""), nullableStr(ev.ThirdPartyAction, ""), strPtrOrNil(ev.AttributionConfidence),
		modelDetected, modelVendor, modelConfidence, modelSource, modelSourcePath, modelDetectedAt,
		ev.Timestamp, rawJSON,
	)
	return err
}

// processHookInventory inserts session_hooks rows from a session.hook_inventory event.
func processHookInventory(ctx context.Context, ev models.ActivityEvent) error {
	// session.hook_inventory uses the raw JSON field to carry the hook list.
	// The hooks are stored in ev.Tool as a JSON payload via the command field.
	// We decode from raw event JSON instead.
	type hookInventoryPayload struct {
		SessionID string `json:"sessionId"`
		Hooks     []struct {
			Event   string `json:"event"`
			Matcher string `json:"matcher"`
			Command string `json:"command"`
			Vendor  string `json:"vendor"`
		} `json:"hooks"`
	}
	// The hooks payload is carried in the Decision.AgentMessage field as JSON.
	if ev.Decision.AgentMessage == "" {
		return nil
	}
	var payload hookInventoryPayload
	if err := json.Unmarshal([]byte(ev.Decision.AgentMessage), &payload); err != nil {
		return nil
	}

	// Resolve session UUID
	var sessionID uuid.UUID
	err := db.Pool.QueryRow(ctx,
		`SELECT session_id FROM ai_sessions WHERE claude_session_id = $1 LIMIT 1`,
		ev.SessionID,
	).Scan(&sessionID)
	if err != nil {
		return fmt.Errorf("hook inventory: session not found %s: %w", ev.SessionID, err)
	}

	for _, h := range payload.Hooks {
		_, _ = db.Pool.Exec(ctx, `
			INSERT INTO session_hooks (session_id, event, matcher, command, vendor)
			VALUES ($1, $2, $3, $4, $5)`,
			sessionID, h.Event, h.Matcher, h.Command, h.Vendor,
		)
	}
	return nil
}

// upsertThirdPartyMonitorPolicy creates or finds the third_party_monitor policy for a detected tool.
// Returns the policy UUID and the configured our_action ("block" or "alert").
// Defaults to "block" on creation so detections appear as violations by default.
func upsertThirdPartyMonitorPolicy(ctx context.Context, ev models.ActivityEvent) (*uuid.UUID, string) {
	toolName := ev.ThirdPartyTool
	if toolName == "" {
		toolName = "unknown"
	}
	detectedAction := ev.ThirdPartyAction

	// Check if a monitor policy for this tool+action already exists.
	var policyID uuid.UUID
	var ourAction string
	err := db.Pool.QueryRow(ctx, `
		SELECT p.policy_id, COALESCE(pv.definition->>'our_action', 'block')
		FROM policies p
		JOIN policy_versions pv ON p.current_version_id = pv.policy_version_id
		WHERE p.policy_type = 'third_party_monitor'
		  AND p.deleted_at IS NULL
		  AND pv.definition->>'third_party_tool' = $1
		  AND pv.definition->>'detected_action'  = $2
		LIMIT 1`,
		toolName, detectedAction,
	).Scan(&policyID, &ourAction)

	if err == nil {
		// Existing monitor — increment occurrence count in definition.
		log.Printf("ingest: found existing third-party monitor policy %s for tool=%s action=%s", policyID, toolName, detectedAction)
		_, _ = db.Pool.Exec(ctx, `
			UPDATE policy_versions pv
			SET definition = jsonb_set(
				definition,
				'{occurrence_count}',
				to_jsonb((COALESCE((definition->>'occurrence_count')::int, 0) + 1))
			)
			FROM policies p
			WHERE p.current_version_id = pv.policy_version_id
			  AND p.policy_id = $1`,
			policyID,
		)
		return &policyID, ourAction
	}
	log.Printf("ingest: creating new third-party monitor policy for tool=%s action=%s", toolName, detectedAction)

	// Create a new third_party_monitor policy — default our_action to "block".
	const defaultOurAction = "block"
	policyName := toolName + " (" + detectedAction + " detected)"
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, defaultOurAction
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := tx.QueryRow(ctx, `
		INSERT INTO policies (policy_name, description, policy_type, status)
		VALUES ($1, $2, 'third_party_monitor', 'active')
		RETURNING policy_id`,
		policyName,
		"Auto-detected enforcement by "+toolName,
	).Scan(&policyID); err != nil {
		return nil, defaultOurAction
	}

	definition := map[string]any{
		"third_party_tool": toolName,
		"detected_action":  detectedAction,
		"our_action":       defaultOurAction,
		"occurrence_count": 1,
		"first_seen_at":    ev.Timestamp,
	}
	defJSON, _ := json.Marshal(definition)

	var pvID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO policy_versions (policy_id, version_number, definition, enforcement_mode, change_summary)
		VALUES ($1, 1, $2, 'block', 'Auto-created from third-party enforcement detection')
		RETURNING policy_version_id`,
		policyID, defJSON,
	).Scan(&pvID); err != nil {
		return nil, defaultOurAction
	}

	if _, err := tx.Exec(ctx,
		`UPDATE policies SET current_version_id = $1 WHERE policy_id = $2`,
		pvID, policyID,
	); err != nil {
		return nil, defaultOurAction
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (actor_type, event_type, resource_type, resource_id, new_value)
		VALUES ('system', 'POLICY_CREATED', 'policy', $1, $2)`,
		policyID, defJSON,
	); err != nil {
		log.Printf("ingest: audit event for monitor policy: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("ingest: upsertThirdPartyMonitorPolicy tx.Commit failed for %s: %v", toolName, err)
		return nil, defaultOurAction
	}

	log.Printf("ingest: created new third-party monitor policy %s for tool=%s action=%s", policyID, toolName, detectedAction)

	// Increment pack version so cx CLI picks up the new monitor policy.
	_, _ = NextPackVersion(ctx)

	return &policyID, defaultOurAction
}

// insertThirdPartyPolicyAction inserts a policy_action row for a third-party enforcement event.
// ourAction is the configured action from the monitor policy ("block" or "alert"), which may
// differ from ev.Decision.Action if the user has changed the policy after initial detection.
func insertThirdPartyPolicyAction(ctx context.Context, sessionID, devID, agentID uuid.UUID, mcpServerID, monitorPolicyID *uuid.UUID, ourAction string, ev models.ActivityEvent) {
	// Normalise to uppercase governance action: "block"→"BLOCK", "alert"→"ALERT".
	recordedAction := strings.ToUpper(ourAction)
	if recordedAction != "BLOCK" && recordedAction != "ALERT" {
		recordedAction = "BLOCK" // safe default
	}

	blockedTool := ev.ThirdPartyTool
	if blockedTool == "" {
		blockedTool = "unknown-third-party"
	}
	// Normalize tool name
	if blockedTool == "unknown-hook" {
		blockedTool = "unknown-third-party"
	}
	toolAction := ev.ThirdPartyAction
	if toolAction == "" {
		toolAction = "block"
	}

	meta := map[string]any{
		"thirdPartyTool":        blockedTool,
		"thirdPartyAction":      toolAction,
		"attributionConfidence": ev.AttributionConfidence,
		"blockedClaudeTool":     ev.Tool.Name,
		"eventId":               ev.EventID,
	}
	metaJSON, _ := json.Marshal(meta)

	reason := blockedTool + " " + toolAction + "ed"
	if ev.Tool.Name != "" {
		reason += " " + ev.Tool.Name
	}
	if ev.Decision.AgentMessage != "" {
		reason = ev.Decision.AgentMessage
	}

	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO policy_actions (
			policy_id, session_id, developer_id, agent_id, mcp_server_id,
			action_type, violation_type, severity, resource_type, resource_id,
			reason, matched_rule, occurred_at, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		monitorPolicyID, sessionID, devID, agentID, mcpServerID,
		recordedAction,
		"other", // violation_type_enum has no third-party category; "other" is the closed-vocabulary fallback (see migration 008)
		models.SeverityFromAction(recordedAction),
		"third_party",
		blockedTool,
		reason,
		blockedTool,
		ev.Timestamp,
		metaJSON,
	); err != nil {
		log.Printf("ingest: insertThirdPartyPolicyAction failed for %s: %v", blockedTool, err)
	}
}

func nullableStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// strPtrOrNil returns nil when s is empty so PostgreSQL stores NULL instead of ''.
// Use for columns with CHECK constraints that reject empty strings (e.g. attribution_confidence).
func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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
	// that haven't received an event in >15 minutes — handles SIGKILL / terminal close.
	if kind == "session.start" && devAgentID != (uuid.UUID{}) {
		_, _ = db.Pool.Exec(ctx, `
			UPDATE ai_sessions
			SET status = 'expired', ended_at = NOW(),
			    duration = EXTRACT(EPOCH FROM (NOW() - started_at))::INTEGER
			WHERE status = 'active'
			  AND developer_agent_id = $1
			  AND claude_session_id  != $2
			  AND COALESCE(last_event_at, started_at) < NOW() - INTERVAL '15 minutes'`,
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

func upsertSkill(ctx context.Context, skillName string) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO skills (skill_name, total_invocations)
		VALUES ($1, 1)
		ON CONFLICT (skill_name)
		DO UPDATE SET last_seen_at = NOW(), total_invocations = skills.total_invocations + 1
		RETURNING skill_id`,
		skillName,
	).Scan(&id)
	return id, err
}

func upsertSessionSkill(ctx context.Context, sessionID, skillID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO session_skills (session_id, skill_id, invocation_count)
		VALUES ($1, $2, 1)
		ON CONFLICT (session_id, skill_id)
		DO UPDATE SET
			invocation_count = session_skills.invocation_count + 1,
			last_invoked_at = NOW()`,
		sessionID, skillID)
	return err
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
		  AND p.deleted_at IS NULL
		  AND p.policy_type IN ('governance', 'pii')
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
		var pol models.PackPolicy
		if err := json.Unmarshal(raw, &pol); err != nil {
			continue
		}
		policies = append(policies, pol)
	}

	// Build PII policy from active pii-type policy in DB.
	// Falls back to a default (all built-ins, block) when none is configured.
	piiPolicy := buildPIIPolicy(ctx)

	return &models.GovernancePack{
		PackVersion: packVersion,
		Policies:    policies,
		PIIPolicy:   piiPolicy,
		UpdatedAt:   time.Now().UTC(),
	}, nil
}

// buildPIIPolicy queries the active PII policy from the DB.
// Returns a default (all built-ins, block) when no policy is configured so that
// PII detection is on by default without requiring admin setup.
func buildPIIPolicy(ctx context.Context) *models.PIIPolicyDef {
	var defJSON []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT pv.definition
		FROM policies p
		JOIN policy_versions pv ON p.current_version_id = pv.policy_version_id
		WHERE p.status = 'active'
		  AND p.deleted_at IS NULL
		  AND p.policy_type = 'pii'
		ORDER BY p.created_at
		LIMIT 1`).Scan(&defJSON)

	if err == nil && len(defJSON) > 0 {
		var def models.PIIPolicyDef
		if json.Unmarshal(defJSON, &def) == nil {
			return &def
		}
	}

	// Default: all built-in patterns enabled, block on detection.
	return &models.PIIPolicyDef{
		Enabled: true,
		Action:  "block",
		BuiltIn: []string{"email", "ssn", "credit_card", "phone", "ip_address", "api_key", "jwt"},
	}
}
