package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/checkmarx/governance-api/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

// expireInactiveSessions marks sessions as expired if they haven't received an
// event in >15 minutes. Handles terminals killed without firing SessionEnd.
func expireInactiveSessions(ctx context.Context) {
	_, _ = db.Pool.Exec(ctx, `
		UPDATE ai_sessions
		SET status = 'expired', ended_at = NOW(),
		    duration = EXTRACT(EPOCH FROM (NOW() - started_at))::INTEGER
		WHERE status = 'active'
		  AND COALESCE(last_event_at, started_at) < NOW() - INTERVAL '15 minutes'`)
}

// ============================================================
// GET /api/v1/sessions
// ============================================================

func ListSessions(w http.ResponseWriter, r *http.Request) {
	expireInactiveSessions(r.Context())
	limit, offset := parsePagination(r.URL.Query())
	rows, err := db.Pool.Query(r.Context(), `
		SELECT
			s.session_id::text, s.claude_session_id, s.started_at, s.ended_at,
			s.duration, s.request_count, s.tool_call_count, s.status::text,
			d.email, COALESCE(d.username,''), COALESCE(d.display_name,''),
			a.agent_name,
			COALESCE(da.machine_id,''),
			COALESCE(s.ide_name,''), COALESCE(s.ide_version,''), COALESCE(s.workspace_path,''), COALESCE(s.ide_entrypoint,''),
			COALESCE(ae.model_detected,''),
			tu.reqs, tu.inp, tu.outp, tu.cr, tu.cc, tu.cost, tu.unpriced
		FROM ai_sessions s
		JOIN developers d       ON s.developer_id = d.developer_id
		JOIN ai_coding_agents a ON s.agent_id     = a.agent_id
		LEFT JOIN developer_agents da ON s.developer_agent_id = da.developer_agent_id
		LEFT JOIN LATERAL (
			SELECT model_detected FROM activity_events
			WHERE session_id = s.session_id AND kind = 'session.start'
			LIMIT 1
		) ae ON TRUE
		LEFT JOIN LATERAL (
			SELECT COUNT(*)::bigint AS reqs,
			       SUM(input_tokens)::bigint AS inp, SUM(output_tokens)::bigint AS outp,
			       SUM(cache_read_tokens)::bigint AS cr, SUM(cache_creation_tokens)::bigint AS cc,
			       SUM(cost_usd)::float8 AS cost,
			       (COUNT(*) FILTER (WHERE cost_usd IS NULL))::bigint AS unpriced
			FROM token_usage_effective WHERE session_id = s.session_id
		) tu ON TRUE
		ORDER BY s.started_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	result := []map[string]any{} // never nil — returns [] not null
	for rows.Next() {
		var (
			sessionID, claudeID, email, username, displayName, agentName, machineID, status string
			ideName, ideVersion, workspacePath, ideEntrypoint, model                        string
			startedAt                                                                       time.Time
			endedAt                                                                         *time.Time
			duration, requestCount, toolCallCount                                           *int
			uReqs, uIn, uOut, uCacheRead, uCacheWrite, uUnpriced                            *int64
			uCost                                                                           *float64
		)
		if err := rows.Scan(&sessionID, &claudeID, &startedAt, &endedAt,
			&duration, &requestCount, &toolCallCount, &status,
			&email, &username, &displayName, &agentName, &machineID,
			&ideName, &ideVersion, &workspacePath, &ideEntrypoint, &model,
			&uReqs, &uIn, &uOut, &uCacheRead, &uCacheWrite, &uCost, &uUnpriced); err != nil {
			log.Printf("api: ListSessions scan: %v", err)
			continue
		}
		session := map[string]any{
			"sessionId":     sessionID,
			"claudeId":      claudeID,
			"startedAt":     startedAt,
			"endedAt":       endedAt,
			"duration":      duration,
			"requestCount":  requestCount,
			"toolCallCount": toolCallCount,
			"status":        status,
			"developer":     map[string]any{"email": email, "username": username, "displayName": displayName},
			"agentName":     agentName,
			"machineId":     machineID,
			"ide":           map[string]any{"name": ideName, "version": ideVersion, "workspacePath": workspacePath, "entrypoint": ideEntrypoint},
		}
		if model != "" {
			session["model"] = model
		}
		// usage is null (not zero) when no token data was ever received for the session.
		if uReqs != nil && *uReqs > 0 {
			total := *uIn + *uOut + *uCacheRead + *uCacheWrite
			session["usage"] = map[string]any{
				"requests":            *uReqs,
				"inputTokens":         *uIn,
				"outputTokens":        *uOut,
				"cacheReadTokens":     *uCacheRead,
				"cacheCreationTokens": *uCacheWrite,
				"totalTokens":         total,
				"costUsd":             uCost,
				"unpricedRequests":    *uUnpriced,
			}
		}
		result = append(result, session)
	}
	jsonOK(w, result)
}

// GET /api/v1/sessions/:id/timeline
// Queries activity_events — contains ALL event kinds (session.start, tool.pre,
// prompt.submit, session.end) with every action type (ALLOW, ALERT, BLOCK).
func GetSessionTimeline(w http.ResponseWriter, r *http.Request) {
	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid session id")
		return
	}
	rows, err := db.Pool.Query(r.Context(), `
		SELECT
			ae.activity_event_id::text,  ae.event_id,
			ae.kind,                     COALESCE(ae.agent_type,''),
			COALESCE(ae.tool_category,''), COALESCE(ae.tool_name,''),
			COALESCE(ae.tool_command,''),  COALESCE(ae.tool_query,''),
			COALESCE(ae.tool_paths::text,''),
			COALESCE(ae.mcp_server,''),    COALESCE(ae.mcp_tool,''),
			ae.prompt_length,
			COALESCE(ae.pii_detected, false),
			COALESCE(ae.prompt_text,''),
			COALESCE(ae.pii_matches::text,'null'),
			COALESCE(ae.third_party_tool,''),
			COALESCE(ae.third_party_action,''),
			COALESCE(ae.attribution_confidence,''),
			COALESCE(ae.detection_method,''),
			COALESCE(ae.third_party_reason,''),
			COALESCE(ae.third_party_hook_event,''),
			ae.action_type,
			COALESCE(ae.policy_name,''),   COALESCE(ae.policy_id,''),
			COALESCE(ae.matched_on,''),    COALESCE(ae.matched_value,''),
			COALESCE(ae.agent_message,''), ae.latency_ms,
			ae.occurred_at
		FROM activity_events ae
		WHERE ae.session_id = $1
		ORDER BY ae.occurred_at`, sessionID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	events := []map[string]any{}
	for rows.Next() {
		var (
			id, eventID, kind, agentType                        string
			toolCat, toolName, toolCmd, toolQuery, toolPathsStr string
			mcpServer, mcpTool                                  string
			piiDetected                                         bool
			promptText, piiMatchesStr                           string
			thirdPartyTool, thirdPartyAction, attrConf          string
			detectionMethod                                     string
			thirdPartyReason, thirdPartyHookEvent               string
			actionType, policyName, policyID                    string
			matchedOn, matchedValue, agentMessage               string
			promptLength                                        *int
			latencyMs                                           *int
			occurredAt                                          time.Time
		)
		if err := rows.Scan(
			&id, &eventID, &kind, &agentType,
			&toolCat, &toolName, &toolCmd, &toolQuery, &toolPathsStr,
			&mcpServer, &mcpTool,
			&promptLength,
			&piiDetected, &promptText, &piiMatchesStr,
			&thirdPartyTool, &thirdPartyAction, &attrConf, &detectionMethod,
			&thirdPartyReason, &thirdPartyHookEvent,
			&actionType, &policyName, &policyID,
			&matchedOn, &matchedValue, &agentMessage, &latencyMs,
			&occurredAt,
		); err != nil {
			log.Printf("api: GetSessionTimeline scan: %v", err)
			continue
		}
		var toolPaths []string
		if toolPathsStr != "" && toolPathsStr != "null" {
			_ = json.Unmarshal([]byte(toolPathsStr), &toolPaths)
		}
		var piiMatches any
		_ = json.Unmarshal([]byte(piiMatchesStr), &piiMatches)

		ev := map[string]any{
			"activityEventId": id,
			"eventId":         eventID,
			"kind":            kind,
			"agentType":       agentType,
			"tool": map[string]any{
				"category": toolCat,
				"name":     toolName,
				"command":  toolCmd,
				"query":    toolQuery,
				"paths":    toolPaths,
				"mcp":      map[string]any{"server": mcpServer, "name": mcpTool},
			},
			"promptLength":          promptLength,
			"piiDetected":           piiDetected,
			"piiMatches":            piiMatches,
			"thirdPartyTool":        thirdPartyTool,
			"thirdPartyAction":      thirdPartyAction,
			"attributionConfidence": attrConf,
			"detectionMethod":       detectionMethod,
			"thirdPartyReason":      thirdPartyReason,
			"thirdPartyHookEvent":   thirdPartyHookEvent,
			"decision": map[string]any{
				"action":       actionType,
				"policyName":   policyName,
				"policyId":     policyID,
				"matchedOn":    matchedOn,
				"matchedValue": matchedValue,
				"agentMessage": agentMessage,
				"latencyMs":    latencyMs,
			},
			"occurredAt": occurredAt,
		}
		if piiDetected && promptText != "" {
			ev["promptText"] = promptText
		}
		events = append(events, ev)
	}
	jsonOK(w, dedupeThirdPartyEvents(events))
}

// thirdPartyDedupeWindow bounds how close two tool.third_party_enforcement
// events must occur to be considered the same underlying tool call reported
// twice (e.g. by two detection mechanisms firing in the same PostToolUse
// handler). Both mechanisms run synchronously back-to-back, so a real
// duplicate lands within a few hundred milliseconds — this window is
// generous without risking merging two genuinely separate tool calls.
const thirdPartyDedupeWindow = 5 * time.Second

// dedupeThirdPartyEvents drops a tool.third_party_enforcement event that
// reports the same tool call as the immediately preceding one: same blocked
// tool, same message, within thirdPartyDedupeWindow. Events are assumed
// ordered by occurred_at ascending. Keeps the first (earliest) occurrence.
func dedupeThirdPartyEvents(events []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(events))
	var lastThirdParty map[string]any
	for _, ev := range events {
		if ev["kind"] != "tool.third_party_enforcement" {
			out = append(out, ev)
			continue
		}
		if lastThirdParty != nil && isSameThirdPartyReport(lastThirdParty, ev) {
			continue // duplicate — skip, keep the earlier one already in out
		}
		lastThirdParty = ev
		out = append(out, ev)
	}
	return out
}

func isSameThirdPartyReport(a, b map[string]any) bool {
	aTime, aOK := a["occurredAt"].(time.Time)
	bTime, bOK := b["occurredAt"].(time.Time)
	if !aOK || !bOK {
		return false
	}
	delta := bTime.Sub(aTime)
	if delta < 0 {
		delta = -delta
	}
	if delta > thirdPartyDedupeWindow {
		return false
	}
	aTool, _ := a["tool"].(map[string]any)
	bTool, _ := b["tool"].(map[string]any)
	if aTool["name"] != bTool["name"] {
		return false
	}
	aDecision, _ := a["decision"].(map[string]any)
	bDecision, _ := b["decision"].(map[string]any)
	return aDecision["agentMessage"] == bDecision["agentMessage"]
}

// GET /api/v1/sessions/:id/timeline/export
func ExportSessionTimeline(w http.ResponseWriter, r *http.Request) {
	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid session id")
		return
	}
	rows, err := db.Pool.Query(r.Context(), `
		SELECT ae.occurred_at, ae.kind, ae.action_type,
		       COALESCE(ae.tool_category,''), COALESCE(ae.tool_name,''),
		       COALESCE(ae.tool_command,''), COALESCE(ae.matched_value,''),
		       COALESCE(ae.policy_name,'')
		FROM activity_events ae
		WHERE ae.session_id = $1
		ORDER BY ae.occurred_at`, sessionID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	headers := []string{"Timestamp", "Kind", "Action", "Category", "Tool", "Command", "MatchedValue", "Policy"}
	var data [][]string
	for rows.Next() {
		var t time.Time
		var kind, action, cat, name, cmd, matched, policy string
		if err := rows.Scan(&t, &kind, &action, &cat, &name, &cmd, &matched, &policy); err != nil {
			continue
		}
		data = append(data, []string{t.Format(time.RFC3339), kind, action, cat, name, cmd, matched, policy})
	}
	streamExport(w, r.URL.Query().Get("format"), "session-timeline", headers, data)
}

// ============================================================
// GET /api/v1/violations
// ============================================================

func ListViolations(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r.URL.Query())
	rows, err := db.Pool.Query(r.Context(), `
		SELECT
			pa.policy_action_id::text,  pa.action_type::text,
			COALESCE(pa.violation_type::text,''),  pa.severity::text,
			COALESCE(pa.resource_type,''),   COALESCE(pa.resource_id,''),
			COALESCE(pa.reason,''),          COALESCE(pa.matched_rule,''),
			pa.occurred_at,                  pa.resolution_status::text,
			d.email,                         COALESCE(d.username,''),
			a.agent_name,
			COALESCE(da.machine_id,''),
			COALESCE(p.policy_name,''),
			COALESCE(pa.metadata::text,'{}')
		FROM policy_actions pa
		JOIN developers d       ON pa.developer_id = d.developer_id
		JOIN ai_coding_agents a ON pa.agent_id     = a.agent_id
		LEFT JOIN ai_sessions sess ON sess.session_id = pa.session_id
		LEFT JOIN developer_agents da ON da.developer_agent_id = sess.developer_agent_id
		LEFT JOIN policies p ON pa.policy_id = p.policy_id
		WHERE pa.action_type IN ('BLOCK', 'ALERT')
		  AND (pa.resource_type != 'third_party' OR pa.metadata->>'detectionMethod' = 'transcript')
		ORDER BY pa.occurred_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var (
			paID, actionType, violationType, severity                           string
			resourceType, resourceID, reason, matchedRule                       string
			resolutionStatus, email, username, agentName, machineID, policyName string
			metaStr                                                             string
			occurredAt                                                          time.Time
		)
		if err := rows.Scan(&paID, &actionType, &violationType, &severity,
			&resourceType, &resourceID, &reason, &matchedRule,
			&occurredAt, &resolutionStatus,
			&email, &username, &agentName, &machineID, &policyName, &metaStr); err != nil {
			log.Printf("api: ListViolations scan: %v", err)
			continue
		}
		var meta map[string]any
		_ = json.Unmarshal([]byte(metaStr), &meta)
		result = append(result, map[string]any{
			"policyActionId":   paID,
			"actionType":       actionType,
			"violationType":    violationType,
			"severity":         severity,
			"resourceType":     resourceType,
			"resourceId":       resourceID,
			"reason":           reason,
			"matchedRule":      matchedRule,
			"policyName":       policyName,
			"occurredAt":       occurredAt,
			"resolutionStatus": resolutionStatus,
			"developer":        map[string]any{"email": email, "username": username},
			"agentName":        agentName,
			"machineId":        machineID,
			"metadata":         meta,
		})
	}
	jsonOK(w, result)
}

