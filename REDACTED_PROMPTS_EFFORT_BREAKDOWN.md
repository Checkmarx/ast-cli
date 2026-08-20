# Redacted User Prompts — Detailed Effort Breakdown

**Objective:** Capture full user prompt text in database, but display only redacted summary in UI  
**Security Model:** Prompts stored encrypted or in separate access-controlled table; full text never shown in regular UI

---

## Overview: What "Redacted" Means

### Storage Layer
- **Full prompt text** stored in database (encrypted or plaintext in restricted table)
- **Retention policy:** Auto-delete after 30 days
- **Access control:** Only admins/security can view

### UI Layer
- **User/developer view:** Shows only `"User submitted 2847-char prompt"` — no text
- **Activity timeline:** Hides prompt content, shows metadata only
- **Session detail:** "Prompt submitted • 2847 chars" (no disclosure of content)

### Admin/Compliance View
- **Optional admin dashboard:** Decrypt and view full prompts (audit trail logged)
- **Export:** Encrypted CSV for compliance review

---

## Implementation Approach: Two Strategies

### Strategy 1: "Separate Encrypted Table" (Recommended)

Store prompts in dedicated table, encrypted at rest, access-controlled.

**Pros:**
- Clean separation (prompts ≠ activity audit)
- Easy to implement retention policy (garbage collection)
- Easy to rotate encryption keys
- Zero compliance risk if done right

**Cons:**
- Slightly more complex schema (join required)
- Encryption overhead (AES-256-GCM encryption/decryption)

### Strategy 2: "Same Table, Redacted in UI" (Simpler)

Store prompts in `activity_events` table directly, redact in UI.

**Pros:**
- Simpler schema (no new table)
- Single query to get activity + prompts

