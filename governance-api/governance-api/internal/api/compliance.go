// Compliance API handlers (POC scope) — see COMPLIANCE_ARCHITECTURE.md §7.
// New file, additive only. All compliance state is read via ComplianceService/
// OversightService; nothing here writes to audit_events/activity_events/policy_actions
// directly except the review endpoint's use of OversightService (§5.5).
package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/checkmarx/governance-api/internal/db"
	"github.com/checkmarx/governance-api/internal/models"
	"github.com/checkmarx/governance-api/internal/services"
	"github.com/checkmarx/governance-api/pkg/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ============================================================
// POST /api/v1/violations/:id/review — closes G2 (§5.5, §7)
// ============================================================

func ReviewViolation(w http.ResponseWriter, r *http.Request) {
	policyActionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid violation id")
		return
	}
	var req models.ReviewViolationRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Decision != "affirmed" && req.Decision != "overridden" && req.Decision != "escalated" {
		jsonError(w, http.StatusBadRequest, "decision must be affirmed, overridden, or escalated")
		return
	}

	reviewerID, err := reviewerActorID(r)
	if err != nil {
		jsonError(w, http.StatusUnauthorized, "unable to resolve reviewer identity")
		return
	}

	svc := services.NewOversightService()
	err = svc.ReviewViolation(r.Context(), policyActionID, reviewerID, req.Decision, req.Rationale)
	switch {
	case err == nil:
		jsonOK(w, map[string]string{"status": statusForDecision(req.Decision)})
	case errors.Is(err, services.ErrSegregationOfDuties):
		jsonError(w, http.StatusForbidden, "reviewer must differ from the violation's actor")
	case errors.Is(err, services.ErrViolationNotFound):
		jsonError(w, http.StatusNotFound, "violation not found")
	default:
		jsonError(w, http.StatusInternalServerError, err.Error())
	}
}

func statusForDecision(decision string) string {
	if decision == "escalated" {
		return "open"
	}
	return "resolved"
}

// reviewerActorID resolves the authenticated caller to a developer UUID for use as
// the reviewer. Unlike resolveActorID (policies.go), a reviewer identity is required
// — an unresolvable actor cannot perform oversight (§5.5's guarantee must be real).
func reviewerActorID(r *http.Request) (uuid.UUID, error) {
	raw := middleware.ActorID(r)
	if id, err := uuid.Parse(raw); err == nil {
		return id, nil
	}
	return uuid.UUID{}, errors.New("actor is not a resolvable developer")
}

// ============================================================
// GET /api/v1/compliance/frameworks (§7)
// ============================================================

func ListComplianceFrameworks(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Pool.Query(r.Context(), `
		SELECT framework_id::text, framework_type::text, display_name
		FROM compliance_frameworks
		ORDER BY display_name`)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	compliance := services.NewComplianceService()
	result := []models.FrameworkStatusResponse{}
	for rows.Next() {
		var id, ftype, name string
		if err := rows.Scan(&id, &ftype, &name); err != nil {
			continue
		}
		fid, _ := uuid.Parse(id)
		resp := models.FrameworkStatusResponse{FrameworkID: id, FrameworkType: ftype, DisplayName: name}
		if snap, err := compliance.LatestSnapshot(r.Context(), fid); err == nil {
			resp.ApplicableControlCount = snap.ApplicableControlCount
			resp.CompliantCount = snap.CompliantCount
			resp.PartiallyCompliantCount = snap.PartiallyCompliantCount
			resp.NonCompliantCount = snap.NonCompliantCount
			resp.PlannedCount = snap.PlannedCount
			t := snap.CalculatedAt
			resp.CalculatedAt = &t
		}
		result = append(result, resp)
	}
	jsonOK(w, result)
}

// ============================================================
// GET /api/v1/compliance/frameworks/:id/controls (§7)
// ============================================================