// GET /api/v1/violations/export
func ExportViolations(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Pool.Query(r.Context(), `
		SELECT pa.occurred_at, d.email, COALESCE(da.machine_id,''), a.agent_name,
		       pa.action_type::text, COALESCE(p.policy_name,''),
		       COALESCE(pa.resource_type,''), COALESCE(pa.violation_type::text,''),
		       COALESCE(pa.resource_id,'')
		FROM policy_actions pa
		JOIN developers d       ON pa.developer_id = d.developer_id
		JOIN ai_coding_agents a ON pa.agent_id     = a.agent_id
		LEFT JOIN ai_sessions sess ON sess.session_id = pa.session_id
		LEFT JOIN developer_agents da ON da.developer_agent_id = sess.developer_agent_id
		LEFT JOIN policies p ON pa.policy_id = p.policy_id
		WHERE pa.action_type IN ('BLOCK','ALERT')
		  AND (pa.resource_type != 'third_party' OR pa.metadata->>'detectionMethod' = 'transcript')
		ORDER BY pa.occurred_at DESC`)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	headers := []string{"Timestamp", "Developer", "MachineID", "Agent", "Action", "Policy", "ResourceType", "ViolationType", "MatchedValue"}
	var data [][]string
	for rows.Next() {
		var t time.Time
		var email, machineID, agentName, action, policy, resourceType, violationType, resourceID string
		if err := rows.Scan(&t, &email, &machineID, &agentName, &action, &policy, &resourceType, &violationType, &resourceID); err != nil {
			continue
		}
		data = append(data, []string{t.Format(time.RFC3339), email, machineID, agentName, action, policy, resourceType, violationType, resourceID})
	}
	streamExport(w, r.URL.Query().Get("format"), "violations", headers, data)
}

