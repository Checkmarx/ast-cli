# AI Governance Compliance Framework — POC Architecture
**Role:** Senior Architect, Checkmarx
**Reference:** AEGIS compliance model · EU AI Act (Regulation 2024/1689) Art. 12/14 · AICPA SOC2 Trust Services Criteria
**Scope:** POC only — database, backend services, and dashboard needed to demonstrate live compliance status, reported as raw counts, calculated from real evidence. Full-scope items (risk tiering, incidents, exceptions, signed bundles, integrity checkpoints) are intentionally out of this document — see §9.

---

## 1. Executive Summary

**The one thing this POC proves:** compliance status is calculated from evidence the platform already produces (audit logs, activity events, policy violations) — not typed in by hand.

**Reused (plumbing only, no changes):**
- `audit_events` — append-only system mutation log
- `activity_events` — full session/tool-call event history
- `policy_actions` — enforcement decisions (ALLOW/ALERT/BLOCK)
- `policies` / `policy_versions` — versioned policy definitions
- `middleware.ActorID(r)` — actor identity extraction (exists today, underused)

**Built new, for the POC:**
- A small, shared control catalog (5–10 controls covering SOC2 + EU AI Act, as data, not code)
- `EvidenceService` — extracts evidence from existing tables automatically
- `ComplianceService` — calculates control status and rolls it up into framework-level raw counts (no blended score)
- `OversightService` + `POST /api/v1/violations/:id/review` — the first mutation path for `policy_actions.resolution_status`, and the mechanism that produces human-oversight evidence for EU AI Act Art.14
- A `/compliance` dashboard page showing live per-status counts per framework

**Explicitly not built for the POC** (see §9 for why and when): agent risk tiering, capability declarations, incident lifecycle, exception/waiver workflow, tamper-evidence checkpoints, signed bundle export, scope-based permissions.

---

## 2. Domain Constraint That Shapes This Design

**AI agent actions are enforced synchronously, not left in an open state.**

| Traditional compliance | This platform |
|---|---|
| A finding (open port, unpatched CVE) is a **persistent state** that stays broken until fixed | A policy decision (BLOCK/ALERT) is a **single, already-completed event** |
| "Remediate within 30 days" makes sense — the exposure is ongoing | Nothing is ongoing — a BLOCK already stopped the action; an ALERT already logged a past one |
| Compliance = is the exposure closed? | Compliance = did a human review the decision, and can the trail prove it? |

**Why this matters for the POC specifically:** the review endpoint (§5.4) is deliberately called "Review," not "Resolve" or "Remediate" — it records that a human looked at a decision, not that something was fixed. This distinction is the reason `OversightService` exists as its own thing instead of being a generic "close this ticket" action.

---

## 3. Current State & Gaps (POC Scope Only)

### 3.1 Reused Components

| Component | Location | Why it's the right substrate |
|---|---|---|
| `audit_events` | `internal/db/migrations/001_initial_schema.sql` | Already actor/resource/old-new-value shaped |
| `activity_events` | `internal/db/migrations/002_activity_events.sql` | Every tool call and decision, full `raw JSONB` payload |
| `policy_actions` | `001_initial_schema.sql` | Already carries severity + `resolution_status` — just needs a way to write to it |
| `middleware.ActorID(r)` | `pkg/middleware/auth.go:63` | Already extracts the actor from the JWT; just not called everywhere it should be |

### 3.2 Gaps This POC Closes

| # | Gap | Why it matters | Fix |
|---|---|---|---|
| G1 | `audit_events.actor_id` always NULL for policy changes — **now present in 6 handlers, not 2** (§3.3) | Can't show "who did this" without it | Wire in `middleware.ActorID(r)` |
| G2 | No API path to `policy_actions.resolution_status` | The column exists; nothing can write to it — no human-oversight evidence is possible without this | New `POST /violations/:id/review` |
| G3 | `violation_type` is free-text `VARCHAR(64)` | Evidence filters (e.g. "count all data-export blocks") aren't reliable against unvalidated text | Enforce as Postgres `ENUM` |
| G4 | No framework/control model | Nothing links a policy or event to an external obligation | New catalog tables (§6) |
| G5 | No evidence extraction | Evidence has to come from somewhere automated, or it's manual bookkeeping | New `EvidenceService`, scheduled |
| G6 | No compliance API or dashboard | Nowhere to see the result | New `/api/v1/compliance/*` + `/compliance` page |
| G7 | No scheduler in the repo | Status calculation must run on a timer, never inline on ingest | In-process ticker |

### 3.3 Recent Implementation Changes — Compliance Impact

The platform has moved since this design was first written. These changes were checked directly against the actual diffs (`policies.go`, `models.go`, `ingest.go`, migrations 004–007, and the governance-ui panel components) — not assumed. Each either strengthens, widens, or newly enables something this design already accounts for.

