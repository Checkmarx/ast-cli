-- Migration 008: Compliance framework (SOC2 + EU AI Act), POC scope.
-- Per COMPLIANCE_ARCHITECTURE.md — closes G3 (violation_type enum) and G4 (framework/control model).
-- Compliance tables are read-only consumers of existing tables (audit_events, activity_events,
-- policy_actions, policies) — none of those tables are altered structurally here beyond the
-- violation_type enum fix, which is itself a standalone gap (G3), independent of compliance.

-- ============================================================
-- G3 — enforce violation_type as a closed vocabulary
-- ============================================================

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'violation_type_enum') THEN
    CREATE TYPE violation_type_enum AS ENUM ('path', 'command', 'mcp_tool', 'query', 'prompt', 'data_export', 'other');
  END IF;
END$$;

ALTER TABLE policy_actions
  ALTER COLUMN violation_type TYPE violation_type_enum
  USING (
    CASE
      WHEN violation_type IS NULL THEN NULL
      WHEN violation_type IN ('path','command','mcp_tool','query','prompt','data_export') THEN violation_type::violation_type_enum
      ELSE 'other'::violation_type_enum
    END
  );

-- ============================================================
-- G4 — compliance framework / control catalog (§6.1)
-- ============================================================

CREATE TYPE compliance_framework_type AS ENUM ('soc2', 'eu_ai_act');
CREATE TYPE evidence_status_enum       AS ENUM ('operational', 'planned');
CREATE TYPE control_status_enum        AS ENUM ('compliant', 'partially_compliant', 'non_compliant', 'planned');
CREATE TYPE mapping_source_enum        AS ENUM ('automatic', 'manual');
CREATE TYPE rule_status_enum           AS ENUM ('active', 'disabled');