func GetFrameworkControls(w http.ResponseWriter, r *http.Request) {
	frameworkID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid framework id")
		return
	}

	compliance := services.NewComplianceService()
	rows, err := compliance.ControlsForFramework(r.Context(), frameworkID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	type controlRow struct {
		ControlID             string `json:"controlId"`
		ControlCode           string `json:"controlCode"`
		FrameworkControlCode  string `json:"frameworkControlCode"`
		Title                 string `json:"title"`
		Description           string `json:"description"`
		EvidenceStatus        string `json:"evidenceStatus"`
		CurrentStatus         string `json:"currentStatus"`
		EvidenceCount         int    `json:"evidenceCount"`
	}

	result := []controlRow{}
	for _, row := range rows {
		status, err := compliance.CalculateControlStatus(r.Context(), row.Control)
		if err != nil {
			continue
		}
		var evidenceCount int
		_ = db.Pool.QueryRow(r.Context(),
			`SELECT COUNT(*) FROM compliance_evidence WHERE control_id = $1`, row.Control.ControlID,
		).Scan(&evidenceCount)

		result = append(result, controlRow{
			ControlID:            row.Control.ControlID.String(),
			ControlCode:          row.Control.ControlCode,
			FrameworkControlCode: row.FrameworkControlCode,
			Title:                row.Control.Title,
			Description:          row.Control.Description,
			EvidenceStatus:       string(row.Control.EvidenceStatus),
			CurrentStatus:        string(status),
			EvidenceCount:        evidenceCount,
		})
	}
	jsonOK(w, result)
}

// ============================================================
// GET /api/v1/compliance/frameworks/:id/status (§7) — cached snapshot only
// ============================================================

func GetFrameworkStatus(w http.ResponseWriter, r *http.Request) {
	frameworkID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid framework id")
		return
	}

	var ftype, name string
	if err := db.Pool.QueryRow(r.Context(),
		`SELECT framework_type::text, display_name FROM compliance_frameworks WHERE framework_id = $1`, frameworkID,
	).Scan(&ftype, &name); err != nil {
		jsonError(w, http.StatusNotFound, "framework not found")
		return
	}

	compliance := services.NewComplianceService()
	resp := models.FrameworkStatusResponse{FrameworkID: frameworkID.String(), FrameworkType: ftype, DisplayName: name}
	snap, err := compliance.LatestSnapshot(r.Context(), frameworkID)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			jsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		// No snapshot calculated yet — return zeros rather than an error (§8: dashboard
		// always reads the latest snapshot; there simply isn't one until the first tick).
		jsonOK(w, resp)
		return
	}
	resp.ApplicableControlCount = snap.ApplicableControlCount
	resp.CompliantCount = snap.CompliantCount
	resp.PartiallyCompliantCount = snap.PartiallyCompliantCount
	resp.NonCompliantCount = snap.NonCompliantCount
	resp.PlannedCount = snap.PlannedCount
	t := snap.CalculatedAt
	resp.CalculatedAt = &t
	jsonOK(w, resp)
}

// ============================================================
// GET /api/v1/compliance/controls/:id/evidence (§7, §10.4 Tier 3)
// ============================================================

func GetControlEvidence(w http.ResponseWriter, r *http.Request) {
	controlID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid control id")
		return
	}

	rows, err := db.Pool.Query(r.Context(), `
		SELECT evidence_id::text,
		       COALESCE(source_audit_event_id::text, ''),
		       COALESCE(source_activity_event_id::text, ''),
		       COALESCE(source_policy_action_id::text, ''),
		       observed_at
		FROM compliance_evidence
		WHERE control_id = $1
		ORDER BY observed_at DESC
		LIMIT 200`, controlID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type evidenceRow struct {
		EvidenceID     string    `json:"evidenceId"`
		SourceType     string    `json:"sourceType"` // audit_event | activity_event | policy_action
		SourceID       string    `json:"sourceId"`
		ObservedAt     time.Time `json:"observedAt"`
	}

	result := []evidenceRow{}
	for rows.Next() {
		var id, auditID, activityID, policyActionID string
		var observedAt time.Time
		if err := rows.Scan(&id, &auditID, &activityID, &policyActionID, &observedAt); err != nil {
			continue
		}
		row := evidenceRow{EvidenceID: id, ObservedAt: observedAt}
		switch {
		case auditID != "":
			row.SourceType, row.SourceID = "audit_event", auditID
		case activityID != "":
			row.SourceType, row.SourceID = "activity_event", activityID
		case policyActionID != "":
			row.SourceType, row.SourceID = "policy_action", policyActionID
		}
		result = append(result, row)
	}
	jsonOK(w, result)
}

