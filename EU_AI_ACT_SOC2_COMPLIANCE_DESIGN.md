# EU AI Act & SOC2 Compliance — Design for AI Governance Platform

**Document Version:** 1.0
**Date:** August 2026
**Author:** Senior Architecture Review
**Scope:** Design only — no implementation. Grounded in AEGIS's existing compliance model and the current AI Governance repo (`governance-api 1`, `governance-ui 1`).

---

## 0. How AEGIS Designed Compliance (Reference Model)

Before designing anything new, it's worth restating precisely what AEGIS does, because the design below is a deliberate adaptation of that model — not a reinvention.

AEGIS treats compliance as a **derived, read-only layer** on top of things it was already doing for other reasons (audit logging, policy enforcement, transparency logging). It never asks an operator to manually assert "we are SOC2 compliant" — it computes that from evidence.

**AEGIS's four building blocks:**

1. **`compliance-controls.ts`** — a static registry mapping each framework (`soc2`, `iso27001`, `nist-ai-rmf`, `eu-ai-act`) to a list of controls. Each control declares an `evidenceSpec`: which audit actions, which detectors, which ontology nodes, and which artifacts would satisfy it. This is metadata, not logic — it says *what counts as proof*, not how to fetch it.

2. **`admin_audit_log`** — a single immutable table that every administrative and security-relevant action writes to (policy changes, approvals, kill-switches, API key lifecycle, proxy calls). This is the raw evidence substrate. Nothing in the compliance layer writes new tables of "compliance data" — it all reads from this one log plus the detector registry and the transparency log.

3. **`compliance-bundle.ts`** — the calculator. For each control, it queries `admin_audit_log` for the counts of the actions named in `evidenceSpec.auditActions`, checks whether the named detectors are registered, checks ontology coverage, and pulls artifacts (transparency root hash, row counts). It classifies each control as `covered` / `partial` / `uncovered` and hashes + Ed25519-signs the resulting bundle, optionally anchoring it in a Merkle transparency log.

4. **`eu-ai-act-evidence.ts`** — a framework-specific specialization for Articles 12–15, because EU AI Act evidence isn't just "count of audit actions" — Art.14 (human oversight) needs approval decisions and kill-switch activations, Art.13 (transparency) needs the actual registered-agent/tool-scope declarations, not just a count.

The critical design insight: **AEGIS never modified its enforcement/logging code to "add compliance."** It added a read-only interpretation layer that maps existing operational telemetry onto external frameworks' language. That is the pattern this document follows for the current AI Governance repo.

---

## 1. Logs & Audit Events Required for Compliance

### 1.1 What the current repo already captures

The current system already has two of the three telemetry primitives AEGIS relies on:

| Existing table | AEGIS equivalent | Compliance role |
|---|---|---|
| `audit_events` (system mutations: policy CRUD, activate/rollback, developer/agent lifecycle) | `admin_audit_log` | Proves *governance change control* — who changed what policy, when, with old/new value diff |
| `activity_events` (every session event — tool.pre/post, prompt.submit, session.start/end — with ALLOW/ALERT/BLOCK) | (closest AEGIS equivalent: proxy call audit rows) | Proves *operational enforcement* — what the AI agent actually did and what was allowed/blocked |
| `policy_actions` (BLOCK/ALERT/ALLOW with severity, resolution_status) | Detector/violation counts | Proves *control effectiveness* — did the guardrail actually fire when it should have |

This is a genuinely strong starting position — better in one respect than AEGIS's single-log model, because activity/audit/policy-action are already separated by concern rather than folded into one table.

### 1.2 Gaps against EU AI Act Art. 12–15

Mapping what Art. 12–15 actually require against what's captured today:

