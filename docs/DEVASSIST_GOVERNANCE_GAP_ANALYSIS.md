# DevAssist Plugin Blocking Activities — AI Governance Gap Analysis

**Date:** 2026-08-20  
**Status:** Analysis of potential tracking gaps between cx-devassist security enforcement and AI Governance system

---

## Executive Summary

The **cx-devassist plugin** is a real-time security blocker that prevents developers from writing code with known vulnerabilities. The **AI Governance system** currently tracks policy enforcement decisions but has **5 critical gaps** in capturing devAssist-specific blocking activities. This analysis identifies what should be tracked to provide complete visibility into developer security behavior.

---

## Part 1: What cx-devAssist Does (Blocking Activities)

### 1.1 Core Function: ASCA Security Analysis

The devAssist plugin performs **Automated Security Code Analysis (ASCA)** at file-write time:

```
Developer → Edit file → devAssist hook fires → ASCA scan → Decision → Block/Allow
```

**Key Finding:** From `delta.go:14-44`, devAssist uses a **deduplication tuple**:
```go
type findingKey struct {
    ruleID          uint32
    problematicLine string // TrimSpace applied
}
```

This means devAssist detects **rule violations** (SAST rules, container vulnerabilities, API misuse, etc.) at the **line level**.

### 1.2 Blocking Actions

When devAssist finds vulnerabilities, it performs **THREE actions**:

| Action | Mechanism | Visibility to Governor |
|--------|-----------|------------------------|
| **BLOCK the write** | Prevents file save via hook rejection | ❌ NOT CAPTURED |
| **Show human-readable denial** | `permissionDecisionReason()` — findings summary | ❌ NOT CAPTURED |
| **Inject remediation guidance** | `additionalContext()` — MCP tool names, suppression commands | ❌ NOT CAPTURED |

**Example block from delta.go:109-112:**
```
"ASCA security scan detected vulnerabilities in src/auth.go.
Findings:
  - auth.go line 42 [HIGH] SQL Injection (rule_id 1001) — Use parameterized queries
  - auth.go line 87 [MEDIUM] Hardcoded Secret (rule_id 5002) — Use environment variables"
```

### 1.3 Vulnerability Context Delivered to Agent

When devAssist blocks, it **injects this context into the agent's message window:**

```
ASCA detected vulnerabilities in {filePath}. 
Do not bypass the scan by writing the same content through another tool or shell command.
ANALYZE each finding to determine if it is a real vulnerability or a false positive...
For each real finding, invoke the cx-devassist:cx-devassist-asca skill...
If that skill is not available, call mcp__Checkmarx__codeRemediation directly...
Use the remediation guidance returned by the tool to fix the vulnerability, then retry the write.
If a finding is a confirmed false positive, suppress it by running:
  cx ignore-vulnerability --scan-type asca --data '{finding}' --optional-flags 'aiProvider=...'
```

**Key insight:** The remediation workflow has **4 branches**:
1. Use devAssist skill (recommended)
2. Call MCP remediation tool directly
3. Suppress as false positive
4. Retry the write after fixing

---

## Part 2: Current AI Governance Tracking

### 2.1 What ActivityEvent Currently Captures

From `models.go:228-246`:

```go
type ActivityEvent struct {
    EventID   string
    Timestamp time.Time
    Kind      string  // session.start | tool.pre | tool.post | prompt.submit | session.end | tool.third_party_enforcement
    SessionID string
    MachineID string
    AgentID   string
    UserID    string
    Agent     ActivityAgent
    Tool      ActivityTool
    Prompt    *ActivityPrompt
    Decision  ActivityDecision      // ← enforcement decision
    IDE       ActivityIDEInfo
    ModelInfo *ModelDetectionInfo
    ThirdPartyTool   string          // ← for third-party hooks
    ThirdPartyAction string
    AttributionConfidence string
}
```

### 2.2 What Decision Captures