| Change | What it is | Compliance impact |
|---|---|---|
| **Policy lifecycle expansion** — `UpdatePolicy`, `DisablePolicy`, `EnablePolicy`, `DeletePolicy` (soft-delete via `deleted_at`) | New audit events: `POLICY_DISABLED`, `POLICY_ENABLED`, `POLICY_DELETED`, alongside existing create/activate | **Strengthens CC8.1 (change management)** — the full policy lifecycle is now audited, not just creation/activation. **But the G1 bug propagated**: all 4 new handlers copy the same `actor_type='system'`, no `actor_id` pattern — this is now a 6-function fix, not 2 (§5.6 updated) |
| **`policy_type` column** — `governance` \| `pii` \| `third_party_monitor` | A new, structured classification of policies, independent of `type`/`action`/`matchTarget` | **New rule-matching dimension for §5.7's auto-mapping engine** — rules can now match on `policyType` directly, not just tool-category attributes (§5.7 updated) |
| **First-class PII detection** — `PIIPolicyDef` (built-in patterns: email, SSN, credit card, phone, IP, API key, JWT), `activity_events.pii_detected`/`pii_matches` | A dedicated, purpose-built detection mechanism, served in the governance pack as `pii_policy`, separate from ordinary pattern-matched policies | **Materially strengthens CC6.7 evidence** — credential/PII protection now has direct, named-pattern detection instead of only inferring intent from generic filesystem-block regexes (§9.2 updated) |
| **Third-party enforcement discovery** — `session_hooks` table (non-governance hooks inventoried at session start), `third_party_monitor` policies (auto-created/incremented on observed third-party blocks, with `attribution_confidence`: definitive/candidate) | The platform now detects and inventories *other* enforcement tooling already present in an agent's configuration | **New evidence for CC9.2 (vendor/third-party risk)**, upgraded from "weak partial" to a real, if narrow, operational source (§9.2 updated) — note this is configuration-level discovery, not network-level shadow-AI discovery |
| **Skills discovery** — `skills`/`session_skills` tables, `/api/v1/skills*`, `app/skills` page | Tracks which Claude Code skills are invoked, by whom, how often; skills are inventoried, not yet policy-enforced (Tier 2+) | **Partial evidence for EU AI Act Art.13** — an observed capability inventory exists now, though it's not yet the full *declared-vs-observed* capability transparency this design describes in §5.5's original (deferred) scope. Treat as a smaller, already-real first step (§9.3 updated) |
| **IDE/workspace capture** — `ai_sessions.ide_name`/`ide_version`/`workspace_path` | Session provenance is now more specific (which IDE, which workspace) | Minor enrichment to agent/session inventory — no control status change, but improves the evidence attached to existing operational controls |
| **Real UI panel components now exist** — `violation-panel.tsx`, `policy-panel.tsx`, `developer-panel.tsx`, `edit-policy-modal.tsx`, `policy-status-badge.tsx` | The panel system this design's UI extensions (§10.5, §10.6) were written against is now real code, not a hypothetical pattern | §10 updated to reference the actual file names. Confirmed by direct inspection: no review/oversight UI exists in `violation-panel.tsx` yet — G2 remains fully open |

**Net effect:** three controls get materially better evidence (CC8.1, CC6.7, CC9.2) without any change to this design's architecture — they're all still control-level status derived from existing/new tables, exactly per §5.3's model. One gap (G1) got wider, not narrower, and needs to be fixed across more surface area than originally scoped. No new architectural pattern was required by any of these changes — the strongest sign that the control-catalog design is holding up as the platform evolves.

---

## 4. Architecture Overview

```
┌───────────────────────────────────────────────────────────────┐
│                  Compliance Layer (POC)                        │
│  ┌────────────────┐   ┌────────────────────────────────────┐ │
│  │ /compliance UI  │   │  Policy API Server (existing Go)    │ │
│  │ (Next.js, new)  │◄─►│  /api/v1/compliance/*               │ │
│  │                 │   │  /api/v1/violations/:id/review (new)│ │
│  └─────────────────┘   └──────────────────┬─────────────────┘ │
└──────────────────────────────────────────┼─────────────────────┘
                                           │ reads only
                    ┌──────────────────────▼──────────────────┐
                    │  internal/services/  (new, 3 services)   │
                    │  ComplianceService · EvidenceService      │
                    │  OversightService                         │
                    └──────────────────────┬──────────────────┘
                                           │ reads only, never writes back
                    ┌──────────────────────▼──────────────────┐
                    │  PostgreSQL 16 — same instance             │
                    │  Existing (untouched): audit_events,       │
                    │  activity_events, policy_actions, policies │
                    │  New: frameworks, controls, evidence,      │
                    │  snapshots, oversight_events                │
                    └───────────────────────────────────────────┘
```

**Cardinal rule:** compliance never sits between an AI agent's action and the ALLOW/BLOCK decision. All compliance writes happen after the fact — on a schedule, or via an explicit dashboard action.

---

## 5. Compliance Domain Design

### 5.1 Package Structure

```
internal/services/               ← new package, 3 files for the POC
├── compliance_service.go        ← control catalog + status calculation
├── evidence_service.go          ← extraction from existing tables
└── oversight_service.go         ← violation review (§5.4)
```

### 5.2 Control Catalog — Data, Not Code

A control is a testable statement plus a machine-readable list of what evidence satisfies it. This is deliberately data, not Go code, so the POC's 5–10 controls can grow into full SOC2/EU AI Act coverage later without a rewrite.

```go
type Control struct {
    ControlID      uuid.UUID
    ControlCode    string          // e.g. CTRL-ACCESS-PROVISIONING
    Title          string
    EvidenceSpec   json.RawMessage
    EvidenceStatus string          // operational | planned — see §5.3
}

type EvidenceSpec struct {
    AuditEventTypes      []string          // e.g. ["DEVELOPER_ADDED", "DEVELOPER_DISABLED"]
    ActivityEventKinds    []string
    PolicyActionFilters   map[string]string // e.g. {"violation_type": "data_export"}
    OversightRequired      bool              // true → needs a human review event, not just the raw event
}
```

**`EvidenceStatus` is the fix for a real modeling bug:** a control referencing event types the platform doesn't emit yet (e.g. an audit action that hasn't been added) will always show zero evidence — not because anything is wrong, but because nothing *can* produce evidence for it. Marking it `planned` instead of `operational` keeps it truthfully out of the compliance calculation until its evidence source actually exists. Only `operational` controls (§5.3) are counted.

### 5.3 Status Calculation — Only Operational Controls Are Measured

Two corrections from a naive "count evidence, low count = non-compliant" model:

1. **Planned controls are excluded entirely**, not scored as failing. A control with no evidence pipeline yet isn't proof of a gap — it's proof we haven't built that pipeline yet. Scoring it as non-compliant would mean *the more thorough the control catalog gets, the worse the score looks*, which is backwards.
2. **For detection/enforcement controls, absence of violations is a good sign, not a bad one.** "Zero data-export blocks this month" usually means nobody tried anything risky — not that the control failed. What actually proves this kind of control works is whether an **active, mapped policy exists** to enforce it at all — not how many times it fired.

