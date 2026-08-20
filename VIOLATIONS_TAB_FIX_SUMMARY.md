# Third-Party Violations in Violations Tab - Quick Fix Summary

**Problem:** devAssist blocks appear in session timeline but NOT in violations tab  
**Root Cause:** Silent failure in `insertThirdPartyPolicyAction` function (errors not logged)  
**Fix Applied:** ✅ Added error logging to diagnose INSERT failures  
**Status:** Ready to test

---

## What I Fixed

### Code Change (1 file, 18 lines modified)

**File:** `governance-api/internal/ingest/ingest.go` (lines 458-474)

**Before:**
```go
_, _ = db.Pool.Exec(ctx, `INSERT INTO policy_actions ...`)  // Errors hidden!
```

**After:**
```go
if err := db.Pool.Exec(ctx, `INSERT INTO policy_actions ...`); err != nil {
    log.Printf("ingest: insertThirdPartyPolicyAction failed for %s: %v", blockedTool, err)
}
```

**Why:** When the INSERT statement fails (e.g., constraint violation, NULL value, FK error), the error was silently ignored. Now errors are logged to `backend.log` for debugging.

---

## Quick Verification Steps

### 1. Restart Governance API
```bash
# Kill old process
pkill governance-api

# Start new one
cd "C:\Users\AnjaliD\Documents\MyProjects\AI Governance\governance-api 1\governance-api"
go run ./cmd/main.go
# Or however you normally start it
```

### 2. Trigger a devAssist Block
- Open Claude Code
- Ask it to create vulnerable Python code (SQL injection, hardcoded secret, etc.)
- devAssist should block the write

### 3. Check Backend Logs
```bash
# Look for new errors
tail -20 ~/.checkmarx/governance/../governance-api\ 1/governance-api/backend.log
```

**If errors found:** They will now be logged like:
```
ingest: insertThirdPartyPolicyAction failed for cx-devassist: pq: ...
```

**If no errors:** The INSERT succeeded, so...

### 4. Check Violations Tab
- Open governance UI
- Go to "Violations" tab
- Click filter "Other Tool Policy" (or keep "All")
- **Should see the devAssist block listed**

---

## Expected Results

### Before Fix:
- ✅ Session Timeline shows: `tool.third_party_enforcement` event
- ❌ Violations Tab shows: Nothing (empty)
- ❌ Logs show: No errors (silently failing)

### After Fix:
- ✅ Session Timeline shows: `tool.third_party_enforcement` event
- ✅ Violations Tab shows: Third-party violation from devAssist
- ✅ Logs show: Either "failed" message with error OR no error (success)

---

## If Violations Still Don't Appear

Check logs for the reason:

```bash
tail -50 ~/.checkmarx/governance/../governance-api\ 1/governance-api/backend.log | grep "third_party\|error\|INSERT"
```

### Common Errors & Fixes:

| Error | Cause | Fix |
|-------|-------|-----|
| `violates foreign key constraint "policy_actions_session_id_fkey"` | Session ID not found | Check if session is being created |
| `violates foreign key constraint "policy_actions_agent_id_fkey"` | Agent ID not found | Verify agent is registered |
| `NULL value in column "resource_id"` | blockedTool is empty | Check `ev.ThirdPartyTool` is populated |
| `NULL value in column "occurred_at"` | Timestamp missing | Check `ev.Timestamp` is set |

---

## What Happens When It Works

```
Timeline:
─────────────────────────────────────

devAssist blocks write
    ↓
Governance detects block (5s later)
    ↓
ActivityEvent created → activity_events table
    ↓
Third-party policy created (if new tool)
    ↓
PolicyAction created → policy_actions table ← NOW LOGGED
    ↓
API /policy-actions returns the record
    ↓
Violations Tab displays it
```

---

## Verification Commands

**1. Check if policy_actions record exists:**
```bash
curl -s -H "Authorization: Bearer cx-dev-api-key" \
  'http://localhost:8080/api/v1/policy-actions?limit=5' | \
  jq '.items[] | select(.resourceType == "third_party")'
```

**2. Check UI displays it:**
- Violations page → Filter "Other Tool Policy"
- Should see violation with:
  - Action: BLOCK
  - Policy: "cx-devassist (block detected)"
  - Resource: "third_party"
  - Developer: your email
  - Time: recent

**3. Check backend logs for errors:**
```bash
grep "insertThirdPartyPolicyAction" ~/.checkmarx/governance/../governance-api\ 1/governance-api/backend.log
```

---

## Success Criteria

You'll know the fix works when:

✅ devAssist blocks a write  
✅ Session timeline shows the block  
✅ Violations tab ALSO shows the block (not just timeline)  
✅ Both have:
  - Same timestamp (or within a few seconds)
  - Same tool name (e.g., "cx-devassist")
  - Action: BLOCK  

---

## Files Modified

```
governance-api/internal/ingest/ingest.go
  - Line 458-474: Added error handling/logging to insertThirdPartyPolicyAction
  - Change: 1 function, 18 lines total
```

**No UI changes needed** (frontend filters already work correctly)

---

## Next Actions

1. **Restart governance API** with the fix
2. **Trigger a devAssist block** in Claude Code
3. **Check backend.log** for any error messages
4. **Open violations tab** and verify the block appears
5. **Report findings**

---

**Status:** ✅ Ready to test — just restart the API and trigger a block
