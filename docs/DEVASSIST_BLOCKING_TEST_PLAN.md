# DevAssist Blocking Event Detection Test Plan

**Date:** 2026-08-20  
**Objective:** Enable devAssist, trigger security blocking events, and verify AI Governance detects them  
**Status:** Test Setup

---

## Part 1: Environment Status Check

### ✅ Confirmed Status

- **Governance API:** Running at `http://localhost:8080`
- **Governance Hooks:** Installed in Claude Code
  - SessionStart ✅
  - PreToolUse ✅
  - PostToolUse ✅
  - UserPromptSubmit ✅
  - SessionEnd ✅
- **Policy Pack:** Synced (`~/.checkmarx/governance/policy-pack.json`)
- **Config:** Governance API key and URL configured

### ⚠️ Items to Verify

1. **devAssist Plugin Status** — Is it enabled in Claude Code?
2. **ASCA Engine** — Is it available?
3. **Hook Verdicts Directory** — Exists at `~/.checkmarx/governance/.hook-verdicts/`

---

## Part 2: Check devAssist Availability

### Step 1: Check Claude Code Settings

```bash
# Check if devAssist is in Claude Code plugins
cat ~/.claude/settings.json | jq '.plugins.["checkmarx-devassist"] // empty'

# Expected output if enabled:
# { "enabled": true, "version": "1.0+" }
```

### Step 2: Check for DevAssist Hook Installation

```bash
# DevAssist should register a hook in the hooks chain
cat ~/.claude/settings.json | jq '.PreToolUse[].hooks[].command' 2>/dev/null | grep -i devassist || echo "DevAssist hook not found"
```

### Step 3: Verify ASCA Engine Availability

```bash
# Check if ASCA is installed locally
cx asca --version 2>/dev/null || echo "ASCA not found in PATH"

# Check cx configuration for ASCA
cat ~/.checkmarx/checkmarxcli.yaml | grep asca
# Expected:
# cx_asca_location: "/path/to/asca/binary"
# cx_asca_port: 65419
```

---

## Part 3: Create Test Scenarios for Triggering Blocks

### Scenario A: SQL Injection Vulnerability

**File:** Create a test file with SQL injection vulnerability

```bash
# Create test directory
mkdir -p /tmp/governance-test
cd /tmp/governance-test

# Create vulnerable Python file
cat > vulnerable_auth.py << 'EOF'
import sqlite3

def authenticate_user(username, user_input):
    # ❌ VULNERABLE: SQL Injection
    # Directly concatenating user input into SQL query
    db = sqlite3.connect(':memory:')
    cursor = db.cursor()
    query = "SELECT * FROM users WHERE username = '" + username + "' AND password = '" + user_input + "'"
    cursor.execute(query)
    return cursor.fetchone()

def get_user_data(user_id):
    # ❌ VULNERABLE: SQL Injection in dynamic query
    db = sqlite3.connect(':memory:')
    cursor = db.cursor()
    sql = "SELECT * FROM users WHERE id = " + str(user_id)
    cursor.execute(sql)
    return cursor.fetchone()
EOF
```

**Expected:** devAssist blocks writes with ASCA violation: "SQL Injection (rule_id 1001)"

---

### Scenario B: Hardcoded Secret/API Key

```bash
cat > api_config.py << 'EOF'
# ❌ VULNERABLE: Hardcoded API Key
API_KEY = "sk-prod-12345abcdefghijklmnop"
SECRET_TOKEN = "ghp_1234567890abcdefghijklmnopqrstuv"

def call_external_api():
    # Using hardcoded credentials
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "X-Token": SECRET_TOKEN
    }
    return headers
EOF
```

**Expected:** devAssist blocks with ASCA violation: "Hardcoded Secret (rule_id 5002)"

---

### Scenario C: Command Injection

```bash
cat > command_exec.py << 'EOF'
import os
import subprocess

def process_user_file(filename):
    # ❌ VULNERABLE: Command Injection
    command = f"cat /etc/passwd | grep {filename}"
    os.system(command)
    
def search_logs(search_term):
    # ❌ VULNERABLE: Unvalidated shell execution
    cmd = "grep -r '" + search_term + "' /var/log/"
    subprocess.call(cmd, shell=True)
EOF
```

**Expected:** devAssist blocks with ASCA violation: "Command Injection (rule_id 2001)"

---

## Part 4: Trigger Blocking Events via Claude Code

### Test Protocol

1. **Start Fresh Session**
   - Open Claude Code
   - Note the SessionID from logs: `~/.checkmarx/governance/logs/`

2. **Use File Write Tool**
   - Ask Claude Code: *"Create a new Python authentication file with user login validation"*
   - Have Claude create one of the vulnerable code samples above
   - devAssist should block the write

3. **Monitor for Block Events**

---

## Part 5: Verify Governance Detection

### Location 1: Correlation Files

