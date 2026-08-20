-- Migration 006: skills discovery tables — skill catalog and per-session tracking.

CREATE TABLE skills (
    skill_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_name            VARCHAR(255) NOT NULL UNIQUE,
    description           TEXT,
    status                VARCHAR(50) DEFAULT 'active',
    first_seen_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_seen_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    total_invocations     INTEGER DEFAULT 0,

    CONSTRAINT valid_skill_status CHECK (status IN ('active', 'deprecated', 'blocked'))
);

CREATE INDEX idx_skills_name ON skills(skill_name);
CREATE INDEX idx_skills_last_seen ON skills(last_seen_at DESC);
CREATE INDEX idx_skills_status ON skills(status);

CREATE TABLE session_skills (
    session_skill_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id            UUID NOT NULL REFERENCES ai_sessions(session_id) ON DELETE CASCADE,
    skill_id              UUID NOT NULL REFERENCES skills(skill_id) ON DELETE CASCADE,
    invocation_count      INTEGER DEFAULT 1,
    first_invoked_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_invoked_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    UNIQUE(session_id, skill_id)
);

CREATE INDEX idx_session_skills_session ON session_skills(session_id);
CREATE INDEX idx_session_skills_skill ON session_skills(skill_id);
CREATE INDEX idx_session_skills_last_invoked ON session_skills(last_invoked_at DESC);