```go
func (s *ComplianceService) CalculateControlStatus(ctx context.Context, controlID uuid.UUID, window time.Duration) (ControlStatus, error) {
    control := s.getControl(ctx, controlID)

    if control.EvidenceStatus != "operational" {
        return ControlStatus{Status: "planned"}, nil // counted separately from measured statuses — see §8
    }

    if control.EvidenceSpec.OversightRequired {
        // Measures a review window (has anyone looked at this?), never a
        // remediation SLA — there's nothing left to "fix" (see §2).
        if s.countUnreviewedPastWindow(ctx, controlID, reviewWindowFor(control)) > 0 {
            return ControlStatus{Status: "partially_compliant"}, nil
        }
        return ControlStatus{Status: "compliant"}, nil
    }

    // Detection/enforcement control: compliant means "a policy is actively
    // enforcing this," not "violations occurred." Silence is not failure —
    // no active mapped policy at all is the actual failure mode.
    if s.hasActiveMappedPolicy(ctx, controlID) {
        return ControlStatus{Status: "compliant"}, nil
    }
    return ControlStatus{Status: "non_compliant"}, nil
}
```

**No percentage is calculated, deliberately** — following AEGIS's own model (§8 explains why). The framework-level result is a raw count per status: how many controls are `compliant`, `partially_compliant`, `non_compliant`, and `planned`, out of the total applicable controls. See §8 for the exact breakdown shown.

### 5.4 Evidence Extraction

Runs on a schedule (§8), never inline on ingest. Scans rows created since the last watermark and inserts *references* — never copies — into `compliance_evidence`:

```go
func (s *EvidenceService) ExtractSince(ctx context.Context, watermark time.Time) error {
    for _, row := range s.db.QueryAuditEvents(ctx, watermark) {
        for _, c := range s.allControls(ctx) {
            if contains(c.EvidenceSpec.AuditEventTypes, row.EventType) {
                s.insertEvidence(ctx, c.ControlID, "audit_event", row.ID, row.Timestamp)
            }
        }
    }
    for _, row := range s.db.QueryPolicyActions(ctx, watermark) {
        for _, c := range s.allControls(ctx) {
            if matchesFilters(c.EvidenceSpec.PolicyActionFilters, row) {
                s.insertEvidence(ctx, c.ControlID, "policy_action", row.ID, row.OccurredAt)
            }
        }
    }
    return nil
}
```

### 5.5 Violation Review — Not Remediation

Direct fix for G2, and the demo centerpiece. Reviewing a violation never means "the risk is closed" (§2) — it means a specific, different human is on record with a judgment.

```go
func (s *OversightService) ReviewViolation(ctx context.Context, policyActionID, reviewerID uuid.UUID, decision, rationale string) error {
    if s.violationActorID(ctx, policyActionID) == reviewerID {
        return ErrSegregationOfDuties // reviewer must differ from the violation's own actor
    }

    tx, _ := s.db.Begin(ctx)
    defer tx.Rollback(ctx)

    tx.Exec(ctx, `INSERT INTO oversight_events (policy_action_id, reviewer_id, decision, rationale) VALUES ($1,$2,$3,$4)`,
        policyActionID, reviewerID, decision, rationale)

    tx.Exec(ctx, `UPDATE policy_actions SET resolution_status='resolved', resolved_at=NOW() WHERE policy_action_id=$1`, policyActionID)

    auditType := models.AuditViolationReviewed
    if decision == "overridden" {
        auditType = models.AuditViolationOverridden
    }
    tx.Exec(ctx, `INSERT INTO audit_events (actor_id, actor_type, event_type, resource_type, resource_id) VALUES ($1,'developer',$2,'policy_action',$3)`,
        reviewerID, auditType, policyActionID)

    return tx.Commit(ctx)
}
```

**Why segregation of duties is enforced in code:** the same developer marking their own BLOCK "reviewed" proves self-clearing, not oversight. The 403 is the actual guarantee — a UI hint alone wouldn't be, and this is the one piece of evidence in the whole POC that must be trustworthy by construction.

### 5.6 Existing-File Fix Required First (G1) — Now Six Functions, Not Two

`policies.go`'s `CreatePolicy` and `ActivatePolicy` hardcode `actor_type='system'` and never call the `ActorID(r)` helper that already exists. Since this design was first written, the same policy file grew four more handlers (`UpdatePolicy`, `DisablePolicy`, `EnablePolicy` via `setPolicyStatus`, `DeletePolicy`) — all copy-pasted from the same pattern, all repeating the identical gap (§3.3):

```go
// Before (all six handlers, including the four added since): 
// INSERT INTO audit_events (actor_type, ...) VALUES ('system', ...)

// After — same fix, applied everywhere it now appears:
actorID := middleware.ActorID(r)
tx.Exec(ctx, `INSERT INTO audit_events (actor_id, actor_type, event_type, resource_type, resource_id, new_value)
              VALUES ($1, 'developer', $2, 'policy', $3, $4)`, actorID, models.AuditPolicyCreated, policyID, defJSON)
```

**Affected functions, confirmed by direct diff inspection:** `CreatePolicy`, `ActivatePolicy`, `UpdatePolicy`, `DisablePolicy`, `EnablePolicy` (via `setPolicyStatus`), `DeletePolicy`. Needed before any control depends on `audit_events.actor_id` — and now more urgent, since the newly-added lifecycle events (`POLICY_DISABLED`/`ENABLED`/`DELETED`) are exactly the richer CC8.1 change-management evidence this design wants (§3.3), and all of it is currently actor-less.

### 5.7 Automated Policy-to-Control Mapping

Policies are authored on the dashboard with structured, fixed-vocabulary fields — `type` (filesystem/shell/mcp/web_search/prompt), `action` (BLOCK/ALERT), `matchTarget` (path/command/mcp_tool/query/prompt). Because these are already a controlled vocabulary (not free text), a policy can be matched to the controls it satisfies **automatically**, with no human tagging step required for the common case.

**The chain of indirection that makes this work:** a rule never points at a framework — only at a **control**. Controls are already framework-agnostic (§5.2). This decoupling is what lets a new framework be added later without touching the rule engine at all (§5.9).

```
Rule (matches policy attributes) ──► Control (shared) ──► Framework Control Mapping ──► "CC6.7" / "Art.15" / any framework
```

**Rule evaluation is not "first match wins."** Every rule whose condition the policy satisfies fires — a single policy is often legitimately evidence for more than one control at once (a BLOCK on an MCP tool is both boundary protection *and* a candidate for human-oversight review *and* change-management evidence, simultaneously, not competing categories):

