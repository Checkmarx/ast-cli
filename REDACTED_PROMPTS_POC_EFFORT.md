# Redacted User Prompts — POC Level Effort

**Goal:** Proof of concept — capture full user prompt text, redact in UI, store for admin viewing  
**Scope:** Minimal viable, no encryption/key management/compliance extras  
**Target Effort:** 4-5 hours total

---

## POC Architecture (Simplified)

### Storage
- Add `user_prompt_text TEXT` column directly to `activity_events` table (no separate table)
- Store full prompt plaintext (acceptable for POC)
- No encryption, no retention policy

### UI
- **Users see:** `"Prompt submitted • 2847 chars [Not shown]"`
- **Admins see:** Full prompt text in admin view (button to toggle)
- No audit logging, no access control (POC)

---

## Implementation: 4-5 Hours Total

### 1. Database Migration (30 minutes)

**File:** `internal/db/migrations/005_user_prompts_poc.sql` (NEW)

```sql
-- POC: Add prompt storage to activity_events
-- Simple plaintext column for proof of concept

ALTER TABLE activity_events 
ADD COLUMN user_prompt_text TEXT,
ADD COLUMN user_prompt_length INT;

CREATE INDEX idx_activity_events_has_prompt ON activity_events(user_prompt_text IS NOT NULL);
```

**That's it.** No new table, no encryption, no retention.

---

### 2. Ingest Logic (1 hour)

**File:** `internal/ingest/ingest.go` — modify `insertActivityEvent()`

```go
// Around line 163, modify the INSERT statement:

_, err := db.Pool.Exec(ctx, `
    INSERT INTO activity_events (
        event_id, session_id, kind,
        agent_type, tool_category, tool_name, tool_command, tool_paths, tool_query,
        mcp_server, mcp_tool, prompt_length,
        action_type, policy_name, policy_id, policy_version,
        matched_on, matched_value, agent_message, latency_ms,
        user_prompt_text, user_prompt_length,  -- ← NEW COLUMNS
        occurred_at, raw
    ) VALUES (
        $1, $2, $3,
        $4, $5, $6, $7, $8, $9,
        $10, $11, $12,
        $13, $14, $15, $16,
        $17, $18, $19, $20,
        $21, $22,  -- ← NEW PARAMETERS
        $23, $24
    ) ON CONFLICT (event_id) DO NOTHING`,
    ev.EventID, sessionID, ev.Kind,
    ev.Agent.Type, ev.Tool.Category, ev.Tool.Name, ev.Tool.Command,
    toolPaths, ev.Tool.Query,
    mcpServer, mcpTool, promptLength,
    nullableStr(ev.Decision.Action, "ALLOW"), ev.Decision.PolicyName, ev.Decision.PolicyID, ev.Decision.PolicyVersion,
    ev.Decision.MatchedOn, ev.Decision.MatchedValue, ev.Decision.AgentMessage, ev.Decision.LatencyMs,
    extractPromptText(ev),  // ← NEW HELPER
    extractPromptLength(ev), // ← NEW HELPER
    ev.Timestamp, rawJSON,
)
```

**Add helper function:**

```go
// extractPromptText gets full prompt from activity event raw JSON
func extractPromptText(ev models.ActivityEvent) string {
    if ev.Kind != "prompt.submit" {
        return ""
    }
    
    var raw struct {
        PromptText string `json:"prompt_text"`
    }
    _ = json.Unmarshal(ev.Raw, &raw)
    return raw.PromptText
}

// extractPromptLength gets prompt character count
func extractPromptLength(ev models.ActivityEvent) *int {
    if ev.Kind != "prompt.submit" {
        return nil
    }
    
    var raw struct {
        PromptText string `json:"prompt_text"`
    }
    if err := json.Unmarshal(ev.Raw, &raw); err != nil {
        return nil
    }
    
    length := len(raw.PromptText)
    return &length
}
```

---

### 3. API Endpoint (1 hour)

**File:** `internal/api/dashboard.go` — add new endpoint

```go
// GET /api/v1/sessions/{id}/prompts — admin POC endpoint
func GetSessionPrompts(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "id")
    
    // POC: No auth check (would add in production)
    rows, err := db.Pool.Query(r.Context(), `
        SELECT event_id, kind, user_prompt_text, user_prompt_length, occurred_at
        FROM activity_events
        WHERE session_id = $1 AND kind = 'prompt.submit'
        ORDER BY occurred_at ASC`, sessionID)
    if err != nil {
        jsonError(w, http.StatusInternalServerError, err.Error())
        return
    }
    defer rows.Close()
    
    prompts := []map[string]any{}
    for rows.Next() {
        var eventID, kind string
        var text *string
        var length *int
        var occurredAt time.Time
        
        if err := rows.Scan(&eventID, &kind, &text, &length, &occurredAt); err != nil {
            continue
        }
        
        prompts = append(prompts, map[string]any{
            "eventId":    eventID,
            "promptText": text,
            "length":     length,
            "timestamp":  occurredAt,
        })
    }
    
    jsonOK(w, map[string]any{
        "sessionId": sessionID,
        "prompts":   prompts,
    })
}
```

**Register in router.go:**

```go
r.Get("/sessions/{id}/prompts", GetSessionPrompts)  // POC — no auth
```

