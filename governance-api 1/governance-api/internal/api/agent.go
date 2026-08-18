package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/checkmarx/governance-api/internal/db"
	"github.com/checkmarx/governance-api/internal/ingest"
	"github.com/checkmarx/governance-api/internal/models"
	"github.com/google/uuid"
)

// GET /api/v1/agent/policies — serve the current governance pack (policy-pack.json)
func GetAgentPolicies(w http.ResponseWriter, r *http.Request) {
	pack, err := ingest.BuildGovernancePack(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "build pack: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Pack-Version", strconv.Itoa(pack.PackVersion))
	if err := json.NewEncoder(w).Encode(pack); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
	}
}

// HEAD /api/v1/agent/policies/version — returns X-Pack-Version header only.
// Used by cx CheckAndSyncVersion to detect policy changes without downloading the full pack.
func HeadAgentPoliciesVersion(w http.ResponseWriter, r *http.Request) {
	packVersion, err := ingest.CurrentPackVersion(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("X-Pack-Version", strconv.Itoa(packVersion))
	w.WriteHeader(http.StatusOK)
}

// POST /api/v1/audit/events — ingest ActivityEvent batch from the cx spool flusher.
func IngestAuditEvents(w http.ResponseWriter, r *http.Request) {
	var req models.IngestAuditRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Events) == 0 {
		jsonOK(w, map[string]any{"accepted": 0})
		return
	}
	proc := ingest.New()
	if err := proc.ProcessBatch(r.Context(), req.Events); err != nil {
		jsonError(w, http.StatusInternalServerError, "ingest: "+err.Error())
		return
	}
	jsonOK(w, map[string]any{"accepted": len(req.Events)})
}

// GET /api/v1/developers?username={u} — resolve OS username to developer_id (§6.1).
func GetDeveloperByUsername(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		jsonError(w, http.StatusBadRequest, "username query param required")
		return
	}
	var devID, email, displayName string
	err := db.Pool.QueryRow(r.Context(),
		`SELECT developer_id::text, email, COALESCE(display_name,'') FROM developers WHERE username = $1`,
		username,
	).Scan(&devID, &email, &displayName)
	if err != nil {
		jsonError(w, http.StatusNotFound, fmt.Sprintf("developer %q not found", username))
		return
	}
	jsonOK(w, map[string]any{
		"developerId": devID,
		"email":       email,
		"displayName": displayName,
	})
}

// POST /api/v1/agents/register — upsert developer_agent row on first governance hook.
func RegisterAgent(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterAgentRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.AgentID == "" || req.AgentType == "" {
		jsonError(w, http.StatusBadRequest, "agentId and agentType are required")
		return
	}
	ctx := r.Context()

	// Resolve or create developer
	email := req.UserID + "@unknown.local"
	var devID uuid.UUID
	if err := db.Pool.QueryRow(ctx, `
		INSERT INTO developers (email, username, display_name)
		VALUES ($1, $2, $2)
		ON CONFLICT (email) DO UPDATE SET last_seen_at = NOW()
		RETURNING developer_id`,
		email, req.UserID,
	).Scan(&devID); err != nil {
		jsonError(w, http.StatusInternalServerError, "resolve developer: "+err.Error())
		return
	}

	// Resolve agent product
	agentName := agentTypeDisplayName(req.AgentType)
	var agentProdID uuid.UUID
	if err := db.Pool.QueryRow(ctx, `
		SELECT agent_id FROM ai_coding_agents
		WHERE LOWER(agent_name) = LOWER($1) LIMIT 1`,
		agentName,
	).Scan(&agentProdID); err != nil {
		// Upsert unknown agent
		if err2 := db.Pool.QueryRow(ctx, `
			INSERT INTO ai_coding_agents (agent_name)
			VALUES ($1)
			ON CONFLICT (agent_name, vendor, version) DO UPDATE SET last_seen_at = NOW()
			RETURNING agent_id`, agentName,
		).Scan(&agentProdID); err2 != nil {
			jsonError(w, http.StatusInternalServerError, "resolve agent: "+err2.Error())
			return
		}
	}

	// Upsert developer_agent bridge row
	var devAgentID uuid.UUID
	if err := db.Pool.QueryRow(ctx, `
		INSERT INTO developer_agents (developer_id, agent_product_id, machine_id, governance_agent_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (governance_agent_id)
		DO UPDATE SET last_seen_at = NOW()
		RETURNING developer_agent_id`,
		devID, agentProdID, req.MachineID, req.AgentID,
	).Scan(&devAgentID); err != nil {
		jsonError(w, http.StatusInternalServerError, "upsert agent: "+err.Error())
		return
	}

	jsonOK(w, map[string]any{"developerAgentId": devAgentID})
}

func agentTypeDisplayName(t string) string {
	switch strings.ToLower(t) {
	case "claude-code", "claude":
		return "Claude Code"
	case "cursor":
		return "Cursor"
	default:
		return t
	}
}