```go
type PolicyMappingRule struct {
    RuleCode        string
    When            map[string]string // e.g. {"type": "mcp", "action": "BLOCK"} — missing keys = wildcard
    MapsToControlID uuid.UUID
}

func AutoMapPolicyToControls(ctx context.Context, tx pgx.Tx, policyID uuid.UUID, def PackPolicy) error {
    for _, rule := range loadActiveRules(ctx) {
        if rule.Matches(def.Type, def.Action, def.MatchTarget) {
            tx.Exec(ctx, `
                INSERT INTO policy_control_mappings (policy_id, control_id, mapping_source, rationale)
                VALUES ($1, $2, 'automatic', $3)
                ON CONFLICT (policy_id, control_id) DO NOTHING`,
                policyID, rule.MapsToControlID, "Auto-mapped via "+rule.RuleCode)
        }
    }
    return nil
}
```

**A catch-all rule guarantees no policy is ever left unmapped:** `{ "when": {}, "mapsToControl": "CTRL-CHANGE-MANAGEMENT" }` matches every policy regardless of type, because every policy going through the versioned policy pipeline is itself change-management evidence (CC8.1) independent of what it's about.

**A second matchable dimension now exists: `policyType`.** The platform added a `policy_type` column (`governance` / `pii` / `third_party_monitor`) independent of a policy's `type`/`action`/`matchTarget` (§3.3). Rules can match on it directly — no design change needed, `when` already accepts any policy attribute as a key:

```json
[
  { "when": { "policyType": "pii" },
    "mapsToControl": "CTRL-PII-PROTECTION" },

  { "when": { "policyType": "third_party_monitor" },
    "mapsToControl": "CTRL-THIRD-PARTY-VISIBILITY" }
]
```

The PII rule gives `CTRL-PII-PROTECTION` a materially stronger evidence source than a generic filesystem-path rule ever could — it's matching against a purpose-built detector (named PII patterns: email, SSN, credit card, API key, JWT, etc.), not inferring intent from a regex. The third-party rule is new evidence entirely — nothing in the original design accounted for the platform discovering *other* enforcement tooling already present in an agent's configuration (§3.3).

**Three distinct moments, not one "mapping step":**

| When | What happens |
|---|---|
| Server startup | Loads rule definitions and the control catalog from their JSON source into the DB — an idempotent sync, never creates a `policy_control_mappings` row itself |
| Policy create/activate (event-driven) | `AutoMapPolicyToControls` runs immediately, same transaction as the existing `audit_events` insert — the policy has its mappings the moment it exists |
| The existing hourly ticker (§8) | Re-runs the same rule evaluation across **every** policy, as an idempotent `INSERT ... ON CONFLICT DO NOTHING` — this is what makes a newly added rule retroactively apply to policies that already existed before that rule was written, without a separate manual backfill step |

**A human correction path remains, but it's no longer required for a mapping to exist.** The Policy Detail Panel (§10) shows which rule produced each mapping and lets someone remove or add a mapping if an auto-rule is wrong or incomplete — a safety net, not a gate.

---

## 6. Database Design (POC Tables Only)

**Database:** PostgreSQL 16, same instance. **Migrations:** continuing from `003_session_last_event.sql`.

### 6.0 Principles

- Compliance tables are never a required FK target for operational tables — `audit_events`/`activity_events`/`policy_actions` stay unaware this layer exists
- `compliance_evidence` stores references, never copies
- Naming/typing matches the existing schema (`gen_random_uuid()`, `TIMESTAMP WITH TIME ZONE`, `idx_<table>_<column>`)

### 6.1 Tables

| Table | Key columns | Notes |
|---|---|---|
| `compliance_frameworks` | `framework_type` (soc2 / eu_ai_act), `display_name` | Just the 2 frameworks for the POC |
| `controls` | `control_code` (unique), `evidence_spec` (JSONB), `evidence_status` (operational / planned) | 5–10 rows, seeded once. `evidence_status` gates whether a control is measured at all (§5.3) |
| `framework_control_mappings` | `framework_id`, `control_id`, `framework_control_code` | e.g. maps one control to both "CC6.1" and "Art.14" |
| `policy_mapping_rules` | `rule_code` (unique), `when_conditions` (JSONB), `maps_to_control_id`, `status` | The auto-mapping rule table (§5.7) — data, not code |
| `policy_control_mappings` | `policy_id`, `control_id`, `mapping_source` (automatic / manual) | Which policy satisfies which control — also what `hasActiveMappedPolicy()` checks (§5.3). `mapping_source` records whether a rule or a human created the row |
| `compliance_evidence` | `control_id`, exactly one of `source_audit_event_id` / `source_activity_event_id` / `source_policy_action_id` (CHECK), `observed_at` | Reference only, never a copy |
| `compliance_status_snapshots` | `framework_id`, `applicable_control_count`, `compliant_count`, `partially_compliant_count`, `non_compliant_count`, `planned_count`, `calculated_at` | Raw counts only — no computed percentage column (§8) |
| `oversight_events` | `policy_action_id`, `reviewer_id`, `decision`, `rationale` | `reviewer_id` ≠ violation's own developer |

### 6.2 Entity Overview

```
Policy Mapping Rule ──(auto-evaluates, §5.7)──► Policy (existing)
                                                    │
Compliance Framework ── Framework Control Mapping ──► Control (shared)
                                                          ├── Policy Control Mapping ──► Policy (existing)
                                                          └── Compliance Evidence ──► audit_event / activity_event / policy_action (existing)
Compliance Status Snapshot (per framework)

Policy Action (existing) ── Oversight Event (review — not remediation, §2)
```

### 6.3 Control Definition Format

```json
{
  "controlCode": "CTRL-HUMAN-OVERSIGHT",
  "title": "Human review of automated high-risk decisions",
  "evidenceSpec": { "auditEventTypes": ["VIOLATION_REVIEWED", "VIOLATION_OVERRIDDEN"], "oversightRequired": true }
}
```

`framework-eu-ai-act.json` maps this to `Art.14`; `framework-soc2.json` maps the same control to `CC7.2` — one evidence pipeline, two frameworks, no duplicated logic.

### 6.4 Rule Definition Format

