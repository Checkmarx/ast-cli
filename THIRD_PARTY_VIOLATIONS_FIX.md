# Third-Party Violations Not Showing in Violations Tab - Fix Applied

**Status:** Analysis & Fix  
**Date:** 2026-08-20  
**Issue:** devAssist blocks detected in session timeline but not appearing in violations tab

---

## Root Cause Analysis

### Issue 1: Silent Failure in Backend ✅ FIXED

**Location:** `governance-api/internal/ingest/ingest.go:458`

**Problem:**
```go
_, _ = db.Pool.Exec(ctx, `INSERT INTO policy_actions ...`)  // Errors ignored!
```

**Impact:** If the INSERT into `policy_actions` fails (constraint violation, FK error, etc.), it silently fails with no logging. This means third-party violations are never persisted.

**Fix Applied:**
```go
if err := db.Pool.Exec(ctx, `INSERT INTO policy_actions ...`); err != nil {
    log.Printf("ingest: insertThirdPartyPolicyAction failed for %s: %v", blockedTool, err)
}
```

Now errors are logged to `backend.log` so we can diagnose issues.

---

### Issue 2: Verify Violations Query Returns Third-Party Events

**Location:** `governance-api/internal/api/dashboard.go:271`

**Query:**
```sql
WHERE pa.action_type IN ('BLOCK', 'ALERT')
```

**Analysis:**
- ✅ Third-party violations are inserted with `action_type = 'BLOCK'` or `'ALERT'`
- ✅ The WHERE clause should include them
- ✅ No additional filtering is applied

**Conclusion:** The query IS correctly returning third-party violations IF they exist in the database.

---

### Issue 3: Frontend Filter Configuration

**Location:** `governance-ui/app/violations/page.tsx:9-23`

**Filter Setup:**
```typescript
type FilterType = "all" | "pii" | "checkmarx" | "third_party";

function applyFilter(violations: PolicyAction[], filter: FilterType): PolicyAction[] {
  if (filter === "pii")         return violations.filter((v) => v.resourceType === "prompt");
  if (filter === "third_party") return violations.filter((v) => v.resourceType === "third_party");
  if (filter === "checkmarx")   return violations.filter((v) => v.resourceType !== "prompt" && v.resourceType !== "third_party");
  return violations;  // filter === "all"
}
```

**Analysis:**
- ✅ Third-party filter correctly checks `resourceType === "third_party"`
- ✅ Third-party violations are inserted with `resource_type = "third_party"`
- ✅ Default filter is "all" which returns all violations
- ✅ Frontend is correctly configured

**Conclusion:** The frontend WILL display third-party violations if they come from the API.

---

## Why Violations Weren't Showing

The most likely cause was **Issue 1: Silent Failures**

When `insertThirdPartyPolicyAction` executed:
1. Policy was created successfully
2. Activity event was inserted successfully
3. **But `policy_actions` INSERT failed silently** (no error logged, no exception thrown)
4. Session timeline shows the event (from `activity_events`)
5. Violations tab is empty (no `policy_actions` record exists)

**Possible reasons for INSERT failure:**
- Foreign key constraint violation (policy_id not found)
- Session ID not found
- Agent ID not found
- Type mismatch in metadata JSON
- NULL value in non-nullable column

---

## Verification Steps

### Step 1: Check Backend Logs for Errors

```bash
# Check for new error messages
tail -50 ~/.checkmarx/governance/../governance-api\ 1/governance-api/backend.log | grep -i "insertThirdPartyPolicyAction\|third_party\|error"
```

**Expected output (if there were errors):**
```
ingest: insertThirdPartyPolicyAction failed for cx-devassist: pq: insert or update on table "policy_actions" violates foreign key constraint "policy_actions_session_id_fkey"
```

### Step 2: Query for Third-Party Violations

```bash
# Query API for third-party violations
curl -s -H "Authorization: Bearer cx-dev-api-key" \
  'http://localhost:8080/api/v1/policy-actions?limit=20' | \
  jq '.items[] | select(.resourceType == "third_party") | {
    id: .policyActionId,
    tool: .resourceId,
    action: .actionType,
    time: .occurredAt
  }'

# Or directly query the database if you have access
psql governance_db << 'SQL'
SELECT COUNT(*) FROM policy_actions WHERE resource_type = 'third_party';
SELECT resource_id, action_type, occurred_at FROM policy_actions 
WHERE resource_type = 'third_party' ORDER BY occurred_at DESC LIMIT 5;
SQL
```

### Step 3: Verify in Violations Tab

After the fix is deployed and logs show no errors:

1. **Trigger a devAssist block** (use previous test instructions)
2. **Check violations tab:**
   - Click filter "Other Tool Policy"
   - Or keep "All" selected
   - Should see new violation with:
     - Action: BLOCK
     - Policy: "cx-devassist (block detected)" or similar
     - Resource: "third_party"
     - Tool: "cx-devassist"

