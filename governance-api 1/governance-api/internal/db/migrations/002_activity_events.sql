-- Migration 002: activity_events table for full session event history
-- Stores every ActivityEvent received from cx (all kinds, all action types).
-- policy_actions stores only BLOCK/ALERT; this table stores everything.

CREATE TABLE activity_events (
    activity_event_id   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id            VARCHAR(255) NOT NULL UNIQUE,   -- eventId from cx spool
    session_id          UUID        NOT NULL REFERENCES ai_sessions(session_id) ON DELETE CASCADE,
    kind                VARCHAR(64) NOT NULL,            -- session.start | tool.pre | tool.post | prompt.submit | session.end
    agent_type          VARCHAR(64),
    tool_category       VARCHAR(64),
    tool_name           VARCHAR(255),
    tool_command        TEXT,
    tool_paths          JSONB,
    tool_query          TEXT,
    mcp_server          VARCHAR(255),
    mcp_tool            VARCHAR(255),
    prompt_length       INTEGER,
    action_type         VARCHAR(16) NOT NULL DEFAULT 'ALLOW',  -- ALLOW | ALERT | BLOCK
    policy_name         VARCHAR(255),
    policy_id           VARCHAR(255),
    policy_version      INTEGER,
    matched_on          VARCHAR(64),
    matched_value       TEXT,
    agent_message       TEXT,
    latency_ms          INTEGER,
    occurred_at         TIMESTAMP WITH TIME ZONE NOT NULL,
    raw                 JSONB
);

CREATE INDEX idx_activity_events_session_id  ON activity_events(session_id);
CREATE INDEX idx_activity_events_occurred_at ON activity_events(occurred_at DESC);
CREATE INDEX idx_activity_events_kind        ON activity_events(kind);
CREATE INDEX idx_activity_events_action_type ON activity_events(action_type);