| Article | Requirement | Currently captured? | Gap |
|---|---|---|---|
| **Art. 12** (record-keeping) | Automatic logging of AI system operation across its lifecycle, traceable events | ✅ `activity_events` + `audit_events` cover this well | Missing: log **integrity proof** (hash chaining / transparency log) — currently nothing prevents silent row deletion or backdating |
| **Art. 13** (transparency) | Documented capabilities, declared tool/data scope, instructions for use | ⚠️ Partial — `ai_coding_agents`, `mcp_servers` describe *what's registered*, but there's no explicit "declared scope" object per agent/session that can be exported as a transparency artifact | Need a **capability declaration record** per agent version (tool scope, data categories touched, model/version used) |
| **Art. 14** (human oversight) | Evidence that a human can intervene / did intervene in high-risk decisions | ⚠️ Partial — `policy_actions.resolution_status` (open/resolved/suppressed) exists but there is **no approval/override event type** — no record of a human reviewing a BLOCK/ALERT and making a decision | Need an **approval/override audit action** (currently the AuditAction enum has no `approval.*` equivalent — AEGIS has `approval.approve`/`approval.reject`, current repo does not) |
| **Art. 15** (accuracy, robustness, security) | Evidence of detection/monitoring accuracy, incident handling, security measures | ⚠️ Partial — violation counts exist (`policy_actions`), but no **rollback/incident event type**, no explicit **integrity/verification artifact** analogous to AEGIS's per-agent trace integrity check | Need incident/rollback audit actions + a periodic integrity verification job producing a storable artifact |

### 1.3 Gaps against SOC2 (Trust Services Criteria)

| Criterion | Requirement | Currently captured? | Gap |
|---|---|---|---|
| CC6.1 (logical access) | Access provisioning/deprovisioning evidence | ✅ `AuditDeveloperAdded/Disabled`, `AuditAgentRegistered/Disabled` | None significant |
| CC6.6 (access review) | Evidence of periodic access review | ❌ Not captured | Need an **access-review event** (who reviewed developer/agent list, when, outcome) — this is a workflow event, not an enforcement event, so it doesn't exist yet in any table |
| CC6.7 (data transmission/removal) | Evidence of controlled data export | ⚠️ Partial — `policy_actions` can capture a BLOCK on an export-type tool, but there's no explicit `resource_type = 'data_export'` classification today | Need to classify violation types more explicitly (path / command / mcp_tool / data_export / prompt) — currently `violation_type` is a free-form field, not a controlled vocabulary |
| CC7.1/CC7.2 (monitoring, incident response) | Evidence of monitoring and incident handling | ⚠️ Partial — `policy_actions` gives raw detections, but there's no **incident lifecycle** (detected → triaged → contained → resolved) | Need incident records that reference one or more `policy_actions` and track a resolution workflow (this is distinct from the simpler `resolution_status` field which is per-violation, not per-incident) |

### 1.4 New audit action types needed (design only)

To close the Art.14/CC6.6/CC7.x gaps, the `AuditAction` enum (currently 28 values in `models.go`) needs additions, mirroring how AEGIS's `admin_audit_log` has explicit `approval.approve`/`approval.reject`/`killswitch.*` actions that the current repo's audit model doesn't yet have an equivalent for:

```
AuditViolationReviewed      -- human reviewed a BLOCK/ALERT (Art.14, CC7.2)
AuditViolationOverridden    -- human overrode an automated decision (Art.14)
AuditAccessReviewCompleted  -- periodic access review performed (CC6.6)
AuditIncidentOpened         -- incident lifecycle started from violation(s) (CC7.1)
AuditIncidentResolved       -- incident closed with remediation notes (CC7.1)
AuditExceptionGranted       -- compliance exception/waiver approved (cross-cutting)
AuditExceptionExpired       -- exception lapsed, control reverts to prior status
```

These are **audit events, not new tables** — they slot directly into the existing `audit_events` table using the existing `actor_id`, `resource_type`, `resource_id`, `old_value`/`new_value` JSONB columns. This is the same principle AEGIS uses: don't build new logging infrastructure, extend the vocabulary of the existing one.

---

## 2. Design Required for Compliance Framework to Be Added

### 2.1 Core design decision: read-only interpretation layer

Following AEGIS exactly: the compliance layer must be **additive and read-only** with respect to the existing governance tables. It reads `audit_events`, `activity_events`, and `policy_actions`; it never writes to them. This preserves:

- The existing fail-open enforcement path (compliance calculation can never block or slow down policy enforcement)
- The immutability guarantees already built into `audit_events` and `policy_versions`
- The ability to add/remove frameworks without touching the enforcement code path at all

### 2.2 Three-tier structure (mirrors AEGIS)

