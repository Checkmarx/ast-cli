# OTel-Based Monitoring for Claude Code & Cowork — Architecture
**Role:** Senior Architect, Checkmarx
**Reference:** [Claude Code monitoring](https://code.claude.com/docs/en/monitoring-usage) · [Claude Cowork monitoring](https://claude.com/docs/cowork/monitoring) · `COMPLIANCE_ARCHITECTURE.md`
**Scope:** Add an OpenTelemetry (OTel) ingestion path alongside the existing `cx` CLI hook pipeline, so governance has a monitoring channel that (a) survives a developer disabling/bypassing hooks and (b) covers Claude Cowork, which has no hook mechanism at all. Structured to match `COMPLIANCE_ARCHITECTURE.md`'s conventions, since every piece of evidence this design produces feeds directly into that document's control-catalog model.

---

## 1. Executive Summary

**The one thing this design proves:** monitoring evidence can exist independent of whether the observed party's hook configuration is intact — closing Cowork's total blind spot in the same stroke — without weakening the hook path at all. It's a second, corroborating channel Anthropic already ships, not a replacement for the first.

**Reused (plumbing only, no changes):**
- `ingest.Processor.ProcessBatch` — every session/developer/agent/MCP-server resolution function it already contains
- `ai_sessions` upsert idempotency (`ON CONFLICT (claude_session_id, agent_id)`) and `activity_events` insert idempotency (`ON CONFLICT (event_id)`)
- The control catalog / policy-to-control auto-mapping engine (`COMPLIANCE_ARCHITECTURE.md` §5.7) — new controls are added as data, not code
- The `middleware.BearerAuth` pattern — mirrored for a new, separate credential space, not modified

**Built new, for this design:**
- `internal/otelingest` package (`receiver.go`, `translate.go`, `identity.go`, `auth.go`)
- `POST /api/v1/otel/v1/logs` — governance-api's own OTLP/HTTP receiver
- `otel_ingest_credentials`, `usage_daily_rollups` tables; `event_source`/`tool_use_id`/cost columns on `activity_events`; coverage counters on `ai_sessions`
- A session-level coverage badge (Hook+OTel / Hook-only / OTel-only) and a `tool_use_id`-grouped timeline query
- `CTRL-TELEMETRY-COVERAGE` / `CTRL-COWORK-OVERSIGHT` control-catalog rows

**Explicitly not built for this design** (see §14 for why and when): OTel-based enforcement, trace/span ingestion, exact third-party-hook disambiguation, metrics-exporter ingestion.

---

## 2. Domain Constraint That Shapes This Design

**OTel evidence is asynchronous and best-effort; it can corroborate an enforcement decision after the fact, but it can never be the decision.**

| Hook pipeline | OTel monitoring channel |
|---|---|
| Runs synchronously, inside the tool-call path — a BLOCK stops the action before it happens | Runs asynchronously, always after the real API call already returned — nothing here can stop or change what happened |
| The single source of truth for `policy_actions` / violations | Never writes `policy_actions`. Corroborates that a decision occurred; never records a second one |
| Developer-editable — fails open if disabled or absent | Admin-enforced for Claude Code (`managed-settings.json` strips developer overrides); admin-*only* for Cowork — no developer override surface exists at all |

**Why this matters for this design specifically:** every other design decision below — the explicit `EventSource=="otel"` guard before `policy_actions` insertion, the deterministic `event_id`, the `tool_use_id` correlation key — exists to enforce this one asymmetry. If OTel evidence were ever allowed to independently create or resolve a violation, it would silently promote a best-effort, admin-configurable telemetry stream to the same authority as the synchronous enforcement path, which is exactly the property that makes hook data trustworthy in the first place.

---

## 3. Current State & Gaps

### 3.1 What's Reused, Unaffected

| Component | Location | Why it's the right substrate |
|---|---|---|
| `ingest.Processor.ProcessBatch` | `governance-api/internal/ingest/ingest.go` | Already resolves developer/agent/session/MCP-server identity from an `ActivityEvent` — OTel needs the same resolution, not a parallel one |
| `activity_events` / `ai_sessions` upsert idempotency | `ingest.go` (`ON CONFLICT` clauses) | Already order-independent and retry-safe; OTel only needs a deterministic `event_id` to inherit this for free (§8) |
| Control catalog + auto-mapping engine | `COMPLIANCE_ARCHITECTURE.md` §5.7 | New controls are onboarded as data rows, never rule-engine code (§9) |
| `middleware.BearerAuth` | `pkg/middleware/auth.go` | Pattern reused for a new, deliberately separate credential space (§5.4) — not modified, not shared |

### 3.2 Gaps This Design Closes

| # | Gap | Why it matters | Fix |
|---|---|---|---|
| G1 | **Hooks are developer-editable and fail open.** A developer can edit `.claude/settings.json`, uninstall `cx`, or the hook process can crash — every decision path in `hooks.go` fails open (`if pack == nil { return claude.ApproveToolUse() }`). | Weak evidence for tamper-resistant record-keeping (EU AI Act Art.12/26, SOC2 CC7.1) — a log source the subject can turn off doesn't prove monitoring was continuous. | Second channel enforced via `managed-settings.json`, which is **not** developer-editable and actively strips conflicting developer-set OTLP vars at startup. |
| G2 | **Claude Cowork has no hook mechanism at all.** It runs in a hosted VM; there is no local CLI, no `.claude/settings.json`, nothing for `cx` to attach to. | Zero visibility today — not a weaker control, a missing product entirely. | OTLP export is Cowork's *only* monitoring surface, period. Wire governance-api into it; no alternative exists. |
| G3 | **No cost/token/model-certainty telemetry.** Hooks only ever observed tool-call decisions — never `cost_usd`, token counts, or a guaranteed-accurate model identifier. | No cost governance possible today; no way to answer "what did this team's AI usage cost." | `api_request` OTel events carry `cost_usd`/token counts per request — new evidence category. |

### 3.3 What Anthropic Actually Exposes

Both products ship the same underlying mechanism — OTLP export — with one decisive difference in who controls it:

- **Claude Code**: `CLAUDE_CODE_ENABLE_TELEMETRY=1` + `OTEL_LOGS_EXPORTER=otlp` + `OTEL_EXPORTER_OTLP_ENDPOINT` (env vars or `.claude/settings.json`). Individually overridable by a developer **unless** set via a managed settings file, in which case Claude Code strips conflicting developer-set OTLP vars at startup.
- **Cowork**: identical OTLP logs stream, but configured **only** at `Admin settings > Cowork` — org-wide, no per-user override surface exists at all. Requires a Team or Enterprise plan.
- Both emit near-identical event names — `user_prompt`, `assistant_response`, `tool_result`, `tool_decision`, `api_request`, `api_error` — correlated by `prompt.id` and `session.id`, carrying `user.email` / `user.account_uuid` / `organization.id`.

**Load-bearing fact for the whole design:** the OTel `session.id` attribute *is* the same Claude Code session UUID the hook pipeline already writes to `ai_sessions.claude_session_id`. OTel-sourced events join onto the existing session graph with no new identity model — a second event source into the same tables, not a parallel system.

---

## 4. Architecture Overview

```
Developer machine (Claude Code)                    Cowork VM (org-wide)
┌─────────────────────────┐                         ┌──────────────────────────┐
│ cx CLI hooks → spool     │  (existing, unchanged)  │ OTel exporter             │
│                          │                         │ (org admin OTLP config)  │
│ OTel exporter            │                         └────────────┬─────────────┘
│ (managed-settings-forced)│                                      │
└───────────┬──────────────┘                                      │
            │ OTLP/HTTP logs                                       │ OTLP/HTTP logs
            ▼                                                     ▼
        ┌───────────────────────────────────────────────────────────┐
        │  OTel Collector (new component; illustrative, §8)          │
        │  otlp receiver → batch/retry processor → dual export:      │
        │    1. otlphttp exporter → governance-api /api/v1/otel/*    │
        │    2. (optional) fan-out to org SIEM/Datadog/Honeycomb     │
        └───────────────────────────┬─────────────────────────────┘
                                    │ HTTPS, dedicated bearer token
                                    ▼
                    POST /api/v1/otel/v1/logs  (governance-api, new)
                                    │
                                    ▼
                translate.go: OTLP LogRecord → models.ActivityEvent
                                    │
                                    ▼
                    ingest.Processor.ProcessBatch (existing, reused)
                                    │
                                    ▼
        activity_events (event_source='otel') / ai_sessions / policy_actions (unreachable, §8)
```

**Cardinal rule:** OTel never sits on the enforcement path, and never independently produces enforcement evidence. It is fire-and-forget from Claude Code's/Cowork's perspective and read-only, corroboration-only from governance-api's — every write this channel makes lands in `activity_events`, never in `policy_actions`.

**Key decision:** Collector-in-front, terminating at a governance-api-owned HTTP endpoint (not a message queue, not a self-hosted general Collector governance-api never touches). The Collector absorbs retry/backoff/buffering and can fan the same stream to an org's existing SIEM without governance-api becoming a general observability sink.

**Corollary:** because governance-api implements the actual OTLP/HTTP receiver contract (path suffix `/v1/logs`, standard content types), the Collector hop is *operationally recommended, not structurally required* — Claude Code or Cowork could point `OTEL_EXPORTER_OTLP_ENDPOINT` straight at governance-api and it would work.

---

## 5. Domain Design

### 5.1 Package Structure

```
governance-api/internal/otelingest/       ← new package, 4 files
├── receiver.go     ← HTTP handlers for /api/v1/otel/v1/logs (and /metrics, phase 2)
├── translate.go     ← OTLP LogRecord → models.ActivityEvent
├── identity.go       ← email/account_uuid-aware developer resolution + backfill
└── auth.go            ← bearer-token validation against otel_ingest_credentials
```

### 5.2 Receiving OTLP (`receiver.go`)

Implements the OTLP/HTTP contract directly, so the Collector's stock `otlphttp` exporter — or Claude Code/Cowork pointed straight at governance-api — both work unmodified:

```go
// POST /api/v1/otel/v1/logs
// Accepts an OTLP ExportLogsServiceRequest (application/json, OTLP JSON mapping).
func IngestOTelLogs(w http.ResponseWriter, r *http.Request) {
    cred, ok := otelingest.AuthenticatedCredential(r)
    if !ok {
        http.Error(w, `{"error":"invalid otel ingest credential"}`, http.StatusUnauthorized)
        return
    }

    var req plogotlp.ExportRequest
    if err := (&plog.JSONUnmarshaler{}).UnmarshalLogs(r.Body); err != nil { ... }

    events := translate.LogsToActivityEvents(req.Logs())   // → []models.ActivityEvent
    if err := ingest.New().ProcessBatch(r.Context(), events); err != nil {
        log.Printf("otelingest: batch process error: %v", err)
    }
    log.Printf("otelingest: ingested %d log records via credential %q", len(events), cred.Label)
    w.WriteHeader(http.StatusOK)
    w.Write(plogotlp.NewExportResponse().MarshalJSON()) // empty success body per OTLP spec
}
```

Dependency: `go.opentelemetry.io/collector/pdata` (`plog`, `plogotlp`) — the standard library for parsing OTLP payloads without hand-rolling protobuf/JSON mapping.

### 5.3 Translating OTLP into `ActivityEvent` (`translate.go`)

One `LogRecord` → one `models.ActivityEvent`, reusing every downstream resolution function in `ingest.go` unchanged:

```go
func LogsToActivityEvents(logs plog.Logs) []models.ActivityEvent {
    var out []models.ActivityEvent
    for i := 0; i < logs.ResourceLogs().Len(); i++ {
        rl := logs.ResourceLogs().At(i)
        agentType := resolveAgentType(rl.Resource().Attributes()) // "service.name"=="cowork" → "cowork"; else "claude-code"
        for j := 0; j < rl.ScopeLogs().Len(); j++ {
            for k := 0; k < rl.ScopeLogs().At(j).LogRecords().Len(); k++ {
                rec := rl.ScopeLogs().At(j).LogRecords().At(k)
                if ev, ok := translateOne(rec, agentType); ok {
                    out = append(out, ev)
                }
            }
        }
    }
    return out
}
```

`translateOne` switches on the OTel event name (the log record's `event.name` attribute) and populates `models.ActivityEvent`:

| OTel event name | `ActivityEvent.Kind` | Notes |
|---|---|---|
| `user_prompt` | `otel.user_prompt` | `prompt.PromptText` populated **only** if `OTEL_LOG_USER_PROMPTS` was enabled upstream — never force-decoded otherwise |
| `assistant_response` | `otel.assistant_response` | maps `model` → `OtelModel` |
| `tool_result` | `otel.tool_result` | maps `tool_use_id` → `ToolUseID`, `decision_source`, `success`, `duration_ms` |
| `tool_decision` | `otel.tool_decision` | maps `tool_use_id` → `ToolUseID`, `decision`/`source` → `DecisionSource` — **never** into `Decision.Action` (§8) |
| `api_request` | `otel.api_request` | maps `cost_usd`, `input_tokens`, `output_tokens`, `cache_read_tokens`, `cache_creation_tokens` |
| `api_error` | `otel.api_error` | maps `error`, `status_code` |
| `auth`, `permission_mode_changed`, `mcp_server_connection` | `otel.<name>` | passthrough into `raw` JSONB for now — no dedicated columns needed yet |

`ActivityEvent` gains the fields this requires (`models.go`):

```go
type ActivityEvent struct {
    // ...existing fields unchanged...
    EventSource    string    `json:"eventSource"`              // "hook" | "otel" — defaults to "hook" for existing callers
    DecisionSource string    `json:"decisionSource,omitempty"`
    Cost           *CostInfo `json:"cost,omitempty"`
    AccountUUID    string    `json:"accountUuid,omitempty"`
    Email          string    `json:"email,omitempty"`
    TerminalType   string    `json:"terminalType,omitempty"`
    OtelModel      string    `json:"otelModel,omitempty"`
}

type CostInfo struct {
    CostUSD             float64 `json:"costUsd"`
    InputTokens         int     `json:"inputTokens"`
    OutputTokens        int     `json:"outputTokens"`
    CacheReadTokens     int     `json:"cacheReadTokens"`
    CacheCreationTokens int     `json:"cacheCreationTokens"`
}
```

### 5.4 Identity Reconciliation (`identity.go`)

The one genuinely new piece of logic, because OTel gives better identity than the hook path ever had:

```go
// ResolveOrCreateDeveloperByIdentity extends ingest.resolveOrCreateDeveloper with
// email/account_uuid awareness. Hook-created developers only ever have a synthetic
// "<username>@unknown.local" email — when a real identity arrives via OTel, we backfill
// that row in place rather than creating a duplicate developer, which would fragment
// session history across two developer_ids for the same person.
func ResolveOrCreateDeveloperByIdentity(ctx context.Context, username, email, accountUUID string) (uuid.UUID, error) {
    if accountUUID != "" {
        if id, err := lookupByAccountUUID(ctx, accountUUID); err == nil {
            return id, nil
        }
    }
    if email != "" && !strings.HasSuffix(email, "@unknown.local") {
        if id, ok := backfillPlaceholderOrMatch(ctx, username, email, accountUUID); ok {
            return id, nil
        }
    }
    return resolveOrCreateDeveloper(ctx, username) // existing ingest.go fallback, unchanged
}
```

This is called from the one change to existing ingest code the whole design requires (§6.2).

### 5.5 Auth (`auth.go` + `pkg/middleware`)

New middleware, deliberately separate from `middleware.BearerAuth` (JWT-for-dashboard, or the single shared `API_KEY` for cx CLI):

```go
// OTelBearerAuth validates against otel_ingest_credentials by hashed lookup.
// Deliberately not the same credential space as API_KEY: this token is
// distributed into every managed-settings.json and the Cowork admin panel,
// so it needs independent rotation without a cx CLI fleet-wide outage.
func OTelBearerAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := extractBearer(r)
        hash := sha256Hex(token)
        cred, err := db.LookupOTelCredential(r.Context(), hash) // WHERE token_hash=$1 AND revoked_at IS NULL
        if err != nil {
            http.Error(w, `{"error":"invalid otel ingest credential"}`, http.StatusUnauthorized)
            return
        }
        ctx := context.WithValue(r.Context(), ctxKeyOTelCredential, cred)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

Rate limiting: a per-credential token bucket (`golang.org/x/time/rate`) in front of `/api/v1/otel/*` only — Cowork is org-wide and can burst; the existing hook ingest endpoint doesn't need this because it's already bounded by one request per developer machine's spool flush interval.

### 5.6 Routes (`router.go`)

```go
r.Route("/api/v1/otel", func(r chi.Router) {
    r.Use(middleware.OTelBearerAuth)   // distinct from the outer BearerAuth group
    r.Post("/v1/logs", IngestOTelLogs)
    // r.Post("/v1/metrics", IngestOTelMetrics) — Phase 2, see §14
})

// admin-facing, inside the existing BearerAuth group
r.Get("/otel/credentials", ListOTelCredentials)
r.Post("/otel/credentials", CreateOTelCredential)      // returns plaintext token once, stores only the hash
r.Delete("/otel/credentials/{id}", RevokeOTelCredential)

// new dashboard analytics, tying directly back into the compliance narrative
r.Get("/analytics/telemetry-coverage", GetTelemetryCoverage) // hook vs otel session counts per developer/agent
r.Get("/analytics/cost-usage", GetCostUsage)                   // usage_daily_rollups reads
```

---

## 6. Database Design

Migration `012_otel_monitoring.sql`, continuing from `011_model_detection.sql` (latest applied).

### 6.1 Schema

```sql
-- Distinguish which channel produced a row, without touching existing rows' meaning.
ALTER TABLE activity_events
    ADD COLUMN event_source      TEXT NOT NULL DEFAULT 'hook'
        CHECK (event_source IN ('hook', 'otel')),
    ADD COLUMN otel_event_name   TEXT,              -- 'user_prompt' | 'tool_result' | 'api_request' | ...
    ADD COLUMN decision_source   TEXT,               -- OTel tool_decision.source: config|hook|user_permanent|user_temporary|user_abort|user_reject
    ADD COLUMN cost_usd          NUMERIC(12,6),
    ADD COLUMN input_tokens      INTEGER,
    ADD COLUMN output_tokens     INTEGER,
    ADD COLUMN cache_read_tokens INTEGER,
    ADD COLUMN cache_creation_tokens INTEGER,
    ADD COLUMN otel_model        TEXT,               -- ground-truth model name from OTel, distinct from the
                                                       -- heuristic model_detected/model_confidence columns (011)
    ADD COLUMN tool_use_id       TEXT;               -- shared correlation key: hook's PreToolUse ToolUseID ==
                                                       -- OTel tool_decision/tool_result's tool_use_id. Lets a
                                                       -- coverage query PROVE a hook row and an OTel row describe
                                                       -- the same real action (corroboration), instead of the
                                                       -- dashboard showing what looks like two unrelated events.

CREATE INDEX idx_activity_events_event_source ON activity_events (event_source);
CREATE INDEX idx_activity_events_otel_event_name ON activity_events (otel_event_name) WHERE event_source = 'otel';
CREATE INDEX idx_activity_events_tool_use_id ON activity_events (tool_use_id) WHERE tool_use_id IS NOT NULL;

-- Per-session coverage counters — the queryable form of "did hooks or OTel see this session."
ALTER TABLE ai_sessions
    ADD COLUMN hook_event_count  INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN otel_event_count  INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN total_cost_usd    NUMERIC(12,6) NOT NULL DEFAULT 0,
    ADD COLUMN terminal_type     TEXT;               -- OTel terminal.type, e.g. 'non-interactive' for Cowork

-- Identity reconciliation: OTel gives a real email/account UUID; hooks only ever had an OS username.
ALTER TABLE developers
    ADD COLUMN account_uuid TEXT UNIQUE;

-- Daily cost/token rollup — avoids aggregating raw activity_events for dashboard reads at scale.
-- Populated by the existing scheduler tick (cmd/server/main.go), same non-blocking pattern as
-- the compliance snapshot cycle — never written inline on ingest.
CREATE TABLE usage_daily_rollups (
    rollup_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usage_date      DATE NOT NULL,
    developer_id    UUID NOT NULL REFERENCES developers(developer_id),
    agent_id        UUID NOT NULL REFERENCES ai_coding_agents(agent_id),
    model_name      TEXT,
    total_cost_usd  NUMERIC(12,6) NOT NULL DEFAULT 0,
    input_tokens    BIGINT NOT NULL DEFAULT 0,
    output_tokens   BIGINT NOT NULL DEFAULT 0,
    request_count   INTEGER NOT NULL DEFAULT 0,
    calculated_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (usage_date, developer_id, agent_id, model_name)
);

-- Dedicated ingest credentials for the OTel path — NOT the shared cx-CLI API_KEY.
-- This token is distributed into every developer's managed-settings.json and into the
-- Cowork admin panel; it needs independent rotation without touching cx CLI ingestion,
-- and a compromised copy has a different blast radius than the CLI-to-server API_KEY.
CREATE TABLE otel_ingest_credentials (
    credential_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    label         TEXT NOT NULL,              -- e.g. "claude-code-managed-settings", "cowork-admin-otlp"
    token_hash    TEXT NOT NULL UNIQUE,       -- SHA-256 of the bearer token, never store plaintext
    created_by    UUID REFERENCES developers(developer_id),
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked_at    TIMESTAMP WITH TIME ZONE
);

-- Seed Cowork as a first-class agent product — same catalog table, no schema change needed
-- beyond this row (COMPLIANCE_ARCHITECTURE.md §6.5 "Case A": onboarding via data, not code).
INSERT INTO ai_coding_agents (agent_name, vendor, agent_type)
VALUES ('Claude Cowork', 'Anthropic', 'cowork')
ON CONFLICT (agent_name, vendor, version) DO NOTHING;
```

### 6.2 Change to Existing `ingest.go` (Minimal, Additive)

```diff
- devID, err := resolveOrCreateDeveloper(ctx, ev.UserID)
+ devID, err := otelingest.ResolveOrCreateDeveloperByIdentity(ctx, ev.UserID, ev.Email, ev.AccountUUID)
```

```diff
  if err := insertActivityEvent(ctx, sessionID, ev); err != nil { ... }
+ incrementSessionEventCounter(ctx, sessionID, ev.EventSource, ev.Cost)
+
+ // OTel is a monitoring channel only — it never independently enforces anything, so it must
+ // never independently RECORD an enforcement outcome either. Enforcement (and every count that
+ // feeds compliance evidence, §9) stays single-sourced from the hook path. Without this guard,
+ // an OTel tool_decision "reject" for the exact same tool call a hook already BLOCKed would
+ // insert a second policy_action row for one real violation — silently doubling every
+ // violation count on the dashboard and in the compliance snapshot.
+ if ev.EventSource == "otel" {
+     return nil
+ }
```

`resolveAgentProduct` gains one new mapping entry (`"cowork" → "Claude Cowork"`) in `agentTypeToCatalogName` — same function, one more `case`.

Everything else in `ingest.go` — `upsertSession`, `updateSessionCounters`, policy_action insertion — is untouched, and **unreachable for OTel-sourced events** per the guard above.

**Consequence, stated plainly:** `translate.go` must never populate `Decision.Action`/`Decision.PolicyID` from OTel's `tool_decision.decision` — even though the guard above would catch it either way, `Decision` stays empty for OTel events by construction, not by relying on the guard as the only line of defense. `decision`/`source` land exclusively in the new `decision_source` column as corroborating evidence, joinable to the hook's own record of the same action via `tool_use_id` — proof the two sources agree, never a second, independently-counted violation.

---

## 7. Backend API

```
POST /api/v1/otel/v1/logs                      → governance-api's own OTLP/HTTP log receiver (§5.2, new)
r.Post("/v1/metrics", ...)                      → Phase 2, not built yet (§14)

GET    /api/v1/otel/credentials                 → list ingest credentials (labels + creation metadata only)
POST   /api/v1/otel/credentials                 → create; returns plaintext token once, stores only the hash
DELETE /api/v1/otel/credentials/{id}            → revoke

GET  /api/v1/analytics/telemetry-coverage       → hook vs OTel session counts per developer/agent (§9)
GET  /api/v1/analytics/cost-usage               → usage_daily_rollups reads, by developer/team/model
```

`GET /api/v1/sessions/{id}/timeline` (existing, `dashboard.go`) is *revised*, not replaced — see §10.

---

## 8. Ingestion, Deduplication & Rollout Strategy

Same discipline `COMPLIANCE_ARCHITECTURE.md` §8 applies to compliance snapshots — never trust a single insertion path to stay correct without an explicit, checkable rule — applied here to a channel with retries, redelivery, and two independent writers to the same session.

### 8.1 The Three Places "Same `session.id`" Could Double-Count

| Layer | Risk | Answer |
|---|---|---|
| **`activity_events` rows** | Hook's `tool.pre` and OTel's `otel.tool_decision`/`otel.tool_result` for one real tool call look like two unrelated rows. | Not deduped — intentional (corroboration, §2). Made *provable* rather than confusing via the shared `tool_use_id` column: a query can join them and show "hook and OTel agree," instead of a dashboard showing two disconnected rows (§10). |
| **`policy_actions` / violation counts** | Mapping OTel's `tool_decision` into `Decision.Action` would record the same real BLOCK twice, inflating every compliance count that reads `policy_actions` (§9's raw-count model depends on these being accurate). | `translate.go` never populates `Decision.Action`/`Decision.PolicyID` for OTel events, **and** `ingest.processOne` has an explicit `if ev.EventSource == "otel" { return nil }` guard (§6.2) before the policy_action insertion branch — two independent lines of defense. Enforcement stays single-sourced from hooks, permanently. |
| **`ai_sessions` counters / cost** | Could `tool_call_count` be incremented by both a hook `tool.pre` and an OTel `tool_decision` for the same call? Could cost be double-added? | No — `tool_call_count` is only ever touched by `updateSessionCounters` on the hook's `tool.pre` kind (unchanged); OTel events instead increment the *separate* `otel_event_count`, never `tool_call_count`. Cost is populated **exclusively** from OTel's `api_request` events — hooks never had cost data, so there is no second writer to collide with. |
| **Batch/session-row idempotency** | Collector redelivering a batch after a timed-out-but-succeeded request; hook and OTel events for the same session arriving out of order. | `event_id` for OTel rows is deterministically derived from `(session.id, event.sequence, otel_event_name)` — using OTel's own documented monotonic per-session counter — not freshly generated. A retried batch produces identical `event_id`s, so the existing `ON CONFLICT (event_id) DO NOTHING` (unchanged code) dedupes it for free. Session-row creation is already order-independent via `upsertSession`'s existing `ON CONFLICT (claude_session_id, agent_id) DO UPDATE` (unchanged) — it doesn't matter whether the hook's `session.start` or the first OTel event lands first. |

```go
// Deterministic, not fresh — a Collector retry must reproduce the identical ID.
ev.EventID = uuid.NewSHA1(otelNamespace, []byte(sessionID+"|"+strconv.FormatInt(sequence, 10)+"|"+otelEventName)).String()
```

### 8.2 Rollout — Who Configures OTel, and in What Order

Claude Code and Cowork have fundamentally different rollout shapes; getting the order wrong either delays closing G1 or takes Cowork's org-wide, no-opt-out traffic on an untested pipeline.

1. **Pilot (Claude Code, per-project `.claude/settings.json`):** a small volunteer cohort, `env` vars at project scope. Developer-editable — **not** the tamper-resistant end state, used only to validate `translate.go` against real payload shapes and confirm the timeline-grouping/coverage-badge UI (§10) before wider rollout.
2. **Fleet-wide (Claude Code, `managed-settings.json`):**
   ```json
   {
     "env": {
       "CLAUDE_CODE_ENABLE_TELEMETRY": "1",
       "OTEL_LOGS_EXPORTER": "otlp",
       "OTEL_EXPORTER_OTLP_PROTOCOL": "http/json",
       "OTEL_EXPORTER_OTLP_ENDPOINT": "https://otel-collector.internal:4318",
       "OTEL_EXPORTER_OTLP_HEADERS": "Authorization=Bearer <otel_ingest_credential>",
       "OTEL_LOG_USER_PROMPTS": "0",
       "OTEL_LOG_ASSISTANT_RESPONSES": "0",
       "OTEL_LOG_TOOL_DETAILS": "1"
     }
   }
   ```
   Pushed via existing MDM/GPO, not editable by the developer — this is the actual G1 answer. `OTEL_LOG_USER_PROMPTS`/`OTEL_LOG_ASSISTANT_RESPONSES` stay **off** by design — the hook pipeline already does PII-aware prompt scanning with redaction (`pii.go`); duplicating raw prompt capture through a second channel would undermine that discipline. `OTEL_LOG_TOOL_DETAILS` stays **on**, since `tool_parameters`/`tool_input` is exactly the signal that strengthens decision-source visibility without capturing conversational content. Treat MDM/GPO coordination as calendar lead time separate from engineering time.
3. **Cowork (org admin, `Admin settings > Cowork`):** last, once the ingest endpoint has proven stable under step 2's real traffic. Same OTLP endpoint/headers, configured once — no per-developer distribution, and no pilot-cohort option exists at all; the switch is org-wide from the moment it's flipped.

---

## 9. Coverage Impact — Mapped to the Compliance Model

Direct tie-back to `COMPLIANCE_ARCHITECTURE.md`'s control-catalog model — new controls added as data only, per its own §5.2/§6.5 "Case A/B" onboarding rule, with zero rule-engine code changes:

```json
{
  "controlCode": "CTRL-TELEMETRY-COVERAGE",
  "title": "Monitoring continues independent of agent-side hook configuration",
  "evidenceSpec": {
    "activityEventKinds": ["otel.user_prompt", "otel.tool_result", "otel.api_request"],
    "oversightRequired": false
  }
}
```
Maps to SOC2 CC7.1 (strengthens, doesn't replace) and EU AI Act Art.12/Art.26 monitoring.

```json
{
  "controlCode": "CTRL-COWORK-OVERSIGHT",
  "title": "Cowork sessions are monitored and human-reviewable",
  "evidenceSpec": {
    "policyActionFilters": {"resource_type": "cowork"},
    "oversightRequired": true
  }
}
```
Starts `evidence_status: "planned"` until the first real Cowork session lands — same honesty rule `COMPLIANCE_ARCHITECTURE.md` §5.3 enforces for every other control.

**A necessary caveat, mirroring that document's §9.1:** OTel's `tool_decision.source` values (`config`/`hook`/`user_permanent`/etc.) give a definitive, latency-free count of *that* a decision was hook-mediated, without the 5-second correlation-sweep inference `correlation.go` currently relies on (Mechanism A, see the loose-coupling detection memory). It does **not** replace that mechanism for *which* third-party hook fired — `source: "hook"` doesn't disambiguate one hook from another. Treat this as a strengthening of CC9.2/third-party-visibility evidence, not a full replacement (§14).

| Result | Before this design | After |
|---|---|---|
| Cowork agent coverage | None — not a catalog entry | First-class `ai_coding_agents` row, flows through existing control-mapping pipeline |
| "Was monitoring tampered with" | Structurally unanswerable | Queryable via `hook_event_count`/`otel_event_count` (§10) |
| Cost governance | Not possible | `usage_daily_rollups`, per developer/team/model |

---

## 10. Dashboard UI

`GetSessionTimeline` (`dashboard.go:104-132`) currently does `SELECT * FROM activity_events WHERE session_id=$1 ORDER BY occurred_at` with no filtering — every `otel.*` row would interleave into the same array `session-panel.tsx` renders today. Its icon fallback (`ICON[event.kind] ?? "○"`) won't break on an unrecognized kind, but three rows for one real tool call (hook `tool.pre` + OTel `tool_decision` + `tool_result`) reads as visual duplication even though nothing is double-counted server-side (§8).

### 10.1 Timeline Query — Group by `tool_use_id`, Not in the Frontend

```sql
-- GetSessionTimeline, revised: hook rows stay primary; OTel rows sharing a tool_use_id
-- attach as corroboration instead of appearing as separate top-level entries.
SELECT ae.*,
       (SELECT jsonb_agg(jsonb_build_object(
                  'decisionSource', o.decision_source, 'success', o.tool_result_size_bytes IS NOT NULL,
                  'durationMs', o.duration_ms, 'occurredAt', o.occurred_at))
        FROM activity_events o
        WHERE o.tool_use_id = ae.tool_use_id AND o.event_source = 'otel' AND o.session_id = ae.session_id
       ) AS otel_corroboration
FROM activity_events ae
WHERE ae.session_id = $1 AND ae.event_source = 'hook'
UNION ALL
-- OTel rows with no matching tool_use_id (user_prompt, or ANY row in a hooks-absent
-- session — Cowork, or a session where hooks never fired) render as standalone entries.
SELECT ae.*, NULL AS otel_corroboration
FROM activity_events ae
WHERE ae.session_id = $1 AND ae.event_source = 'otel'
  AND (ae.tool_use_id IS NULL OR NOT EXISTS (
        SELECT 1 FROM activity_events h WHERE h.tool_use_id = ae.tool_use_id AND h.event_source = 'hook'))
ORDER BY occurred_at;
```

```
┌───────────────────────────────────────────────────────────┐
│  Session Timeline                                    ✕ Close │
├───────────────────────────────────────────────────────────┤
│  ⚡ session.start                              10:12:40      │
│  ○  Bash — "npm install"           ✓ verified by OTel  10:12:47 │
│  ⛔ BLOCK — write .env (filesystem policy)      10:13:02      │
│  ○  otel.user_prompt (Cowork, no hook coverage) 10:14:10      │
└───────────────────────────────────────────────────────────┘
```

### 10.2 Session-Level Coverage Badge (New — Session List + Panel Header)

Sourced directly from `ai_sessions.hook_event_count`/`otel_event_count`. A session with `hook_event_count=0, otel_event_count>0` on a Claude Code (not Cowork) session is the actual tamper signal `CTRL-TELEMETRY-COVERAGE` exists for — visually distinct (warning-toned), not styled the same as ordinary full coverage:

```
┌─────────────────────────────────────────────────────┐
│  Sessions                                             │
├─────────────────────────────────────────────────────┤
│  dev@corp.com   claude-code   [Hook+OTel]   14 events →│
│  dev2@corp.com  claude-code   [⚠ OTel-only]   6 events →│
│  dev3@corp.com  cowork        [OTel-only]    9 events →│
└─────────────────────────────────────────────────────┘
```

| `hook_event_count` | `otel_event_count` | Badge | Meaning |
|---|---|---|---|
| > 0 | > 0 | `Hook+OTel` (neutral) | Full coverage, corroborated |
| > 0 | 0 | `Hook-only` (neutral) | OTel not yet configured for this developer/session — expected during rollout (§8.2) |
| 0 | > 0, agent=Claude Code | `⚠ OTel-only` (**warning**) | Hooks were absent/disabled for this session — the signal this design exists to surface |
| 0 | > 0, agent=Claude Cowork | `OTel-only` (neutral) | Expected — Cowork has no hook mechanism, this is its normal state |

---

## 11. Implementation Plan

| Step | Deliverables | Gaps closed |
|---|---|---|
| **1 — Core ingestion** | Migration `012`, `otelingest` package, `/api/v1/otel/v1/logs`, identity backfill, session coverage counters, dedup/idempotency guards (§8.1), pilot rollout (§8.2 step 1–2) | G1, G2 |
| **2 — Cost & dashboards** | `/api/v1/otel/v1/metrics`, `usage_daily_rollups` scheduler job, `/analytics/cost-usage` + `/analytics/telemetry-coverage` endpoints, timeline-grouping and coverage-badge UI (§10) | G3 |
| **3 (deferred)** | Beta detailed tracing for exact third-party hook disambiguation | Not required for this design's baseline (§14) |

---

## 12. Critical Design Principles

1. **OTel never sits on the enforcement path.** It is fire-and-forget from Claude Code's/Cowork's perspective and read-only from governance-api's — nothing here can block, delay, or alter a tool call.
2. **Corroboration, not duplication.** Two sources describing one real action is the evidence model, not a bug — but every representation of "the same action" must be provably linked (`tool_use_id`), never independently counted.
3. **Enforcement stays single-sourced from hooks, permanently.** OTel-derived events can never create a `policy_actions` row — by construction in the translator, and by an explicit guard in `ingest.processOne`. Two independent lines of defense, not one.
4. **Identity is reconciled, never duplicated.** A real email/account UUID backfills the hook path's synthetic placeholder developer row in place; it never creates a second `developer_id` for the same person.
5. **Retries must be idempotent by construction.** OTel `event_id` is derived deterministically from `(session.id, event.sequence)`, not freshly generated, so a redelivered batch converges on the same rows via the existing `ON CONFLICT`.
6. **Absence of hook coverage is itself evidence, not silence.** A session with `otel_event_count>0` and `hook_event_count=0` is the tamper signal this design exists to surface — for Claude Code it must be visually flagged, never treated as an ordinary sparse session.
7. **Admin-enforced for Claude Code, admin-only for Cowork.** A developer-editable OTel config isn't the end state for the tamper-resistance goal; Cowork's org-wide admin switch has no other shape by design.
8. **Reused plumbing, not a parallel system.** Every session/developer/agent resolution function, the identity model, and the control-mapping engine are inherited unchanged — OTel is a second event source into the same tables, not a second architecture.

---

## 13. Gap Resolution Summary

| Gap | Resolution | Step |
|---|---|---|
| G1 — hooks tamperable/fail-open | `otelingest` package + `managed-settings.json` (admin-enforced, non-editable) | 1 |
| G2 — Cowork blind spot | Same OTLP endpoint; `Admin settings > Cowork`; Claude Cowork seeded into `ai_coding_agents` | 1 |
| G3 — no cost/token telemetry | `api_request` event → `cost_usd`/token columns; `usage_daily_rollups` | 2 |

---

## 14. Deferred Beyond This Design

Not designed in detail here — each needs its own design pass if prioritized later:

| Capability | Why deferred |
|---|---|
| OTel trace/span ingestion | Logs alone close G1–G3; traces add real cardinality/cost with no governance value yet |
| Exact third-party hook disambiguation (`claude_code.hook` spans, `hook_definitions`) | Requires enabling Claude Code's beta detailed tracing (`CLAUDE_CODE_ENHANCED_TELEMETRY_BETA`) — not required for this design's baseline; OTel's `decision_source` already gives a coarser, latency-free signal (§9) |
| OTel-based enforcement/blocking | Structurally impossible — export is async/best-effort by Anthropic's own design, never preventive (§2) |
| `/api/v1/otel/v1/metrics` ingestion | `api_request` LOG events already carry per-request cost/tokens for Claude Code; metrics only add value for aggregate counters (session count, lines of code) with no log equivalent — low priority |
| Real-time Collector alerting / SIEM fan-out rules | Operational/ops-team concern once the Collector is stood up — not a governance-api change |

---

## 15. Testing

- **Unit:** one test per OTel event name in `translate.go`'s mapping table, using recorded sample OTLP JSON payloads (captured via `OTEL_LOGS_EXPORTER=console` against a real Claude Code session — not hand-guessed shapes).
- **Dedup-specific (§8.1):** (a) assert `translateOne` never sets `Decision.Action` for any OTel event name; (b) assert `ingest.processOne` returns before any `policy_actions` INSERT when `EventSource == "otel"`, even if a malformed translator did populate `Decision`; (c) assert re-translating the identical LogRecord twice yields the identical `EventID`; (d) assert a hook `tool.pre` row and an OTel `tool_decision` row sharing a `tool_use_id` are queryable as a corroborating pair, not summed as two violations.
- **Contract:** replay a real Claude Code OTLP export against `/v1/logs`, assert `activity_events` (`event_source='otel'`) + `ai_sessions.otel_event_count` + developer identity backfill all land correctly, including the placeholder-email-to-real-email dedup path.
- **Regression:** existing hook-path tests in `governance/hooks_test.go` and `ingest` — assert zero behavior change (`event_source` defaults to `'hook'`, all existing call sites unaffected, violation counts identical to a pre-OTel baseline on a hook-only fixture).