From `models.go:292-302`:

```go
type ActivityDecision struct {
    Action        string  // "BLOCK" | "ALLOW" | "ALERT"
    PolicyID      string
    PolicyName    string
    PolicyVersion int
    PackVersion   int
    MatchedOn     string  // what triggered the match
    MatchedValue  string
    AgentMessage  string  // message shown to user
    LatencyMs     int64
}
```

### 2.3 What Policy Actions Track

From `models.go:151-170`:

```go
type PolicyAction struct {
    PolicyActionID   uuid.UUID
    PolicyID         *uuid.UUID
    PolicyVersionID  *uuid.UUID
    SessionID        uuid.UUID
    DeveloperID      uuid.UUID
    AgentID          uuid.UUID
    MCPServerID      *uuid.UUID
    ActionType       string  // "BLOCK" | "ALLOW" | "ALERT"
    ViolationType    string
    Severity         string  // "high" | "medium" | "low"
    ResourceType     string
    ResourceID       string
    Reason           string
    MatchedRule      string
    OccurredAt       time.Time
    ResolutionStatus string  // tracking if resolved
    ResolvedAt       *time.Time
    Metadata         map[string]any
}
```

### 2.4 Current Tracking Gaps

✅ **Currently tracked:**
- Policy blocks/allows
- Session lifecycle
- Tool invocations
- PII detection
- MCP usage
- Skill invocations

❌ **NOT tracked:**
- devAssist-specific blocks
- Vulnerability rule details
- ASCA scan results
- Remediation guidance delivery
- Suppression commands issued
- False positive confirmations
- Remediation tool invocations

---

## Part 3: The 5 Critical Gaps

### Gap 1: No devAssist Block Events

**Problem:** When devAssist blocks a write, there's no event recorded in AI Governance.

**Current state:**
- devAssist fires a hook → blocks the write → nothing goes to governance-api

**Impact:**
- No visibility into "how many times were developers blocked by security?"
- No per-developer security incident tracking
- No rule-violation trending

**What should be tracked:**
```json
{
  "kind": "tool.security_enforcement",
  "tool": {
    "category": "security",
    "name": "asca",
    "command": "file-write-check"
  },
  "decision": {
    "action": "BLOCK",
    "reason": "ASCA vulnerability detected",
    "findings": [
      {
        "ruleId": 1001,
        "ruleName": "SQL Injection",
        "severity": "HIGH",
        "filePath": "src/auth.go",
        "line": 42,
        "problemCode": "SELECT * FROM users WHERE id = " + userId,
        "remediation": "Use parameterized queries"
      }
    ]
  }
}
```

### Gap 2: No Vulnerability Rule Aggregation

**Problem:** Each ASCA finding has a rule_id (uint32) but governance only tracks generic "BLOCK" action.

**Current limitation:** From `delta.go:20-25`, devAssist deduplicates by (ruleID, problematicLine) but governance doesn't track this tuple.

**Impact:**
- Can't answer: "Which 3 rules account for 50% of blocks?"
- Can't measure: "Has rule 1001 (SQL Injection) improved after we added detection?"
- Can't correlate: "Do certain models trigger more security violations?"

**What should be tracked:**
```sql
-- Missing schema
ALTER TABLE policy_actions ADD COLUMN (
  security_rule_id INTEGER,
  security_rule_name VARCHAR(255),
  problematic_line TEXT TRIM,
  remediation_advice TEXT
);
```

### Gap 3: No Remediation Workflow Tracking

**Problem:** devAssist injects remediation guidance, but governance doesn't track if the developer:
1. Uses devAssist skill
2. Calls MCP remediation tool
3. Suppresses as false positive
4. Retries the write

**Current state:** The developer reads the guidance and acts, but nothing is logged.

