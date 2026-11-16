// Package services — compliance domain services (POC scope).
// New package, additive only — see COMPLIANCE_ARCHITECTURE.md §5.1.
//
// ComplianceService owns the control catalog, the auto-mapping rule engine (§5.7),
// and control/framework status calculation (§5.3). It only ever reads existing
// operational tables (audit_events, activity_events, policy_actions, policies) —
// it never writes to them (§4, cardinal rule).
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/checkmarx/governance-api/internal/db"
	"github.com/checkmarx/governance-api/internal/models"
	"github.com/google/uuid"
)

// ReviewWindow is how long an oversight-required violation can sit unreviewed
// before it drags a control to partially_compliant (§2, §5.3). Not a remediation
// SLA — it measures whether a human is watching, not whether something was "fixed".
const ReviewWindow = 7 * 24 * time.Hour

// EvidenceWindow bounds how far back evidence extraction looks when deciding
// whether a control's evidence pipeline has ever produced anything (§5.4).
const EvidenceWindow = 90 * 24 * time.Hour

type ComplianceService struct{}

func NewComplianceService() *ComplianceService { return &ComplianceService{} }

// ============================================================
// Control catalog (§5.2)
// ============================================================

// AllControls returns every control in the shared catalog, with EvidenceSpec parsed.
func (s *ComplianceService) AllControls(ctx context.Context) ([]models.Control, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT control_id, control_code, title, COALESCE(description,''), evidence_spec, evidence_status::text
		FROM controls
		ORDER BY control_code`)
	if err != nil {
		return nil, fmt.Errorf("query controls: %w", err)
	}
	defer rows.Close()

	var out []models.Control
	for rows.Next() {
		var c models.Control
		var specRaw []byte
		if err := rows.Scan(&c.ControlID, &c.ControlCode, &c.Title, &c.Description, &specRaw, &c.EvidenceStatus); err != nil {
			continue
		}
		c.EvidenceSpecRaw = specRaw
		if err := c.ParseEvidenceSpec(); err != nil {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// ControlsForFramework returns every control mapped to a framework, plus its own code under that framework.
func (s *ComplianceService) ControlsForFramework(ctx context.Context, frameworkID uuid.UUID) ([]FrameworkControlRow, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT c.control_id, c.control_code, c.title, COALESCE(c.description,''), c.evidence_spec,
		       c.evidence_status::text, fcm.framework_control_code
		FROM framework_control_mappings fcm
		JOIN controls c ON c.control_id = fcm.control_id
		WHERE fcm.framework_id = $1
		ORDER BY fcm.framework_control_code`, frameworkID)
	if err != nil {
		return nil, fmt.Errorf("query framework controls: %w", err)
	}
	defer rows.Close()

	var out []FrameworkControlRow
	for rows.Next() {
		var row FrameworkControlRow
		var specRaw []byte
		if err := rows.Scan(&row.Control.ControlID, &row.Control.ControlCode, &row.Control.Title,
			&row.Control.Description, &specRaw, &row.Control.EvidenceStatus, &row.FrameworkControlCode); err != nil {
			continue
		}
		row.Control.EvidenceSpecRaw = specRaw
		_ = row.Control.ParseEvidenceSpec()
		out = append(out, row)
	}
	return out, nil
}

// FrameworkControlRow pairs a shared control with its label under one specific framework.
type FrameworkControlRow struct {
	Control              models.Control
	FrameworkControlCode string
}

// ============================================================
// Status calculation (§5.3) — the corrected model:
//   1. Planned controls are excluded, not scored as failing.
//   2. Detection/enforcement controls: compliant means "an active policy is
//      mapped", not "violations occurred" — silence isn't failure.
//   3. Oversight-required controls: compliant means every triggered violation
//      under this control was reviewed within the window.
// ============================================================

// CalculateControlStatus implements §5.3 exactly.
func (s *ComplianceService) CalculateControlStatus(ctx context.Context, control models.Control) (models.ControlStatusValue, error) {
	if control.EvidenceStatus != models.EvidenceOperational {
		return models.StatusPlanned, nil
	}

	if control.EvidenceSpec.OversightRequired {
		unreviewed, err := s.countUnreviewedPastWindow(ctx, control.ControlID, ReviewWindow)
		if err != nil {
			return "", err
		}
		if unreviewed > 0 {
			return models.StatusPartiallyCompliant, nil
		}
		return models.StatusCompliant, nil
	}

	// Detection/enforcement or audit-log-only control. The distinguishing signal
	// is whether the control declares any audit/activity event types at all — those
	// are properties of the *platform* (record-keeping, access provisioning, policy
	// lifecycle), true the moment the capture mechanism is operational, independent
	// of any single policy. Everything else (boundary protection, data protection,
	// PII detection, third-party visibility, system integrity) is a *mechanism*
	// whose proof is "is a policy actively enforcing this" — checking
	// PolicyActionFilters alone was wrong for controls like CTRL-PII-PROTECTION
	// that have no violation_type signature to filter on but still need an active
	// mapped policy to be true evidence.
	isPlatformCapability := len(control.EvidenceSpec.AuditEventTypes) > 0 || len(control.EvidenceSpec.ActivityEventKinds) > 0
	if isPlatformCapability {
		// Being operational already proves the capture mechanism exists. Silence
		// (no events yet) is not failure, same reasoning as principle §12.7.
		return models.StatusCompliant, nil
	}

	active, err := s.hasActiveMappedPolicy(ctx, control.ControlID)
	if err != nil {
		return "", err
	}
	if active {
		return models.StatusCompliant, nil
	}
	return models.StatusNonCompliant, nil
}