CREATE TABLE compliance_frameworks (
    framework_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    framework_type compliance_framework_type NOT NULL UNIQUE,
    display_name   VARCHAR(255) NOT NULL,
    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Shared control catalog — not owned by any single framework (§5.2, §6.1).
CREATE TABLE controls (
    control_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    control_code    VARCHAR(64) NOT NULL UNIQUE,
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    evidence_spec   JSONB NOT NULL,
    evidence_status evidence_status_enum NOT NULL DEFAULT 'planned',
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_controls_evidence_status ON controls(evidence_status);

-- Crosswalk: one control can satisfy many frameworks under different codes (§6.1).
CREATE TABLE framework_control_mappings (
    mapping_id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    framework_id           UUID NOT NULL REFERENCES compliance_frameworks(framework_id) ON DELETE CASCADE,
    control_id             UUID NOT NULL REFERENCES controls(control_id) ON DELETE CASCADE,
    framework_control_code VARCHAR(64) NOT NULL,
    UNIQUE (framework_id, control_id)
);

CREATE INDEX idx_framework_control_mappings_framework ON framework_control_mappings(framework_id);
CREATE INDEX idx_framework_control_mappings_control   ON framework_control_mappings(control_id);

-- Auto-mapping rule table — data, not code (§5.7, §6.4).
CREATE TABLE policy_mapping_rules (
    rule_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_code          VARCHAR(64) NOT NULL UNIQUE,
    when_conditions    JSONB NOT NULL,
    maps_to_control_id UUID NOT NULL REFERENCES controls(control_id) ON DELETE CASCADE,
    description        TEXT,
    status             rule_status_enum NOT NULL DEFAULT 'active'
);

CREATE INDEX idx_policy_mapping_rules_status ON policy_mapping_rules(status);

-- Which policy satisfies which control, and how the mapping was created (§5.7, §6.1).
CREATE TABLE policy_control_mappings (
    mapping_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id       UUID NOT NULL REFERENCES policies(policy_id) ON DELETE CASCADE,
    control_id      UUID NOT NULL REFERENCES controls(control_id) ON DELETE CASCADE,
    mapping_source  mapping_source_enum NOT NULL DEFAULT 'automatic',
    rationale       TEXT,
    mapped_by       UUID REFERENCES developers(developer_id),
    mapped_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (policy_id, control_id)
);

CREATE INDEX idx_policy_control_mappings_policy  ON policy_control_mappings(policy_id);
CREATE INDEX idx_policy_control_mappings_control ON policy_control_mappings(control_id);

-- Evidence — references only, never copies (§5.4, §6.1, principle §12.3).
CREATE TABLE compliance_evidence (
    evidence_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    control_id               UUID NOT NULL REFERENCES controls(control_id) ON DELETE CASCADE,
    source_audit_event_id    UUID REFERENCES audit_events(audit_event_id),
    source_activity_event_id UUID REFERENCES activity_events(activity_event_id),
    source_policy_action_id  UUID REFERENCES policy_actions(policy_action_id),
    observed_at              TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at               TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CHECK (num_nonnulls(source_audit_event_id, source_activity_event_id, source_policy_action_id) = 1)
);

CREATE INDEX idx_compliance_evidence_control      ON compliance_evidence(control_id);
CREATE INDEX idx_compliance_evidence_observed_at  ON compliance_evidence(observed_at DESC);
-- Prevent the same source row from being linked to the same control twice across scheduler runs.
CREATE UNIQUE INDEX idx_compliance_evidence_dedupe ON compliance_evidence(
    control_id,
    COALESCE(source_audit_event_id, '00000000-0000-0000-0000-000000000000'),
    COALESCE(source_activity_event_id, '00000000-0000-0000-0000-000000000000'),
    COALESCE(source_policy_action_id, '00000000-0000-0000-0000-000000000000')
);

-- Raw counts only — no computed percentage column (§8.1).
CREATE TABLE compliance_status_snapshots (
    snapshot_id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    framework_id               UUID NOT NULL REFERENCES compliance_frameworks(framework_id) ON DELETE CASCADE,
    applicable_control_count   INTEGER NOT NULL,
    compliant_count            INTEGER NOT NULL DEFAULT 0,
    partially_compliant_count  INTEGER NOT NULL DEFAULT 0,
    non_compliant_count        INTEGER NOT NULL DEFAULT 0,
    planned_count              INTEGER NOT NULL DEFAULT 0,
    calculated_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_compliance_snapshots_framework_calc ON compliance_status_snapshots(framework_id, calculated_at DESC);

-- Human review of a violation — not remediation (§2, §5.5).
CREATE TABLE oversight_events (
    oversight_event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_action_id   UUID NOT NULL REFERENCES policy_actions(policy_action_id) ON DELETE CASCADE,
    reviewer_id        UUID NOT NULL REFERENCES developers(developer_id),
    decision           VARCHAR(32) NOT NULL CHECK (decision IN ('affirmed', 'overridden', 'escalated')),
    rationale          TEXT,
    reviewed_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oversight_events_policy_action ON oversight_events(policy_action_id);
CREATE INDEX idx_oversight_events_reviewer      ON oversight_events(reviewer_id);

-- ============================================================
-- Seed data — frameworks, controls, rules, framework mappings (§9)
-- ============================================================

INSERT INTO compliance_frameworks (framework_type, display_name) VALUES
    ('soc2',     'SOC2 Type II'),
    ('eu_ai_act','EU AI Act (Regulation 2024/1689)');

INSERT INTO controls (control_code, title, description, evidence_spec, evidence_status) VALUES
    ('CTRL-ACCESS-PROVISIONING', 'Access provisioning and deprovisioning',
     'Developer and agent access is added and removed through a controlled, audited process.',
     '{"auditEventTypes":["DEVELOPER_ADDED","DEVELOPER_DISABLED","AGENT_REGISTERED","AGENT_DISABLED"],"activityEventKinds":[],"policyActionFilters":{},"oversightRequired":false}',
     'operational'),

    ('CTRL-BOUNDARY-PROTECTION', 'Boundary protection for external tools',
     'Unauthorized MCP/external tool access is actively blocked.',
     '{"auditEventTypes":[],"activityEventKinds":[],"policyActionFilters":{"violation_type":"mcp_tool"},"oversightRequired":false}',
     'operational'),

    ('CTRL-DATA-PROTECTION', 'Restricted data transmission',
     'Filesystem access to sensitive data paths (credentials, exports) is actively blocked.',
     '{"auditEventTypes":[],"activityEventKinds":[],"policyActionFilters":{"violation_type":"path"},"oversightRequired":false}',
     'operational'),

    ('CTRL-PII-PROTECTION', 'PII / credential detection',
     'Purpose-built PII detection (email, SSN, credit card, API key, JWT, etc.) is active and enforcing.',
     '{"auditEventTypes":[],"activityEventKinds":[],"policyActionFilters":{},"oversightRequired":false}',
     'operational'),

    ('CTRL-SYSTEM-INTEGRITY', 'Shell/system command integrity',
     'Dangerous shell commands are actively detected and blocked.',
     '{"auditEventTypes":[],"activityEventKinds":[],"policyActionFilters":{"violation_type":"command"},"oversightRequired":false}',
     'operational'),

    ('CTRL-HUMAN-OVERSIGHT', 'Human review of automated high-risk decisions',
     'A different human reviews BLOCK/ALERT decisions within the review window.',
     '{"auditEventTypes":["VIOLATION_REVIEWED","VIOLATION_OVERRIDDEN"],"activityEventKinds":[],"policyActionFilters":{},"oversightRequired":true}',
     'operational'),

    ('CTRL-CHANGE-MANAGEMENT', 'Policy change management',
     'Policy changes (create/update/activate/disable/enable/delete) go through controlled, versioned, actor-attributed pipeline.',
     '{"auditEventTypes":["POLICY_CREATED","POLICY_VERSION_CREATED","POLICY_ACTIVATED","POLICY_UPDATED","POLICY_DISABLED","POLICY_ENABLED","POLICY_DELETED"],"activityEventKinds":[],"policyActionFilters":{},"oversightRequired":false}',
     'operational'),

    ('CTRL-THIRD-PARTY-VISIBILITY', 'Third-party tool visibility',
     'Other (non-governance) enforcement tooling already present is discovered and inventoried.',
     '{"auditEventTypes":[],"activityEventKinds":[],"policyActionFilters":{},"oversightRequired":false}',
     'operational'),

    ('CTRL-RECORD-KEEPING', 'Automatic record-keeping across the AI system lifecycle',
     'Every session/tool-call event is automatically logged.',
     '{"auditEventTypes":[],"activityEventKinds":["session.start","tool.pre","tool.post","prompt.submit","session.end"],"policyActionFilters":{},"oversightRequired":false}',
     'operational');

-- Auto-mapping rules (§5.7, §6.4) — data, evaluated by ComplianceService.AutoMapPolicyToControls.
INSERT INTO policy_mapping_rules (rule_code, when_conditions, maps_to_control_id, description) VALUES
    ('RULE-MCP-BLOCK',    '{"type":"mcp","action":"BLOCK"}',
        (SELECT control_id FROM controls WHERE control_code = 'CTRL-BOUNDARY-PROTECTION'),
        'MCP tool blocks demonstrate boundary protection'),

    ('RULE-FS-PATH-BLOCK','{"type":"filesystem","matchTarget":"path","action":"BLOCK"}',
        (SELECT control_id FROM controls WHERE control_code = 'CTRL-DATA-PROTECTION'),
        'Filesystem path blocks demonstrate data protection'),

    ('RULE-SHELL-BLOCK',  '{"type":"shell","action":"BLOCK"}',
        (SELECT control_id FROM controls WHERE control_code = 'CTRL-SYSTEM-INTEGRITY'),
        'Shell command blocks demonstrate system integrity enforcement'),

    ('RULE-ANY-BLOCK',    '{"action":"BLOCK"}',
        (SELECT control_id FROM controls WHERE control_code = 'CTRL-HUMAN-OVERSIGHT'),
        'Any BLOCK is a candidate for human oversight review'),

    ('RULE-PII',          '{"policyType":"pii"}',
        (SELECT control_id FROM controls WHERE control_code = 'CTRL-PII-PROTECTION'),
        'PII-typed policies are purpose-built PII/credential detection'),

    ('RULE-THIRD-PARTY',  '{"policyType":"third_party_monitor"}',
        (SELECT control_id FROM controls WHERE control_code = 'CTRL-THIRD-PARTY-VISIBILITY'),
        'Third-party-monitor policies are auto-discovered third-party enforcement'),

    ('RULE-CATCHALL',     '{}',
        (SELECT control_id FROM controls WHERE control_code = 'CTRL-CHANGE-MANAGEMENT'),
        'Every policy is itself change-management evidence, regardless of category');

-- Framework mappings (§9.2, §9.3) — crosswalk from shared controls to each framework's own codes.
INSERT INTO framework_control_mappings (framework_id, control_id, framework_control_code)
SELECT (SELECT framework_id FROM compliance_frameworks WHERE framework_type = 'soc2'), control_id, code
FROM (VALUES
    ('CTRL-ACCESS-PROVISIONING',  'CC6.1'),
    ('CTRL-BOUNDARY-PROTECTION',  'CC6.6'),
    ('CTRL-DATA-PROTECTION',      'CC6.7'),
    ('CTRL-PII-PROTECTION',       'CC6.7'),
    ('CTRL-SYSTEM-INTEGRITY',     'CC7.1'),
    ('CTRL-HUMAN-OVERSIGHT',      'CC7.2'),
    ('CTRL-CHANGE-MANAGEMENT',    'CC8.1'),
    ('CTRL-THIRD-PARTY-VISIBILITY','CC9.2')
) AS m(control_code, code)
JOIN controls c ON c.control_code = m.control_code;

INSERT INTO framework_control_mappings (framework_id, control_id, framework_control_code)
SELECT (SELECT framework_id FROM compliance_frameworks WHERE framework_type = 'eu_ai_act'), control_id, code
FROM (VALUES
    ('CTRL-RECORD-KEEPING',  'Art.12'),
    ('CTRL-HUMAN-OVERSIGHT', 'Art.14')
) AS m(control_code, code)
JOIN controls c ON c.control_code = m.control_code;