```json
{
  "ruleCode": "RULE-FS-PATH-BLOCK",
  "when": { "type": "filesystem", "matchTarget": "path", "action": "BLOCK" },
  "mapsToControl": "CTRL-DATA-PROTECTION",
  "description": "Filesystem path blocks demonstrate data protection"
}
```

`when` conditions are matched against a policy's own `type`/`action`/`matchTarget` fields; any key omitted from `when` is a wildcard (matches any value for that field). The empty object `{}` is the catch-all rule guaranteeing every policy maps to at least `CTRL-CHANGE-MANAGEMENT`.

### 6.5 Adding a New Framework — What Actually Changes

Because rules target controls, not frameworks, adding a framework is almost always a pure data change:

```
Case A — the new framework's requirement is already satisfied by an existing control
  (common — access provisioning, boundary protection, and change management
   are asked for by nearly every framework in some form)

  → Add ONE row to framework_control_mappings.
  → No changes to rules, controls, or code.
  → Every policy already auto-mapped to that control retroactively counts
    as evidence for the new framework, immediately — including policies
    created long before the new framework was added.

Case B — the new framework needs evidence that genuinely doesn't exist yet

  → Add a new row to `controls` (evidence_status = "planned" until a real
    evidence source exists).
  → Optionally add a new rule, IF the evidence can be derived from a
    policy's own attributes — many can't (e.g. personnel competence
    records aren't a policy attribute), and that control simply stays
    "planned" until something new is built.
  → Add ONE row to framework_control_mappings.
  → Still no changes to the rule engine's code — only new data rows.
```

The rule engine itself (`AutoMapPolicyToControls`, §5.7) never needs a code change to onboard a new framework — only Case B ever adds new data rows, and even then the matching logic is unchanged.

---

## 7. Backend API (POC Endpoints Only)

```
POST /api/v1/violations/:id/review              → record a human review (§5.5) — closes G2

GET  /api/v1/compliance/frameworks               → frameworks + current status as raw counts
GET  /api/v1/compliance/frameworks/:id/controls  → controls + status
GET  /api/v1/compliance/frameworks/:id/status    → cached snapshot, never live-calculated
GET  /api/v1/compliance/controls/:id/evidence    → evidence records for a control

GET    /api/v1/policies/:id/control-mappings              → mappings for a policy, with mapping_source (§5.7)
POST   /api/v1/policies/:id/control-mappings              → manual correction path — add a mapping an auto-rule missed
DELETE /api/v1/policies/:id/control-mappings/:mappingId   → manual correction path — remove a wrong auto-mapping
```

**Review request/response:**

```
POST /api/v1/violations/550e8400-.../review
{ "decision": "overridden", "rationale": "False positive — path is a test fixture" }

200 OK  { "status": "resolved" }

403 Forbidden  (reviewer == the violation's own actor)
{ "error": "reviewer must differ from the violation's actor" }
```

**Router additions** (no scope middleware for the POC — existing all-authenticated-callers behavior is fine at this stage):

```go
r.Post("/violations/{id}/review", ReviewViolation)
r.Get("/compliance/frameworks", ListComplianceFrameworks)
r.Get("/compliance/frameworks/{id}/controls", GetFrameworkControls)
r.Get("/compliance/frameworks/{id}/status", GetFrameworkStatus)
r.Get("/compliance/controls/{id}/evidence", GetControlEvidence)
```

---

## 8. Compliance Calculation Strategy

Status is calculated on a schedule, never per-request — this bounds dashboard latency and matches how the platform already keeps enforcement fast.

```
In-process ticker (cmd/server/main.go, hourly)
  └── AutoMapPolicyToControls for every policy, against every active rule   (§5.7 — idempotent, applies new rules retroactively)
  └── EvidenceService.ExtractSince(watermark)
  └── ComplianceService.CalculateFrameworkSnapshot(fw)   → one row per framework

Dashboard reads
  └── Always the latest snapshot — worst-case staleness = one hour
```

### 8.1 Raw Counts, Not a Percentage

**No percentage is computed anywhere in this design — this is a deliberate correction, not an omission.** AEGIS's own compliance engine (`compliance-bundle.ts`) was checked directly for this: it classifies every control into one of three categorical states (`covered` / `partial` / `uncovered`) and reports only **raw counts** at the framework level (`total_controls`, `covered`, `partial`, `uncovered`) — it never divides one by the other. Its own code comment states the reason plainly: *"The point is NOT to claim audit readiness on the customer's behalf — only the auditor can do that."*

A computed percentage — even a carefully-labeled one — visually reads as a score or a certification. That's a claim this platform shouldn't make on an org's behalf. So the snapshot reports counts only:

```
ComplianceStatusSnapshot for a framework:
  applicable_control_count     — every control mapped into this framework's catalog
  compliant_count               — operational, and evidence shows the control is satisfied
  partially_compliant_count      — operational, but some oversight-required items are unreviewed
  non_compliant_count            — operational, and no active mechanism/evidence satisfies it
  planned_count                  — not yet operational; no evidence pipeline exists for it yet
```

`compliant_count + partially_compliant_count + non_compliant_count + planned_count = applicable_control_count`, always. No ratio is derived from these four numbers anywhere in the backend — if a percentage is ever wanted for a specific report, it should be computed and clearly labeled at the point of display, not stored or treated as the system's own assessment.

---

## 9. Applicable Controls — SOC2 & EU AI Act

This is the concrete answer to "which controls, specifically." Both frameworks' full control lists are below, each marked against what this platform's evidence can actually measure — following §5.3's `operational` / `planned` distinction, not a rounded-up estimate.

### 9.1 A Necessary Caveat for EU AI Act — Deployer, Not Provider

The EU AI Act splits obligations by role. **Art.9–15 are *provider* obligations** — they belong to whoever built the AI system (e.g. Anthropic, for the underlying model). **Art.26 is the *deployer* obligation set** — it applies to the organization *using* the AI system, which is who actually runs this platform. For most customers running AI coding agents internally, **Art.26 is the more directly relevant article**, and it happens to line up closely with what this platform already produces (automatic logging, human oversight assignment, monitoring). Art.9–15 are included below because they're the commonly-cited articles and matter if a customer is *also* a provider of a high-risk AI system — but Art.26 shouldn't be an afterthought.

