# Prompt & File Path Tracking Analysis

**Date:** 2026-08-18  
**Status:** Requirements analysis for UI display and allowlist/blocklist capabilities  
**Objective:** Understand feasibility of tracking prompt text and file paths in audit/governance UI

---

## Executive Summary

### Current State (Verified via Code Inspection)

| Feature | Currently Tracked | Stored | Displayed in UI | Details |
|---------|-------------------|--------|-----------------|---------|
| **File Paths** | ✅ YES | ✅ YES (activity_events.tool_paths JSONB) | ❌ NOT YET | Paths extracted from Read/Write/Edit/Grep operations; stored but not surfaced in UI |
| **Prompt Text** | ❌ PARTIAL | ❌ PARTIAL (only length) | ❌ NO | Only prompt_length stored; full text not captured |
| **File Allowlist/Blocklist** | ❌ NO | ❌ NO | ❌ NO | Can be implemented via policies, but not yet |

### Recommendation

**Both features are feasible and complementary.**

- **File paths**: 80% done — just needs UI display + optional allowlist feature
- **Prompt text**: Needs 3-tier design choice due to privacy/storage tradeoffs; recommend **limited capture** (system prompts only, user prompts redacted in UI)

---

## 1. File Path Tracking — Current State & Enhancement

### What's Already Working

File paths are **already being extracted and stored**. Evidence:

1. **ast-cli normalization** (`normalize.go`, line 135):
   ```go
   case isFilesystemTool(toolName):
       return ToolInfo{
           Category: "filesystem",
           Name:     toolName,
           Paths:    extractPaths(rawInput),  // ← Extracts paths from tool input
       }
   ```

2. **extractPaths() implementation** (normalize.go, line ~190-210):
   - Parses file_path from `{file_path: "...", ...}` input
   - Returns `[]string` of all file paths accessed

3. **Database storage** (activity_events schema, line 14):
   ```sql
   tool_paths JSONB,  -- Stored as JSON array
   ```

4. **Ingest writes paths** (ingest.go, line 149-150):
   ```go
   if len(ev.Tool.Paths) > 0 {
       toolPaths, _ = json.Marshal(ev.Tool.Paths)
   }
   ```

### What's Missing

**Only the UI display layer** — the data exists in the database but is not shown.

### Enhancement 1: Display File Paths in Session Timeline

**What to add:**
1. UI column in activity events timeline: "File(s)" showing the file paths
2. Truncate long paths with hover tooltip for full path
3. Make paths clickable to navigate to file in UI (if applicable)

**Changes required:**

#### governance-ui: `components/panels/session-panel.tsx`

```typescript
// In the activity timeline table, add a new column after "Tool"

<td className="border p-2 text-sm">
  {event.tool_paths && event.tool_paths.length > 0 ? (
    <div className="space-y-1">
      {event.tool_paths.slice(0, 3).map((path, idx) => (
        <div 
          key={idx} 
          className="font-mono text-xs text-gray-700 truncate" 
          title={path}
        >
          {path.split('/').pop() || path} {/* Show filename only */}
        </div>
      ))}
      {event.tool_paths.length > 3 && (
        <div className="text-xs text-gray-500">
          +{event.tool_paths.length - 3} more
        </div>
      )}
    </div>
  ) : (
    <span className="text-gray-400">—</span>
  )}
</td>
```

**Effort:** Low (1-2 hours) — purely UI, no backend changes

---

### Enhancement 2: File Allowlist/Blocklist Feature

**What to add:**
1. Policy type: "filesystem" with patterns for allowed/blocked files
2. UI policy editor: regex patterns for `allowed_files` and `blocked_files`
3. Enforcement: Check file paths against policy before tool execution

**Changes required:**

#### Step 1: Extend Policy Schema (governance-api)

**New policy type definition in `models.go`:**

