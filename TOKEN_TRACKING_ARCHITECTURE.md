# Per-Session Token Tracking: Current State and Production Approach

**Role:** Senior Architect review
**Scope:** How to capture, store and report token consumption per AI-agent session (Claude Code first; Cowork via OTel; Cursor best-effort).
**Status:** Phase 1 (transcript source) implemented on branch `feature/token-tracking` (root repo and the nested `ast-cli/ast-cli` repo). OTel source (Phase 2) is still design only; the ledger already accepts `source='otel'`.

**Implemented in Phase 1**
- Hook: `governance/usage.go` reads new transcript lines from a persisted offset on `Stop`, `SubagentStop` and `SessionEnd` and spools `usage.delta` events (token counts, model and ids only, no message content).
- Backend: migration `015_token_usage.sql` (`token_usage_events`, `model_prices`, `token_usage_effective` view), `ingest/usage.go`, and the `/sessions/{id}/usage`, `/analytics/token-usage`, `/token-usage/prices` and `/token-usage/reprice` endpoints. Session and overview responses carry usage totals.
- UI: Tokens and Cost columns on Sessions, a Token Usage section in the session panel, a `/usage` page, and Tokens/Cost Today on Overview.
- Verified against this session's real transcript: ledger totals equal an independent calculation, and replaying the whole transcript changes nothing.
- Open items: prices in `model_prices` are placeholders from the 011 seed and must be verified, and models without a price row (for example `claude-fable`) show tokens but no cost.

---

## 1. Current State: nothing captures tokens today

| Layer | What exists | Gap |
|---|---|---|
| Hooks (`ast-cli/internal/commands/agenthooks/governance/hooks.go`) | `session.start`, `tool.pre`, `tool.post`, `prompt.submit`, `session.end` events. | None carries usage data. `handleClaudeSessionEnd` (line 404) sends only a timestamp. |
| `ai_sessions` (`001_initial_schema.sql:161`) | `request_count`, `tool_call_count`. | `request_count` counts prompts (incremented in `updateSessionCounters`). It is unrelated to API calls or tokens. No token or cost columns. |
| `ai_models` (`011_model_detection.sql`) | Prices in a JSONB blob. | Prices are unversioned and the seed is stale (`claude-haiku-4`). Cost cannot be computed correctly over time. |
| OTel design (`OTEL_MONITORING_ARCHITECTURE.md`) | `api_request` events, `usage_daily_rollups`, shared `session.id`. | Design only: the Go code has no `otel`, `cost_usd` or `input_tokens` references. The doc names the migration `012`, which is taken by `012_detection_method`. Migrations run to `014`, so the next free number is `015`. |
| Model attribution | Detected once at session start. | `/model` switches mid-session, and the model can change at sub-agent or compaction boundaries, so one `model_id` per session mis-prices mixed-model sessions. |

Structural limit: Claude Code hooks fire on events, not on API calls, so they never see usage. The OTel design only rolls up per day and per developer, so neither path gives per-session totals.

## 2. Capture Options

| Option | Fidelity | Tamper resistance | Infra | Coverage |
|---|---|---|---|---|
| A. Transcript reader in hooks | High: per-message `usage` (input, output, cache-read, cache-create). | Low: the file is local and user-editable. | None. Reuses spool and ingest. | Claude Code only. |
| B. OTel `api_request` events | Highest: per request, with `cost_usd`, model and `session.id`. | High, if pushed via managed-settings. | Collector or OTLP endpoint. | Claude Code and Cowork. |
| C. Estimate from prompt length | Poor: misses tool output and context. | n/a | None. | Last resort only (Cursor). |
| D. Vendor admin or usage API | Org-level only. | High. | API key. | No per-session data. |

## 3. Recommendation: one ledger, two sources, OTel wins on conflict

This follows the cardinal rule in the OTel doc: a second source into the same pipeline, with no parallel data model.

```
Hooks (Stop / SessionEnd / PreCompact)         OTel api_request
   └─ read transcript from stored offset            └─ translateOne()
        └─ emit `usage.delta` events                      │
                  └──────────► ingest.Processor ◄─────────┘
                                    │  INSERT ... ON CONFLICT DO NOTHING
                                    ▼
                        token_usage_events  (immutable ledger)
                                    │
              scheduler tick ───────┴──► session_usage (view or table) ► usage_daily_rollups
```