**Impact:**
- No insight into remediation success rate
- Can't measure: "How effective is the MCP remediation tool?"
- Can't identify: "Which developers need security training?"
- Can't optimize: "Does devAssist skill guidance vs MCP tool guidance lead to different outcomes?"

**What should be tracked:**
```json
{
  "kind": "tool.remediation",
  "tool": {
    "category": "skill",
    "name": "cx-devassist:cx-devassist-asca"
  },
  "decision": {
    "action": "REMEDIATION_APPLIED",
    "relatedBlockEventId": "block-event-123",
    "remediationType": "skill|mcp|suppress|manual"
  }
}
```

### Gap 4: No False Positive Tracking

**Problem:** When developers suppress a finding as false positive, that decision isn't captured.

**Execution path from delta.go:127:**
```bash
cx ignore-vulnerability --scan-type asca --data '{finding}' --optional-flags 'aiProvider=...'
```

This command:
1. Writes to workspace `.checkmarx/ignore.json`
2. Suppresses the rule going forward
3. But **governance doesn't know this happened**

**Impact:**
- No insight into false positive rate
- Can't validate: "Is ASCA too noisy?"
- Can't track: "Which rules are frequently suppressed?"
- Miss development of trust: "Is the developer learning the rule or just suppressing?"

**What should be tracked:**
```json
{
  "kind": "tool.false_positive_suppression",
  "tool": {
    "category": "security",
    "name": "asca"
  },
  "decision": {
    "action": "SUPPRESS",
    "ruleId": 1001,
    "reason": "false_positive",
    "developedVerification": true/false,
    "relatedBlockEventId": "block-event-123"
  }
}
```

### Gap 5: No ASCA Rule Performance Metrics

**Problem:** Even after tracking blocks, there's no aggregation by rule/severity/remediation path.

**Missing insights:**
- "Rule 1001 (SQL Injection) has 100 blocks, 60% suppressed as false positive, 30% fixed via MCP, 10% fixed manually"
- "Developer X is suppressing 80% of security findings instead of fixing"
- "Model Y (claude-haiku) triggers 3x more security violations than Model Z (claude-opus)"

**What should be tracked:**
```sql
CREATE TABLE security_metrics (
  metric_id UUID PRIMARY KEY,
  developer_id UUID,
  rule_id INTEGER,
  rule_name VARCHAR(255),
  severity VARCHAR(50),
  period_start TIMESTAMP,
  period_end TIMESTAMP,
  block_count INTEGER,
  fixed_via_skill_count INTEGER,
  fixed_via_mcp_count INTEGER,
  suppressed_as_fp_count INTEGER,
  fixed_manually_count INTEGER,
  remediation_success_rate FLOAT,
  created_at TIMESTAMP
);
```

---

## Part 4: Impact Matrix

| Gap | Current State | Missing Capability | Business Impact | Tracking Effort |
|-----|---------------|-------------------|-----------------|-----------------|
| **Gap 1** | No block events | Security incident visibility | Can't measure policy effectiveness | 2-4 hrs |
| **Gap 2** | Generic "BLOCK" | Rule-level aggregation | Can't trend specific vulnerabilities | 4-6 hrs |
| **Gap 3** | Block only | Remediation workflow tracking | Can't measure developer compliance | 6-8 hrs |
| **Gap 4** | Suppression lost | False positive tracking | Can't validate detection quality | 3-4 hrs |
| **Gap 5** | Events only | Rule performance dashboard | Can't optimize security training | 8-12 hrs |

---

## Part 5: Recommended Tracking Additions

### 5.1 New ActivityEvent Kinds

```go
// Add to models.go ActivityEvent Kind values
const (
    "tool.asca_check"              // Pre-write ASCA scan
    "tool.asca_block"              // ASCA vulnerability block
    "tool.security_remediation"    // Developer remediation action
    "tool.false_positive_suppress"  // Suppression recorded
    "tool.vulnerability_retried"   // Write retried after fix
)
```

### 5.2 New ActivityTool Structure for Security

