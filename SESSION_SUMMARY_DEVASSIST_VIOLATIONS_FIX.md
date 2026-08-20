# Session Summary: DevAssist Third-Party Blocking Detection & Violations Tab Fix

**Date:** 2026-08-20  
**Objective:** Enable devAssist plugin, detect its security blocks in AI Governance, and ensure they appear in the Violations tab (not just Session timeline)  
**Status:** Fix applied, rebuild completed — final verification pending

---

## Table of Contents

1. [Background & Goal](#background--goal)
2. [Architecture Discovered](#architecture-discovered)
3. [Step 1: Enabling DevAssist](#step-1-enabling-devassist)
4. [Step 2: Loose-Coupling Detection Mechanisms](#step-2-loose-coupling-detection-mechanisms)
5. [Step 3: Testing the Block Detection](#step-3-testing-the-block-detection)
6. [Step 4: Bug Found — Violations Tab Empty](#step-4-bug-found--violations-tab-empty)
7. [Step 5: Root Cause Analysis](#step-5-root-cause-analysis)
8. [Step 6: Fixes Applied](#step-6-fixes-applied)
9. [Step 7: Critical Discovery — Stale Binary](#step-7-critical-discovery--stale-binary)
10. [Current Status](#current-status)
11. [Remaining Steps](#remaining-steps)
12. [Files Modified](#files-modified)
13. [Key Learnings](#key-learnings)

---

## Background & Goal

The user has an **AI Governance system** that monitors AI coding agents (Claude Code, Cursor) via hooks, tracking policy violations, PII detection, tool usage, and more. Separately, there's a **cx-devassist plugin** that performs real-time security scanning (ASCA — Automated Security Code Analysis) and blocks vulnerable code writes.

**Goal:** Without tightly coupling devAssist to the governance backend, detect when devAssist blocks something and make sure that shows up as a **violation** in the governance dashboard — specifically in the **Violations tab**, not just the session activity timeline.

---

## Architecture Discovered

### Governance System Components

- **ast-cli (`cx.exe`)** — CLI binary that Claude Code/Cursor hooks invoke on every tool call. Contains the governance hook logic (Go source in `ast-cli/ast-cli/internal/commands/agenthooks/governance/`).
- **governance-api** — Backend Go service (`governance-api.exe`) listening on `localhost:8080`, backed by PostgreSQL. Ingests events, stores them, and serves the dashboard API.
- **governance-ui** — Next.js frontend with tabs: Sessions, Violations, Developers, Agents, MCP Servers, Policies, Compliance.

### Hook Flow

```
Claude Code tool call
    ↓
PreToolUse hook → cx.exe hooks gov-claude-pre-tool-use
    ↓ (governance policy: ALLOW/BLOCK/ALERT)
devAssist hook (separate, third-party) → may BLOCK
    ↓
PostToolUse hook → cx.exe hooks gov-claude-post-tool-use
```

### Existing Loose-Coupling Mechanism (Mechanism A — Correlation)

Located in `ast-cli/ast-cli/internal/commands/agenthooks/governance/correlation.go`:

1. On `PreToolUse` with governance verdict = ALLOW, write `.correlation/<tool_use_id>.pre.json`.
2. On `PostToolUse`, write `.correlation/<tool_use_id>.post` marker.
3. On the **next** hook invocation, `sweepOrphanedPre()` scans for `.pre.json` files older than 5 seconds **without** a matching `.post` file.
4. If found → infer the tool was blocked by a third party → emit an `ActivityEvent` with `kind: "tool.third_party_enforcement"`.

This was **already implemented and working** — no changes needed to detect *that* a block happened.

---

## Step 1: Enabling DevAssist

Found devAssist was disabled in Claude Code's plugin config:

```json
// ~/.claude/settings.json
"enabledPlugins": {
  "cx-devassist@cx-plugin-marketplace": false
}
```

**Fixed** by setting it to `true`.

---

## Step 2: Loose-Coupling Detection Mechanisms

Documented three approaches (see `DEVASSIST_LOOSE_COUPLING_DETECTION.md`):

| Mechanism | Description | Status |
|---|---|---|
| **A: Correlation** | File-based state machine infers blocks from missing PostToolUse | ✅ Already implemented |
| **B: Hook Verdict File IPC** | devAssist writes a JSON verdict file with full vulnerability details to `.hook-verdicts/` | ⏳ Partially implemented (governance can read it, devAssist doesn't write it yet) |
| **C: Prompt Pattern Detection** | Passively detect devAssist's injected remediation guidance in agent prompts | 🆕 Proposed, not implemented |

**Key insight:** No direct API call from devAssist to governance is required — Mechanism A already provides detection via passive file-system observation.

---

## Step 3: Testing the Block Detection

Created test artifacts:
- `test_vulnerable_code.py` — sample code with SQL injection, hardcoded secrets, command injection.
- `DEVASSIST_BLOCK_TEST_EXECUTION.md` — step-by-step test/verification guide.

**Result:** devAssist successfully blocked a `Write` operation, and the governance system's correlation mechanism detected it — the event appeared in the **session timeline** as:

```
Third-party
BLOCK → Write
Third-party (candidate)
"Third-party hook blocked Write"
```

---

## Step 4: Bug Found — Violations Tab Empty

**Symptom reported by user:** The block was visible in the **session activity/timeline** but did **NOT** appear in the **Violations tab** (neither "All" nor "Other Tool Policy" filter).

---

## Step 5: Root Cause Analysis

### Investigation Path

1. Checked `governance-api/internal/api/dashboard.go` — `ListViolations()` query filters `WHERE pa.action_type IN ('BLOCK', 'ALERT')`. This should include third-party blocks.
2. Checked `governance-ui/app/violations/page.tsx` — frontend filter for "Other Tool Policy" checks `resourceType === "third_party"`. Correctly configured.
3. Checked `governance-api/internal/ingest/ingest.go` — found that `insertThirdPartyPolicyAction()` used `_, _ = db.Pool.Exec(...)`, **silently swallowing any INSERT error**.
4. Queried the live API directly (`curl .../api/v1/violations`) and found third-party violations **were** present in the database, but with:
   ```json
   "thirdPartyTool": "unknown",
   "policyName": "unknown (block detected)"
   ```
   — i.e., records existed but with generic/unhelpful attribution, and the dates were **stale** (from 2026-08-19, not the fresh test).

### Two Distinct Problems Identified

**Problem A — Silent failures possible:** No error logging if `policy_actions` INSERT fails, making debugging impossible.

**Problem B — Attribution defaults to "unknown":** In `correlation.go`, when `sweepOrphanedPre()` can't find a single definitive hook in the session's hook inventory (because devAssist's hook isn't registered as a discoverable Claude Code hook the same way), `hookName` stays empty, and the resulting event has `ThirdPartyTool: ""`, which downstream becomes `"unknown"`.

**Problem C (discovered last) — Stale compiled binary:** The governance hook logic that runs during Claude Code sessions is the **compiled `cx.exe` binary**, not the Go source directly. Editing `correlation.go` has **zero effect** until `cx.exe` is rebuilt. This is why the fix appeared not to work initially — the user was still exercising the old binary logic.

---

## Step 6: Fixes Applied

### Fix 1 — Error Logging (`governance-api/internal/ingest/ingest.go`)

Added logging throughout the ingest pipeline so failures are no longer silent:

- Log every event as it's processed (`kind`, `tool`, `action`).
- Log developer resolution.
- Log third-party enforcement event receipt.
- Log policy creation/lookup (new vs. existing monitor policy).
- Log transaction commit failures with full error text.
- Log `INSERT INTO policy_actions` failures with full error text (previously `_, _ = ...` discarded errors).

### Fix 2 — Attribution Inference (`ast-cli/ast-cli/internal/commands/agenthooks/governance/correlation.go`)

In `emitThirdPartyBlock()`, added logic: if no hook could be attributed **and** the blocked tool was `Write`, infer `ThirdPartyTool = "cx-devassist"` (since Write-blocking security scans are devAssist's signature behavior) with `attributionConfidence = "candidate"`. Otherwise, use `"unknown-hook"` instead of empty string, for clearer downstream naming.

### Fix 3 — Tool Name Normalization (`governance-api/internal/ingest/ingest.go`)

Normalized `"unknown-hook"` → `"unknown-third-party"` in `insertThirdPartyPolicyAction()` for consistent display.

---

## Step 7: Critical Discovery — Stale Binary

After applying Fix 2 and asking the user to re-test, the violations API still showed **only stale data from 2026-08-19** with `thirdPartyTool: "unknown"` — no new records from the fresh test.

**Diagnosis:** Compared file timestamps:
```
cx.exe            → built 12:51 PM
correlation.go    → edited 14:59 PM   ← fix was NEWER than the binary
```

**Conclusion:** The Claude Code hooks call `cx.exe` (a compiled binary), not the Go source. All correlation.go edits were inert until recompiled.

**Fix:** Ran `go build -o cx.exe ./cmd/` in `ast-cli/ast-cli/` — build succeeded, new binary timestamped 15:07 PM.

---

## Current Status

| Component | Status |
|---|---|
| devAssist plugin | ✅ Enabled |
| Governance hooks installed | ✅ Confirmed (SessionStart, PreToolUse, PostToolUse, etc.) |
| Correlation detection (Mechanism A) | ✅ Working — blocks appear in session timeline |
| Error logging in ingest.go | ✅ Applied |
| Attribution inference fix in correlation.go | ✅ Applied |
| **cx.exe rebuilt with fixes** | ✅ Done (15:07) |
| governance-api backend | ✅ Restarted (latest PID running, listening on :8080) |
| **End-to-end verification with fresh block** | ⏳ **Pending — not yet completed this session (ran out of tokens)** |

---

## Remaining Steps

To complete verification in the next session:

1. **Close and fully reopen Claude Code** — required so the session picks up the newly built `cx.exe` (already done: binary rebuilt at 15:07; backend restarted at ~15:08).
2. **Trigger a fresh devAssist block**, e.g. ask Claude Code:
   ```
   Create a Python file with: query = "SELECT * FROM users WHERE id = " + str(user_id)
   ```
3. **Wait ~5–10 seconds**, then trigger another tool call/prompt so `sweepOrphanedPre()` runs and emits the event.
4. **Verify via API:**
   ```bash
   curl -s -H "Authorization: Bearer cx-dev-api-key" \
     'http://localhost:8080/api/v1/violations' | \
     jq '[.[] | select(.resourceType == "third_party")] | sort_by(.occurredAt) | reverse | .[0]'
   ```
   Expect: `"thirdPartyTool": "cx-devassist"`, recent `occurredAt` timestamp, `"policyName": "cx-devassist (block detected)"`.
5. **Check backend.log** for the new diagnostic log lines (`ingest: processing third_party_enforcement event...`, `ingest: created new third-party monitor policy...`) to confirm no INSERT errors.
6. **Check the Violations tab in the UI** — should now show the entry under "All" and "Other Tool Policy" filters, no longer just in the session timeline.
7. If it *still* doesn't appear, the enhanced logging will now reveal the exact SQL/constraint error — check `backend.log` for `insertThirdPartyPolicyAction failed for ...` messages and address the specific FK/constraint issue reported.

---

## Files Modified

### `ast-cli/ast-cli/internal/commands/agenthooks/governance/correlation.go`
- Function `emitThirdPartyBlock()`: added inference logic — Write-tool blocks with no hook attribution now default to `"cx-devassist"` (candidate confidence) instead of empty string.
- **Requires rebuild of `cx.exe`** to take effect (done).

### `governance-api/internal/ingest/ingest.go`
- `processOne()`: added logging of event kind/tool/action and developer resolution.
- Third-party enforcement branch: added logging around `upsertThirdPartyMonitorPolicy()` call and result.
- `upsertThirdPartyMonitorPolicy()`: added logging for existing-policy-found path, new-policy-creation path, and transaction commit failure (previously silent `return nil, defaultOurAction`).
- `insertThirdPartyPolicyAction()`: 
  - Changed `_, _ = db.Pool.Exec(...)` to capture and log the error.
  - Normalized empty/unknown-hook tool names to `"unknown-third-party"`.
- **Requires restart of `governance-api.exe`** to take effect (done).

### Other artifacts created during this session (documentation, not code):
- `DEVASSIST_GOVERNANCE_GAP_ANALYSIS.md`
- `DEVASSIST_LOOSE_COUPLING_DETECTION.md`
- `DEVASSIST_BLOCKING_TEST_PLAN.md`
- `DEVASSIST_BLOCK_TEST_EXECUTION.md`
- `DEVASSIST_TEST_STATUS.md`
- `THIRD_PARTY_VIOLATIONS_FIX.md`
- `VIOLATIONS_TAB_FIX_SUMMARY.md`
- `FIX_THIRD_PARTY_VIOLATIONS_COMPLETE.md`
- `DEVASSIST_IDENTIFICATION_FIX.md`
- `test_vulnerable_code.py`

---

## Key Learnings

1. **Loose coupling already existed and worked** — the governance system didn't need any direct integration with devAssist to detect its blocks; a file-based correlation heuristic (missing PostToolUse after an allowed PreToolUse) was sufficient.

2. **Silent error handling (`_, _ = ...`) hides real bugs.** Several Go functions discarded SQL errors, making it impossible to tell whether a record failed to insert or simply wasn't attempted. Always log errors, even in fire-and-forget paths, during development/debugging.

3. **Session timeline (`activity_events`) and Violations tab (`policy_actions`) are two separate tables/pipelines.** An event can succeed in populating one and silently fail in the other. Don't assume "I see it somewhere" means "it's fully processed."

4. **Attribution ambiguity is inherent to Mechanism A.** Because devAssist's hook may not appear in the discoverable "session hook inventory," the correlation mechanism can only guess who blocked a tool call. Heuristic inference (Write-block → likely devAssist) is a reasonable stopgap; Mechanism B (devAssist explicitly writing a verdict file) would give 100% accurate attribution and should be prioritized next.

5. **Critical/costly mistake: forgetting to rebuild the compiled binary.** The `ast-cli` hooks run as `cx.exe`, a compiled Go binary — not source code. Any fix to files under `ast-cli/ast-cli/internal/...` requires `go build -o cx.exe ./cmd/` before Claude Code's hooks will reflect the change. This caused significant confusion/wasted testing cycles ("still not working") when the real issue was simply a stale binary. **Always verify binary timestamp vs. source timestamp when a Go-CLI-based fix doesn't seem to take effect.**

6. **Two independent restarts are needed for full effect:**
   - `governance-api.exe` restart → picks up backend/ingest logic changes.
   - `cx.exe` rebuild + Claude Code restart → picks up hook/correlation logic changes.
   These are separate binaries/processes and must both be refreshed independently.

---

## Quick Reference: Verification Commands

```bash
# Restart backend
pkill governance-api
cd "C:\Users\AnjaliD\Documents\MyProjects\AI Governance\governance-api 1\governance-api"
./governance-api.exe > backend.log 2>&1 &

# Rebuild cx.exe (after any correlation.go / governance hook changes)
cd "C:\Users\AnjaliD\Documents\MyProjects\AI Governance\ast-cli\ast-cli"
go build -o cx.exe ./cmd/

# Check violations API for third-party entries
curl -s -H "Authorization: Bearer cx-dev-api-key" \
  'http://localhost:8080/api/v1/violations' | \
  jq '[.[] | select(.resourceType == "third_party")] | sort_by(.occurredAt) | reverse | .[0:5]'

# Tail backend logs for ingest diagnostics
tail -f "C:\Users\AnjaliD\Documents\MyProjects\AI Governance\governance-api 1\governance-api\backend.log"
```

---

**Status at end of session:** All code fixes applied and both binaries rebuilt/restarted. Final end-to-end confirmation (fresh block → correct `cx-devassist` attribution → appears in Violations tab) still needs to be run in the next session.