// ============================================================
// GET /api/v1/developers/list
// ============================================================

func ListDevelopers(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r.URL.Query())
	rows, err := db.Pool.Query(r.Context(), `
		SELECT d.developer_id::text, d.email,
		       COALESCE(d.display_name,''), COALESCE(d.username,''),
		       COALESCE(d.role,'developer'), COALESCE(d.status::text,'active'),
		       d.first_seen_at, d.last_seen_at,
		       COUNT(DISTINCT pa.policy_action_id)
		         FILTER (WHERE pa.action_type IN ('BLOCK','ALERT'))::int AS violation_count
		FROM developers d
		LEFT JOIN policy_actions pa ON pa.developer_id = d.developer_id
		GROUP BY d.developer_id
		ORDER BY d.last_seen_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var id, email, display, username, role, status string
		var firstSeen, lastSeen time.Time
		var violations int
		if err := rows.Scan(&id, &email, &display, &username, &role, &status, &firstSeen, &lastSeen, &violations); err != nil {
			log.Printf("api: ListDevelopers scan: %v", err)
			continue
		}
		result = append(result, map[string]any{
			"developerId": id, "email": email, "displayName": display,
			"username": username, "role": role, "status": status,
			"firstSeenAt": firstSeen, "lastSeenAt": lastSeen,
			"violationCount": violations,
		})
	}
	jsonOK(w, result)
}

// ============================================================
// GET /api/v1/agents
// ============================================================

func ListAgents(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Pool.Query(r.Context(), `
		SELECT a.agent_id::text, a.agent_name,
		       COALESCE(a.vendor,''),    COALESCE(a.agent_type::text,''),
		       COALESCE(a.version,''),   COALESCE(a.status::text,''),
		       a.first_seen_at, a.last_seen_at,
		       COUNT(DISTINCT da.developer_agent_id)::int AS instance_count
		FROM ai_coding_agents a
		LEFT JOIN developer_agents da ON da.agent_product_id = a.agent_id
		GROUP BY a.agent_id
		ORDER BY a.agent_name`)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var id, name, vendor, agentType, version, status string
		var firstSeen, lastSeen time.Time
		var instances int
		if err := rows.Scan(&id, &name, &vendor, &agentType, &version, &status, &firstSeen, &lastSeen, &instances); err != nil {
			log.Printf("api: ListAgents scan: %v", err)
			continue
		}
		result = append(result, map[string]any{
			"agentId": id, "agentName": name, "vendor": vendor,
			"agentType": agentType, "version": version, "status": status,
			"firstSeenAt": firstSeen, "lastSeenAt": lastSeen,
			"instanceCount": instances,
		})
	}
	jsonOK(w, result)
}

// ============================================================
// GET /api/v1/mcp-servers
// ============================================================

func ListMCPServers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Pool.Query(r.Context(), `
		SELECT m.mcp_server_id::text, m.server_name,
		       COALESCE(m.server_url,''), COALESCE(m.vendor,''), m.status::text,
		       m.first_seen_at, m.last_seen_at,
		       COUNT(DISTINCT sm.session_id)::int            AS session_count,
		       COALESCE(SUM(sm.tool_call_count),0)::int      AS total_calls
		FROM mcp_servers m
		LEFT JOIN session_mcp_servers sm ON sm.mcp_server_id = m.mcp_server_id
		GROUP BY m.mcp_server_id
		ORDER BY m.last_seen_at DESC`)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var id, name, url, vendor, status string
		var firstSeen, lastSeen time.Time
		var sessions, calls int
		if err := rows.Scan(&id, &name, &url, &vendor, &status, &firstSeen, &lastSeen, &sessions, &calls); err != nil {
			log.Printf("api: ListMCPServers scan: %v", err)
			continue
		}
		result = append(result, map[string]any{
			"mcpServerId": id, "serverName": name, "serverUrl": url,
			"vendor": vendor, "status": status,
			"firstSeenAt": firstSeen, "lastSeenAt": lastSeen,
			"sessionCount": sessions, "totalCalls": calls,
		})
	}
	jsonOK(w, result)
}

// GET /api/v1/mcp-servers/:id
func GetMCPServer(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid mcp server id")
		return
	}
	var serverName, serverURL, vendor, status string
	var firstSeen, lastSeen time.Time
	if err := db.Pool.QueryRow(r.Context(), `
		SELECT server_name, COALESCE(server_url,''), COALESCE(vendor,''),
		       status::text, first_seen_at, last_seen_at
		FROM mcp_servers WHERE mcp_server_id = $1`, id,
	).Scan(&serverName, &serverURL, &vendor, &status, &firstSeen, &lastSeen); err != nil {
		jsonError(w, http.StatusNotFound, "mcp server not found")
		return
	}
	jsonOK(w, map[string]any{
		"mcpServerId": id, "serverName": serverName, "serverUrl": serverURL,
		"vendor": vendor, "status": status,
		"firstSeenAt": firstSeen, "lastSeenAt": lastSeen,
	})
}

// ============================================================
// GET /api/v1/skills
// GET /api/v1/skills/{id}
// GET /api/v1/sessions/{id}/skills
// ============================================================

func ListSkills(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r.URL.Query())
	rows, err := db.Pool.Query(r.Context(), `
		SELECT
			s.skill_id::text, s.skill_name, s.status,
			s.first_seen_at, s.last_seen_at,
			s.total_invocations,
			COUNT(DISTINCT ss.session_id)::int AS session_count
		FROM skills s
		LEFT JOIN session_skills ss ON s.skill_id = ss.skill_id
		GROUP BY s.skill_id, s.skill_name, s.status, s.first_seen_at, s.last_seen_at, s.total_invocations
		ORDER BY s.last_seen_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var skillID, skillName, status string
		var firstSeenAt, lastSeenAt time.Time
		var totalInvocations, sessionCount int
		if err := rows.Scan(&skillID, &skillName, &status, &firstSeenAt, &lastSeenAt, &totalInvocations, &sessionCount); err != nil {
			log.Printf("api: ListSkills scan: %v", err)
			continue
		}
		result = append(result, map[string]any{
			"skillId":          skillID,
			"skillName":        skillName,
			"status":           status,
			"firstSeenAt":      firstSeenAt,
			"lastSeenAt":       lastSeenAt,
			"totalInvocations": totalInvocations,
			"sessionCount":     sessionCount,
		})
	}
	jsonOK(w, result)
}

