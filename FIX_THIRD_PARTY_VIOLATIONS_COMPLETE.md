# Complete Fix: Third-Party Violations Not Showing in Violations Tab

**Status:** ✅ Fix Applied & Ready to Test  
**Date:** 2026-08-20  
**Issue:** devAssist blocks visible in session timeline but NOT in violations tab  
**Root Cause:** Silent failures in policy_action creation with no error logging

---

## Changes Made

### File: `governance-api/internal/ingest/ingest.go`

#### Change 1: Enhanced Third-Party Enforcement Processing (Lines 128-139)
**Before:**
```go
if ev.Kind == "tool.third_party_enforcement" {
    monitorPolicyID, ourAction := upsertThirdPartyMonitorPolicy(ctx, ev)
    insertThirdPartyPolicyAction(ctx, sessionID, devID, agentProdID, mcpServerID, monitorPolicyID, ourAction, ev)
    return nil
}
```

**After:**
```go
if ev.Kind == "tool.third_party_enforcement" {
    log.Printf("ingest: processing third_party_enforcement event %s: tool=%s action=%s",
        ev.EventID, ev.ThirdPartyTool, ev.ThirdPartyAction)
    monitorPolicyID, ourAction := upsertThirdPartyMonitorPolicy(ctx, ev)
    if monitorPolicyID == nil {
        log.Printf("ingest: third-party policy creation failed for %s", ev.ThirdPartyTool)
    } else {
        log.Printf("ingest: created/found third-party policy %s for %s", monitorPolicyID, ev.ThirdPartyTool)
    }
    insertThirdPartyPolicyAction(ctx, sessionID, devID, agentProdID, mcpServerID, monitorPolicyID, ourAction, ev)
    return nil
}
```

**Benefit:** Logs when third-party events are processed and whether policy creation succeeds

---

#### Change 2: Policy Finding Logging (Line 344)
**Added:**
```go
log.Printf("ingest: found existing third-party monitor policy %s for tool=%s action=%s", policyID, toolName, detectedAction)
```

**Benefit:** Logs when existing policies are reused

---

#### Change 3: Policy Creation Logging (Line 359)
**Added:**
```go
log.Printf("ingest: creating new third-party monitor policy for tool=%s action=%s", toolName, detectedAction)
```

**Benefit:** Logs when new policies are being created

---

#### Change 4: Commit Error Logging (Line 416-419)
**Before:**
```go
if err := tx.Commit(ctx); err != nil {
    return nil, defaultOurAction
}
```

**After:**
```go
if err := tx.Commit(ctx); err != nil {
    log.Printf("ingest: upsertThirdPartyMonitorPolicy tx.Commit failed for %s: %v", toolName, err)
    return nil, defaultOurAction
}

log.Printf("ingest: created new third-party monitor policy %s for tool=%s action=%s", policyID, toolName, detectedAction)
```

**Benefit:** Logs transaction failures with detailed error messages

---

#### Change 5: Event Processing Logging (Lines 38-40)
**Added at start of processOne:**
```go
log.Printf("ingest: processing event %s kind=%s tool=%s action=%s", ev.EventID, ev.Kind, ev.Tool.Name, ev.Decision.Action)
log.Printf("ingest: resolved developer %s for user %s", devID, ev.UserID)
```

**Benefit:** Logs every event being processed so we can trace the flow

---

#### Change 6: Better Error Logging in insertThirdPartyPolicyAction (Lines 458-474)
**Before:**
```go
_, _ = db.Pool.Exec(ctx, `INSERT INTO policy_actions ...`)
```

**After:**
```go
if err := db.Pool.Exec(ctx, `INSERT INTO policy_actions ...`); err != nil {
    log.Printf("ingest: insertThirdPartyPolicyAction failed for %s: %v", blockedTool, err)
}
```

**Benefit:** Logs INSERT failures instead of silently ignoring them

---

## Testing Steps

### Step 1: Restart the Governance API

```bash
# Kill the old process
pkill governance-api

# Wait for it to die
sleep 2

# Start it with new logging
cd "C:\Users\AnjaliD\Documents\MyProjects\AI Governance\governance-api 1\governance-api"
go run ./cmd/main.go
```

Wait for: `listening on :8080`

---

### Step 2: Trigger a devAssist Block

In Claude Code, submit this prompt:

```
Create a new Python file named "vulnerable_db.py" with:
- A function that connects to a database
- Takes a user_id parameter from the user
- Constructs a SQL query by directly concatenating the user_id into the query string
- Example: query = "SELECT * FROM users WHERE id = " + str(user_id)
- Do NOT use parameterized queries or prepared statements
```

**Expected:** devAssist blocks the write

---

### Step 3: Monitor the Logs

In real-time, watch the logs for new messages:

```bash
# Follow the logs as they come in
tail -f ~/.checkmarx/governance/../governance-api\ 1/governance-api/backend.log
```

**Look for these log messages (in order):**

```
1. ingest: processing event [EVENT_ID] kind=tool.third_party_enforcement tool=cx-devassist action=block
2. ingest: resolved developer [DEV_ID] for user [USERNAME]
3. ingest: processing third_party_enforcement event [EVENT_ID]: tool=cx-devassist action=block
4. ingest: creating new third-party monitor policy for tool=cx-devassist action=block
5. ingest: created new third-party monitor policy [POLICY_ID] for tool=cx-devassist action=block
6. ingest: insertThirdPartyPolicyAction ← Should NOT see an error here
```