```
Tier 1 — Framework Registry (static metadata)
   compliance_frameworks, framework_controls
   "What does SOC2 CC6.1 require as evidence?"

Tier 2 — Policy-to-Control Mapping (operator-curated)
   policy_control_mappings
   "Which of our existing policies satisfy that control?"

Tier 3 — Evidence Calculator (derived, computed on read or on schedule)
   compliance_evidence, compliance_status_snapshots
   "Given our actual audit/activity/policy_action rows, is CC6.1 covered?"
```

This mirrors AEGIS's `compliance-controls.ts` (Tier 1) → operator-defined mapping (Tier 2, implicit in AEGIS via `evidenceSpec.auditActions` naming actual action strings) → `compliance-bundle.ts` classify() (Tier 3).

### 2.3 Framework definition format

Like AEGIS's `custom-compliance.ts`, frameworks should be **data, not code** — a JSON/YAML definition loaded at startup or via admin API, so EU AI Act and SOC2 ship as seed data, and future frameworks (ISO27001, NIST AI RMF, internal frameworks) don't require new Go code:

```yaml
framework: eu-ai-act
version: "2024-1689"
controls:
  - code: Art.12
    title: Record-keeping
    evidence:
      audit_actions: [policy.create, policy.update, policy.activate]
      activity_kinds: [tool.pre, tool.post, prompt.submit]
      artifacts: [audit-row-count, log-integrity-hash]
  - code: Art.14
    title: Human oversight
    evidence:
      audit_actions: [violation.reviewed, violation.overridden, killswitch.revoke]
      artifacts: [pending-review-count]
```

This is the single most important design choice to carry over from AEGIS: **evidence requirements are declared as references to existing event vocabulary, not as new bespoke queries per control.** Adding a control means naming which existing (or newly-added, per §1.4) audit actions/activity kinds count as proof — never writing new extraction code per control.

### 2.4 EU AI Act specialization (like `eu-ai-act-evidence.ts`)

EU AI Act Art.12–15 evidence isn't uniform "count of actions" like most SOC2 controls — it needs structured, human-readable evidence packs. Design a parallel, framework-specific evidence assembler (not a generic one) exactly as AEGIS separates `eu-ai-act-evidence.ts` from the generic `compliance-bundle.ts`:

- **Art.12** → audit row counts + activity event counts + (new) log integrity artifact
- **Art.13** → snapshot of registered agents (`ai_coding_agents`), declared MCP tool scope (`mcp_servers`), active policy definitions — exported as structured JSON, not just a count
- **Art.14** → counts of `AuditViolationReviewed`/`AuditViolationOverridden`/kill-switch-equivalent events, plus `policy_actions.resolution_status` breakdown
- **Art.15** → detector/policy chain composition, violation tally, integrity verification result, incident/rollback events

Each article gets a `compliant: boolean` + `gaps: string[]` — directly reusing AEGIS's `ArticleEvidence` shape, since it's a clean, auditor-legible pattern.

### 2.5 Status classification logic (per control)

```
COMPLIANT           evidence present for all required event types
                     AND no unresolved violations older than N days
                     AND no expired exceptions

PARTIALLY_COMPLIANT  evidence present for some but not all required types
                     OR unresolved violations within remediation window

NON_COMPLIANT        required evidence missing entirely
                     OR unresolved violations past remediation window

UNKNOWN              no policies mapped to this control yet
```

Exceptions (§2.6) can force a control to COMPLIANT despite missing evidence, with a full audit trail — this is a deliberate design carryover, not present verbatim in AEGIS but consistent with how any real audit program handles accepted risk.

### 2.6 Exceptions / waivers

Not present in AEGIS, but necessary for a real-world rollout: a control that's legitimately not applicable (e.g., no MCP servers in use yet) shouldn't show as perpetually NON_COMPLIANT. Design a lightweight `control_exceptions` concept: reason, approver, expiry. On expiry, status reverts automatically and an `AuditExceptionExpired` event fires — keeping the audit trail honest.

---

## 3. Backend & DB Changes Required

### 3.1 New tables (design, not DDL-final)

