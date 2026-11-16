-- Migration 004: PII detection columns, session hook inventory, MCP server source tracking.

-- PII scan results on prompt.submit events.
-- prompt_text is only populated when PII is detected — it stores the redacted version.
ALTER TABLE activity_events
  ADD COLUMN IF NOT EXISTS pii_detected  BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS prompt_text   TEXT,
  ADD COLUMN IF NOT EXISTS pii_matches   JSONB;

-- Session hook inventory: non-governance hooks discovered at session start.
CREATE TABLE IF NOT EXISTS session_hooks (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id   UUID NOT NULL REFERENCES ai_sessions(session_id) ON DELETE CASCADE,
  event        TEXT NOT NULL,
  matcher      TEXT NOT NULL DEFAULT '',
  command      TEXT NOT NULL,
  vendor       TEXT NOT NULL DEFAULT 'unknown',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_session_hooks_session_id ON session_hooks(session_id);

-- Third-party enforcement fields on activity_events.
ALTER TABLE activity_events
  ADD COLUMN IF NOT EXISTS third_party_tool         TEXT,
  ADD COLUMN IF NOT EXISTS third_party_action       TEXT,
  ADD COLUMN IF NOT EXISTS attribution_confidence   TEXT
    CHECK (attribution_confidence IN ('definitive', 'candidate'));

-- MCP server source: was it configured (session-start inventory) or called (observed via hook)?
ALTER TABLE mcp_servers
  ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'called'
    CHECK (source IN ('configured', 'called', 'configured+called'));