### Source A mechanics (hook side, ships first)
- Every hook payload carries `transcript_path`. Verify that `ast-cx-hooks`' `SessionEndEvent` exposes it.
- Trigger on `Stop` (end of each turn), not only `SessionEnd`. `SessionEnd` is unreliable, which migration 003 already works around for stale sessions.
- Keep a byte offset per session next to `saveSessionState`. Read only new JSONL lines and emit one `usage.delta` per assistant message.
- Also hook `PreCompact`. Compaction rewrites the transcript, so offsets can break without it.
- Include sub-agent (sidechain) transcripts, which are separate files and often a large share of spend.

### Source B mechanics (OTel)
Implement the already-designed `api_request` mapping, with its deterministic `EventID`.

### Precedence
For a given `(session, request)` pair, prefer OTel, otherwise transcript. The `source` column keeps both rows, so a mismatch between the two is itself a tamper signal.

## 4. Data Model (migration 015)

```sql
CREATE TABLE token_usage_events (
  usage_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id      UUID NOT NULL REFERENCES ai_sessions ON DELETE CASCADE,
  request_key     TEXT NOT NULL,        -- Anthropic message.id / requestId
  source          TEXT NOT NULL CHECK (source IN ('otel','transcript','estimate')),
  model_name      TEXT NOT NULL,        -- per request, not per session
  is_sidechain    BOOLEAN NOT NULL DEFAULT false,
  input_tokens BIGINT, output_tokens BIGINT,
  cache_read_tokens BIGINT, cache_creation_tokens BIGINT,
  cost_usd        NUMERIC(12,6),        -- NULL if unknown, never 0
  price_version   INT,                  -- which price row computed it
  occurred_at     TIMESTAMPTZ NOT NULL,
  UNIQUE (session_id, request_key, source)
);

CREATE TABLE model_prices (
  model_name, effective_from,
  input_per_mtok, output_per_mtok,
  cache_read_per_mtok, cache_write_per_mtok,
  PRIMARY KEY (model_name, effective_from)
);
```

Per-session totals are `SUM` over the ledger, with the precedence above applied. Do not use `UPDATE ai_sessions SET tokens = tokens + $n`: it is not idempotent, so retries and replays double-count. If the session list needs fast reads, add a cached `session_usage` table that the scheduler rebuilds from the ledger. The ingest path never writes it.

## 5. Correctness Pitfalls

1. **Streaming duplicates.** Transcripts write several lines per `message.id`, each with cumulative usage. Dedupe by `message.id` and take the final or max values. Summing lines overcounts.
2. **Cache tokens are separate and priced differently.** Cache reads cost roughly 10% of input and cache writes roughly 125%. Storing only input and output understates cost.
3. **Resume and fork.** A resumed session can reuse a session id, and a fork gets a new one. `UNIQUE(claude_session_id, agent_id)` plus the ledger key handles this. Never attribute by "last session".
4. **Cost is derived, not stored truth.** Store tokens as the fact. Compute cost from the effective-dated `model_prices` row and keep `price_version` so history can be restated.
5. **Unknown is not zero.** Cursor and Copilot hooks expose no usage. Record `source='estimate'` or NULL and show "n/a" in the UI. A fake 0 breaks budgets.
6. **Privacy.** The transcript reader must extract only the `usage`, `model` and `id` fields and never copy message content, or it reintroduces the PII problem the prompt-redaction work solves.
7. **Fail-open and latency.** The transcript read runs in the hook subprocess. Cap it (for example 50 ms and 1 MB per run), keep the offset, and catch up on the next `Stop`. Never block the agent.

## 6. Rollout

| Phase | Scope | Exit criterion |
|---|---|---|
| 1 | Migration 015, `usage.delta` event, transcript reader on `Stop` and `SessionEnd`, ledger insert, per-session total in `GetSessions` (`dashboard.go:40`) and `session-panel.tsx`. | A session's total matches Claude's own `/cost` within about 2%. |
| 2 | OTel `api_request` ingest, precedence logic, `usage_daily_rollups`, `/analytics/cost-usage`. | Transcript-vs-OTel variance under 1% on pilot machines. |
| 3 | Managed-settings rollout, budget alerts, a control for usage anomalies, Cursor estimates. | Fleet coverage above 95%. |

Operational alerts:
- A session with tool events but zero usage rows.
- Variance between the two sources.
- Ingest lag.

Index the ledger on `(session_id)` and `(occurred_at)`, and partition by month if volume warrants.

## 7. Open Decisions

1. Is Phase 1 (transcript-only) acceptable as the first ship, with OTel following? Recommended: yes. It needs no infrastructure, but the numbers are not tamper-proof until Phase 2.
2. Are Cursor or Copilot token numbers needed? Their hooks give none, so it is estimate-only unless a proxy is added.
3. Should budgets apply per session, per developer-day, or both? Only hooks can block, and a `UserPromptSubmit` check against the cached `session_usage` is the only low-latency enforcement point.
