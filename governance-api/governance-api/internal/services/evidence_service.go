package services

import (
	"context"
	"fmt"
	"time"

	"github.com/checkmarx/governance-api/internal/db"
	"github.com/checkmarx/governance-api/internal/models"
	"github.com/google/uuid"
)

// EvidenceService extracts evidence from existing tables into compliance_evidence (§5.4).
// It only reads audit_events / activity_events / policy_actions and only ever inserts
// references (control_id + source row id) — never a copy of the source row's content.
type EvidenceService struct {
	compliance *ComplianceService
	watermark  time.Time // in-memory, reset to EvidenceWindow lookback on first run
}

func NewEvidenceService(compliance *ComplianceService) *EvidenceService {
	return &EvidenceService{
		compliance: compliance,
		watermark:  time.Now().Add(-EvidenceWindow),
	}
}

// ExtractSince scans rows created since the last watermark against every control's
// evidence spec and inserts matching references into compliance_evidence (§5.4).
// Runs on a schedule (§8), never inline on ingest.
func (s *EvidenceService) ExtractSince(ctx context.Context) error {
	controls, err := s.compliance.AllControls(ctx)
	if err != nil {
		return fmt.Errorf("load controls: %w", err)
	}

	runStart := time.Now()

	if err := s.extractAuditEvents(ctx, controls); err != nil {
		return fmt.Errorf("extract audit events: %w", err)
	}
	if err := s.extractActivityEvents(ctx, controls); err != nil {
		return fmt.Errorf("extract activity events: %w", err)
	}
	if err := s.extractPolicyActions(ctx, controls); err != nil {
		return fmt.Errorf("extract policy actions: %w", err)
	}

	s.watermark = runStart
	return nil
}

func (s *EvidenceService) extractAuditEvents(ctx context.Context, controls []models.Control) error {
	rows, err := db.Pool.Query(ctx, `
		SELECT audit_event_id, event_type, timestamp
		FROM audit_events
		WHERE timestamp > $1`, s.watermark)
	if err != nil {
		return err
	}
	defer rows.Close()

	type row struct {
		id        uuid.UUID
		eventType string
		ts        time.Time
	}
	var eventRows []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.eventType, &r.ts); err != nil {
			continue
		}
		eventRows = append(eventRows, r)
	}
	rows.Close()

	for _, r := range eventRows {
		for _, c := range controls {
			if !contains(c.EvidenceSpec.AuditEventTypes, r.eventType) {
				continue
			}
			s.insertEvidence(ctx, c.ControlID, "audit_event", r.id, r.ts)
		}
	}
	return nil
}

func (s *EvidenceService) extractActivityEvents(ctx context.Context, controls []models.Control) error {
	rows, err := db.Pool.Query(ctx, `
		SELECT activity_event_id, kind, occurred_at
		FROM activity_events
		WHERE occurred_at > $1`, s.watermark)
	if err != nil {
		return err
	}
	defer rows.Close()

	type row struct {
		id   uuid.UUID
		kind string
		ts   time.Time
	}
	var eventRows []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.kind, &r.ts); err != nil {
			continue
		}
		eventRows = append(eventRows, r)
	}
	rows.Close()

	for _, r := range eventRows {
		for _, c := range controls {
			if !contains(c.EvidenceSpec.ActivityEventKinds, r.kind) {
				continue
			}
			s.insertEvidence(ctx, c.ControlID, "activity_event", r.id, r.ts)
		}
	}
	return nil
}

func (s *EvidenceService) extractPolicyActions(ctx context.Context, controls []models.Control) error {
	rows, err := db.Pool.Query(ctx, `
		SELECT policy_action_id, COALESCE(violation_type::text, ''), occurred_at
		FROM policy_actions
		WHERE occurred_at > $1`, s.watermark)
	if err != nil {
		return err
	}
	defer rows.Close()

	type row struct {
		id            uuid.UUID
		violationType string
		ts            time.Time
	}
	var actionRows []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.violationType, &r.ts); err != nil {
			continue
		}
		actionRows = append(actionRows, r)
	}
	rows.Close()

	for _, r := range actionRows {
		for _, c := range controls {
			want, ok := c.EvidenceSpec.PolicyActionFilters["violation_type"]
			if !ok {
				continue // this control's evidence doesn't come from policy_actions at all
			}
			if want != r.violationType {
				continue
			}
			s.insertEvidence(ctx, c.ControlID, "policy_action", r.id, r.ts)
		}
	}
	return nil
}

func (s *EvidenceService) insertEvidence(ctx context.Context, controlID uuid.UUID, sourceKind string, sourceID uuid.UUID, observedAt time.Time) {
	var query string
	switch sourceKind {
	case "audit_event":
		query = `INSERT INTO compliance_evidence (control_id, source_audit_event_id, observed_at)
		         VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`
	case "activity_event":
		query = `INSERT INTO compliance_evidence (control_id, source_activity_event_id, observed_at)
		         VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`
	case "policy_action":
		query = `INSERT INTO compliance_evidence (control_id, source_policy_action_id, observed_at)
		         VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`
	default:
		return
	}
	// Best-effort — a failed evidence insert never blocks the rest of the scan (§4, cardinal rule).
	_, _ = db.Pool.Exec(ctx, query, controlID, sourceID, observedAt)
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