// hasActiveMappedPolicy checks whether at least one policy mapped to this control
// is currently active (§5.3, point 2 — the mechanism existing is the proof, not
// how many times it fired).
func (s *ComplianceService) hasActiveMappedPolicy(ctx context.Context, controlID uuid.UUID) (bool, error) {
	var exists bool
	err := db.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM policy_control_mappings pcm
			JOIN policies p ON p.policy_id = pcm.policy_id
			WHERE pcm.control_id = $1
			  AND p.status = 'active'
			  AND p.deleted_at IS NULL
		)`, controlID).Scan(&exists)
	return exists, err
}

// countUnreviewedPastWindow counts BLOCK/ALERT policy_actions, under policies
// mapped to this control, that have no oversight_events row and are older than
// the review window (§5.3, §5.5). Pooled across every mapped policy — the
// question is "has every triggered event under this control been reviewed",
// not "is each policy individually reviewed".
func (s *ComplianceService) countUnreviewedPastWindow(ctx context.Context, controlID uuid.UUID, window time.Duration) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM policy_actions pa
		JOIN policy_control_mappings pcm ON pcm.policy_id = pa.policy_id
		WHERE pcm.control_id = $1
		  AND pa.action_type IN ('BLOCK', 'ALERT')
		  AND pa.occurred_at < NOW() - $2::interval
		  AND NOT EXISTS (
			SELECT 1 FROM oversight_events oe WHERE oe.policy_action_id = pa.policy_action_id
		  )`, controlID, fmt.Sprintf("%d seconds", int(window.Seconds())),
	).Scan(&count)
	return count, err
}

// CalculateFrameworkSnapshot rolls every control mapped to a framework up into
// raw counts and persists one snapshot row (§8, §8.1). No percentage is ever derived.
func (s *ComplianceService) CalculateFrameworkSnapshot(ctx context.Context, frameworkID uuid.UUID) (*models.ComplianceStatusSnapshot, error) {
	rows, err := s.ControlsForFramework(ctx, frameworkID)
	if err != nil {
		return nil, err
	}

	snap := &models.ComplianceStatusSnapshot{
		FrameworkID:            frameworkID,
		ApplicableControlCount: len(rows),
	}
	for _, row := range rows {
		status, err := s.CalculateControlStatus(ctx, row.Control)
		if err != nil {
			continue
		}
		switch status {
		case models.StatusCompliant:
			snap.CompliantCount++
		case models.StatusPartiallyCompliant:
			snap.PartiallyCompliantCount++
		case models.StatusNonCompliant:
			snap.NonCompliantCount++
		case models.StatusPlanned:
			snap.PlannedCount++
		}
	}

	err = db.Pool.QueryRow(ctx, `
		INSERT INTO compliance_status_snapshots
			(framework_id, applicable_control_count, compliant_count, partially_compliant_count, non_compliant_count, planned_count)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING snapshot_id, calculated_at`,
		frameworkID, snap.ApplicableControlCount, snap.CompliantCount,
		snap.PartiallyCompliantCount, snap.NonCompliantCount, snap.PlannedCount,
	).Scan(&snap.SnapshotID, &snap.CalculatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert snapshot: %w", err)
	}
	return snap, nil
}

