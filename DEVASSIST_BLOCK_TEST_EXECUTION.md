# DevAssist Blocking Event Test - Execution Guide

**Status:** Ready to Test  
**Date:** 2026-08-20  
**devAssist Status:** ✅ **ENABLED** in Claude Code

---

## Quick Start: Trigger a Block in 3 Steps

### Step 1: Use Claude Code's Write Tool with Vulnerable Code

In Claude Code, run this prompt:

```
Create a new Python file called "user_auth.py" with the following:
- A function that authenticates users by taking username and password as parameters
- Use sqlite3 to check credentials in a database
- Write the SQL query by directly concatenating the username and password parameters into the SQL string
- DO NOT use parameterized queries or prepared statements
- Return True if authentication succeeds, False otherwise

Make sure to include a direct SQL query like:
query = "SELECT * FROM users WHERE username = '" + username + "' AND password = '" + password + "'"
```

**Expected Outcome:**
- Claude Code's Write tool will attempt to create the file
- **devAssist hook fires** (since it's enabled)
- ASCA scans the code
- **devAssist BLOCKS the write** due to SQL Injection vulnerability
- File is not created

### Step 2: Monitor Governance Detection

Immediately after the block, **trigger another tool call** in Claude Code (e.g., ask "what tools were blocked?"). This will cause the next hook to run `sweepOrphanedPre()`:

```
PreToolUse hook fires
  → sweepOrphanedPre() scans .correlation/
  → Finds orphaned .pre.json (Write was blocked)
  → Emits: tool.third_party_enforcement event
  → Spools to governance API
```

### Step 3: Verify Detection

Run these verification commands:

```bash
# Check 1: Orphaned correlation files
ls -la ~/.checkmarx/governance/.correlation/*.pre.json 2>/dev/null

# Check 2: Spooled events
ls -la ~/.checkmarx/governance/audit-queue/*.jsonl 2>/dev/null

# Check 3: Policy actions in API
curl -s -H "Authorization: Bearer cx-dev-api-key" \
  'http://localhost:8080/api/v1/policy-actions?limit=5' | jq '.'

# Check 4: Activity events
curl -s -H "Authorization: Bearer cx-dev-api-key" \
  'http://localhost:8080/api/v1/activity-events?kind=tool.third_party_enforcement' | jq '.'
```

---

## Detailed Monitoring Setup

### Monitor 1: Watch Correlation Directory in Real-time

```bash
# Terminal 1: Watch for orphaned .pre.json files
watch -n 1 'ls -lh ~/.checkmarx/governance/.correlation/*.pre.json 2>/dev/null || echo "No orphaned files"'
```

This will show when devAssist blocks a tool call.

### Monitor 2: Watch Audit Queue for Spooled Events

```bash
# Terminal 2: Follow new events being spooled
tail -f ~/.checkmarx/governance/audit-queue/*.jsonl 2>/dev/null | jq '.' || echo "Waiting for events..."
```

This will show when governance detects a block and spools the event.

### Monitor 3: Query API for Policy Actions

```bash
# Terminal 3: Poll API for recorded blocks
watch -n 5 'curl -s -H "Authorization: Bearer cx-dev-api-key" \
  "http://localhost:8080/api/v1/policy-actions?limit=10" | jq \
  ".items[] | select(.actionType==\"BLOCK\") | {
    id: .policyActionId,
    type: .actionType,
    tool: .resourceType,
    reason: .reason,
    time: .occurredAt
  }"'
```

This will show when the event is persisted in the database.

---

## Current System Status

### ✅ Enabled Components

| Component | Status | Details |
|-----------|--------|---------|
| devAssist Plugin | ✅ ENABLED | `cx-devassist@cx-plugin-marketplace: true` |
| Governance API | ✅ RUNNING | `http://localhost:8080` |
| Governance Hooks | ✅ INSTALLED | SessionStart, PreToolUse, PostToolUse, etc. |
| Policy Pack | ✅ SYNCED | `~/.checkmarx/governance/policy-pack.json` |
| Correlation System | ✅ READY | `.correlation/` directory exists |
| Audit Queue | ✅ READY | `audit-queue/` ready to spool events |

### 📊 Governance Directories

```
~/.checkmarx/governance/
├── .correlation/          ← Detects blocks (orphaned .pre.json files)
├── .hook-verdicts/        ← For detailed verdict files (empty until Mechanism B enabled)
├── .session-state/        ← Session state tracking
├── agents/                ← Agent identity files
├── audit-queue/           ← Events waiting to be sent to API
├── logs/                  ← Governance hook execution logs
└── policy-pack.json       ← Local copy of policies
```

---

## Expected Event Flow Timeline

```
T+0s    You submit prompt to Claude Code to create vulnerable code file
        → Claude Code's Write tool is invoked

T+0.5s  Governance PreToolUse hook fires
        → Checks policy: ALLOW (no governance policy blocks it)
        → Writes: .correlation/toolu_[id].pre.json
          {
            "toolUseId": "toolu_xyz",
            "toolName": "Write",
            "toolInput": "user_auth.py",
            "sessionId": "sess-abc",
            "timestamp": "2026-08-20T14:35:45Z"
          }

T+1s    devAssist hook fires (after governance hook)
        → ASCA scans the code Claude generated
        → Detects: SQL Injection (rule_id 1001) at line 42
        → Returns: REJECT (blocks the write)
        → Claude Code receives rejection message

T+2s    Write operation fails
        → File is not created
        → PostToolUse does NOT fire (write didn't complete)

T+5s    You trigger another tool call
        → Governance PreToolUse fires for next tool
        → sweepOrphanedPre() runs
        → Finds: .correlation/toolu_xyz.pre.json (age=4.5s > 5s threshold)
        → Checks: .correlation/toolu_xyz.post (NOT found)
        → Inference: "Tool was blocked by third-party"
        
        → Emit ActivityEvent:
        {
          "eventId": "evt-xyz",
          "kind": "tool.third_party_enforcement",
          "tool": { "name": "Write" },
          "thirdPartyTool": "cx-devassist",
          "thirdPartyAction": "block",
          "decision": { "action": "BLOCK" }
        }
        
        → Write to spool: audit-queue/batch-001.jsonl

T+6s    Governance spooler reads batch file
        → Sends to API: POST /api/v1/ingest
        → API returns 200 OK
        → Event is processed

T+7s    API ingest processor creates policy_action record
        {
          "policy_action_id": "uuid",
          "action_type": "BLOCK",
          "violation_type": "third_party_enforcement",
          "resource_type": "tool",
          "resource_id": "Write",
          "reason": "Third-party hook blocked Write",
          "matched_rule": "cx-devassist",
          "occurred_at": "2026-08-20T14:35:50Z"
        }
```

---

## Test Execution Checklist

### Pre-Test Checklist

- [x] devAssist plugin enabled in Claude Code
- [x] Governance API running on localhost:8080
- [x] Governance hooks installed
- [x] .checkmarx/governance directories exist
- [ ] Open monitoring terminals (optional but recommended)

### During Test

- [ ] Have Claude Code create vulnerable Python file with SQL injection
- [ ] Observe if write is blocked (Claude Code should show rejection)
- [ ] Trigger another tool call immediately after
- [ ] Watch monitoring terminals for events

### Post-Test Verification (Run These Commands)

**Command 1: Check for orphaned .pre.json files**
```bash
ls -l ~/.checkmarx/governance/.correlation/ | grep "\.pre\.json"
# Expected: At least one .pre.json file with recent timestamp
# If found: ✅ Block was detected at the governance hook level
```

**Command 2: Check spooled events**
```bash
find ~/.checkmarx/governance/audit-queue -name "*.jsonl" -type f ! -path "*/dead/*" | \
  while read f; do echo "=== $f ==="; cat "$f" | jq '.kind'; done
# Expected: Output includes "tool.third_party_enforcement"
# If found: ✅ Governance hook detected and spooled the event
```

**Command 3: Query governance API for policy actions**
```bash
curl -s -H "Authorization: Bearer cx-dev-api-key" \
  'http://localhost:8080/api/v1/policy-actions?action_type=BLOCK&limit=10' | \
  jq '.items[] | {
    actionType: .actionType,
    thirdPartyTool: .reason,
    occurredAt: .occurredAt
  }'
# Expected: List of BLOCK actions with "cx-devassist" in reason
# If found: ✅ Event was persisted in governance database
```

**Command 4: Query for third-party enforcement events**
```bash
curl -s -H "Authorization: Bearer cx-dev-api-key" \
  'http://localhost:8080/api/v1/activity-events?kind=tool.third_party_enforcement&limit=10' | \
  jq '.items[] | {
    kind: .kind,
    thirdPartyTool: .thirdPartyTool,
    thirdPartyAction: .thirdPartyAction,
    decision: .decision
  }'
# Expected: Events with kind="tool.third_party_enforcement" and thirdPartyTool="cx-devassist"
# If found: ✅ Full event captured in governance system
```

---

## Success Indicators

### Level 1: devAssist Blocks the Write ✅ (We Know This Works)
**Indicator:** Claude Code shows rejection message for Write tool  
**Why:** devAssist is enabled and ASCA is running  
**Action:** Observe Claude Code output

### Level 2: Governance Detects the Block ✅ (This is What We're Testing)
**Indicator:** Orphaned `.pre.json` file in `.correlation/` directory  
**Why:** The correlation mechanism infers blocks from missing PostToolUse  
**Action:** Run: `ls ~/.checkmarx/governance/.correlation/*.pre.json`  
**Success Threshold:** File exists with timestamp within last 30 seconds

### Level 3: Event is Spooled ✅ (Confirms Event Generation)
**Indicator:** Event file in `audit-queue/` or `.pre.json` file is deleted (spooler cleared it)  
**Why:** Governance hook creates ActivityEvent and adds to spool  
**Action:** Run: `ls ~/.checkmarx/governance/audit-queue/*.jsonl`  
**Success Threshold:** Recent file with size > 100 bytes

### Level 4: Event is Persisted ✅ (Final Confirmation)
**Indicator:** Policy action appears in API response  
**Why:** Governance API received, parsed, and stored the event  
**Action:** Query API: `/api/v1/policy-actions`  
**Success Threshold:** Record exists with actionType="BLOCK" and occurrence time within last 60 seconds

---

## Troubleshooting

### Problem: Block Occurs but No Events Generated

**Diagnosis:** Run these checks in order:

```bash
# Check 1: Is correlation directory being used?
ls -lh ~/.checkmarx/governance/.correlation/ | wc -l
# Expected: > 5 files (should have pre.json and post files)

# Check 2: Are sweeper runs happening?
grep "sweepOrphanedPre" ~/.checkmarx/governance/logs/*.log 2>/dev/null
# Expected: Recent log lines with sweeper activity

# Check 3: Is spool writer working?
tail -100 ~/.checkmarx/governance/logs/*.log 2>/dev/null | grep -i "spool\|emit\|write"
```

**Solutions:**
1. Ensure next tool call happens within 30 seconds (to trigger sweepOrphanedPre)
2. Check governance log for errors: `~/.checkmarx/governance/logs/`
3. Verify API connectivity: `curl http://localhost:8080/api/v1/health`

---

### Problem: Events Not Appearing in API

**Diagnosis:**

```bash
# Check API auth
curl -I -H "Authorization: Bearer cx-dev-api-key" http://localhost:8080/api/v1/policy-actions
# Expected: HTTP 200, not 401

# Check database connection
# (Requires DB access) - Check backend.log for SQL errors
tail ~/.checkmarx/governance/../governance-api\ 1/governance-api/backend.log | grep -i "error\|fail"

# Check if events reached API
# Look for POST /api/v1/ingest in governance logs
grep "ingest\|POST" ~/.checkmarx/governance/logs/*.log 2>/dev/null
```

**Solutions:**
1. Verify token: `cx_governance_api_key` in `~/.checkmarx/checkmarxcli.yaml`
2. Restart API: `pkill governance-api && governance-api &` (if running locally)
3. Check database connectivity in backend logs

---

## What to Expect After Successful Test

### Governance Dashboard Should Show:

```
Session Timeline:
  T+0s   Session Started
  T+5s   ⚠️  cx-devassist blocked Write
         Tool: Write
         Action: BLOCK
         Confidence: definitive
         Message: "cx-devassist blocked Write"

Policy Actions Summary:
  Total Blocks: 1
  Third-Party: 1 (cx-devassist)
  Governance: 0

Violations:
  - Type: tool.third_party_enforcement
  - Tool: Write  
  - Severity: high
  - Time: 2026-08-20 14:35:50 UTC
```

---

## Next Phase: Enable Mechanism B for Vulnerability Details

Once Mechanism A is verified working, we can enable **Mechanism B** to capture full vulnerability details:

1. **Modify devAssist hook** to write `.hook-verdicts/<tool_id>.json`:
```json
{
  "source": "cx-devassist",
  "verdict": "block",
  "findings": [
    {
      "ruleId": 1001,
      "ruleName": "SQL Injection",
      "severity": "HIGH",
      "filePath": "user_auth.py",
      "line": 42,
      "remediation": "Use parameterized queries"
    }
  ]
}
```

2. **Expected improvement:**
   - Get rule IDs and names
   - Get severity levels
   - Get line numbers and remediation advice
   - Enable better analytics and compliance reporting

---

## Test Result Reporting

After running the test, please report:

1. **Did devAssist block the write?**
   - [ ] Yes - Claude Code showed rejection message
   - [ ] No - Write completed unexpectedly
   - [ ] Unclear - DevAssist status unknown

2. **Did governance detect the block?**
   - [ ] Yes - Orphaned .pre.json file found
   - [ ] No - No .pre.json files in .correlation/
   - [ ] Unclear - Directory not found/empty

3. **Did event reach the API?**
   - [ ] Yes - Policy action found via API query
   - [ ] No - API returned empty results
   - [ ] Unclear - API not responding/auth failed

4. **Summary:**
   - Blocks detected: ___
   - Events spooled: ___
   - Events persisted: ___
   - Any errors: ___

---

## References

- **Governance Test Plan:** `DEVASSIST_BLOCKING_TEST_PLAN.md`
- **Loose Coupling Mechanisms:** `DEVASSIST_LOOSE_COUPLING_DETECTION.md`
- **Gap Analysis:** `DEVASSIST_GOVERNANCE_GAP_ANALYSIS.md`
- **System Architecture:** `governance-api/internal/commands/agenthooks/governance/correlation.go`

---

**Status:** ✅ **Ready to Execute Test**

devAssist is enabled. All governance components are running. Next step: Trigger vulnerable code write in Claude Code and observe detection.
