-- Migration 003: track last activity time per session for stale session detection.
-- Sessions killed without firing SessionEnd get auto-expired after 15 minutes of inactivity.

ALTER TABLE ai_sessions ADD COLUMN IF NOT EXISTS last_event_at TIMESTAMP WITH TIME ZONE;

-- Seed existing rows with started_at so the column is never null
UPDATE ai_sessions SET last_event_at = started_at WHERE last_event_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ai_sessions_last_event_at ON ai_sessions(last_event_at);
