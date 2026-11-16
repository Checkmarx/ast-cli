package api

import (
	"encoding/json"
	"net/http"

	"github.com/checkmarx/governance-api/internal/db"
	"github.com/checkmarx/governance-api/internal/ingest"
	"github.com/checkmarx/governance-api/internal/models"
	"github.com/checkmarx/governance-api/internal/services"
	"github.com/checkmarx/governance-api/pkg/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// resolveActorID resolves the authenticated actor into a *uuid.UUID for audit_events.actor_id,
// and an actor_type ('developer' when resolvable, 'system' otherwise). Closes G1 —
// middleware.ActorID(r) already existed but was never called from these handlers
// (see COMPLIANCE_ARCHITECTURE.md §5.6).
func resolveActorID(r *http.Request) (*uuid.UUID, string) {
	raw := middleware.ActorID(r)
	if id, err := uuid.Parse(raw); err == nil {
		return &id, "developer"
	}
	return nil, "system"
}

// autoMapPolicyControls runs the auto-mapping rule engine (§5.7) for a single policy.
// Best-effort: a failure here never blocks the policy write that triggered it.
func autoMapPolicyControls(r *http.Request, policyID uuid.UUID, policyType string, def json.RawMessage) {
	var pack struct {
		Type        string `json:"type"`
		Action      string `json:"action"`
		MatchTarget string `json:"matchTarget"`
	}
	_ = json.Unmarshal(def, &pack)
	svc := services.NewComplianceService()
	_ = svc.AutoMapPolicyToControls(r.Context(), policyID, services.PolicyAttributes{
		PolicyType:  policyType,
		Type:        pack.Type,
		Action:      pack.Action,
		MatchTarget: pack.MatchTarget,
	})
}

// GET /api/v1/policies — list all non-deleted policies with their current active version.
// Query params: ?status=active|disabled|all  ?type=governance|pii|third_party_monitor|all
func ListPolicies(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")
	typeFilter := r.URL.Query().Get("type")

	query := `
		SELECT
			p.policy_id::text,
			p.policy_name,
			COALESCE(p.description, ''),
			p.status::text,
			COALESCE(p.policy_type, 'governance'),
			p.created_at,
			p.updated_at,
			COALESCE(pv.policy_version_id::text, ''),
			COALESCE(pv.version_number, 0),
			COALESCE(pv.enforcement_mode::text, 'alert'),
			COALESCE(pv.change_summary, ''),
			COALESCE(pv.definition::text, '{}')
		FROM policies p
		LEFT JOIN policy_versions pv ON p.current_version_id = pv.policy_version_id
		WHERE p.deleted_at IS NULL`

	if statusFilter != "" && statusFilter != "all" {
		query += ` AND p.status = '` + statusFilter + `'`
	}
	if typeFilter != "" && typeFilter != "all" {
		query += ` AND p.policy_type = '` + typeFilter + `'`
	}
	query += ` ORDER BY p.created_at DESC`

	rows, err := db.Pool.Query(r.Context(), query)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type policyRow struct {
		PolicyID        string          `json:"policyId"`
		PolicyName      string          `json:"policyName"`
		Description     string          `json:"description"`
		Status          string          `json:"status"`
		PolicyType      string          `json:"policyType"`
		CreatedAt       string          `json:"createdAt"`
		UpdatedAt       string          `json:"updatedAt"`
		VersionID       string          `json:"currentVersionId"`
		VersionNumber   int             `json:"versionNumber"`
		EnforcementMode string          `json:"enforcementMode"`
		ChangeSummary   string          `json:"changeSummary"`
		Definition      json.RawMessage `json:"definition"`
	}

	result := []policyRow{}
	for rows.Next() {
		var row policyRow
		var defStr string
		var createdAt, updatedAt interface{}
		if err := rows.Scan(
			&row.PolicyID, &row.PolicyName, &row.Description, &row.Status, &row.PolicyType,
			&createdAt, &updatedAt,
			&row.VersionID, &row.VersionNumber, &row.EnforcementMode,
			&row.ChangeSummary, &defStr,
		); err != nil {
			continue
		}
		if t, ok := createdAt.(interface{ String() string }); ok {
			row.CreatedAt = t.String()
		}
		if t, ok := updatedAt.(interface{ String() string }); ok {
			row.UpdatedAt = t.String()
		}
		if defStr != "" {
			row.Definition = json.RawMessage(defStr)
		}
		result = append(result, row)
	}
	if result == nil {
		result = []policyRow{}
	}
	jsonOK(w, result)
}

// POST /api/v1/policies — create a new policy with its initial version
func CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var req models.CreatePolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.PolicyName == "" {
		jsonError(w, http.StatusBadRequest, "policyName is required")
		return
	}
	if req.EnforcementMode == "" {
		req.EnforcementMode = "alert"
	}

	ctx := r.Context()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var policyID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO policies (policy_name, description)
		VALUES ($1, $2)
		RETURNING policy_id`,
		req.PolicyName, req.Description,
	).Scan(&policyID); err != nil {
		jsonError(w, http.StatusInternalServerError, "insert policy: "+err.Error())
		return
	}

	var pvID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO policy_versions (policy_id, version_number, definition, enforcement_mode, change_summary)
		VALUES ($1, 1, $2, $3, $4)
		RETURNING policy_version_id`,
		policyID, req.Definition, req.EnforcementMode, req.ChangeSummary,
	).Scan(&pvID); err != nil {
		jsonError(w, http.StatusInternalServerError, "insert version: "+err.Error())
		return
	}

	// Point current_version_id at the new version
	if _, err := tx.Exec(ctx,
		`UPDATE policies SET current_version_id = $1, updated_at = NOW() WHERE policy_id = $2`,
		pvID, policyID,
	); err != nil {
		jsonError(w, http.StatusInternalServerError, "update policy: "+err.Error())
		return
	}

	// Audit: POLICY_CREATED
	actorID, actorType := resolveActorID(r)
	defJSON, _ := json.Marshal(req)
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (actor_id, actor_type, event_type, resource_type, resource_id, new_value)
		VALUES ($1, $2, $3, 'policy', $4, $5)`,
		actorID, actorType, models.AuditPolicyCreated, policyID, defJSON,
	); err != nil {
		jsonError(w, http.StatusInternalServerError, "audit: "+err.Error())
		return
	}

	if err := tx.Commit(ctx); err != nil {
		jsonError(w, http.StatusInternalServerError, "commit: "+err.Error())
		return
	}

	// Auto-map to compliance controls (§5.7) — event-driven, immediate.
	autoMapPolicyControls(r, policyID, "governance", req.Definition)

	jsonCreated(w, map[string]any{
		"policyId":        policyID,
		"policyVersionId": pvID,
		"versionNumber":   1,
	})
}

// PUT /api/v1/policies/:id/activate — activate a specific version and bump packVersion
func ActivatePolicy(w http.ResponseWriter, r *http.Request) {
	policyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid policy id")
		return
	}

	var req models.ActivatePolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := r.Context()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Verify policyVersionId belongs to policyId
	var versionNumber int
	if err := tx.QueryRow(ctx,
		`SELECT version_number FROM policy_versions WHERE policy_version_id = $1 AND policy_id = $2`,
		req.PolicyVersionID, policyID,
	).Scan(&versionNumber); err != nil {
		jsonError(w, http.StatusNotFound, "policy version not found")
		return
	}

	// Update current_version_id
	if _, err := tx.Exec(ctx,
		`UPDATE policies SET current_version_id = $1, updated_at = NOW() WHERE policy_id = $2`,
		req.PolicyVersionID, policyID,
	); err != nil {
		jsonError(w, http.StatusInternalServerError, "update: "+err.Error())
		return
	}

	// Increment global packVersion
	packVersion, err := ingest.NextPackVersion(ctx)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "pack version: "+err.Error())
		return
	}

	// Audit: POLICY_ACTIVATED
	actorID, actorType := resolveActorID(r)
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (actor_id, actor_type, event_type, resource_type, resource_id)
		VALUES ($1, $2, $3, 'policy', $4)`,
		actorID, actorType, models.AuditPolicyActivated, policyID,
	); err != nil {
		jsonError(w, http.StatusInternalServerError, "audit: "+err.Error())
		return
	}

	var activeDef json.RawMessage
	var activePolicyType string
	_ = tx.QueryRow(ctx, `
		SELECT pv.definition, COALESCE(p.policy_type, 'governance')
		FROM policy_versions pv JOIN policies p ON p.policy_id = pv.policy_id
		WHERE pv.policy_version_id = $1`, req.PolicyVersionID,
	).Scan(&activeDef, &activePolicyType)

	if err := tx.Commit(ctx); err != nil {
		jsonError(w, http.StatusInternalServerError, "commit: "+err.Error())
		return
	}

	// Auto-map to compliance controls (§5.7) — activation is the other event-driven moment.
	if activeDef != nil {
		autoMapPolicyControls(r, policyID, activePolicyType, activeDef)
	}

	jsonOK(w, models.ActivatePolicyResponse{
		PolicyID:      policyID,
		PolicyVersion: versionNumber,
		PackVersion:   packVersion,
	})
}