// CalculateAllFrameworkSnapshots runs CalculateFrameworkSnapshot for every framework (§8).
func (s *ComplianceService) CalculateAllFrameworkSnapshots(ctx context.Context) error {
	rows, err := db.Pool.Query(ctx, `SELECT framework_id FROM compliance_frameworks`)
	if err != nil {
		return err
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()

	for _, id := range ids {
		if _, err := s.CalculateFrameworkSnapshot(ctx, id); err != nil {
			return fmt.Errorf("snapshot for framework %s: %w", id, err)
		}
	}
	return nil
}

// LatestSnapshot returns the most recently calculated snapshot for a framework, if any.
func (s *ComplianceService) LatestSnapshot(ctx context.Context, frameworkID uuid.UUID) (*models.ComplianceStatusSnapshot, error) {
	var snap models.ComplianceStatusSnapshot
	err := db.Pool.QueryRow(ctx, `
		SELECT snapshot_id, framework_id, applicable_control_count, compliant_count,
		       partially_compliant_count, non_compliant_count, planned_count, calculated_at
		FROM compliance_status_snapshots
		WHERE framework_id = $1
		ORDER BY calculated_at DESC
		LIMIT 1`, frameworkID,
	).Scan(&snap.SnapshotID, &snap.FrameworkID, &snap.ApplicableControlCount, &snap.CompliantCount,
		&snap.PartiallyCompliantCount, &snap.NonCompliantCount, &snap.PlannedCount, &snap.CalculatedAt)
	if err != nil {
		return nil, err // caller decides how to handle "no snapshot yet" (e.g. pgx.ErrNoRows)
	}
	return &snap, nil
}

// ============================================================
// Automated policy-to-control mapping (§5.7)
// ============================================================

// loadActiveRules loads every active rule and parses its when-conditions.
func (s *ComplianceService) loadActiveRules(ctx context.Context) ([]models.PolicyMappingRule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT rule_id, rule_code, when_conditions, maps_to_control_id, COALESCE(description,''), status::text
		FROM policy_mapping_rules
		WHERE status = 'active'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.PolicyMappingRule
	for rows.Next() {
		var rule models.PolicyMappingRule
		var whenRaw []byte
		if err := rows.Scan(&rule.RuleID, &rule.RuleCode, &whenRaw, &rule.MapsToControlID, &rule.Description, &rule.Status); err != nil {
			continue
		}
		rule.WhenConditionsRaw = whenRaw
		_ = json.Unmarshal(whenRaw, &rule.WhenConditions)
		out = append(out, rule)
	}
	return out, nil
}

// PolicyAttributes is the subset of a policy's own fields that rules match against (§5.7).
// PolicyType comes from policies.policy_type; Type/Action/MatchTarget come from the
// PackPolicy definition JSON (empty for non-"governance" policy types, which is fine —
// rules matching on those keys simply won't match, and policyType-only rules still will).
type PolicyAttributes struct {
	PolicyType  string
	Type        string
	Action      string
	MatchTarget string
}

// AutoMapPolicyToControls evaluates every active rule against a policy's attributes
// and inserts a policy_control_mappings row for each rule that matches (§5.7).
// Not "first match wins" — every satisfied rule fires, since a policy is often
// legitimately evidence for more than one control at once.
func (s *ComplianceService) AutoMapPolicyToControls(ctx context.Context, policyID uuid.UUID, attrs PolicyAttributes) error {
	rules, err := s.loadActiveRules(ctx)
	if err != nil {
		return fmt.Errorf("load rules: %w", err)
	}
	for _, rule := range rules {
		if !rule.Matches(attrs.PolicyType, attrs.Type, attrs.Action, attrs.MatchTarget) {
			continue
		}
		if _, err := db.Pool.Exec(ctx, `
			INSERT INTO policy_control_mappings (policy_id, control_id, mapping_source, rationale)
			VALUES ($1, $2, 'automatic', $3)
			ON CONFLICT (policy_id, control_id) DO NOTHING`,
			policyID, rule.MapsToControlID, "Auto-mapped via "+rule.RuleCode,
		); err != nil {
			return fmt.Errorf("insert mapping for rule %s: %w", rule.RuleCode, err)
		}
	}
	return nil
}

// AutoMapAllPolicies re-runs AutoMapPolicyToControls for every non-deleted policy
// against every currently active rule (§5.7, §8) — idempotent (ON CONFLICT DO NOTHING),
// this is what makes a newly added rule retroactively apply to policies that already
// existed before that rule was written.
func (s *ComplianceService) AutoMapAllPolicies(ctx context.Context) error {
	rows, err := db.Pool.Query(ctx, `
		SELECT p.policy_id, COALESCE(p.policy_type, 'governance'), COALESCE(pv.definition::text, '{}')
		FROM policies p
		LEFT JOIN policy_versions pv ON p.current_version_id = pv.policy_version_id
		WHERE p.deleted_at IS NULL`)
	if err != nil {
		return fmt.Errorf("query policies for auto-mapping: %w", err)
	}

	type policyRow struct {
		id         uuid.UUID
		policyType string
		defJSON    string
	}
	var policyRows []policyRow
	for rows.Next() {
		var pr policyRow
		if err := rows.Scan(&pr.id, &pr.policyType, &pr.defJSON); err != nil {
			continue
		}
		policyRows = append(policyRows, pr)
	}
	rows.Close()

	for _, pr := range policyRows {
		attrs := PolicyAttributes{PolicyType: pr.policyType}
		var def struct {
			Type        string `json:"type"`
			Action      string `json:"action"`
			MatchTarget string `json:"matchTarget"`
		}
		if json.Unmarshal([]byte(pr.defJSON), &def) == nil {
			attrs.Type = def.Type
			attrs.Action = def.Action
			attrs.MatchTarget = def.MatchTarget
		}
		if err := s.AutoMapPolicyToControls(ctx, pr.id, attrs); err != nil {
			return err
		}
	}
	return nil
}