func GetSkillDetail(w http.ResponseWriter, r *http.Request) {
	skillID := chi.URLParam(r, "id")

	var skillName, status string
	var firstSeenAt, lastSeenAt time.Time
	var totalInvocations int
	if err := db.Pool.QueryRow(r.Context(), `
		SELECT skill_name, status, first_seen_at, last_seen_at, total_invocations
		FROM skills WHERE skill_id = $1`, skillID,
	).Scan(&skillName, &status, &firstSeenAt, &lastSeenAt, &totalInvocations); err != nil {
		jsonError(w, http.StatusNotFound, "skill not found")
		return
	}

	rows, err := db.Pool.Query(r.Context(), `
		SELECT
			s.session_id::text, s.claude_session_id,
			ss.invocation_count,
			ss.first_invoked_at, ss.last_invoked_at,
			d.email, a.agent_name
		FROM session_skills ss
		JOIN ai_sessions s       ON ss.session_id = s.session_id
		JOIN developers d        ON s.developer_id = d.developer_id
		JOIN ai_coding_agents a  ON s.agent_id = a.agent_id
		WHERE ss.skill_id = $1
		ORDER BY ss.last_invoked_at DESC`, skillID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	sessions := []map[string]any{}
	for rows.Next() {
		var sessionID, claudeID, email, agentName string
		var invocationCount int
		var firstInvokedAt, lastInvokedAt time.Time
		if err := rows.Scan(&sessionID, &claudeID, &invocationCount, &firstInvokedAt, &lastInvokedAt, &email, &agentName); err != nil {
			continue
		}
		sessions = append(sessions, map[string]any{
			"sessionId":       sessionID,
			"claudeId":        claudeID,
			"invocationCount": invocationCount,
			"firstInvokedAt":  firstInvokedAt,
			"lastInvokedAt":   lastInvokedAt,
			"developer":       email,
			"agent":           agentName,
		})
	}

	jsonOK(w, map[string]any{
		"skillId":          skillID,
		"skillName":        skillName,
		"status":           status,
		"firstSeenAt":      firstSeenAt,
		"lastSeenAt":       lastSeenAt,
		"totalInvocations": totalInvocations,
		"sessions":         sessions,
	})
}

func GetSessionSkills(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")

	rows, err := db.Pool.Query(r.Context(), `
		SELECT
			s.skill_id::text, s.skill_name, s.status,
			ss.invocation_count,
			ss.first_invoked_at, ss.last_invoked_at
		FROM session_skills ss
		JOIN skills s ON ss.skill_id = s.skill_id
		WHERE ss.session_id = $1
		ORDER BY ss.last_invoked_at DESC`, sessionID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var skillID, skillName, status string
		var invocationCount int
		var firstInvokedAt, lastInvokedAt time.Time
		if err := rows.Scan(&skillID, &skillName, &status, &invocationCount, &firstInvokedAt, &lastInvokedAt); err != nil {
			continue
		}
		result = append(result, map[string]any{
			"skillId":         skillID,
			"skillName":       skillName,
			"status":          status,
			"invocationCount": invocationCount,
			"firstInvokedAt":  firstInvokedAt,
			"lastInvokedAt":   lastInvokedAt,
		})
	}
	jsonOK(w, result)
}

// ============================================================
// GET /api/v1/audit
// ============================================================

func ListAuditEvents(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r.URL.Query())
	rows, err := db.Pool.Query(r.Context(), `
		SELECT ae.audit_event_id::text, ae.actor_type::text,
		       ae.event_type, COALESCE(ae.resource_type,''),
		       ae.timestamp, COALESCE(ae.metadata::text,'{}'),
		       COALESCE(d.email, '') AS actor_email
		FROM audit_events ae
		LEFT JOIN developers d ON ae.actor_id = d.developer_id
		ORDER BY ae.timestamp DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var id, actorType, eventType, resourceType, actorEmail, metaStr string
		var ts time.Time
		if err := rows.Scan(&id, &actorType, &eventType, &resourceType, &ts, &metaStr, &actorEmail); err != nil {
			log.Printf("api: ListAuditEvents scan: %v", err)
			continue
		}
		var meta map[string]any
		_ = json.Unmarshal([]byte(metaStr), &meta)
		result = append(result, map[string]any{
			"auditEventId": id, "actorType": actorType, "eventType": eventType,
			"resourceType": resourceType, "timestamp": ts,
			"metadata": meta, "actorEmail": actorEmail,
		})
	}
	jsonOK(w, result)
}

// ============================================================
// GET /api/v1/overview
// ============================================================

func GetOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	expireInactiveSessions(ctx)

	var devCount, activeSessionCount, violationsToday, mcpServerCount int
	_ = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM developers WHERE status = 'active'`).Scan(&devCount)
	_ = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM ai_sessions WHERE status = 'active'`).Scan(&activeSessionCount)
	_ = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM policy_actions
		WHERE action_type IN ('BLOCK','ALERT')
		  AND occurred_at >= NOW() - INTERVAL '24 hours'`).Scan(&violationsToday)
	_ = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM mcp_servers WHERE status = 'active'`).Scan(&mcpServerCount)

	var tokensToday int64
	var costToday *float64
	_ = db.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(input_tokens + output_tokens + cache_read_tokens + cache_creation_tokens),0)::bigint,
		       SUM(cost_usd)::float8
		FROM token_usage_effective
		WHERE occurred_at >= date_trunc('day', NOW())`).Scan(&tokensToday, &costToday)

	jsonOK(w, map[string]any{
		"tokensToday":          tokensToday,
		"costTodayUsd":         costToday,
		"developerCount":       devCount,
		"activeSessionCount":   activeSessionCount,
		"violationsToday":      violationsToday,
		"mcpServersDiscovered": mcpServerCount,
	})
}

