-- Migration 015: per-session token usage ledger + effective-dated model prices.
--
-- token_usage_events is an immutable, idempotent ledger: one row per
-- (session, Anthropic message id, source). Session totals are always derived
-- from it (never incremented counters), so retries and replays cannot double-count.
-- Cost is derived: tokens are the fact, cost_usd is computed from model_prices and
-- can be restated by re-running the pricing UPDATE after a price change.

CREATE TABLE IF NOT EXISTS model_prices (
    price_id                 SERIAL PRIMARY KEY,
    model_prefix             TEXT        NOT NULL,   -- longest-prefix match on model name
    effective_from           DATE        NOT NULL DEFAULT DATE '2000-01-01',
    input_per_mtok           NUMERIC(12,4) NOT NULL,
    output_per_mtok          NUMERIC(12,4) NOT NULL,
    cache_read_per_mtok      NUMERIC(12,4) NOT NULL,
    cache_write_5m_per_mtok  NUMERIC(12,4) NOT NULL,
    cache_write_1h_per_mtok  NUMERIC(12,4) NOT NULL,
    note                     TEXT,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (model_prefix, effective_from)
);

-- PLACEHOLDER base prices carried over from the 011 ai_models seed. Verify against
-- current vendor pricing and INSERT a new row with a later effective_from to change
-- a price; do not edit rows in place. Cache multipliers: read 0.1x, 5m write 1.25x,
-- 1h write 2x of the input price. Models with no matching prefix stay unpriced
-- (cost_usd NULL) rather than being shown as $0.
INSERT INTO model_prices
    (model_prefix, input_per_mtok, output_per_mtok, cache_read_per_mtok, cache_write_5m_per_mtok, cache_write_1h_per_mtok, note)
VALUES
    ('claude-opus',   15.00, 75.00, 1.5000, 18.7500, 30.0000, 'placeholder from 011 seed - verify'),
    ('claude-sonnet',  3.00, 15.00, 0.3000,  3.7500,  6.0000, 'placeholder from 011 seed - verify'),
    ('claude-haiku',   0.80,  4.00, 0.0800,  1.0000,  1.6000, 'placeholder from 011 seed - verify')
ON CONFLICT (model_prefix, effective_from) DO NOTHING;

CREATE TABLE IF NOT EXISTS token_usage_events (
    usage_id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id                UUID        NOT NULL REFERENCES ai_sessions(session_id) ON DELETE CASCADE,
    request_key               TEXT        NOT NULL,   -- Anthropic message.id (OTel: request id)
    source                    TEXT        NOT NULL CHECK (source IN ('otel', 'transcript', 'estimate')),
    model_name                TEXT        NOT NULL,   -- per request, not per session (/model can change mid-session)
    is_sidechain              BOOLEAN     NOT NULL DEFAULT FALSE,   -- sub-agent request
    input_tokens              BIGINT      NOT NULL DEFAULT 0,
    output_tokens             BIGINT      NOT NULL DEFAULT 0,      -- includes thinking tokens
    cache_read_tokens         BIGINT      NOT NULL DEFAULT 0,
    cache_creation_5m_tokens  BIGINT      NOT NULL DEFAULT 0,
    cache_creation_1h_tokens  BIGINT      NOT NULL DEFAULT 0,
    cache_creation_tokens     BIGINT GENERATED ALWAYS AS (cache_creation_5m_tokens + cache_creation_1h_tokens) STORED,
    thinking_tokens           BIGINT      NOT NULL DEFAULT 0,      -- informational subset of output_tokens
    service_tier              TEXT,
    cost_usd                  NUMERIC(14,6),                       -- NULL = unpriced, never 0
    price_id                  INTEGER     REFERENCES model_prices(price_id),
    client_version            TEXT,
    occurred_at               TIMESTAMPTZ NOT NULL,
    ingested_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (session_id, request_key, source)
);

CREATE INDEX IF NOT EXISTS idx_token_usage_session     ON token_usage_events (session_id);
CREATE INDEX IF NOT EXISTS idx_token_usage_occurred_at ON token_usage_events (occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_token_usage_model       ON token_usage_events (model_name);

-- One row per real API request: when several sources saw the same request, the most
-- authoritative wins (otel > transcript > estimate). Totals must read this view.
CREATE OR REPLACE VIEW token_usage_effective AS
SELECT DISTINCT ON (session_id, request_key) *
FROM token_usage_events
ORDER BY session_id, request_key,
         CASE source WHEN 'otel' THEN 0 WHEN 'transcript' THEN 1 ELSE 2 END;
