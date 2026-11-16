// Package models — compliance domain types (POC scope).
// New file, additive only — see COMPLIANCE_ARCHITECTURE.md §5, §6.
// Kept separate from models.go so the existing file is untouched.
package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ============================================================
// New audit action constants (closes part of G1/G2 evidence — §5.5, §5.6)
// ============================================================

const (
	AuditViolationReviewed   = "VIOLATION_REVIEWED"
	AuditViolationOverridden = "VIOLATION_OVERRIDDEN"
)

// ============================================================
// Control Catalog (§5.2, §6.1) — data, not code
// ============================================================

// EvidenceStatus gates whether a control is counted at all (§5.3).
type EvidenceStatus string

const (
	EvidenceOperational EvidenceStatus = "operational"
	EvidencePlanned     EvidenceStatus = "planned"
)

// ControlStatusValue is the four-way raw-count status (§5.3, §8.1). No percentage is ever derived from it.
type ControlStatusValue string

const (
	StatusCompliant           ControlStatusValue = "compliant"
	StatusPartiallyCompliant  ControlStatusValue = "partially_compliant"
	StatusNonCompliant        ControlStatusValue = "non_compliant"
	StatusPlanned             ControlStatusValue = "planned"
)

// EvidenceSpec is the JSON shape stored in controls.evidence_spec (§5.2, §6.3).
type EvidenceSpec struct {
	AuditEventTypes     []string          `json:"auditEventTypes"`
	ActivityEventKinds  []string          `json:"activityEventKinds"`
	PolicyActionFilters map[string]string `json:"policyActionFilters"`
	OversightRequired   bool              `json:"oversightRequired"`
}

// Control is a testable statement plus a machine-readable evidence spec (§5.2).
type Control struct {
	ControlID      uuid.UUID       `json:"controlId"      db:"control_id"`
	ControlCode    string          `json:"controlCode"    db:"control_code"`
	Title          string          `json:"title"          db:"title"`
	Description    string          `json:"description"    db:"description"`
	EvidenceSpecRaw json.RawMessage `json:"evidenceSpec"  db:"evidence_spec"`
	EvidenceStatus EvidenceStatus  `json:"evidenceStatus" db:"evidence_status"`

	// Parsed lazily by services — not scanned directly from the DB.
	EvidenceSpec EvidenceSpec `json:"-" db:"-"`
}

// ParseEvidenceSpec decodes EvidenceSpecRaw into EvidenceSpec.
func (c *Control) ParseEvidenceSpec() error {
	return json.Unmarshal(c.EvidenceSpecRaw, &c.EvidenceSpec)
}

// ComplianceFramework (§6.1) — SOC2 / EU AI Act for the POC.
type ComplianceFramework struct {
	FrameworkID   uuid.UUID `json:"frameworkId"   db:"framework_id"`
	FrameworkType string    `json:"frameworkType" db:"framework_type"`
	DisplayName   string    `json:"displayName"   db:"display_name"`
	CreatedAt     time.Time `json:"createdAt"     db:"created_at"`
}

// FrameworkControlMapping — crosswalk: one control, many frameworks (§6.1, §6.2).
type FrameworkControlMapping struct {
	MappingID            uuid.UUID `json:"mappingId"            db:"mapping_id"`
	FrameworkID          uuid.UUID `json:"frameworkId"          db:"framework_id"`
	ControlID            uuid.UUID `json:"controlId"            db:"control_id"`
	FrameworkControlCode string    `json:"frameworkControlCode" db:"framework_control_code"`
}

// PolicyMappingRule (§5.7, §6.4) — data-driven auto-mapping rule.
type PolicyMappingRule struct {
	RuleID           uuid.UUID       `json:"ruleId"           db:"rule_id"`
	RuleCode         string          `json:"ruleCode"         db:"rule_code"`
	WhenConditions   map[string]string `json:"-" db:"-"`
	WhenConditionsRaw json.RawMessage `json:"whenConditions"  db:"when_conditions"`
	MapsToControlID  uuid.UUID       `json:"mapsToControlId"  db:"maps_to_control_id"`
	Description      string          `json:"description"      db:"description"`
	Status           string          `json:"status"            db:"status"`
}

// Matches reports whether a policy's own attributes satisfy this rule's condition.
// Missing keys in WhenConditions are wildcards (§6.4). The empty rule matches everything.
func (r *PolicyMappingRule) Matches(policyType, toolType, action, matchTarget string) bool {
	for key, want := range r.WhenConditions {
		var got string
		switch key {
		case "policyType":
			got = policyType
		case "type":
			got = toolType
		case "action":
			got = action
		case "matchTarget":
			got = matchTarget
		default:
			return false // unknown condition key — never matches
		}
		if got != want {
			return false
		}
	}
	return true
}