// ============================================================
// Export helpers
// ============================================================

func streamExport(w http.ResponseWriter, format, filename string, headers []string, rows [][]string) {
	if format == "xlsx" {
		streamExcel(w, filename, headers, rows)
		return
	}
	streamCSV(w, filename, headers, rows)
}

func streamCSV(w http.ResponseWriter, filename string, headers []string, rows [][]string) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, filename))
	cw := csv.NewWriter(w)
	_ = cw.Write(headers)
	for _, row := range rows {
		_ = cw.Write(row)
	}
	cw.Flush()
}

func streamExcel(w http.ResponseWriter, filename string, headers []string, rows [][]string) {
	f := excelize.NewFile()
	sheet := "Sheet1"
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for ri, row := range rows {
		for ci, val := range row {
			cell, _ := excelize.CoordinatesToCellName(ci+1, ri+2)
			_ = f.SetCellValue(sheet, cell, val)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		jsonError(w, http.StatusInternalServerError, "excel: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.xlsx"`, filename))
	_, _ = w.Write(buf.Bytes())
}

func parsePagination(q interface{ Get(string) string }) (limit, offset int) {
	limit, offset = 100, 0
	if v := q.Get("limit"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &limit); err != nil {
			limit = 100
		}
	}
	if v := q.Get("offset"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &offset); err != nil {
			offset = 0
		}
	}
	if limit > 500 {
		limit = 500
	}
	return
}
