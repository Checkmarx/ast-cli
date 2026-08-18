package api

import (
	"encoding/json"
	"net/http"

	"github.com/checkmarx/governance-api/internal/db"
	"github.com/checkmarx/governance-api/internal/ingest"
	"github.com/checkmarx/governance-api/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// GET /api/v1/policies — list all policies with their current active version
func ListPolicies(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Pool.Query(r.Context(), `
		SELECT
			p.policy_id::text,
			p.policy_name,
			COALESCE(p.description, ''),
			p.status::text,
			p.created_at,
			p.updated_at,
			COALESCE(pv.policy_version_id::text, ''),
			COALESCE(pv.version_number, 0),
			COALESCE(pv.enforcement_mode::text, 'alert'),
			COALESCE(pv.change_summary, ''),
			COALESCE(pv.definition::text, '{}')
		FROM policies p
		LEFT JOIN policy_versions pv ON p.current_version_id = pv.policy_version_id
		ORDER BY p.created_at DESC`)
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
			&row.PolicyID, &row.PolicyName, &row.Description, &row.Status,
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
	defJSON, _ := json.Marshal(req)
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (actor_type, event_type, resource_type, resource_id, new_value)
		VALUES ('system', $1, 'policy', $2, $3)`,
		models.AuditPolicyCreated, policyID, defJSON,
	); err != nil {
		jsonError(w, http.StatusInternalServerError, "audit: "+err.Error())
		return
	}

	if err := tx.Commit(ctx); err != nil {
		jsonError(w, http.StatusInternalServerError, "commit: "+err.Error())
		return
	}

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
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (actor_type, event_type, resource_type, resource_id)
		VALUES ('system', $1, 'policy', $2)`,
		models.AuditPolicyActivated, policyID,
	); err != nil {
		jsonError(w, http.StatusInternalServerError, "audit: "+err.Error())
		return
	}

	if err := tx.Commit(ctx); err != nil {
		jsonError(w, http.StatusInternalServerError, "commit: "+err.Error())
		return
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
