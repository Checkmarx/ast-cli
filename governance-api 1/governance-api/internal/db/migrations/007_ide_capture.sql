-- Migration 007: IDE capture — track which IDE or runtime launched each session.

ALTER TABLE ai_sessions
    ADD COLUMN IF NOT EXISTS ide_name       VARCHAR(50),
    ADD COLUMN IF NOT EXISTS ide_version    VARCHAR(50),
    ADD COLUMN IF NOT EXISTS workspace_path TEXT;

CREATE INDEX IF NOT EXISTS idx_sessions_ide_name ON ai_sessions(ide_name);