### 9.2 SOC2 — All 9 Common Criteria Categories

| # | Category | What it asks | Measurable now? | Evidence source |
|---|---|---|---|---|
| CC1 | Control Environment | Tone at the top, org structure, accountability | ❌ Not measurable | Org culture/HR artifact — not a log |
| CC2 | Communication & Information | Policies communicated internally/externally | ❌ Not measurable | Requires distribution/acknowledgment records |
| CC3 | Risk Assessment | Formal risk identification and analysis process | ❌ Not measurable | A documented process, not runtime data |
| CC4 | Monitoring Activities | Ongoing evaluation of the control environment itself (meta-monitoring) | ❌ Not measurable | Program-design artifact, distinct from CC7's runtime detection |
| CC5 | Control Activities | Policies/procedures selected and deployed | ⚠️ Partial | `policies`/`policy_versions` prove *a* control activity exists; doesn't prove the broader control-selection process |
| **CC6.1** | Logical access — security measures implemented | Access control mechanisms exist | ✅ **Operational** | `audit_events` (developer/agent lifecycle) |
| **CC6.2 / CC6.5** | User registration/authorization; access removed when no longer needed | Provisioning and deprovisioning are tracked | ✅ **Operational** | `AuditDeveloperAdded/Disabled`, `AuditAgentRegistered/Disabled` |
| CC6.3 | Role-based access, least privilege | Access is scoped by role | ❌ Not measurable | No role/permission model exists yet in the platform itself (G12, deferred) |
| CC6.4 | Physical access restriction | Physical facility controls | ❌ N/A | Not applicable to this platform's scope |
| **CC6.6** | Boundary protection / external threat controls | Unauthorized external tool access is restricted | ✅ **Operational** | `policy_actions` blocking MCP/tool access |
| **CC6.7** | Restricts data transmission/movement | Data exfiltration attempts are blocked | ✅ **Operational**, once `violation_type` enum (G3) lands | `policy_actions` where `violation_type = 'data_export'`, **now strengthened by first-class PII detection** — `activity_events.pii_detected`/`pii_matches` against named patterns (email, SSN, credit card, API key, JWT), not just inferred filesystem-path regexes (§3.3) |
| CC6.8 | Prevents/detects malicious software | Malware controls | ❌ N/A | Not applicable to this platform's scope |
| **CC7.1** | Detects and monitors for anomalies | Detection capability exists and fires | ✅ **Operational** | `policy_actions` (ALLOW/ALERT/BLOCK) |
| **CC7.2** | Monitors and evaluates security events | Violations are reviewed by a human | ✅ **Operational**, once G2 lands | `oversight_events` |
| CC7.3–CC7.5 | Incident evaluation, response, recovery | Formal incident lifecycle | ❌ Planned (deferred, §13) | Needs `IncidentService` — out of POC scope |
| **CC8.1** | Change management — changes are authorized, tested, approved | Policy changes go through controlled versioning | ✅ **Operational** | `policy_versions`, immutable, actor-attributed once G1 lands (now covering the full lifecycle — create/update/activate/disable/enable/delete, §3.3) |
| CC9.1 | Business disruption / continuity risk | BCP/DR planning | ❌ N/A | Not applicable to this platform's scope |
| **CC9.2** | Vendor / business-partner risk | Third-party risk is managed | ✅ **Operational** (upgraded from weak partial, §3.3) | `session_hooks` (non-governance hooks inventoried at session start) + `third_party_monitor` policies (auto-discovered third-party enforcement, with `attribution_confidence`) — still narrower than a formal per-vendor risk assessment, but now a real, populated evidence source rather than just an MCP inventory |

**Result: 8 of ~19 applicable criteria operational** (CC6.1, CC6.2/6.5, CC6.6, CC6.7, CC7.1, CC7.2, CC8.1, CC9.2), 1 weak partial (CC5), 1 planned (CC7.3–7.5), the remainder (CC1–4, CC6.3/6.4/6.8, CC9.1) genuinely out of this platform's reach regardless of how much is built, because they're organizational/physical, not operational-log evidence.

### 9.3 EU AI Act — Deployer Obligations (Art.26) and Provider Obligations (Art.9–15)

**Art.26 — Deployer obligations (most relevant for most customers):**

| Obligation | What it asks | Measurable now? | Evidence source |
|---|---|---|---|
| Assign human oversight to competent personnel | A named, trained person reviews AI decisions | ✅ **Operational**, once G2 lands | `oversight_events.reviewer_id` |
| Monitor operation per instructions | Ongoing monitoring of the AI system's behavior | ✅ **Operational** | `activity_events` continuous capture |
| Keep automatically generated logs (min. 6 months) | Retained, complete logs of system operation | ✅ **Operational** | `audit_events`/`activity_events` — retention policy is a config check, not new evidence |
| Inform provider/authority of risks or serious incidents | Incident escalation | ❌ Planned (deferred, §13) | Needs `IncidentService` |

**Art.9–15 — Provider obligations (relevant only if the customer is also a provider of a high-risk system):**

| Article | What it asks | Measurable now? | Evidence source |
|---|---|---|---|
| Art.9 — Risk management system | Documented, lifecycle-wide risk identification/analysis | ⚠️ Partial | Policies function as a risk *control*; the full documented risk management *system* is external |
| Art.10 — Data and data governance | Training/validation data quality and governance | ❌ N/A | Not applicable — these are pre-trained third-party models, not trained by the customer |
| Art.11 — Technical documentation | Narrative documentation of design, intended purpose, testing | ❌ Planned (deferred, §13) | Would need the capability declaration work, deferred out of POC |
| **Art.12 — Record-keeping** | Automatic logging across the system's lifecycle | ✅ **Operational** | `audit_events` + `activity_events` — closest to a direct match of any control in this document |
| Art.13 — Transparency to deployers | Declared capabilities, limitations, accuracy metrics | ⚠️ Weak partial (upgraded from planned, §3.3) | `skills`/`session_skills` now give a real, observed capability inventory (which skills exist, who invokes them, how often) — but this is *observed* usage, not the *declared* capability + drift-detection model §5.5 originally described. Full Art.13 still needs the declaration side |
| **Art.14 — Human oversight** | A human can review/intervene in automated decisions | ✅ **Operational**, once G2 lands | `oversight_events` — purpose-built match |
| Art.15 — Accuracy, robustness, cybersecurity | Adversarial robustness and accuracy testing | ⚠️ Weak partial | Violation tally shows enforcement is active; doesn't include actual robustness/accuracy testing |