```go
type FilesystemPolicy struct {
    Type             string   `json:"type"`                 // "filesystem"
    AllowedFiles     []string `json:"allowed_files,omitempty"`     // whitelist patterns
    BlockedFiles     []string `json:"blocked_files,omitempty"`     // blacklist patterns
    BlockedDirs      []string `json:"blocked_directories,omitempty"` // directory prefixes
    Action           string   `json:"action"`               // ALERT | BLOCK
    Reason           string   `json:"reason"`
    AgentMessage     string   `json:"agentMessage,omitempty"`
}
```

**Examples:**

```json
{
  "type": "filesystem",
  "blockable_files": [
    ".*\\.env$",           // Block .env files
    ".*\\.pem$",           // Block certificates
    ".*\\.key$",           // Block keys
    "/etc/.*",             // Block /etc system files
    ".*\\/secrets\\/.*"    // Block /secrets/* anywhere
  ],
  "allowed_files": [
    ".*\\.md$",            // Allow markdown (if needed)
    "/home/[^/]+/projects.*"  // Allow home/projects
  ],
  "action": "BLOCK",
  "agentMessage": "Access to sensitive files is not permitted"
}
```

#### Step 2: Policy Evaluation (ast-cli)

**In `pack.go`, add filesystem policy evaluator:**

```go
func (p *GovernancePack) evaluateFilesystem(tool ToolInfo) Decision {
    for _, policy := range p.Policies {
        if policy.Type != "filesystem" {
            continue
        }
        
        for _, path := range tool.Paths {
            // Check blocklist
            for _, pattern := range policy.BlockedFiles {
                if matchPattern(path, pattern) {
                    return Decision{
                        Action:       policy.Action,
                        PolicyID:     policy.PolicyID,
                        PolicyName:   policy.PolicyName,
                        MatchedValue: path,
                        AgentMessage: policy.AgentMessage,
                    }
                }
            }
            
            // Check allowlist (if present)
            if len(policy.AllowedFiles) > 0 {
                allowed := false
                for _, pattern := range policy.AllowedFiles {
                    if matchPattern(path, pattern) {
                        allowed = true
                        break
                    }
                }
                if !allowed && policy.Action != "" {
                    return Decision{
                        Action:       policy.Action,
                        PolicyID:     policy.PolicyID,
                        PolicyName:   policy.PolicyName,
                        MatchedValue: path,
                        AgentMessage: policy.AgentMessage,
                    }
                }
            }
        }
    }
    return Decision{Action: "ALLOW"}
}
```

#### Step 3: Hook Integration

**In `handleClaudePreToolUse()`, call filesystem evaluator:**

```go
decision := pack.Evaluate(event)  // Existing line

// Add filesystem-specific evaluation if tool is filesystem
if event.Tool.Category == "filesystem" {
    fsDecision := pack.evaluateFilesystem(event.Tool)
    if fsDecision.Action != "ALLOW" {
        decision = fsDecision
    }
}
```

**Effort:** Medium (8-12 hours) — involves policy schema extension, evaluation logic, and UI policy editor

---

## 2. Prompt Text Tracking — Current State & Options

### Current State (Verified)

**Only prompt length is tracked:**

```go
type ActivityPrompt struct {
    Length int `json:"length"`
}
```

```sql
prompt_length INTEGER,  -- Only length stored, not text
```

### Why Prompt Text is Complex

**Privacy concerns:**
- Prompts may contain: system architecture, customer data, passwords, API keys, etc.
- Storing verbatim violates GDPR right to deletion, CCPA, SOC2
- Compliance risk if prompts are cached/indexed unintentionally

**Storage cost:**
- Claude prompts can be 10K-100K tokens (30KB-300KB)
- Storing for every prompt.submit event = massive DB growth
- Example: 1000 sessions × 10 prompts × 50KB = 500MB per day

**Practical utility:**
- For audit: knowing "user submitted 868-char prompt" is often enough
- For governance: what matters is *what Claude did*, not what user asked
- If you need to audit prompts, that's usually a compliance requirement → encrypt and separate

### Three Options (Increasing Complexity & Risk)

#### Option A: Don't Track Prompts (Current State)

**Pros:**
- Zero privacy risk
- Zero storage cost
- Compliant with all regulations
- Simplest implementation

