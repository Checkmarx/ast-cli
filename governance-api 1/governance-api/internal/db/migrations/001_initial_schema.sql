-- ============================================================
-- Migration 001: Initial governance schema
-- Architecture §5 — all tables in dependency order
-- ============================================================

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- §5.1 Policies & Versions
-- ============================================================

CREATE TYPE policy_status AS ENUM ('active', 'inactive');
CREATE TYPE enforcement_mode AS ENUM ('alert', 'block');

CREATE TABLE policies (
    policy_id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_name         VARCHAR(255) NOT NULL,
    description         TEXT,
    current_version_id  UUID,                       -- FK added after policy_versions is created
    status              policy_status NOT NULL DEFAULT 'active',
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE policy_versions (
    policy_version_id   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id           UUID        NOT NULL REFERENCES policies(policy_id) ON DELETE CASCADE,
    version_number      INTEGER     NOT NULL,
    definition          JSONB       NOT NULL,        -- full GovernancePack JSON — write-once
    enforcement_mode    enforcement_mode NOT NULL DEFAULT 'alert',
    created_by          UUID,                        -- FK → developers added later
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    change_summary      TEXT,
    UNIQUE (policy_id, version_number)
);

-- Now add FK from policies to the version table
ALTER TABLE policies
    ADD CONSTRAINT fk_policies_current_version
    FOREIGN KEY (current_version_id) REFERENCES policy_versions(policy_version_id);

-- Global pack version counter: incremented every time any policy is activated
CREATE SEQUENCE pack_version_seq START 1;

-- ============================================================
-- §5.1 Audit Events (append-only system mutation log)
-- ============================================================

CREATE TYPE actor_type AS ENUM ('developer', 'system');

CREATE TABLE audit_events (
    audit_event_id  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id        UUID,                            -- FK → developers (nullable for system events)
    actor_type      actor_type  NOT NULL DEFAULT 'system',
    event_type      VARCHAR(64) NOT NULL,
    resource_type   VARCHAR(64),
    resource_id     UUID,
    old_value       JSONB,
    new_value       JSONB,
    timestamp       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    metadata        JSONB
);

CREATE INDEX idx_audit_events_timestamp     ON audit_events(timestamp DESC);
CREATE INDEX idx_audit_events_event_type    ON audit_events(event_type);
CREATE INDEX idx_audit_events_actor_id      ON audit_events(actor_id);

-- ============================================================
-- §5.2 Developer & Agent Inventory
-- ============================================================

CREATE TYPE developer_status AS ENUM ('active', 'disabled');

CREATE TABLE developers (
    developer_id    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) NOT NULL UNIQUE,
    display_name    VARCHAR(255),
    username        VARCHAR(255),                    -- OS username used for resolution
    team            VARCHAR(255),
    department      VARCHAR(255),
    role            VARCHAR(64) NOT NULL DEFAULT 'developer',
    status          developer_status NOT NULL DEFAULT 'active',
    first_seen_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_seen_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_developers_email      ON developers(email);
CREATE INDEX idx_developers_username   ON developers(username);
CREATE INDEX idx_developers_status     ON developers(status);

-- Add FK from policy_versions.created_by to developers
ALTER TABLE policy_versions
    ADD CONSTRAINT fk_policy_versions_created_by
    FOREIGN KEY (created_by) REFERENCES developers(developer_id);

-- Add FK from audit_events.actor_id to developers
ALTER TABLE audit_events
    ADD CONSTRAINT fk_audit_events_actor_id
    FOREIGN KEY (actor_id) REFERENCES developers(developer_id);

CREATE TYPE agent_type_enum AS ENUM ('ide_extension', 'cli', 'standalone');
CREATE TYPE agent_catalog_status AS ENUM ('active', 'disabled');

CREATE TABLE ai_coding_agents (
    agent_id        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_name      VARCHAR(255) NOT NULL,           -- "Claude Code", "Cursor"
    vendor          VARCHAR(255),                    -- "Anthropic", "Anysphere"
    agent_type      agent_type_enum NOT NULL DEFAULT 'cli',
    version         VARCHAR(64),
    environment     VARCHAR(128),                    -- "vscode", "terminal"
    status          agent_catalog_status NOT NULL DEFAULT 'active',
    first_seen_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_seen_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    metadata        JSONB,
    UNIQUE (agent_name, vendor, version)
);

-- Seed known agents
INSERT INTO ai_coding_agents (agent_name, vendor, agent_type, environment)
VALUES
    ('Claude Code', 'Anthropic',   'cli',           'terminal'),
    ('Cursor',      'Anysphere',   'ide_extension', 'vscode'),
    ('Windsurf',    'Codeium',     'ide_extension', 'vscode'),
    ('Gemini CLI',  'Google',      'cli',           'terminal'),
    ('Copilot CLI', 'GitHub',      'cli',           'terminal'),
    ('Amazon Q',    'AWS',         'ide_extension', 'vscode');

CREATE TABLE ai_models (
    model_id        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    provider        VARCHAR(128) NOT NULL,
    model_name      VARCHAR(255) NOT NULL,
    model_version   VARCHAR(64),
    model_family    VARCHAR(128),
    status          VARCHAR(32) NOT NULL DEFAULT 'active',
    metadata        JSONB,
    UNIQUE (provider, model_name, model_version)
);

-- §5.2 developer_agents bridge: developer × machine × agent product
CREATE TABLE developer_agents (
    developer_agent_id  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    developer_id        UUID        NOT NULL REFERENCES developers(developer_id),
    agent_product_id    UUID        NOT NULL REFERENCES ai_coding_agents(agent_id),
    machine_id          VARCHAR(255) NOT NULL,
    governance_agent_id VARCHAR(255) NOT NULL UNIQUE,  -- SHA256 hex deterministic id
    first_seen_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_seen_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_developer_agents_developer_id      ON developer_agents(developer_id);
CREATE INDEX idx_developer_agents_governance_id     ON developer_agents(governance_agent_id);
CREATE INDEX idx_developer_agents_machine_id        ON developer_agents(machine_id);

-- ============================================================
-- §5.3 Session Tracking
-- ============================================================

CREATE TYPE session_status AS ENUM ('active', 'ended', 'expired');

CREATE TABLE ai_sessions (
    session_id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    claude_session_id   VARCHAR(255) NOT NULL,       -- raw session string from Claude/Cursor stdin
    developer_id        UUID        NOT NULL REFERENCES developers(developer_id),
    agent_id            UUID        NOT NULL REFERENCES ai_coding_agents(agent_id),
    developer_agent_id  UUID        REFERENCES developer_agents(developer_agent_id),
    model_id            UUID        REFERENCES ai_models(model_id),
    started_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    ended_at            TIMESTAMP WITH TIME ZONE,
    duration            INTEGER,                     -- seconds, computed on session-end
    request_count       INTEGER     NOT NULL DEFAULT 0,
    tool_call_count     INTEGER     NOT NULL DEFAULT 0,
    status              session_status NOT NULL DEFAULT 'active',
    metadata            JSONB,
    UNIQUE (claude_session_id, agent_id)
);

CREATE INDEX idx_ai_sessions_developer_id   ON ai_sessions(developer_id);
CREATE INDEX idx_ai_sessions_agent_id       ON ai_sessions(agent_id);
CREATE INDEX idx_ai_sessions_started_at     ON ai_sessions(started_at DESC);
CREATE INDEX idx_ai_sessions_status         ON ai_sessions(status);
CREATE INDEX idx_ai_sessions_claude_id      ON ai_sessions(claude_session_id);

-- ============================================================
-- §5.4 MCP Governance
-- ============================================================

CREATE TYPE mcp_status AS ENUM ('active', 'disabled');

CREATE TABLE mcp_servers (
    mcp_server_id   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    server_name     VARCHAR(255) NOT NULL UNIQUE,
    server_url      VARCHAR(1024),
    vendor          VARCHAR(255),
    server_version  VARCHAR(64),
    status          mcp_status  NOT NULL DEFAULT 'active',
    first_seen_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_seen_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    metadata        JSONB
);

CREATE TABLE session_mcp_servers (
    session_id          UUID        NOT NULL REFERENCES ai_sessions(session_id) ON DELETE CASCADE,
    mcp_server_id       UUID        NOT NULL REFERENCES mcp_servers(mcp_server_id),
    first_used_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_used_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    tool_call_count     INTEGER     NOT NULL DEFAULT 0,
    blocked_call_count  INTEGER     NOT NULL DEFAULT 0,
    PRIMARY KEY (session_id, mcp_server_id)
);

-- ============================================================
-- §5.5 Policy Actions (enforcement decisions)
-- ============================================================

CREATE TYPE action_type AS ENUM ('ALLOW', 'ALERT', 'BLOCK');
CREATE TYPE severity_level AS ENUM ('low', 'medium', 'high');
CREATE TYPE resolution_status AS ENUM ('open', 'resolved', 'suppressed');

CREATE TABLE policy_actions (
    policy_action_id    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id           UUID        REFERENCES policies(policy_id),
    policy_version_id   UUID        REFERENCES policy_versions(policy_version_id),
    session_id          UUID        NOT NULL REFERENCES ai_sessions(session_id),
    developer_id        UUID        NOT NULL REFERENCES developers(developer_id),
    agent_id            UUID        NOT NULL REFERENCES ai_coding_agents(agent_id),
    mcp_server_id       UUID        REFERENCES mcp_servers(mcp_server_id),
    action_type         action_type NOT NULL,
    violation_type      VARCHAR(64),                 -- path | command | mcp_tool | query | prompt
    severity            severity_level NOT NULL,     -- derived from action_type on ingest
    resource_type       VARCHAR(64),                 -- filesystem | shell | mcp | web_search | prompt
    resource_id         TEXT,                        -- matched value (file path / command / mcp tool)
    reason              TEXT,                        -- policyName from decision
    matched_rule        VARCHAR(255),                -- policyId from decision
    occurred_at         TIMESTAMP WITH TIME ZONE NOT NULL,
    resolution_status   resolution_status NOT NULL DEFAULT 'open',
    resolved_at         TIMESTAMP WITH TIME ZONE,
    metadata            JSONB                        -- latencyMs, packVersion
);

CREATE INDEX idx_policy_actions_developer_id    ON policy_actions(developer_id);
CREATE INDEX idx_policy_actions_session_id      ON policy_actions(session_id);
CREATE INDEX idx_policy_actions_policy_id       ON policy_actions(policy_id);
CREATE INDEX idx_policy_actions_action_type     ON policy_actions(action_type);
CREATE INDEX idx_policy_actions_occurred_at     ON policy_actions(occurred_at DESC);
CREATE INDEX idx_policy_actions_severity        ON policy_actions(severity);