**Result: 3 of 11 EU AI Act obligations operational** (Art.26 oversight assignment, Art.26 monitoring/logging, Art.12, Art.14 — counting Art.26's two operational sub-obligations together with Art.12/14), 3 partials (Art.9, Art.13, Art.15), 2 planned (Art.26 incident reporting, Art.11), 1 N/A (Art.10).

### 9.4 What This Means for the Snapshot Counts (§8.1)

Plugging §9.2/§9.3 into the raw-count model, with no ratio taken:

```
SOC2 (applicable_control_count ≈ 19)
  compliant_count:           8   (CC6.1, CC6.2/6.5, CC6.6, CC6.7, CC7.1, CC7.2, CC8.1, CC9.2)  ← CC9.2 upgraded, §3.3
  partially_compliant_count: 0
  non_compliant_count:       0
  planned_count:            11   (CC1–4, CC5, CC6.3/6.4/6.8, CC7.3–7.5, CC9.1)

EU AI Act (applicable_control_count = 8, counting Art.26 as 2 sub-obligations)
  compliant_count:            4   (Art.26 oversight assignment, Art.26 monitoring/logging, Art.12, Art.14)
  partially_compliant_count:  3   (Art.9, Art.13, Art.15)  ← Art.13 upgraded from planned, §3.3
  non_compliant_count:        0
  planned_count:              1   (Art.26 incident reporting, Art.11 — grouped for brevity)
```

Both movements came from real implementation changes (§3.3), not a redesign of the scoring model itself — the raw-count structure absorbed new evidence sources without needing to change.

This is the honest starting count for this POC's catalog before any additional evidence pipelines are built — a growing `planned_count` over time is expected and not something to hide; it's the accurate baseline the roadmap (§14) grows from. Nothing here is divided into a percentage — a reader can see at a glance that most of the catalog isn't measured yet, without that fact being compressed into a single low-looking number.

---

## 10. Dashboard UI (POC Pages Only)

Follows the existing panel pattern — full-screen overlay panels that stack, each with its own **✕ Close**, no URL change on open. Compliance gets its own clearly separated section with three tiers, matching how the rest of the dashboard already drills down (Sessions → Violation, Policy → Violation): **Acts defined → Controls covered → Evidence records.** These panels are now real, existing files (§3.3) — `components/panels/violation-panel.tsx` and `components/panels/policy-panel.tsx` — each gains one new section; nothing else changes shape. (Confirmed by direct inspection: `violation-panel.tsx` currently has no review/oversight UI at all — §10.5 below is a net-new section, not an extension of something partially built.)

### 10.1 Pages

| Page | Tier | Purpose |
|---|---|---|
| `/compliance` | 1 — Acts defined | Every framework (SOC2, EU AI Act) as a card, with raw status counts from the latest snapshot |
| `/compliance/[frameworkId]` | 2 — Controls covered | Every control mapped to this framework, its current status, and evidence count — click a row to drill in |
| `/compliance/control/[controlId]` | 3 — Evidence records | The actual evidence rows backing this control's status, plus which policies feed it and how they were mapped |
| Violations (existing, extended) | — | Adds a Review action per row |
| Policy Detail (existing, extended) | — | Adds a "Mapped Controls" section showing auto-mapped rules with a manual correction option |

### 10.2 Tier 1 — Acts Defined (`/compliance`)

```
┌─────────────────────────────────────────────────────────────────┐
│  Compliance                                                      │
├─────────────────────────────────────────────────────────────────┤
│  SOC2                                    EU AI Act                │
│  19 applicable controls                  8 applicable obligations │
│  ● Compliant: 8                          ● Compliant: 4           │
│  ▲ Partial: 0                            ▲ Partial: 3             │
│  ✕ Non-compliant: 0                      ✕ Non-compliant: 0       │
│  ○ Planned: 11                    →      ○ Planned: 1       →     │
└─────────────────────────────────────────────────────────────────┘
```

Every framework the org has defined is a card here, always showing raw counts (§8.1) — never a blended score. Clicking a card opens Tier 2 for that framework.

### 10.3 Tier 2 — Controls Covered (`/compliance/[frameworkId]`)

```
┌───────────────────────────────────────────────────────────────────┐
│  Observe / Compliance / SOC2                            ✕ Close   │
├───────────────────────────────────────────────────────────────────┤
│  SOC2                    19 applicable controls                    │
│  ● Compliant: 8    ▲ Partial: 0    ✕ Non-compliant: 0    ○ Planned: 11│
├───────────────────────────────────────────────────────────────────┤
│  Code    Title                        Status         Evidence      │
│  CC6.1   Access provisioning          ● compliant     14 records → │
│  CC6.7   Data transmission            ● compliant     6 records  → │
│  CC7.2   Incident evaluation          ● compliant     31 records → │
│  CC9.2   Vendor/third-party risk      ● compliant     9 records  → │
│  CC1     Control environment          ○ planned       —          → │
└───────────────────────────────────────────────────────────────────┘
```

Raw counts, not a percentage (§8.1) — `planned` controls are shown separately, never blended into compliant/partial/non-compliant. Clicking any row opens Tier 3, nested on top of this panel.

### 10.4 Tier 3 — Evidence Records (`/compliance/control/[controlId]`)

```
┌───────────────────────────────────────────────────────────────────┐
│  Observe / Compliance / SOC2 / CC6.7                     ✕ Close  │
├───────────────────────────────────────────────────────────────────┤
│  CC6.7 — Data transmission — restricted access     ● compliant     │
│  "Restricts data transmission/movement"                            │
├──────────────────────────────────┬──────────────────────────────────┤
│  Mapped Policies                 │  Evidence Records (6)              │
│  ──────────────────────────────  │  ────────────────────────────────  │
│  "block .env access"             │  policy_action  2026-08-12 10:12   │
│    [auto: RULE-FS-PATH-BLOCK]    │    BLOCK — .env read attempt   →   │
│                                   │                                    │
│  "block AWS key regex"           │  policy_action  2026-08-11 14:02   │
│    [auto: RULE-FS-PATH-BLOCK]    │    BLOCK — AWS key pattern     →   │
│                                   │                                    │
│  "block Slack webhooks" [disabled]│  ... 4 more                       │
│    [manual, added by @secops]    │                                    │
└──────────────────────────────────┴──────────────────────────────────┘
```

This is the tier that answers *"prove it"* — every evidence row links back to the exact `policy_action`/`audit_event`/`activity_event` that produced it (§5.4: reference, never a copy), and every mapped policy shows **how** it got mapped — `[auto: RULE-...]` or `[manual, added by @user]` — so the provenance of every piece of evidence is inspectable, not just its existence.

### 10.5 Violation Review — New Section in `violation-panel.tsx`

```
┌───────────────────────────────────────────────────────┐
│  [● BLOCK]  2026-08-12 10:12:47              ✕ Close   │
├──────────────────────────────────┬─────────────────────┤
│  Context (unchanged)             │  Full Payload         │
│                                   │  (unchanged)          │
│  Review  (NEW)                    │                      │
│  ( ) Affirm  ( ) Override  ( ) Escalate                   │
│  Rationale: [____________]                                │
│  [ Submit Review ]                                        │
└──────────────────────────────────┴─────────────────────┘
```

Labeled **"Review,"** not "Resolve" — reviewing is the accurate word for what's happening (§5.5).

### 10.6 Policy Detail — New "Mapped Controls" Section in `policy-panel.tsx`

`policy-panel.tsx` already renders policy config, status (via `policy-status-badge.tsx` — now showing `active`/`disabled` per the real lifecycle work in §3.3), and version history. This adds one section:

```
┌───────────────────────────────────────────────────────┐
│  Block credential files  [BLOCK] [filesystem] [active] ✕ Close │
├───────────────────────────────────────────────────────┤
│  ... (existing policy config, unchanged)                │
│                                                          │
│  Mapped Controls  (NEW)                                  │
│  ──────────────────────────────────────────────────────  │
│  CC6.7  Data transmission     [auto: RULE-FS-PATH-BLOCK] ✕│
│  CC8.1  Change management     [auto: catch-all]          ✕│
│  + Add control mapping                                    │
└───────────────────────────────────────────────────────┘
```

This is the correction path from §5.7 — every mapping shows which rule produced it, and can be removed (✕) or a new one added manually if an auto-rule got it wrong. No mapping requires this screen to exist at all; it's a safety net, not a required step.

---

## 11. Implementation Plan

| Step | Deliverables | Gaps closed |
|---|---|---|
| **1 — Prerequisites** | `ActorID` wiring, `violation_type` enum, framework/control/rule schema, seed the controls in §9 and the rule table (§5.7, §6.4) | G1, G3, G4 |
| **2 — Services** | `EvidenceService`, `ComplianceService`, `OversightService`, `AutoMapPolicyToControls` (§5.7), review endpoint, ticker (now also re-running auto-mapping, §8) | G2, G5, G7 |
| **3 — API & Dashboard** | `/api/v1/compliance/*` routes, mapping-correction endpoints (§7), three-tier `/compliance` pages (§10.1–10.4), Review section in `ViolationPanel`, Mapped Controls section in Policy Detail | G6 |

---

## 12. Critical Design Principles

1. **Compliance never sits on the enforcement path.** One non-blocking call from `ingest.processOne`; everything else is scheduled.
2. **Review, not remediation.** A violation is a completed event, not an open exposure (§2).
3. **Evidence is referenced, never copied.** Exactly one source FK per `compliance_evidence` row.
4. **Controls are shared; frameworks are labels.** One control can map to both SOC2 and EU AI Act — no duplicated evidence logic.
5. **Oversight is enforced server-side.** Reviewer ≠ actor is a 403, not a UI convention.
6. **Only what's measurable is measured, and it's never blended into one number.** A `planned` control (no evidence pipeline built yet) is counted in its own `planned_count`, never folded into `non_compliant_count` — a bigger catalog should never make the compliant/partial/non-compliant counts look worse (§5.3, §8.1).
7. **Silence isn't failure for enforcement controls.** Zero violations usually means nothing risky was attempted, not that a control failed — compliance for these controls is proven by an active, mapped policy existing, not by violation counts.
8. **Deployer obligations come first for EU AI Act.** Most customers are deployers, not providers — Art.26 (§9.3) is prioritized over Art.9–15 for that reason.

---

## 13. Gap Resolution Summary

| Gap | Resolution | Step |
|---|---|---|
| G1 — actor_id never populated | Wire `ActorID(r)` into `policies.go` | 1 |
| G2 — no review endpoint | `oversight_events` + `POST /violations/:id/review` | 1–2 |
| G3 — unenforced violation_type | Postgres `ENUM` | 1 |
| G4 — no framework/control model | Catalog + mapping tables | 1 |
| G5 — no evidence extraction | `EvidenceService.ExtractSince()` | 2 |
| G6 — no compliance API/dashboard | `/api/v1/compliance/*` + `/compliance` pages | 3 |
| G7 — no scheduler | In-process ticker | 2 |

---

## 14. Deferred Beyond This POC

Not designed in this document — each needs its own design pass when the platform moves past POC:

| Capability | Why deferred |
|---|---|
| Agent risk tiering + capability declaration/drift | EU AI Act Art.13 needs it eventually; hardcode one agent's risk tier for the demo instead |
| Incident lifecycle | Only needed if something gets through an ALERT and needs investigating — rare in a demo |
| Exception/waiver workflow | Needed once real controls legitimately don't apply to every org — not before |
| Integrity checkpoints (tamper-evidence) | Matters for a real auditor, not a demo audience |
| Signed evidence bundles | Ed25519 signing matters once there's an actual external auditor consuming exports |
| Scope-based permissions (`RequireScope`) | Real access control matters in production; invisible in a demo |
| NIST AI RMF / ISO 42001 / additional frameworks | Cheap to add later via the same catalog mechanism — not needed to prove the concept |