```bash
# Check for orphaned .pre.json files (indicates blocks)
ls -la ~/.checkmarx/governance/.correlation/*.pre.json 2>/dev/null

# Expected when blocked:
# -rw------- 1 anjali anjali 234 Aug 20 14:30 tool-abc123.pre.json
# -rw------- 1 anjali anjali   0 Aug 20 14:30 tool-abc123.post  ← Won't exist if blocked

# File contents should show:
# {
#   "toolUseId": "tool-abc123",
#   "toolName": "Write",
#   "toolInput": "vulnerable_auth.py",
#   "sessionId": "sess-xyz",
#   "timestamp": "2026-08-20T14:30:45Z"
# }
```

### Location 2: Spool Queue

```bash
# Check audit event spool
ls -la ~/.checkmarx/governance/audit-queue/ | head -20

# Expected file when block detected:
# -rw------- 1 anjali anjali 1024 Aug 20 14:35 batch-001.jsonl

# Check contents:
tail -50 ~/.checkmarx/governance/audit-queue/batch-*.jsonl | jq '.'
```

### Location 3: Governance API Database

**Check if events were ingested:**

```bash
# Use governance API to query events
curl -s -H "Authorization: Bearer cx-dev-api-key" \
  'http://localhost:8080/api/v1/policy-actions?action_type=BLOCK&limit=10' \
  | jq '.items[] | {actionType, thirdPartyTool, occurredAt}'

# Expected response:
# {
#   "actionType": "BLOCK",
#   "thirdPartyTool": "cx-devassist",
#   "occurredAt": "2026-08-20T14:35:00Z"
# }
```

### Location 4: Activity Events

```bash
# Check raw activity events
curl -s -H "Authorization: Bearer cx-dev-api-key" \
  'http://localhost:8080/api/v1/activity-events?kind=tool.third_party_enforcement&limit=10' \
  | jq '.items[] | {kind, thirdPartyTool, thirdPartyAction, decision}'

# Expected:
# {
#   "kind": "tool.third_party_enforcement",
#   "thirdPartyTool": "cx-devassist",
#   "thirdPartyAction": "block",
#   "decision": {
#     "action": "BLOCK",
#     "agentMessage": "cx-devassist blocked Write"
#   }
# }
```

---

## Part 6: Expected Event Flow

```
Timeline:
─────────────────────────────────────────────────────────────

T+0s   Claude Code receives prompt to create vulnerable file
       → Governance SessionStart hook fires
       → Session created: sess-abc123

T+2s   Claude writes code via Write tool
       → Governance PreToolUse hook fires
         → Our policy: ALLOW
         → Write: .correlation/tool-xyz.pre.json
       → devAssist hook fires (via hook chain)
         → Scans file: DETECTS SQL Injection (rule_id 1001)
         → Returns: REJECT (blocks write)
       → Write operation fails
       → PostToolUse does NOT fire (because write failed)

T+5s   Claude submits next prompt/tool call
       → Governance PreToolUse fires again
         → sweepOrphanedPre() runs
           → Finds: .correlation/tool-xyz.pre.json (age=3s)
           → Checks: .correlation/tool-xyz.post (NOT found)
           → Inference: "Tool was blocked"
         → Emit: ActivityEvent {
             kind: "tool.third_party_enforcement",
             tool: "Write",
             thirdPartyTool: "cx-devassist",
             thirdPartyAction: "block"
           }
         → Spool to governance API

T+6s   Governance API receives spooled event
       → Ingest processor handles it
       → Creates policy_action record:
         {
           policy_id: null,
           action_type: "BLOCK",
           violation_type: "third_party_enforcement",
           resource_type: "tool",
           resource_id: "Write",
           reason: "Third-party hook blocked Write",
           matched_rule: "cx-devassist",
           occurred_at: "2026-08-20T14:35:00Z"
         }

T+10s  Governance UI shows event in Activity Timeline
       → Orange badge: "cx-devassist blocked Write"
```

---

## Part 7: Test Execution Checklist

### Pre-Test Setup

- [ ] Verify governance API running: `curl http://localhost:8080/api/v1/health`
- [ ] Confirm hooks installed: `cat ~/.claude/settings.json | jq '.PreToolUse'`
- [ ] Check devAssist availability (enable if needed)
- [ ] Clear old test files: `rm -rf /tmp/governance-test && mkdir /tmp/governance-test`
- [ ] Check governance directories exist:
  ```bash
  ls -d ~/.checkmarx/governance/.{correlation,audit-queue,hook-verdicts}
  ```

### During Test

- [ ] Have Claude Code create one vulnerable code sample (A, B, or C)
- [ ] Observe if write is blocked
- [ ] Note timestamp of block attempt
- [ ] Submit another action to trigger `sweepOrphanedPre()` scan

### Post-Test Validation

- [ ] [ ] Check `.correlation/` for orphaned `.pre.json` file
  ```bash
  ls -la ~/.checkmarx/governance/.correlation/ | grep "\.pre\.json"
  ```

- [ ] Check spool queue for event
  ```bash
  tail ~/.checkmarx/governance/audit-queue/*.jsonl | jq '.kind'
  ```