```
compliance_frameworks
  framework_id, framework_type (enum: eu_ai_act, soc2, custom),
  display_name, version, source (built_in | custom), status

framework_controls
  control_id, framework_id (FK), control_code, title, description,
  evidence_spec (JSONB: {audit_actions[], activity_kinds[], artifacts[]}),
  current_status (enum), status_updated_at

policy_control_mappings
  policy_id (FK -> policies), control_id (FK), mapped_by, mapped_at, rationale

compliance_evidence
  evidence_id, control_id (FK),
  source_audit_event_id (FK -> audit_events, nullable),
  source_activity_event_id (FK -> activity_events, nullable),
  source_policy_action_id (FK -> policy_actions, nullable),
  evidence_timestamp, evidence_status (valid | disputed | expired)

compliance_status_snapshots
  snapshot_id, framework_id (FK), compliance_percentage,
  compliant_count, partial_count, non_compliant_count, calculated_at

control_exceptions
  exception_id, control_id (FK), reason, approved_by, expires_at, status

evidence_bundles / bundle_signatures
  (as in the general architecture doc — Ed25519 + SHA256, mirrors AEGIS's
   compliance-bundle.ts signing + optional transparency log anchoring)
```

**Design note:** `compliance_evidence` deliberately stores **references** (FKs) to existing rows rather than copying their data — same principle as AEGIS, which never duplicates `admin_audit_log` rows into a separate compliance store. This keeps the evidence trail provably tied to the original immutable record.

### 3.2 Backend service layer additions

```
internal/services/
  compliance_service.go     -- status calculation, evidence linking
  eu_ai_act_service.go       -- Art.12–15 specialized evidence assembly
  bundle_service.go          -- signed export generation (Ed25519 + SHA256)
  exception_service.go       -- waiver lifecycle
```