**Cons:**
- Limited audit trail for "what was Claude asked"
- Can't reconstruct user intent from activity log

**When to use:** Most organizations (default safe choice)

---

#### Option B: Limited Prompt Capture (Recommended)

**What to track:**
- Only **system context prompts** (governance instructions, policy summaries)
- **Redact user prompts** in UI, show only length and timestamp
- Store encrypted in a separate, rotatable table with retention policy

**Pros:**
- Captures governance context without user privacy risk
- Helps debug policy decisions
- Compliant (separate PII table, retention policy, encryption)
- Modest storage cost (system prompts are small)

**Cons:**
- Slightly more complex implementation
- Requires encryption key management

**Implementation:**

1. **New table: `system_prompts`** (encrypted, optional, separate from activity_events)
   ```sql
   CREATE TABLE system_prompts (
       prompt_id UUID PRIMARY KEY,
       session_id UUID,
       prompt_text_encrypted BYTEA,  -- encrypted with key-vault
       created_at TIMESTAMP,
       retention_days INTEGER DEFAULT 30  -- auto-delete after 30 days
   );
   ```

2. **In activity_events.sql, add:**
   ```sql
   user_prompt_summary VARCHAR(100),  -- "user submitted 2500-char prompt"
   system_prompt_id UUID REFERENCES system_prompts(prompt_id)  -- link only
   ```

3. **In UI:** Show user prompt summary only (length, timestamp), no text
4. **Admin dashboard (optional):** Encrypted system prompts visible to security team only

**Effort:** Medium-High (12-18 hours) — encryption/decryption, separate table, UI changes

---

#### Option C: Full Prompt Capture (Not Recommended)

**What to track:**
- Full user prompt text stored plaintext in activity_events

**Pros:**
- Complete audit trail
- Easiest to implement (just add a column)

**Cons:**
- **GDPR violation** — verbatim user data without consent
- **SOC2 violation** — logging PII without encryption
- **CCPA liability** — can't delete user prompts on request
- **Massive storage** — 500MB+ per day
- **Security risk** — if DB is breached, user data exposed

**When to use:** Only if:
- You have explicit user consent
- Data is encrypted at rest
- You have automated retention/deletion
- Your compliance team approves

**Not recommended for most orgs.**

---

## 3. Recommended Implementation Path

### Tier A: File Path Display (High Priority)

**Do this first.** Low effort, immediate value, zero risk.

1. Add file paths column to session timeline UI
2. Show filename only with tooltip for full path
3. Create 1 new UI component (reusable)

**Effort:** 1-2 hours  
**Risk:** None  
**Value:** Immediate auditability of file access

### Tier B: File Policy (Optional, Medium Priority)

**Do this if you need file-level enforcement.**

1. Extend policy schema to support `blockable_files`, `allowed_files`
2. Add filesystem evaluator to pack
3. Create policy editor UI for file patterns
4. Hook into PreToolUse handler

**Effort:** 8-12 hours  
**Risk:** Medium (policy evaluation logic needs testing)  
**Value:** Prevent accidental access to sensitive files (credentials, configs, source)

### Tier C: Limited Prompt Capture (Low Priority, Optional)

**Do this only if compliance requires it.**

1. Separate table for system prompts (encrypted)
2. Store prompt summary + reference in activity_events
3. Admin-only UI to view system prompts
4. Auto-retention policy

**Effort:** 12-18 hours  
**Risk:** High (encryption/key management, compliance)  
**Value:** Reconstructs governance decision context

---

## 4. Database Schema Changes

### For Tier A (File Path Display)

**No database changes needed.** Data already exists in `activity_events.tool_paths`.

### For Tier B (File Policy)

**Extend `policy_versions` table to store filesystem policies:**

```sql
-- Add to policy_versions definition:
-- definition already stores full policy JSON, so filesystem policies can be:
{
  "type": "filesystem",
  "blockable_files": [".*\\.env$", ".*\\.pem$"],
  "action": "BLOCK",
  "agentMessage": "Credential files cannot be accessed"
}
```