// GET /api/v1/policies/:id/history — version history for the policy
func GetPolicyHistory(w http.ResponseWriter, r *http.Request) {
	policyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid policy id")
		return
	}

	rows, err := db.Pool.Query(r.Context(), `
		SELECT pv.policy_version_id, pv.version_number, pv.enforcement_mode,
		       pv.created_at, pv.change_summary,
		       (p.current_version_id = pv.policy_version_id) AS is_active
		FROM policy_versions pv
		JOIN policies p ON pv.policy_id = p.policy_id
		WHERE pv.policy_id = $1
		ORDER BY pv.version_number DESC`,
		policyID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type historyRow struct {
		PolicyVersionID string `json:"policyVersionId"`
		VersionNumber   int    `json:"versionNumber"`
		EnforcementMode string `json:"enforcementMode"`
		CreatedAt       string `json:"createdAt"`
		ChangeSummary   string `json:"changeSummary"`
		IsActive        bool   `json:"isActive"`
	}
	history := []historyRow{} // never nil — returns [] not null
	for rows.Next() {
		var h historyRow
		if err := rows.Scan(&h.PolicyVersionID, &h.VersionNumber, &h.EnforcementMode,
			&h.CreatedAt, &h.ChangeSummary, &h.IsActive); err != nil {
			continue
		}
		history = append(history, h)
	}
	jsonOK(w, history)
}