// ============================================================
// GET /api/v1/compliance/controls — flat list, needed by the §10.6 "Add control
// mapping" UI to populate its control picker.
// ============================================================

func ListControls(w http.ResponseWriter, r *http.Request) {
	compliance := services.NewComplianceService()
	controls, err := compliance.AllControls(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	type controlRow struct {
		ControlID   string `json:"controlId"`
		ControlCode string `json:"controlCode"`
		Title       string `json:"title"`
	}
	result := []controlRow{}
	for _, c := range controls {
		result = append(result, controlRow{ControlID: c.ControlID.String(), ControlCode: c.ControlCode, Title: c.Title})
	}
	jsonOK(w, result)
}

// ============================================================
// Policy ↔ Control mapping — manual correction path (§5.7, §7, §10.6)
// ============================================================

func ListPolicyControlMappings(w http.ResponseWriter, r *http.Request) {
	policyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid policy id")
		return
	}
	rows, err := db.Pool.Query(r.Context(), `
		SELECT pcm.mapping_id::text, pcm.control_id::text, c.control_code, c.title,
		       pcm.mapping_source::text, COALESCE(pcm.rationale,''), pcm.mapped_at
		FROM policy_control_mappings pcm
		JOIN controls c ON c.control_id = pcm.control_id
		WHERE pcm.policy_id = $1
		ORDER BY c.control_code`, policyID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	type mappingRow struct {
		MappingID     string    `json:"mappingId"`
		ControlID     string    `json:"controlId"`
		ControlCode   string    `json:"controlCode"`
		Title         string    `json:"title"`
		MappingSource string    `json:"mappingSource"`
		Rationale     string    `json:"rationale"`
		MappedAt      time.Time `json:"mappedAt"`
	}
	result := []mappingRow{}
	for rows.Next() {
		var m mappingRow
		if err := rows.Scan(&m.MappingID, &m.ControlID, &m.ControlCode, &m.Title, &m.MappingSource, &m.Rationale, &m.MappedAt); err != nil {
			continue
		}
		result = append(result, m)
	}
	jsonOK(w, result)
}

func CreatePolicyControlMapping(w http.ResponseWriter, r *http.Request) {
	policyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid policy id")
		return
	}
	var req models.CreatePolicyControlMappingRequest
	if err := decodeJSON(r, &req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	controlID, err := uuid.Parse(req.ControlID)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid controlId")
		return
	}

	actorID, _ := resolveActorID(r)
	var mappingID uuid.UUID
	err = db.Pool.QueryRow(r.Context(), `
		INSERT INTO policy_control_mappings (policy_id, control_id, mapping_source, rationale, mapped_by)
		VALUES ($1, $2, 'manual', $3, $4)
		ON CONFLICT (policy_id, control_id) DO UPDATE SET mapping_source = 'manual', rationale = $3, mapped_by = $4, mapped_at = NOW()
		RETURNING mapping_id`,
		policyID, controlID, req.Rationale, actorID,
	).Scan(&mappingID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonCreated(w, map[string]string{"mappingId": mappingID.String()})
}

func DeletePolicyControlMapping(w http.ResponseWriter, r *http.Request) {
	mappingID, err := uuid.Parse(chi.URLParam(r, "mappingId"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid mapping id")
		return
	}
	res, err := db.Pool.Exec(r.Context(), `DELETE FROM policy_control_mappings WHERE mapping_id = $1`, mappingID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.RowsAffected() == 0 {
		jsonError(w, http.StatusNotFound, "mapping not found")
		return
	}
	jsonOK(w, map[string]bool{"deleted": true})
}