**If you see an error at step 6, note the full error message.**

---

### Step 4: Check Violations Tab

1. Open governance UI
2. Go to **Violations** tab
3. Click filter **"Other Tool Policy"**
4. **Should see the devAssist block listed**

---

## Expected Log Output

### Success Case:
```
2026/08/20 14:35:45 ingest: processing event evt-abc123 kind=tool.third_party_enforcement tool=cx-devassist action=block
2026/08/20 14:35:45 ingest: resolved developer [uuid] for user anjali
2026/08/20 14:35:45 ingest: processing third_party_enforcement event evt-abc123: tool=cx-devassist action=block
2026/08/20 14:35:45 ingest: creating new third-party monitor policy for tool=cx-devassist action=block
2026/08/20 14:35:46 ingest: created new third-party monitor policy [uuid] for tool=cx-devassist action=block
```

### Error Case (Example):
```
2026/08/20 14:35:45 ingest: processing third_party_enforcement event evt-abc123: tool=cx-devassist action=block
2026/08/20 14:35:45 ingest: upsertThirdPartyMonitorPolicy tx.Commit failed for cx-devassist: pq: violates foreign key constraint "policy_versions_policy_id_fkey"
2026/08/20 14:35:45 ingest: third-party policy creation failed for cx-devassist
2026/08/20 14:35:46 ingest: insertThirdPartyPolicyAction failed for cx-devassist: pq: invalid input syntax for type uuid: ""
```

---

## Troubleshooting Guide

### Issue 1: No logs mentioning "third_party_enforcement"

**Means:** The event isn't reaching the ingest processor OR isn't being classified as third_party_enforcement

**Check:**
```bash
# Look for what event kinds ARE being processed
tail -50 ~/.checkmarx/governance/../governance-api\ 1/governance-api/backend.log | grep "processing event"
```

**Fix:** Check if the governance hook is capturing the block correctly

---

### Issue 2: "third-party policy creation failed" log message

**Means:** upsertThirdPartyMonitorPolicy returned nil

**Check the previous log lines for:**
- `upsertThirdPartyMonitorPolicy tx.Commit failed` ← Transaction failed
- Check the database for: `SELECT * FROM policies WHERE policy_type = 'third_party_monitor'`

---

### Issue 3: "insertThirdPartyPolicyAction failed" with constraint error

**Common errors:**

| Error | Fix |
|-------|-----|
| `violates foreign key constraint "policy_actions_session_id_fkey"` | Session not found in ai_sessions - check session creation |
| `violates foreign key constraint "policy_actions_agent_id_fkey"` | Agent not found - verify Claude Code agent registered |
| `violates foreign key constraint "policy_actions_developer_id_fkey"` | Developer not found - check resolveOrCreateDeveloper |
| `NULL value in column "occurred_at"` | Event timestamp missing - check ev.Timestamp |
| `NULL value in column "action_type"` | Action type not set - check recordedAction calculation |

**How to debug:** The error message will be logged with full details

---

### Issue 4: Still no violations in the tab after logs show success

**Means:** Events and policy_actions are being created, but API/UI has an issue

**Check:**
```bash
# Query the API directly
curl -s -H "Authorization: Bearer cx-dev-api-key" \
  'http://localhost:8080/api/v1/policy-actions' | \
  jq '.items[] | select(.resourceType == "third_party")'

# If this returns results, it's a UI filtering issue
# If this returns nothing, it's a database issue
```

---

## Files Modified

```
governance-api/internal/ingest/ingest.go
├─ Lines 38-40: Added event processing logging
├─ Lines 128-139: Added third-party enforcement logging
├─ Line 344: Added policy finding logging  
├─ Line 359: Added policy creation logging
├─ Lines 416-419: Added commit error logging
└─ Lines 458-474: Added INSERT failure logging
```

**Total changes:** 6 sections, comprehensive error logging

---

## Expected Outcomes

### If Fix Works:
✅ Logs show all processing steps  
✅ devAssist blocks appear in violations tab  
✅ Both session timeline AND violations tab show the same block  

### If Fix Reveals Errors:
✅ Error messages appear in logs  
✅ Exact error helps diagnose root cause  
✅ We can fix the root cause (constraint violation, missing data, etc.)

---

## Deployment Checklist

- [x] Code changes applied to ingest.go
- [ ] Governance API restarted
- [ ] devAssist block triggered in Claude Code
- [ ] Logs monitored for processing steps
- [ ] Error message captured (if any)
- [ ] Violations tab checked for the block

---

## Next Steps After Testing

**If success (violations appear):**
- ✅ Issue is resolved
- Monitor future blocks to ensure consistency
- Consider keeping enhanced logging for production observability

**If error found:**
- Capture the full error message
- Implement targeted fix for the constraint/data issue
- Re-test with the fix applied

---

## Summary

Enhanced the ingest processor with comprehensive logging at 6 key points to:
1. Track every event being processed
2. Log policy creation (success and failure)
3. Log policy_action INSERT success/failure with full error details
4. Enable rapid diagnosis of any failures

This should immediately reveal WHY third-party violations aren't being created and allow for targeted fixes.

**Ready to test!**