// PUT /api/v1/policies/:id — edit a policy by creating a new version and activating it.
func UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	policyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid policy id")
		return
	}
	var req models.UpdatePolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx := r.Context()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Get next version number
	var nextVersion int
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(MAX(version_number), 0) + 1 FROM policy_versions WHERE policy_id = $1`,
		policyID,
	).Scan(&nextVersion); err != nil {
		jsonError(w, http.StatusInternalServerError, "version query: "+err.Error())
		return
	}

	var pvID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO policy_versions (policy_id, version_number, definition, enforcement_mode, change_summary)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING policy_version_id`,
		policyID, nextVersion, req.Definition, req.EnforcementMode, req.ChangeSummary,
	).Scan(&pvID); err != nil {
		jsonError(w, http.StatusInternalServerError, "insert version: "+err.Error())
		return
	}

	if _, err := tx.Exec(ctx, `
		UPDATE policies SET current_version_id = $1, policy_name = $2, description = $3, updated_at = NOW()
		WHERE policy_id = $4`,
		pvID, req.PolicyName, req.Description, policyID,
	); err != nil {
		jsonError(w, http.StatusInternalServerError, "update policy: "+err.Error())
		return
	}

	packVersion, err := ingest.NextPackVersion(ctx)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "pack version: "+err.Error())
		return
	}

	actorID, actorType := resolveActorID(r)
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (actor_id, actor_type, event_type, resource_type, resource_id)
		VALUES ($1, $2, $3, 'policy', $4)`,
		actorID, actorType, models.AuditPolicyUpdated, policyID,
	); err != nil {
		jsonError(w, http.StatusInternalServerError, "audit: "+err.Error())
		return
	}

	var updatedPolicyType string
	_ = tx.QueryRow(ctx, `SELECT COALESCE(policy_type, 'governance') FROM policies WHERE policy_id = $1`, policyID).Scan(&updatedPolicyType)

	if err := tx.Commit(ctx); err != nil {
		jsonError(w, http.StatusInternalServerError, "commit: "+err.Error())
		return
	}

	autoMapPolicyControls(r, policyID, updatedPolicyType, req.Definition)

	jsonOK(w, map[string]any{"policyId": policyID, "policyVersionId": pvID, "versionNumber": nextVersion, "packVersion": packVersion})
}

// PUT /api/v1/policies/:id/disable — disable a policy (excluded from pack, re-enableable).
func DisablePolicy(w http.ResponseWriter, r *http.Request) {
	setPolicyStatus(w, r, "disabled", models.AuditPolicyDisabled)
}

// PUT /api/v1/policies/:id/enable — re-enable a disabled policy.
func EnablePolicy(w http.ResponseWriter, r *http.Request) {
	setPolicyStatus(w, r, "active", models.AuditPolicyEnabled)
}

func setPolicyStatus(w http.ResponseWriter, r *http.Request, status, auditEvent string) {
	policyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid policy id")
		return
	}
	ctx := r.Context()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	res, err := tx.Exec(ctx,
		`UPDATE policies SET status = $1, updated_at = NOW() WHERE policy_id = $2 AND deleted_at IS NULL`,
		status, policyID,
	)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "update: "+err.Error())
		return
	}
	if res.RowsAffected() == 0 {
		jsonError(w, http.StatusNotFound, "policy not found")
		return
	}

	packVersion, _ := ingest.NextPackVersion(ctx)

	actorID, actorType := resolveActorID(r)
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (actor_id, actor_type, event_type, resource_type, resource_id)
		VALUES ($1, $2, $3, 'policy', $4)`,
		actorID, actorType, auditEvent, policyID,
	); err != nil {
		jsonError(w, http.StatusInternalServerError, "audit: "+err.Error())
		return
	}

	if err := tx.Commit(ctx); err != nil {
		jsonError(w, http.StatusInternalServerError, "commit: "+err.Error())
		return
	}
	jsonOK(w, map[string]any{"policyId": policyID, "status": status, "packVersion": packVersion})
}