**No new table; use existing `definition` JSONB column.**

### For Tier C (Prompt Capture)

```sql
-- NEW TABLE: system_prompts (optional, for compliance)
CREATE TABLE system_prompts (
    prompt_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES ai_sessions(session_id) ON DELETE CASCADE,
    prompt_text_encrypted BYTEA NOT NULL,  -- AES-256-GCM encrypted
    encryption_key_version INT,            -- for key rotation
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    retention_until TIMESTAMP WITH TIME ZONE,  -- auto-delete date
    
    INDEX idx_system_prompts_session ON session_id,
    INDEX idx_system_prompts_retention ON retention_until
);

-- MODIFY activity_events table:
ALTER TABLE activity_events ADD COLUMN user_prompt_length INT;      -- length only
ALTER TABLE activity_events ADD COLUMN system_prompt_id UUID REFERENCES system_prompts(prompt_id);
```

---

## 5. API Changes (Tier A + B + C)

### Tier A (File Paths) — No API Changes

Session timeline API already returns `tool_paths` in activity events; UI just needs to display it.

### Tier B (File Policy) — No New APIs

File policy is stored as part of existing `policy_versions.definition` (JSONB).

### Tier C (Prompt Capture) — New Endpoints (Optional)

```go
// Admin-only endpoint to view system prompts
GET /api/v1/admin/sessions/{id}/system-prompts
Response: {
    prompts: [
        {
            promptId: "...",
            createdAt: "...",
            decrypted_text: "..." // Only for admin role
        }
    ]
}

// Compliance export (encrypted)
GET /api/v1/admin/export/sessions/{id}/with-prompts?format=csv
Response: encrypted CSV with system prompts
```

---

## 6. UI Changes

### Tier A: File Paths Display

**File:** `components/panels/session-panel.tsx`

```typescript
// Add column to activity timeline table
<thead>
  <tr>
    <th>Event</th>
    <th>Tool</th>
    <th>Files</th>           {/* NEW */}
    <th>Decision</th>
    <th>Time</th>
  </tr>
</thead>

<tbody>
  {events.map(event => (
    <tr key={event.eventId}>
      <td>{event.kind}</td>
      <td>{event.tool_name}</td>
      <td>
        {/* NEW: File paths component */}
        {event.tool_paths?.length > 0 ? (
          <div className="space-y-1 text-xs">
            {event.tool_paths.slice(0, 2).map((path, i) => (
              <div key={i} className="font-mono text-gray-700 truncate" title={path}>
                {path.split('/').pop()}
              </div>
            ))}
            {event.tool_paths.length > 2 && (
              <div className="text-gray-500">+{event.tool_paths.length - 2}</div>
            )}
          </div>
        ) : <span className="text-gray-400">—</span>}
      </td>
      ...
    </tr>
  ))}
</tbody>
```

**Effort:** 2 hours

### Tier B: File Policy Editor

**New page:** `app/policies/create` — extend existing policy creation UI

```typescript
// Add new "Filesystem" policy type option
// Show pattern inputs for:
// - blockable_files: textarea with regex patterns (one per line)
// - allowed_files: textarea with optional allowlist patterns

// Example form:
<form>
  <fieldset>
    <legend>Policy Type</legend>
    <select>
      <option>Shell Commands</option>
      <option>MCP Tools</option>
      <option>Filesystem Paths (NEW)</option>
    </select>
  </fieldset>
  
  <fieldset>
    <legend>Blocked File Patterns (Regex)</legend>
    <textarea placeholder="\\.env$&#10;\\.pem$&#10;/etc/.*" />
    <p className="text-xs">One pattern per line</p>
  </fieldset>
  
  <fieldset>
    <legend>Action</legend>
    <select>
      <option>ALERT</option>
      <option>BLOCK</option>
    </select>
  </fieldset>
</form>
```

**Effort:** 6 hours (form + validation + preview matching)

### Tier C: System Prompts (Admin Only)

**New admin page:** `app/admin/audit/system-prompts`