// PolicyControlMapping (§5.7, §6.1) — which policy satisfies which control, and how.
type PolicyControlMapping struct {
	MappingID     uuid.UUID  `json:"mappingId"     db:"mapping_id"`
	PolicyID      uuid.UUID  `json:"policyId"      db:"policy_id"`
	ControlID     uuid.UUID  `json:"controlId"     db:"control_id"`
	MappingSource string     `json:"mappingSource" db:"mapping_source"` // automatic | manual
	Rationale     string     `json:"rationale"     db:"rationale"`
	MappedBy      *uuid.UUID `json:"mappedBy"      db:"mapped_by"`
	MappedAt      time.Time  `json:"mappedAt"      db:"mapped_at"`
}

// ComplianceEvidence (§5.4, §6.1) — a reference to exactly one source row, never a copy.
type ComplianceEvidence struct {
	EvidenceID             uuid.UUID  `json:"evidenceId"             db:"evidence_id"`
	ControlID              uuid.UUID  `json:"controlId"              db:"control_id"`
	SourceAuditEventID     *uuid.UUID `json:"sourceAuditEventId"     db:"source_audit_event_id"`
	SourceActivityEventID  *uuid.UUID `json:"sourceActivityEventId"  db:"source_activity_event_id"`
	SourcePolicyActionID   *uuid.UUID `json:"sourcePolicyActionId"   db:"source_policy_action_id"`
	ObservedAt             time.Time  `json:"observedAt"             db:"observed_at"`
	CreatedAt              time.Time  `json:"createdAt"              db:"created_at"`
}

// ComplianceStatusSnapshot (§8, §8.1) — raw counts only, no computed percentage.
type ComplianceStatusSnapshot struct {
	SnapshotID               uuid.UUID `json:"snapshotId"               db:"snapshot_id"`
	FrameworkID              uuid.UUID `json:"frameworkId"              db:"framework_id"`
	ApplicableControlCount   int       `json:"applicableControlCount"   db:"applicable_control_count"`
	CompliantCount           int       `json:"compliantCount"           db:"compliant_count"`
	PartiallyCompliantCount  int       `json:"partiallyCompliantCount"  db:"partially_compliant_count"`
	NonCompliantCount        int       `json:"nonCompliantCount"        db:"non_compliant_count"`
	PlannedCount             int       `json:"plannedCount"             db:"planned_count"`
	CalculatedAt             time.Time `json:"calculatedAt"             db:"calculated_at"`
}

// OversightEvent (§5.5, §6.1) — a human review, distinct from the violation itself.
type OversightEvent struct {
	OversightEventID uuid.UUID `json:"oversightEventId" db:"oversight_event_id"`
	PolicyActionID   uuid.UUID `json:"policyActionId"   db:"policy_action_id"`
	ReviewerID       uuid.UUID `json:"reviewerId"       db:"reviewer_id"`
	Decision         string    `json:"decision"         db:"decision"` // affirmed | overridden | escalated
	Rationale        string    `json:"rationale"        db:"rationale"`
	ReviewedAt       time.Time `json:"reviewedAt"       db:"reviewed_at"`
}

// ============================================================
// Request/response types (§5.5, §7)
// ============================================================

// ReviewViolationRequest is the body for POST /api/v1/violations/:id/review.
type ReviewViolationRequest struct {
	Decision  string `json:"decision"`  // affirmed | overridden | escalated
	Rationale string `json:"rationale"`
}

// CreatePolicyControlMappingRequest is the body for the manual correction path (§7, §10.6).
type CreatePolicyControlMappingRequest struct {
	ControlID string `json:"controlId"`
	Rationale string `json:"rationale"`
}

// FrameworkStatusResponse is what GET /compliance/frameworks/:id/status returns (§7, §8.1).
type FrameworkStatusResponse struct {
	FrameworkID              string    `json:"frameworkId"`
	FrameworkType            string    `json:"frameworkType"`
	DisplayName              string    `json:"displayName"`
	ApplicableControlCount   int       `json:"applicableControlCount"`
	CompliantCount           int       `json:"compliantCount"`
	PartiallyCompliantCount  int       `json:"partiallyCompliantCount"`
	NonCompliantCount        int       `json:"nonCompliantCount"`
	PlannedCount             int       `json:"plannedCount"`
	CalculatedAt             *time.Time `json:"calculatedAt"`
}