```go
type SecurityFinding struct {
    RuleID          uint32 `json:"ruleId"`
    RuleName        string `json:"ruleName"`
    Severity        string `json:"severity"`         // HIGH | MEDIUM | LOW
    FilePath        string `json:"filePath"`
    Line            int    `json:"line"`
    ProblematicLine string `json:"problematicLine"` // TrimSpace'd
    Remediation     string `json:"remediation"`
}

type SecurityDecision struct {
    Findings       []SecurityFinding `json:"findings"`
    FindingCount   int               `json:"findingCount"`
    SuggestedTools []string          `json:"suggestedTools"` // ["cx-devassist-asca", "mcp_codeRemediation"]
}
```

### 5.3 Enhanced Policy Action for Security

```go
// Extend policy_actions table
ALTER TABLE policy_actions ADD COLUMN (
    security_rule_id        INTEGER,
    security_rule_name      VARCHAR(255),
    vulnerability_severity  VARCHAR(50),
    problematic_code_line   TEXT,
    remediation_suggestion  TEXT,
    remediation_path_taken  VARCHAR(50),  -- "skill" | "mcp" | "manual" | "suppressed"
    false_positive_confirmed BOOLEAN,
    block_to_resolution_seconds INTEGER
);
```

### 5.4 New Security Metrics Table

```sql
CREATE TABLE security_rule_metrics (
    metric_id UUID PRIMARY KEY,
    developer_id UUID REFERENCES developers(developer_id),
    ai_session_id UUID REFERENCES ai_sessions(session_id),
    rule_id INTEGER,
    rule_name VARCHAR(255),
    rule_category VARCHAR(100),  -- "injection" | "hardcoded_secret" | "xss" | etc
    severity VARCHAR(50),         -- "HIGH" | "MEDIUM" | "LOW"
    
    -- Tracking the block
    blocked_at TIMESTAMP,
    
    -- Remediation tracking
    remediation_started_at TIMESTAMP,
    remediation_completed_at TIMESTAMP,
    remediation_type VARCHAR(50),  -- "skill" | "mcp" | "manual" | "suppressed"
    was_false_positive BOOLEAN,
    
    -- Metrics
    time_to_remediate_seconds INTEGER,
    tool_used VARCHAR(255),  -- "cx-devassist-asca" | "mcp_codeRemediation" | etc
    
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### 5.5 Query Examples to Enable

```sql
-- Which rules cause most blocks?
SELECT rule_id, rule_name, COUNT(*) as blocks, 
       SUM(CASE WHEN was_false_positive THEN 1 ELSE 0 END) as false_positives,
       ROUND(100.0 * SUM(CASE WHEN was_false_positive THEN 1 ELSE 0 END) / COUNT(*), 2) as fp_rate
FROM security_rule_metrics
WHERE blocked_at >= NOW() - INTERVAL '30 days'
GROUP BY rule_id, rule_name
ORDER BY blocks DESC;

-- Developer security pattern analysis
SELECT d.email, 
       COUNT(*) as total_blocks,
       SUM(CASE WHEN remediation_type = 'skill' THEN 1 ELSE 0 END) as skill_fixes,
       SUM(CASE WHEN remediation_type = 'mcp' THEN 1 ELSE 0 END) as mcp_fixes,
       SUM(CASE WHEN remediation_type = 'suppressed' THEN 1 ELSE 0 END) as suppressions,
       AVG(time_to_remediate_seconds) as avg_remediation_time_sec
FROM security_rule_metrics srm
JOIN developers d ON srm.developer_id = d.developer_id
WHERE blocked_at >= NOW() - INTERVAL '30 days'
GROUP BY d.email
ORDER BY total_blocks DESC;

-- Model impact on security violations
SELECT am.model_name,
       COUNT(*) as block_count,
       ROUND(100.0 * SUM(CASE WHEN srm.was_false_positive THEN 1 ELSE 0 END) / COUNT(*), 2) as false_positive_rate