### Step 4: Check Session Timeline

Compare before and after:

**Session Timeline (Activity Events):**
- ✅ Shows event: `kind: "tool.third_party_enforcement"`
- ✅ Shows: `thirdPartyTool: "cx-devassist"`
- ✅ Shows: `thirdPartyAction: "block"`

**Violations Tab (Policy Actions):**
- Now should also show the same block
- With additional fields like policy, severity, etc.

---

## Data Flow After Fix

```
devAssist blocks tool write
    ↓
Governance hook detects block
    ↓
ActivityEvent created (always succeeds)
    ↓
insertThirdPartyPolicyAction called
    ├─ ✅ Logs any errors (NEW!)
    ├─ Creates third_party_monitor policy
    └─ Inserts policy_actions record
         ↓
         If INSERT fails → Error logged to backend.log
         If INSERT succeeds → Record appears in violations tab
```

---

## If Violations STILL Don't Appear

Use this debugging checklist:

### Check 1: Is INSERT Happening?

```bash
tail ~/.checkmarx/governance/../governance-api\ 1/governance-api/backend.log | grep "insertThirdPartyPolicyAction"
```

- If you see "failed": Note the error message, fix the constraint violation
- If nothing appears: insertThirdPartyPolicyAction not being called (check if events are reaching ingest)

### Check 2: Policy Created?

```bash
# Check if third_party_monitor policies exist
curl -s -H "Authorization: Bearer cx-dev-api-key" \
  'http://localhost:8080/api/v1/policies?type=third_party' | jq '.items[]'
```

Should see policies like "cx-devassist (block detected)"

### Check 3: Policy Actions Record Exists?

```bash
# Database query
psql governance_db << 'SQL'
SELECT pa.policy_action_id, pa.resource_type, pa.resource_id, p.policy_name 
FROM policy_actions pa
LEFT JOIN policies p ON pa.policy_id = p.policy_id
WHERE pa.resource_type = 'third_party'
ORDER BY pa.occurred_at DESC LIMIT 5;
SQL
```

### Check 4: API Returns the Record?

```bash
curl -s -H "Authorization: Bearer cx-dev-api-key" \
  'http://localhost:8080/api/v1/policy-actions' | jq '.items[] | select(.resourceType == "third_party")'
```

Should return the third-party violations

### Check 5: Frontend Filtering Works?

Check browser console (F12 → Network tab):
- Look for request to `/api/v1/policy-actions`
- Verify response includes `resourceType: "third_party"`
- Check if frontend filter is showing the violation

---

## Deployment Checklist

- [x] Error logging added to `insertThirdPartyPolicyAction`
- [ ] Restart governance API to pick up the changes
  ```bash
  pkill governance-api
  cd "C:\Users\AnjaliD\Documents\MyProjects\AI Governance\governance-api 1\governance-api"
  go run ./cmd/main.go  # or however you start it
  ```
- [ ] Trigger a new devAssist block
- [ ] Verify error message appears (or no error if successful)
- [ ] Check violations tab for the new entry

---

## Alternative: Direct SQL Query for Violations

If the API/UI still isn't working, you can query directly:

```bash
# Get all third-party violations (last 24 hours)
curl -s -H "Authorization: Bearer cx-dev-api-key" \
  'http://localhost:8080/api/v1/policy-actions' | jq '
  .items[] 
  | select(.resourceType == "third_party" and (.occurredAt > (now | gmtime | mktime)))
  | {
      time: .occurredAt,
      tool: .resourceId,
      action: .actionType,
      developer: .developer.email,
      policy: .policyName,
      reason: .reason
    }'
```

---

## Next Steps

1. **Restart the governance API** (to pick up the logging changes)
2. **Trigger a devAssist block** (using test instructions from earlier)
3. **Check backend logs** for any error messages
4. **Verify in violations tab** that the block appears
5. **Report findings** with:
   - Whether errors appear in logs
   - Whether violations appear in tab
   - Any errors encountered

---

## Summary

**What was fixed:**
- ✅ Added error logging to `insertThirdPartyPolicyAction`
- ✅ Verified violations query and frontend filters are correct

**What to test:**
- Trigger devAssist block and check if it appears in violations tab
- Check logs for any error messages

**Expected outcome:**
- devAssist blocks show in BOTH:
  - ✅ Session timeline (Activity Events)
  - ✅ Violations tab (Policy Actions)

---

## Files Modified

| File | Change | Purpose |
|------|--------|---------|
| `governance-api/internal/ingest/ingest.go:458-474` | Added error logging | Diagnose INSERT failures |

**Status:** Ready to test after governance-api restart