```typescript
// Lists system prompts from sessions
// Requires admin role
// Shows: timestamp, decrypted text, session link
// Export button: download as encrypted CSV
```

**Effort:** 8 hours (encryption handling, admin gating, export)

---

## 7. Threat Model & Risk Mitigation

### Risk: Storing User Prompts

**Threat:** GDPR violation if user prompt contains PII

**Mitigation (Option B):**
- Don't store user prompts at all
- Only store system/governance prompts
- Store separately (different table, different retention)
- Encrypt with AES-256
- Auto-delete after 30 days
- Audit access logs

### Risk: File Access Evasion

**Threat:** User tries to access blocked file via different tool/path

**Mitigation:**
- Evaluate ALL filesystem tools (Read, Write, Edit, Grep, LS)
- Normalize paths (resolve symlinks, relative → absolute)
- Pattern matching is case-insensitive on Windows
- Test with known evasion attempts (relative paths, symlinks, etc.)

### Risk: Performance Impact

**Threat:** Filesystem policy evaluation slows down every tool call

**Mitigation:**
- Precompile regex patterns at policy load time
- Cache pattern compilation in `GovernancePack`
- Short-circuit on first blocklist match (BLOCK vs ALERT)
- Typical evaluation: <1ms

---

## 8. Test Strategy

### Tier A (File Paths Display)

- **Unit:** File path extraction (`extractPaths()`)
- **UI:** Session timeline renders file paths correctly, truncation works, tooltip shows full path
- **E2E:** Read a file → timeline shows filename + full path on hover

### Tier B (File Policy)

- **Unit:** Filesystem policy evaluator with test patterns
- **Integration:** File access against blocklist/allowlist policies
- **E2E:** Create file policy → try to read blocked file → policy enforced
- **Edge cases:** Relative paths, symlinks, case sensitivity, path normalization

### Tier C (Prompt Capture)

- **Unit:** Encryption/decryption of prompts
- **Integration:** System prompts stored and encrypted, retention dates honored
- **Security:** Only admins can decrypt; access logged
- **Compliance:** Export format meets audit requirements

---

## 9. Implementation Priority & Effort Summary

| Tier | Feature | Effort | Risk | Value | Recommendation |
|------|---------|--------|------|-------|-----------------|
| **A** | File paths display | 2h | None | High | **Do immediately** |
| **B** | File policy/blocklist | 10h | Medium | High | **Do next** |
| **C** | Prompt capture (encrypted) | 16h | High | Medium | **Only if compliance required** |

**Total for A+B:** ~12 hours, covers file audit and enforcement  
**Total for A+B+C:** ~28 hours, adds prompt audit trail

---

## 10. Recommended Next Steps

### This Week (Tier A):
1. Add file paths column to session timeline UI
2. Test with real file operations
3. Deploy to staging

### Next Sprint (Tier B):
1. Extend policy schema for filesystem rules
2. Implement filesystem evaluator in pack.go
3. Create policy editor UI for file patterns
4. End-to-end test with blocklist scenarios

### Later (Tier C, if needed):
1. Design system-prompts table with encryption
2. Implement prompt capture in ingest
3. Build admin UI for prompt audit
4. Get compliance approval before deploying

---

## Appendix: Code Locations (Already Verified)

**File path extraction (already working):**
- `ast-cli/internal/commands/agenthooks/governance/normalize.go:135` — `extractPaths()`
- Storage: `governance-api/internal/db/migrations/002_activity_events.sql:14` — `tool_paths JSONB`
- Ingest: `governance-api/internal/ingest/ingest.go:149-150` — writes tool_paths

**Prompt capture (partial):**
- Model: `governance-api/internal/models/models.go:225` — `ActivityPrompt struct`
- Storage: `governance-api/internal/db/migrations/002_activity_events.sql:18` — `prompt_length INTEGER`
- Ingest: `governance-api/internal/ingest/ingest.go:157-160` — writes prompt_length only

**UI display (missing):**
- Session panel: `governance-ui/components/panels/session-panel.tsx` — needs file paths column
- Activity timeline: Needs to unpack and display `tool_paths` JSONB

