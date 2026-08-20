package services

import (
	"context"
	"errors"

	"github.com/checkmarx/governance-api/internal/db"
	"github.com/checkmarx/governance-api/internal/models"
	"github.com/google/uuid"
)

// ErrSegregationOfDuties is returned when a reviewer tries to review their own violation.
var ErrSegregationOfDuties = errors.New("reviewer must differ from the violation's actor")

// ErrViolationNotFound is returned when the referenced policy_action doesn't exist.
var ErrViolationNotFound = errors.New("violation not found")

// OversightService records human review of violations (§5.5). Reviewing a violation
// never means "the risk is closed" (§2) — it means a specific, different human is
// on record with a judgment. This is the mechanism that produces human-oversight
// evidence for EU AI Act Art.14 / SOC2 CC7.2, and the fix for G2.
type OversightService struct{}

func NewOversightService() *OversightService { return &OversightService{} }

// ReviewViolation records a human review of a policy_action. Enforces segregation
// of duties server-side (§5.5) — the same developer marking their own BLOCK
// "reviewed" would prove self-clearing, not oversight.
func (s *OversightService) ReviewViolation(ctx context.Context, policyActionID, reviewerID uuid.UUID, decision, rationale string) error {
	var actorID uuid.UUID
	err := db.Pool.QueryRow(ctx,
		`SELECT developer_id FROM policy_actions WHERE policy_action_id = $1`, policyActionID,
	).Scan(&actorID)
	if err != nil {
		return ErrViolationNotFound
	}
	if actorID == reviewerID {
		return ErrSegregationOfDuties
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `
		INSERT INTO oversight_events (policy_action_id, reviewer_id, decision, rationale)
		VALUES ($1, $2, $3, $4)`,
		policyActionID, reviewerID, decision, rationale,
	); err != nil {
		return err
	}

	// "resolved" means "reviewed" — it does not imply anything was fixed (§2, §5.5).
	// Escalated reviews stay open, but now carry a review on record.
	newStatus := "resolved"
	if decision == "escalated" {
		newStatus = "open"
	}
	if _, err := tx.Exec(ctx,
		`UPDATE policy_actions SET resolution_status = $1, resolved_at = NOW() WHERE policy_action_id = $2`,
		newStatus, policyActionID,
	); err != nil {
		return err
	}

	auditType := models.AuditViolationReviewed
	if decision == "overridden" {
		auditType = models.AuditViolationOverridden
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (actor_id, actor_type, event_type, resource_type, resource_id)
		VALUES ($1, 'developer', $2, 'policy_action', $3)`,
		reviewerID, auditType, policyActionID,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