- `compliance_service.go` is purely **read-side**: it queries `audit_events`/`activity_events`/`policy_actions` filtered by the `evidence_spec` on each control, and writes only to `compliance_evidence`/`compliance_status_snapshots` — never touches enforcement tables.
- Calculation should run as a **scheduled job** (hourly, matching AEGIS's on-demand bundle generation model) rather than inline in the ingest path, so compliance status can never add latency to policy enforcement — this preserves the current system's fail-open, low-latency ingest design.

### 3.3 New audit action integration point

The one place enforcement code *does* need to change: wherever a human currently marks `policy_actions.resolution_status = 'resolved'` in the dashboard, that action should also emit one of the new audit events from §1.4 (`AuditViolationReviewed`/`AuditViolationOverridden`). This is a small, additive change to the existing violation-resolution API handler — not a new subsystem.

### 3.4 API surface (design)

```
GET  /api/v1/compliance/frameworks
GET  /api/v1/compliance/frameworks/:id/controls
GET  /api/v1/compliance/frameworks/:id/status
GET  /api/v1/compliance/eu-ai-act/evidence        -- Art.12-15 structured pack
GET  /api/v1/compliance/controls/:id/evidence
POST /api/v1/compliance/bundles                   -- signed export
POST /api/v1/compliance/controls/:id/exceptions
```

Directly parallel to AEGIS's `/api/v1/compliance/*` route set (`frameworks`, `controls/:fw`, `bundle/:fw`).

---

## 4. How Compliance Is Calculated — Data Identification

This is the crux: **which existing rows prove which control**, stated explicitly rather than left implicit (this is the single biggest gap between "we have logs" and "we have compliance").

### 4.1 EU AI Act — data identification

| Article | Data source | Query shape |
|---|---|---|
| Art.12 | `audit_events` (all rows), `activity_events` (all rows) | Row counts within reporting window; presence of continuous timestamps (gap detection) |
| Art.13 | `ai_coding_agents`, `mcp_servers`, `policies` (active versions) | Snapshot export — not a count, a document: "these agents, these tools, these policies were declared active as of date X" |
| Art.14 | `policy_actions` where `resolution_status`, new `AuditViolationReviewed`/`AuditViolationOverridden` events | Count of BLOCK/ALERT actions that received a human review event within N hours; ratio reviewed/total |
| Art.15 | `policy_actions` (violation tally by severity), detector/policy registry, new integrity-check artifact | Violation trend, plus a computed "last integrity check passed" boolean |

### 4.2 SOC2 — data identification

| Control | Data source | Query shape |
|---|---|---|
| CC6.1 | `audit_events` where `event_type IN (developer.added, developer.disabled, agent.registered, agent.disabled)` | Count > 0 within window = provisioning process evidenced |
| CC6.6 | New `AuditAccessReviewCompleted` events | Must exist at least once per defined review cadence (e.g., quarterly) — absence = NON_COMPLIANT regardless of other evidence |
| CC6.7 | `policy_actions` where `violation_type = 'data_export'` (requires the controlled-vocabulary change from §1.3) | BLOCK count > 0 with no unresolved exports = control working as intended |
| CC7.1 | `policy_actions` (all), new incident records | Detection coverage: are there policies mapped, are violations non-zero-but-trending-down |
| CC7.2 | New `AuditIncidentOpened`/`AuditIncidentResolved` pairs | Mean time-to-resolution; unresolved incident count |

### 4.3 The identification principle

Every control's evidence must resolve to a **deterministic, re-runnable query** against existing tables plus the small set of new audit action types. If a control's evidence can't be expressed this way, it's a signal that either (a) a new audit action type is needed (§1.4), or (b) the control genuinely requires a manual attestation (handled via the exception mechanism, §2.6) rather than automated evidence — this distinction should be made explicit per-control in the framework definition, not discovered ad hoc.

---

## 5. Additional Design Changes (Cross-Cutting)

These aren't part of the four sections above but are necessary for the design to hold together in practice:

1. **Controlled vocabulary for `violation_type`** — currently a free-form field on `policy_actions`. Needs to become a fixed enum (`path`, `command`, `mcp_tool`, `data_export`, `prompt`) so SOC2/EU AI Act evidence queries can filter reliably. Without this, Art.15/CC6.7 evidence extraction is unreliable.

2. **Log integrity artifact** — Art.12 implicitly requires tamper-evidence, which AEGIS provides via a Merkle transparency log. The current repo has no equivalent. Design options range from a lightweight periodic hash-chain-of-audit-rows job to a full transparency log; either way this is a genuine new capability, not a mapping exercise, and should be scoped as its own design decision rather than bundled silently into "compliance."

3. **Dashboard additions** — a `/compliance` section mirroring the existing page structure (`/overview`, `/policies`, `/violations` pattern), showing framework cards, per-control status, and an evidence-bundle export action — consistent with the existing UI's panel-based navigation model, not a new UI paradigm.

4. **Retention alignment** — `audit_events`/`activity_events` retention policy must be checked against the compliance frameworks' evidence retention requirements (SOC2 audits typically want 1 year+, EU AI Act record-keeping requirements are longer). If current retention/archival is shorter than what a control's `evidenceSpec` implies, that's a policy gap to flag before framework rollout, not after.

5. **Ownership boundary** — compliance framework definitions (Tier 1) should be reviewed/approved by whoever owns compliance/legal, not hardcoded by engineering — this is a process design point, not a technical one, but it determines whether the JSON/YAML framework format (§2.3) needs an admin UI for editing or stays a seed file initially.

---

## Summary

| Question | Answer |
|---|---|
| Does the current repo have the raw telemetry AEGIS relies on? | Mostly yes — `audit_events` + `activity_events` + `policy_actions` cover the bulk of it |
| What's structurally missing? | Human-oversight events (Art.14/CC7.2), access-review events (CC6.6), controlled vocabulary for violation types, and any log-integrity/tamper-evidence mechanism |
| What's the right architecture? | A read-only interpretation layer (frameworks → controls → evidence spec → calculated status), never touching enforcement code, exactly as AEGIS built it |
| What's genuinely new vs. just "mapping"? | New: audit action vocabulary additions, log integrity, exception workflow. Mapping-only: everything else — frameworks, controls, evidence calculation, bundle export |

**Recommendation:** Treat §1.4 (new audit actions) and §5.1 (controlled vocabulary) as prerequisites — they're small, additive schema/enum changes that unblock accurate evidence identification for *both* EU AI Act and SOC2. Everything else (frameworks, controls, calculation, bundles, UI) can be built once those two prerequisites land, following AEGIS's layering exactly.