// DELETE /api/v1/policies/:id — soft-delete a policy (sets deleted_at, never hard-deleted).
func DeletePolicy(w http.ResponseWriter, r *http.Request) {
	policyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid policy id")
		return
	}
	ctx := r.Context()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	res, err := tx.Exec(ctx,
		`UPDATE policies SET deleted_at = NOW(), status = 'disabled', updated_at = NOW() WHERE policy_id = $1 AND deleted_at IS NULL`,
		policyID,
	)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "delete: "+err.Error())
		return
	}
	if res.RowsAffected() == 0 {
		jsonError(w, http.StatusNotFound, "policy not found or already deleted")
		return
	}

	packVersion, _ := ingest.NextPackVersion(ctx)

	actorID, actorType := resolveActorID(r)
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (actor_id, actor_type, event_type, resource_type, resource_id)
		VALUES ($1, $2, $3, 'policy', $4)`,
		actorID, actorType, models.AuditPolicyDeleted, policyID,
	); err != nil {
		jsonError(w, http.StatusInternalServerError, "audit: "+err.Error())
		return
	}

	if err := tx.Commit(ctx); err != nil {
		jsonError(w, http.StatusInternalServerError, "commit: "+err.Error())
		return
	}
	jsonOK(w, map[string]any{"policyId": policyID, "deleted": true, "packVersion": packVersion})
}

// GET /api/v1/sessions/:id/hooks — return session_hooks for a session.
func GetSessionHooks(w http.ResponseWriter, r *http.Request) {
	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid session id")
		return
	}
	rows, err := db.Pool.Query(r.Context(), `
		SELECT id::text, session_id::text, event, matcher, command, vendor, created_at
		FROM session_hooks WHERE session_id = $1 ORDER BY created_at`, sessionID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var id, sid, event, matcher, command, vendor string
		var createdAt interface{}
		if err := rows.Scan(&id, &sid, &event, &matcher, &command, &vendor, &createdAt); err != nil {
			continue
		}
		result = append(result, map[string]any{
			"id": id, "sessionId": sid, "event": event,
			"matcher": matcher, "command": command, "vendor": vendor, "createdAt": createdAt,
		})
	}
	jsonOK(w, result)
}