- [ ] Query API for policy actions
  ```bash
  curl -s -H "Authorization: Bearer cx-dev-api-key" \
    'http://localhost:8080/api/v1/policy-actions?action_type=BLOCK&limit=5' \
    | jq '.'
  ```

- [ ] Verify event in activity timeline
  ```bash
  curl -s -H "Authorization: Bearer cx-dev-api-key" \
    'http://localhost:8080/api/v1/activity-events?kind=tool.third_party_enforcement' \
    | jq '.'
  ```

---

## Part 8: Troubleshooting

### Problem: Block Not Detected

**Symptom:** Write completes even though it should be blocked

**Diagnosis:**
```bash
# Check if devAssist hook is in the hook chain
cat ~/.claude/settings.json | jq '.PreToolUse[].hooks[].command' | grep -i devassist

# Check ASCA version
cx asca --version

# Look for devAssist errors in logs
grep -r "devassist" ~/.checkmarx/governance/logs/
```

**Solution:**
- Enable devAssist plugin in Claude Code
- Install ASCA engine: `cx asca install`
- Verify hook chain order (governance first, then devAssist)

---

### Problem: Block Detected but Not Recorded in Governance

**Symptom:** File blocked but no `tool.third_party_enforcement` event appears

**Diagnosis:**
```bash
# Check if sweepOrphanedPre is running
grep "sweepOrphanedPre" ~/.checkmarx/governance/logs/*.log

# Check if spool is enabled
cat ~/.checkmarx/governance/audit-queue/batch-*.jsonl | wc -l

# Check governance API connectivity
curl -v http://localhost:8080/api/v1/health
```

**Solution:**
- Verify governance API is running
- Check auth token is valid
- Ensure `cx_governance_url` is configured
- Try manual spool flush: `cx hooks governance sync`

---

### Problem: Post-Test Cleanup

```bash
# Clear test files
rm -rf /tmp/governance-test

# Clear correlation files
rm -f ~/.checkmarx/governance/.correlation/*.{pre.json,post}

# Clear spool (archive old events)
mv ~/.checkmarx/governance/audit-queue/batch-*.jsonl \
   ~/.checkmarx/governance/audit-queue/.archive/
```

---

## Part 9: Expected Outputs

### Success Case: Block Detected and Recorded

```
✅ Governance Timeline Shows:
   T+5s  [Orange Block Icon] cx-devassist blocked Write
         tool: Write
         action: BLOCK
         confidence: definitive
         message: "cx-devassist blocked Write"

✅ API Query Returns:
   {
     "actionType": "BLOCK",
     "violationType": "third_party_enforcement",
     "severity": "high",
     "resourceType": "tool",
     "resourceId": "Write",
     "reason": "Third-party hook blocked Write",
     "thirdPartyTool": "cx-devassist",
     "attributionConfidence": "definitive",
     "occurredAt": "2026-08-20T14:35:00Z"
   }

✅ Correlation Files Show:
   .correlation/tool-xyz.pre.json  (exists, age > 5 sec)
   .correlation/tool-xyz.post      (does NOT exist)
   → Inference: Tool was blocked
```

---

## Part 10: Next Steps After Successful Test

1. **If detection works:**
   - Enable Mechanism B (File-based IPC with vulnerability details)
   - Modify devAssist to write `.hook-verdicts/` file with ASCA findings
   - Extend schema to store rule details

2. **If detection fails:**
   - Check devAssist logs for errors
   - Verify hook execution order
   - Enable debug logging in governance hooks

3. **For production:**
   - Schedule regular blocking tests
   - Set up alerts for high block rates
   - Dashboard to visualize security violations by rule

---

## Reference: Governance Event Schema

**Event that should be generated:**

```json
{
  "eventId": "evt-20260820-143500-abc123",
  "timestamp": "2026-08-20T14:35:00.000Z",
  "kind": "tool.third_party_enforcement",
  "sessionId": "sess-abc123",
  "machineId": "dev-machine-001",
  "agentId": "claude-code-001",
  "userId": "anjali",
  "agent": {
    "type": "claude-code",
    "id": "claude-code-001"
  },
  "tool": {
    "category": "third_party",
    "name": "Write",
    "rawName": "Write"
  },
  "thirdPartyTool": "cx-devassist",
  "thirdPartyAction": "block",
  "attributionConfidence": "definitive",
  "decision": {
    "action": "BLOCK",
    "agentMessage": "cx-devassist blocked Write"
  }
}
```

---

## Success Criteria

✅ **Test Passes When:**

1. Claude Code attempts to write vulnerable code
2. devAssist blocks the write operation
3. Governance hook detects the block (orphaned .pre.json after 5 seconds)
4. ActivityEvent is generated with `kind: "tool.third_party_enforcement"`
5. Event is spooled to governance API
6. Policy action record is created in database
7. Event appears in governance UI timeline

**Current Status:** Ready to execute test  
**Next Action:** Verify devAssist is enabled, then trigger test scenario