FROM security_rule_metrics srm
JOIN ai_sessions s ON srm.ai_session_id = s.session_id
JOIN ai_models am ON s.model_id = am.model_id
WHERE srm.blocked_at >= NOW() - INTERVAL '30 days'
GROUP BY am.model_name
ORDER BY block_count DESC;
```

---

## Part 6: Implementation Roadmap

### Phase 1: Core Instrumentation (Week 1-2)
- [ ] Update `ActivityEvent` struct to include security findings
- [ ] Add migration for `security_rule_metrics` table
- [ ] Modify devAssist hook integration to emit block events
- [ ] Add `tool.asca_block` event kind handler in ingest

### Phase 2: Remediation Tracking (Week 3-4)
- [ ] Instrument devAssist skill to log invocation
- [ ] Instrument MCP codeRemediation tool to log results
- [ ] Capture `cx ignore-vulnerability` suppressions
- [ ] Add false positive confirmation tracking

### Phase 3: Dashboard & Metrics (Week 5-6)
- [ ] Add security dashboard to governance-ui
- [ ] Rule leaderboard (top blocks, top FP rates)
- [ ] Developer security patterns view
- [ ] Model impact on security violations chart

### Phase 4: Optimization & Compliance (Week 7-8)
- [ ] Automated alerts for high FP rates
- [ ] Security training recommendations
- [ ] Compliance reporting (SOC2, ISO)
- [ ] Rule effectiveness scoring

---

## Part 7: Risk Analysis

### 7.1 Current Risks (Without Enhanced Tracking)

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| **Security blind spot** | High | Critical | Implement Gap 1 + Gap 2 |
| **Ineffective remediation** | Medium | High | Implement Gap 3 |
| **Alert fatigue** | High | Medium | Implement Gap 4 (measure FP rate) |
| **Model regression** | Medium | High | Implement Gap 5 (track by model) |
| **Compliance audit failure** | Low | Critical | Full implementation needed |

### 7.2 Implementation Risks

- **Performance:** Adding 5 new tables could slow ingest → Mitigate with batching, async metrics
- **Data volume:** Security_rule_metrics could be large → Mitigate with retention policy
- **Hook compatibility:** devAssist changes might break existing flows → Mitigate with feature flags

---

## Part 8: Success Metrics

After implementation, you should be able to answer:

1. ✅ "How many times was the developer blocked by security this week?" (Gap 1)
2. ✅ "Which 3 rules cause 50% of all blocks?" (Gap 2)
3. ✅ "Did the developer fix the vulnerability or suppress it?" (Gap 3)
4. ✅ "What's our false positive rate on ASCA scans?" (Gap 4)
5. ✅ "Which model (Claude vs GPT vs Gemini) triggers more security violations?" (Gap 5)
6. ✅ "Are developers improving or just suppressing more rules?" (All gaps)

---

## References

- **devAssist code:** `ast-cli/internal/commands/agenthooks/guardrails/asca/delta.go`
- **Governance models:** `governance-api/internal/models/models.go`
- **Ingest processor:** `governance-api/internal/ingest/ingest.go`
- **Current decision tracking:** `PolicyAction` model (lines 151-170 in models.go)

---

## Conclusion

The cx-devAssist plugin provides **real-time vulnerability blocking**, but AI Governance is currently **blind to these decisions**. The 5 gaps represent a ~30-40 hour implementation to capture:

1. Block events themselves
2. Vulnerability rule details
3. Remediation guidance acceptance
4. False positive confirmations
5. Aggregate security metrics

**Recommendation:** Prioritize **Gaps 1 & 2** (8-10 hrs) for immediate visibility into security enforcement, then add **Gap 3** (6-8 hrs) to measure remediation effectiveness. Gaps 4 & 5 enable optimization but have lower immediate priority.