---

### 4. UI Changes (1.5-2 hours)

#### 4.1 Session Timeline — Redacted View (45 minutes)

**File:** `components/panels/session-panel.tsx` — modify activity event row rendering

```typescript
// In the activity timeline table, modify the render for prompt.submit events:

{event.kind === 'prompt.submit' ? (
  <td className="border p-2 text-sm text-gray-600">
    <div className="flex items-center gap-2">
      <span>📝 {event.user_prompt_length || 0} chars</span>
      <span className="text-xs text-gray-400">[Redacted]</span>
    </div>
  </td>
) : (
  <td className="border p-2">—</td>
)}
```

**Effort:** 45 minutes

#### 4.2 Admin View — Show Full Prompts (1 hour)

**File:** `app/sessions/[id]/prompts/page.tsx` (NEW)

```typescript
'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';

interface Prompt {
  eventId: string;
  promptText: string;
  length: number;
  timestamp: string;
}

export default function SessionPromptsPage() {
  const params = useParams();
  const sessionId = params.id as string;
  const [prompts, setPrompts] = useState<Prompt[]>([]);
  const [loading, setLoading] = useState(true);
  const [showFullText, setShowFullText] = useState(false);

  useEffect(() => {
    const fetchPrompts = async () => {
      try {
        const res = await fetch(`/api/v1/sessions/${sessionId}/prompts`);
        const data = await res.json();
        setPrompts(data.prompts || []);
      } catch (error) {
        console.error('Failed to load prompts:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchPrompts();
  }, [sessionId]);

  if (loading) return <div className="p-8">Loading prompts...</div>;
  if (prompts.length === 0) return <div className="p-8">No prompts in this session.</div>;

  return (
    <div className="p-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Session Prompts</h1>
        <button
          onClick={() => setShowFullText(!showFullText)}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700"
        >
          {showFullText ? 'Hide Text' : 'Show Full Text'}
        </button>
      </div>

      <div className="space-y-4">
        {prompts.map((prompt, i) => (
          <div key={prompt.eventId} className="border p-4 rounded bg-gray-50">
            <div className="flex justify-between text-sm text-gray-600 mb-2">
              <span>Prompt #{i + 1}</span>
              <span>{new Date(prompt.timestamp).toLocaleString()}</span>
              <span className="text-gray-500">{prompt.length} chars</span>
            </div>
            
            {showFullText ? (
              <pre className="bg-white p-3 rounded border text-sm overflow-auto max-h-64 whitespace-pre-wrap break-words">
                {prompt.promptText}
              </pre>
            ) : (
              <p className="text-gray-500 italic">
                [Click "Show Full Text" above to view prompt content]
              </p>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
```

**Add to lib/api.ts:**

```typescript
async sessionPrompts(sessionId: string): Promise<any> {
  const response = await fetch(`${this.apiUrl}/sessions/${sessionId}/prompts`, {
    headers: { Authorization: `Bearer ${this.apiKey}` },
  });
  if (!response.ok) throw new Error('Failed to fetch prompts');
  return response.json();
}
```

**Effort:** 1 hour

---

## Total POC Effort: **4-5 Hours**

| Component | Effort | What It Does |
|-----------|--------|-------------|
| DB migration | 30m | Add two columns to activity_events |
| Ingest logic | 1h | Extract prompt from event, store in DB |
| API endpoint | 1h | Return prompts for a session (no auth) |
| Timeline UI | 45m | Show "📝 2847 chars [Redacted]" in timeline |
| Admin page | 1h | New page with toggle to show/hide full prompts |
| **TOTAL** | **4-5h** | **End-to-end POC working** |

---

## What's NOT Included (Production Stuff)

❌ Encryption  
❌ Key management  
❌ Encryption key rotation  
❌ Audit logging of who viewed prompts  
❌ Role-based access control  
❌ Auto-deletion / retention policy  
❌ Compliance export  
❌ Performance optimization  

---

## POC Test Flow

1. **Start Claude Code session**
2. **Submit a prompt** (e.g., "analyze this file")
3. **Check activity timeline** → see `"📝 145 chars [Redacted]"`
4. **Go to `/sessions/{id}/prompts`** → click "Show Full Text" → see the actual prompt
5. **Proof of concept complete** ✅

---

## Next Steps for Production (If Approved)

If POC works and you want to productionize:

1. **Add encryption** — wrap `user_prompt_text` in `crypto.Encrypt()`
2. **Add retention policy** — schedule daily cleanup of old prompts
3. **Add audit logging** — log when admin views prompts
4. **Add role check** — only admins can view full text
5. **Add compliance export** — CSV with encrypted prompts

**Productionization effort:** ~8 additional hours

---

## Files to Create/Modify

**Create:**
- `internal/db/migrations/005_user_prompts_poc.sql`
- `app/sessions/[id]/prompts/page.tsx`

**Modify:**
- `internal/ingest/ingest.go` (add helpers + column to INSERT)
- `internal/api/dashboard.go` (add GetSessionPrompts endpoint)
- `internal/api/router.go` (register /sessions/{id}/prompts route)
- `lib/api.ts` (add sessionPrompts method)
- `components/panels/session-panel.tsx` (show redacted prompt in timeline)