**Cons:**
- Prompts visible to anyone with DB access
- Harder to implement retention (must delete rows)
- All activity tied to prompts (can't delete one without other)
- Compliance risk (PII in main audit table)

**Recommendation: Use Strategy 1** (Separate table) — slightly more effort but much safer.

---

## Effort Breakdown: Strategy 1 (Separate Encrypted Table)

### Phase 1: Infrastructure Setup (2 hours)

#### 1.1 Encryption Key Management (1 hour)

**What:** Set up encryption key storage and rotation

**Changes:**
- Create `encryption_keys` table in DB (for key rotation)
- Add `.env` variable: `ENCRYPTION_KEY_BASE64` (primary key)
- Add helper functions: `encrypt()`, `decrypt()`, `rotateKey()`

**Code example:**
```go
// internal/crypto/crypto.go (NEW FILE)
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "os"
)

var encryptionKey []byte

func init() {
    keyB64 := os.Getenv("ENCRYPTION_KEY_BASE64")
    if keyB64 == "" {
        panic("ENCRYPTION_KEY_BASE64 not set")
    }
    var err error
    encryptionKey, err = base64.StdEncoding.DecodeString(keyB64)
    if err != nil || len(encryptionKey) != 32 {
        panic("invalid ENCRYPTION_KEY_BASE64: must be base64-encoded 256-bit key")
    }
}

// Encrypt encrypts plaintext with AES-256-GCM
func Encrypt(plaintext string) (string, error) {
    block, err := aes.NewCipher(encryptionKey)
    if err != nil {
        return "", err
    }
    
    aead, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonce := make([]byte, aead.NonceSize())
    if _, err := rand.Read(nonce); err != nil {
        return "", err
    }
    
    ciphertext := aead.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext with AES-256-GCM
func Decrypt(ciphertextB64 string) (string, error) {
    ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
    if err != nil {
        return "", err
    }
    
    block, err := aes.NewCipher(encryptionKey)
    if err != nil {
        return "", err
    }
    
    aead, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonceSize := aead.NonceSize()
    if len(ciphertext) < nonceSize {
        return "", errors.New("ciphertext too short")
    }
    
    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }
    
    return string(plaintext), nil
}
```

**Configuration:**
```bash
# Generate a key (32 bytes = 256 bits)
python3 -c "import secrets, base64; print(base64.b64encode(secrets.token_bytes(32)).decode())"

# Set in .env
ENCRYPTION_KEY_BASE64=<output from above>
```

**Effort:** 1 hour

---

#### 1.2 Database Migration (1 hour)

**What:** Create `user_prompts` table to store encrypted prompts

**File:** `internal/db/migrations/005_user_prompts.sql` (NEW)

```sql
-- Migration 005: user prompts table with encryption support
-- Stores full user prompt text (encrypted) with retention policy.

CREATE TABLE user_prompts (
    prompt_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES ai_sessions(session_id) ON DELETE CASCADE,
    
    -- Encrypted prompt text (AES-256-GCM)
    prompt_text_encrypted VARCHAR(65535) NOT NULL,
    
    -- Metadata for monitoring
    prompt_length INT NOT NULL,        -- original character count
    token_estimate INT,                 -- rough token count (length / 4)
    
    -- Retention policy
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    retention_until TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() + INTERVAL '30 days'),
    
    -- Audit trail
    decrypted_by UUID,                  -- admin who last viewed it (nullable)
    decrypted_at TIMESTAMP WITH TIME ZONE,
    
    INDEX idx_user_prompts_session ON session_id,
    INDEX idx_user_prompts_retention ON retention_until,
    INDEX idx_user_prompts_created ON created_at DESC
);

-- MODIFY activity_events to reference prompts
ALTER TABLE activity_events 
ADD COLUMN user_prompt_id UUID REFERENCES user_prompts(prompt_id),
ADD COLUMN user_prompt_length INT;  -- cache for quick display
```

**Effort:** 1 hour

---

### Phase 2: Event Capture (3 hours)

#### 2.1 Hook Event Enhancement (1 hour)

**What:** Ensure hook events include full prompt text

**Current state:** `claude/types.go` in ast-cx-hooks already has `PreToolUseEvent` and `UserPromptSubmitEvent` that include the prompt, but it might only be the length.

**Check:** Does `UserPromptSubmitEvent` include full prompt text?

Looking at ast-cx-hooks source (assumed):
```go
type UserPromptSubmitEvent struct {
    EventBase
    UserMessage string          `json:"user_message"`  // ← Full text (if present)
    PromptText  string          `json:"prompt_text"`   // ← Or here
    // ...
}
```

**If missing:** Need to verify hook provides full prompt. If not, this is a blocker (would require ast-cx-hooks PR upstream).

**Effort:** 1 hour (verification) or blocked if hook doesn't provide it

---

#### 2.2 Ingest Logic Enhancement (2 hours)

**What:** Capture prompts on `prompt.submit` events, encrypt, and store

**File:** `internal/ingest/ingest.go` — add functions

```go
package ingest

import (
    "github.com/checkmarx/governance-api/internal/crypto"
    "github.com/google/uuid"
)

// upsertUserPrompt encrypts and stores a user prompt
func upsertUserPrompt(ctx context.Context, sessionID uuid.UUID, promptText string) (uuid.UUID, error) {
    promptLength := len(promptText)
    tokenEstimate := (promptLength + 3) / 4  // rough estimate: 1 token ≈ 4 chars
    
    // Encrypt the prompt
    encrypted, err := crypto.Encrypt(promptText)
    if err != nil {
        return uuid.Nil, fmt.Errorf("encrypt prompt: %w", err)
    }
    
    // Store encrypted prompt
    var promptID uuid.UUID
    err = db.Pool.QueryRow(ctx, `
        INSERT INTO user_prompts (
            session_id, prompt_text_encrypted, prompt_length, token_estimate, retention_until
        ) VALUES ($1, $2, $3, $4, NOW() + INTERVAL '30 days')
        RETURNING prompt_id`,
        sessionID, encrypted, promptLength, tokenEstimate,
    ).Scan(&promptID)
    
    return promptID, err
}

// Update processOne() to capture prompts
func (p *Processor) processOne(ctx context.Context, ev models.ActivityEvent) error {
    // ... existing code ...
    
    // NEW: Capture user prompt if this is a prompt.submit event
    var userPromptID *uuid.UUID
    if ev.Kind == "prompt.submit" && ev.Prompt != nil && ev.Prompt.Length > 0 {
        // Note: ev.Prompt only has Length; need full text from raw event
        // Extract from ev.RawPromptText (would need to add to ActivityEvent struct)
        // OR get from raw JSON
        var rawPrompt struct {
            PromptText string `json:"prompt_text"`
        }
        if err := json.Unmarshal(ev.Raw, &rawPrompt); err == nil && rawPrompt.PromptText != "" {
            if id, err := upsertUserPrompt(ctx, sessionID, rawPrompt.PromptText); err == nil {
                userPromptID = &id
            } else {
                log.Printf("ingest: upsert user prompt failed: %v", err)
            }
        }
    }
    
    // ... rest of existing logic ...
    return nil
}
```

**Note:** This assumes the raw event includes full prompt text. If ast-cx-hooks doesn't provide it, need to coordinate upstream.

**Effort:** 2 hours (implementation + testing)

---

### Phase 3: API Endpoints (3 hours)

#### 3.1 Admin API Endpoints (3 hours)

**File:** `internal/api/admin.go` (NEW) or extend `dashboard.go`

```go
// GET /api/v1/admin/sessions/{id}/prompts — admin-only endpoint
// Returns decrypted user prompts for a session (with audit trail)
func GetSessionPromptsAdmin(w http.ResponseWriter, r *http.Request) {
    // Check admin role
    if !isAdmin(r.Context()) {
        jsonError(w, http.StatusForbidden, "admin access required")
        return
    }
    
    sessionID := chi.URLParam(r, "id")
    adminID := getUserID(r.Context())
    
    rows, err := db.Pool.Query(r.Context(), `
        SELECT prompt_id, prompt_text_encrypted, prompt_length, created_at
        FROM user_prompts
        WHERE session_id = $1
        ORDER BY created_at ASC`, sessionID)
    if err != nil {
        jsonError(w, http.StatusInternalServerError, err.Error())
        return
    }
    defer rows.Close()
    
    prompts := []map[string]any{}
    for rows.Next() {
        var promptID, encrypted string
        var length int
        var createdAt time.Time
        
        if err := rows.Scan(&promptID, &encrypted, &length, &createdAt); err != nil {
            continue
        }
        
        // Decrypt (logs access)
        decrypted, err := crypto.Decrypt(encrypted)
        if err != nil {
            log.Printf("decrypt prompt %s: %v", promptID, err)
            continue
        }
        
        // Log audit trail
        _ = db.Pool.Exec(r.Context(), `
            UPDATE user_prompts
            SET decrypted_by = $1, decrypted_at = NOW()
            WHERE prompt_id = $2`, adminID, promptID)
        
        prompts = append(prompts, map[string]any{
            "promptId":      promptID,
            "promptText":    decrypted,
            "length":        length,
            "createdAt":     createdAt,
        })
    }
    
    jsonOK(w, map[string]any{
        "sessionId": sessionID,
        "prompts":   prompts,
        "decryptedBy": adminID,
        "decryptedAt": time.Now(),
    })
}

// GET /api/v1/admin/export/sessions/{id}/prompts — export encrypted prompts
func ExportSessionPromptsAdmin(w http.ResponseWriter, r *http.Request) {
    // Similar to above, but returns CSV (encrypted blob + metadata)
    // No decryption in response; raw encrypted text for compliance storage
}
```

**Register routes:**
```go
// In router.go, admin-only subrouter
r.Route("/admin", func(r chi.Router) {
    r.Use(requireAdmin)  // middleware
    r.Get("/sessions/{id}/prompts", GetSessionPromptsAdmin)
    r.Get("/export/sessions/{id}/prompts", ExportSessionPromptsAdmin)
})
```

**Effort:** 3 hours (2 endpoints + audit logging + CSV export)

---

### Phase 4: UI Changes (4 hours)

#### 4.1 Regular User UI — Redacted View (2 hours)

**File:** `components/panels/session-panel.tsx`

```typescript
// Show prompt summary only (redacted)
<div className="p-4 bg-blue-50 border border-blue-200 rounded">
  <h3 className="font-semibold">Prompts Submitted</h3>
  {prompts.map((event, i) => (
    <div key={i} className="text-sm text-gray-600 py-2">
      <span className="text-gray-900 font-mono">
        {new Date(event.occurredAt).toLocaleTimeString()}
      </span>
      {' • '}
      <span className="text-blue-600 font-semibold">
        User submitted {event.user_prompt_length}-char prompt
      </span>
      <p className="text-xs text-gray-500 mt-1">
        [Full text stored encrypted, visible to admins only]
      </p>
    </div>
  ))}
</div>
```

**Session timeline table addition:**
```typescript
// In activity events table, for "prompt.submit" rows:
<td className="border p-2 text-sm">
  {event.kind === 'prompt.submit' ? (
    <div className="text-gray-600">
      📝 {event.user_prompt_length} chars
      <p className="text-xs text-gray-400">[Redacted]</p>
    </div>
  ) : (
    '—'
  )}
</td>
```

**Effort:** 2 hours

---

#### 4.2 Admin UI — Unredacted View (2 hours)

**File:** `app/admin/sessions/[id]/prompts/page.tsx` (NEW)

```typescript
'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';

interface DecryptedPrompt {
  promptId: string;
  promptText: string;
  length: number;
  createdAt: string;
}

export default function AdminSessionPromptsPage({ params }) {
  const [prompts, setPrompts] = useState<DecryptedPrompt[]>([]);
  const [loading, setLoading] = useState(true);
  const [role, setRole] = useState('');

  useEffect(() => {
    // Check if user is admin
    const checkAuth = async () => {
      const user = await api.getCurrentUser();
      if (user.role !== 'admin') {
        // Redirect to unauthorized
        window.location.href = '/unauthorized';
        return;
      }
      setRole(user.role);
    };

    const fetchPrompts = async () => {
      try {
        const data = await api.adminSessionPrompts(params.id);
        setPrompts(data.prompts);
      } catch (error) {
        console.error('Failed to load prompts:', error);
      } finally {
        setLoading(false);
      }
    };

    checkAuth();
    fetchPrompts();
  }, [params.id]);

  if (role !== 'admin') return <div>Unauthorized</div>;
  if (loading) return <div>Loading...</div>;

  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold mb-4">Session Prompts (Admin Only)</h1>
      <p className="text-sm text-gray-600 mb-6">
        ⚠️ This view is encrypted and logged. Your access is audited.
      </p>
      
      <div className="space-y-4">
        {prompts.map((prompt) => (
          <div key={prompt.promptId} className="border p-4 rounded bg-gray-50">
            <div className="flex justify-between mb-2">
              <span className="text-sm text-gray-600">
                {new Date(prompt.createdAt).toLocaleString()}
              </span>
              <span className="text-xs text-gray-500">{prompt.length} chars</span>
            </div>
            <pre className="bg-white p-3 rounded border text-sm overflow-auto max-h-96">
              {prompt.promptText}
            </pre>
          </div>
        ))}
      </div>
    </div>
  );
}
```

**API method in `lib/api.ts`:**
```typescript
async adminSessionPrompts(sessionId: string): Promise<any> {
  const response = await fetch(
    `${this.apiUrl}/admin/sessions/${sessionId}/prompts`,
    { headers: { Authorization: `Bearer ${this.apiKey}` } }
  );
  if (!response.ok) throw new Error('Failed to fetch prompts');
  return response.json();
}
```

**Effort:** 2 hours (admin page + role checking + audit logging)

---

### Phase 5: Retention & Cleanup (1 hour)

#### 5.1 Auto-Deletion Job

**What:** Background job to delete prompts older than 30 days

**File:** `internal/jobs/retention.go` (NEW)

```go
package jobs

import (
    "context"
    "log"
    "time"
    
    "github.com/checkmarx/governance-api/internal/db"
)

// RunRetentionCleanup deletes expired user prompts daily
func RunRetentionCleanup(ctx context.Context) {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    
    for range ticker.C {
        deleted, err := db.Pool.Exec(ctx, `
            DELETE FROM user_prompts
            WHERE retention_until < NOW()`)
        if err != nil {
            log.Printf("retention cleanup failed: %v", err)
            continue
        }
        log.Printf("retention cleanup: deleted %d prompts", deleted.RowsAffected())
    }
}

// Start this in main.go:
// go jobs.RunRetentionCleanup(context.Background())
```

**Effort:** 1 hour

---

## Total Effort Summary

| Phase | Task | Effort | Notes |
|-------|------|--------|-------|
| 1 | Encryption setup | 1h | Key generation + crypto package |
| 1 | Database migration | 1h | New `user_prompts` table |
| 2 | Hook verification | 1h | Check if ast-cx-hooks provides full text |
| 2 | Ingest logic | 2h | Capture, encrypt, store prompts |
| 3 | Admin API | 3h | Decrypt endpoints, audit logging |
| 4 | User UI (redacted) | 2h | Show metadata only |
| 4 | Admin UI (unredacted) | 2h | Admin dashboard with decryption |
| 5 | Retention job | 1h | Auto-delete after 30 days |
| **TOTAL** | | **13 hours** | |

---

## Hidden Complexity: Things That Add Time

### 1. **Encryption Key Rotation** (+2 hours)

If you want to rotate encryption keys without decrypting all prompts:

```sql
-- Version column tracks which key encrypted this
ALTER TABLE user_prompts ADD COLUMN key_version INT;

-- Lazy rotation: re-encrypt on next admin view
-- OR batch job to rotate all keys
```

### 2. **Compliance Export** (+2 hours)

If you need encrypted CSV export for auditors:

```go
// ExportSessionPromptsAdmin returns:
// prompt_id | encrypted_text | created_at | metadata
// All in one encrypted ZIP file with manifest
```

### 3. **Role-Based Access Control** (+1-2 hours)

If some admins should see prompts but others shouldn't:

```sql
-- Add permission check
-- CREATE ROLE analyst_read_only
-- GRANT SELECT ON user_prompts TO analyst_read_only
```

### 4. **Audit Logging for Decryption** (+1 hour)

Already included (see code above), but if you need:
- Slack notification when admin views prompts
- Dashboard showing "who viewed what when"
- Differential audit (changed vs. not changed)

---

## Dependencies & Blockers

### Hard Blockers

1. **Hook provides full prompt text?**
   - If `UserPromptSubmitEvent` doesn't include full prompt, need upstream ast-cx-hooks PR
   - **Check first:** Can't proceed without this
   - **If blocked:** 3-4 weeks for upstream PR + release

2. **Encryption library available?**
   - Go's `crypto/aes` and `crypto/cipher` are standard library ✅
   - Should not be a blocker

### Soft Risks

1. **Performance:** Encryption adds ~5ms per prompt (acceptable)
2. **Key management:** If key is lost, prompts can't be decrypted (recommend key vault integration)
3. **Compliance approval:** Need legal/compliance sign-off before storing prompts

---

## Recommendation: Phased Rollout

### Week 1 (MVP, 6 hours)
- Phase 1: Encryption + DB migration
- Phase 2: Ingest logic (capture + encrypt)
- Phase 4.1: User UI (redacted view)

**Result:** Prompts captured and encrypted, invisible in regular UI.

### Week 2 (Admin Access, 5 hours)
- Phase 3: Admin API endpoints
- Phase 4.2: Admin UI dashboard
- Phase 5: Retention job

**Result:** Admins can decrypt and audit prompts; auto-delete after 30 days.

### Week 3+ (Optional Enhancements)
- Key rotation
- Compliance export
- Role-based access

---

## Cost-Benefit Analysis

**Cost:** 13 hours engineering + ongoing encryption/storage overhead  
**Benefit:**
- ✅ Full audit trail of user intent
- ✅ Compliance-ready (PII in separate table, encrypted, retention policy)
- ✅ Non-invasive (hidden from regular users, admin-only)
- ✅ Reversible (deletion job, key rotation options)

**Trade-off:**
- Slightly higher DB storage (prompts are often 1KB-10KB each)
- Encryption overhead (~5ms per prompt)
- Key management responsibility

**When to do it:**
- **YES:** If compliance/auditors require it
- **YES:** If you need to debug "why did Claude do X" (user intent reconstruction)
- **NO:** If you only care about action audit (what was done, not why)

---

## Implementation Checklist

- [ ] Verify hook provides full prompt text (or coordinate upstream PR)
- [ ] Generate encryption key, add to .env
- [ ] Create migration 005_user_prompts.sql
- [ ] Implement crypto package (encrypt/decrypt)
- [ ] Add upsertUserPrompt() in ingest.go
- [ ] Add admin API endpoints
- [ ] Register /admin routes
- [ ] Build user UI (redacted view)
- [ ] Build admin UI (unredacted dashboard)
- [ ] Implement retention job
- [ ] Test end-to-end with real prompts
- [ ] Get compliance approval
- [ ] Document encryption key backup/rotation procedure

