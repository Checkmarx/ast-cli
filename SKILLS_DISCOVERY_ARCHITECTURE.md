# AI Governance Skills Discovery Platform — Architecture Plan

**Role:** Senior Architect, Checkmarx  
**Phase:** Tier 1 — Accurate skill discovery and session-level skill inventory  
**Scope:** ast-cli · governance-api · governance-ui · Database · Skills Inventory  
**Date:** 2026-08-18

---

## 1. Executive Summary

This is a **Tier 1 skills discovery initiative** that brings skill tracking to parity with MCP server discovery. Currently, skill invocations are logged as indistinguishable `category: "other"` entries; after implementation, each skill is discovered, tracked, and inventoried with full usage metrics.

**What we build (entirely new):**
- `skills` table — skill catalog, mirroring `mcp_servers` structure
- `session_skills` bridge table — per-session skill usage with invocation counts
- `GET /api/v1/skills` — skill inventory API with usage statistics
- `GET /api/v1/skills/{id}` — skill detail view (per-session breakdown)
- `GET /api/v1/sessions/{id}/skills` — session-level skill list
- Skills UI panel — inventory view in governance dashboard
- Skill normalization in ast-cli — extract skill name from `Skill` tool input

**What this does NOT do (Tier 2+):**
- Activity attribution — linking individual tool calls back to which skill invoked them
- Skill approval workflows — approval required before use
- Skill metadata enrichment — description, owner, documentation
- Skill blocking/deprecation — enforcement policies (future)

---

## 2. Current State Analysis

### 2.1 Current Skill Tracking Behavior

| Aspect | Current | Problem |
|---|---|---|
| Skill detection | Logged as `toolName="Skill"` | Input JSON not parsed |
| Skill name | Discarded in normalization | 50 invocations all appear as `"Skill"` |
| Categorization | `category: "other"` | No distinction from other tools |
| DB storage | No `skills` table | No inventory, no history |
| Session tracking | No `session_skills` bridge | Can't query "which skills in session X" |
| UI visibility | Hidden in `category: other` | Not discoverable in dashboard |

### 2.2 Gaps to Build

| # | Gap | What needs to be built |
|---|---|---|
| G1 | No skill name extraction | `extractSkillName()` in normalize.go |
| G2 | No skill category | New `case toolName == "Skill"` in normalizeTool() |
| G3 | No skills table | New `skills` table (mirrors `mcp_servers`) |
| G4 | No session-skill bridge | New `session_skills` table |
| G5 | No skill upsert logic | `upsertSkill()` and `upsertSessionSkill()` functions |
| G6 | No skill discovery flow | Call upsert functions during event ingest |
| G7 | No skills inventory API | `ListSkills`, `GetSkillDetail`, `GetSessionSkills` endpoints |
| G8 | No skills UI | New skills page and session detail integration |

---

## 3. Architecture Overview

### Skills Discovery Data Flow

```
┌──────────────────────────────────────────────────────────────────────┐
│                      AI Governance Skills Platform                    │
│                                                                       │
│  ┌──────────────────┐    ┌───────────────────────────────────────┐   │
│  │   Dashboard UI   │    │    Policy API Server + Skills API     │   │
│  │  (Next.js/React) │◄──►│                                       │   │
│  │                  │    │  GET /api/v1/skills                   │   │
│  │  • Skills panel  │    │  GET /api/v1/skills/{id}              │   │
│  │  • Session skills│    │  GET /api/v1/sessions/{id}/skills    │   │
│  │  • Inventory     │    │  POST /api/v1/audit/events            │   │
│  └──────────────────┘    └───────────────────┬───────────────────┘   │
└─────────────────────────────────────────────┬┘                        │
                                              │ HTTPS                    │
                    ┌─────────────────────────▼────────────────────────┐
                    │              Developer Machine                     │
                    │                                                    │
                    │  cx hooks governance CLI                          │
                    │  ├── governance pack eval (local)                 │
                    │  ├── skill invocation detection                   │
                    │  └── activity event spool                         │
                    │                                                    │
                    │  ~/.checkmarx/governance/                         │
                    │  ├── policy-pack.json                             │
                    │  └── audit-queue/  ← skill events spooled here    │
                    └──────────────────────────────────────────────────┘
                                          │
                    ┌─────────────────────▼────────────────────────────┐
                    │          AI Coding Agent                          │
                    │   Claude Code / Cursor                            │
                    │   fires hook → cx hooks <route>                  │
                    │   ├── PreToolUse fires with ToolName="Skill"      │
                    │   │   ├── ToolInput={"skill":"code-review"}      │
                    │   │   └── governance pack eval (local, ~1–4ms)    │
                    │   └── skill audit event spooled async             │
                    └──────────────────────────────────────────────────┘
```

**Cardinal rule:** Skills are discovered on **first invocation** in the event ingest pipeline, not during hook execution.

---

## 4. Database Design

**Database:** PostgreSQL 16  
**Scope:** New tables only; existing tables untouched

### 4.1 Entity Relationship Overview

```
Skill (catalog)
    │
    ├── session_skills (bridge: session + skill)
    │       │
    │       └── ai_sessions  (existing)
    │
    └── activity_events (existing — tool_category="skill", tool_name=skill_name)
```

### 4.2 Database Schema Changes

#### `skills` Table

Mirrors `mcp_servers` structure. One row per unique skill name. Discovered on first invocation.

| Column | Type | Notes |
|---|---|---|
| `skill_id` | UUID PK | |
| `skill_name` | VARCHAR UNIQUE | "code-review", "dataviz", "update-config", etc. |
| `description` | TEXT | Optional — for future UI enrichment |
| `status` | ENUM | `active`, `deprecated`, `blocked` — default `active` |
| `first_seen_at` | TIMESTAMP | Auto-populated on discovery |
| `last_seen_at` | TIMESTAMP | Updated on each invocation |
| `total_invocations` | INTEGER | Denormalized counter — incremented on each invocation |

**Indexes:**
```sql
CREATE INDEX idx_skills_name ON skills(skill_name);
CREATE INDEX idx_skills_last_seen ON skills(last_seen_at DESC);
CREATE INDEX idx_skills_status ON skills(status);
```

#### `session_skills` Table

Bridge table — tracks per-session skill usage with invocation counts.

| Column | Type | Notes |
|---|---|---|
| `session_skill_id` | UUID PK | |
| `session_id` | UUID FK → ai_sessions | |
| `skill_id` | UUID FK → skills | |
| `invocation_count` | INTEGER | How many times this skill was invoked in this session |
| `first_invoked_at` | TIMESTAMP | First invocation timestamp in this session |
| `last_invoked_at` | TIMESTAMP | Latest invocation timestamp in this session |

**Unique constraint:** `UNIQUE(session_id, skill_id)` — one row per unique session + skill pair.

**Indexes:**
```sql
CREATE INDEX idx_session_skills_session ON session_skills(session_id);
CREATE INDEX idx_session_skills_skill ON session_skills(skill_id);
CREATE INDEX idx_session_skills_last_invoked ON session_skills(last_invoked_at DESC);
```

#### Existing Table: `activity_events`

**No schema changes required.** The `tool_category` and `tool_name` columns already exist.

- When a skill is invoked: `tool_category = "skill"`, `tool_name = "code-review"` (actual skill name extracted from input)
- Before this change: `tool_category = "other"`, `tool_name = "Skill"` (name discarded)
- Migration: Historical skill invocations remain as `category: "other"`. Going forward, new invocations are properly categorized.

---

## 5. Implementation: ast-cli Changes

### 5.1 Package Structure (no new packages)

Skills detection reuses the existing `governance/normalize.go` file — no new files needed.

```
internal/commands/agenthooks/
└── governance/
    ├── normalize.go       ← MODIFIED: add skill extraction + detection
    ├── hooks.go           (unchanged)
    ├── pack.go            (unchanged)
    └── ... (other files)
```

### 5.2 Skill Name Extraction

#### File: `internal/commands/agenthooks/governance/normalize.go`

#### Change 1: Add skill name extraction function

**Location:** After `parseMCPName()` function (line ~180)

```go
// extractSkillName extracts the skill name from Skill tool input.
// Input format: {"skill":"code-review","args":"..."}
// Returns the skill name or empty string if unparseable.
func extractSkillName(raw json.RawMessage) string {
	var input struct {
		Skill string `json:"skill"`
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		return ""
	}
	return input.Skill
}
```

#### Change 2: Add skill detection case to `normalizeTool()`

**Location:** In `normalizeTool()` function, add new case **before** the `default` (line ~159)

```go
case toolName == "Skill":
	skillName := extractSkillName(rawInput)
	if skillName == "" {
		skillName = "unknown"
	}
	return ToolInfo{
		Category: "skill",
		Name:     skillName,
		RawName:  toolName,
	}
```

**Complete function after changes:**

```go
func normalizeTool(toolName string, rawInput json.RawMessage) ToolInfo {
	switch {
	case isFilesystemTool(toolName):
		return ToolInfo{
			Category: "filesystem",
			Name:     toolName,
			RawName:  toolName,
			Paths:    extractPaths(rawInput),
		}
	case toolName == "Bash":
		return ToolInfo{
			Category: "shell",
			Name:     "Bash",
			RawName:  "Bash",
			Command:  extractCommand(rawInput),
		}
	case strings.HasPrefix(toolName, "mcp__"):
		server, tool := parseMCPName(toolName)
		return ToolInfo{
			Category: "mcp",
			Name:     tool,
			RawName:  toolName,
			MCP:      &MCPInfo{Server: server, Name: tool},
		}
	case toolName == "WebSearch" || toolName == "WebFetch":
		return ToolInfo{
			Category: "web_search",
			Name:     toolName,
			RawName:  toolName,
			Query:    extractQuery(rawInput),
		}
	case toolName == "Skill":
		skillName := extractSkillName(rawInput)
		if skillName == "" {
			skillName = "unknown"
		}
		return ToolInfo{
			Category: "skill",
			Name:     skillName,
			RawName:  toolName,
		}
	default:
		return ToolInfo{Category: "other", Name: toolName, RawName: toolName}
	}
}
```

### Summary of ast-cli Changes
- **1 new function:** `extractSkillName()`
- **1 new case:** in `normalizeTool()` to detect and extract skills
- **No hook changes:** PreToolUse/PostToolUse/UserPromptSubmit handlers unchanged; they already call `normalizeTool()`
- **No policy changes:** Skill invocations flow through existing audit/decision logic unchanged

---

## 6. Implementation: governance-api Changes

### 6.1 Database Migration

#### File: `internal/db/migrations/004_skills.sql` (new)

```sql
-- Migration 004: add skills discovery tables for skill inventory and per-session tracking.

CREATE TABLE skills (
    skill_id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_name            VARCHAR(255) NOT NULL UNIQUE,
    description           TEXT,
    status                VARCHAR(50) DEFAULT 'active',
    first_seen_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_seen_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    total_invocations     INTEGER DEFAULT 0,
    
    CONSTRAINT valid_status CHECK (status IN ('active', 'deprecated', 'blocked'))
);

CREATE INDEX idx_skills_name ON skills(skill_name);
CREATE INDEX idx_skills_last_seen ON skills(last_seen_at DESC);
CREATE INDEX idx_skills_status ON skills(status);

CREATE TABLE session_skills (
    session_skill_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id            UUID NOT NULL REFERENCES ai_sessions(session_id) ON DELETE CASCADE,
    skill_id              UUID NOT NULL REFERENCES skills(skill_id) ON DELETE CASCADE,
    invocation_count      INTEGER DEFAULT 1,
    first_invoked_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_invoked_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    UNIQUE(session_id, skill_id)
);

CREATE INDEX idx_session_skills_session ON session_skills(session_id);
CREATE INDEX idx_session_skills_skill ON session_skills(skill_id);
CREATE INDEX idx_session_skills_last_invoked ON session_skills(last_invoked_at DESC);
```

### 6.2 Event Ingestion

#### File: `internal/ingest/ingest.go`

#### Change 1: Add skill upsert functions

**Location:** After `upsertMCPServer()` function (line ~333)

```go
// upsertSkill creates or updates a skill entry on discovery.
func upsertSkill(ctx context.Context, skillName string) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO skills (skill_name)
		VALUES ($1)
		ON CONFLICT (skill_name)
		DO UPDATE SET last_seen_at = NOW(), total_invocations = total_invocations + 1
		RETURNING skill_id`,
		skillName,
	).Scan(&id)
	return id, err
}

// upsertSessionSkill creates or updates the session-skill relationship.
func upsertSessionSkill(ctx context.Context, sessionID, skillID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO session_skills (session_id, skill_id, invocation_count)
		VALUES ($1, $2, 1)
		ON CONFLICT (session_id, skill_id)
		DO UPDATE SET
			invocation_count = session_skills.invocation_count + 1,
			last_invoked_at = NOW()`,
		sessionID, skillID)
	return err
}
```

#### Change 2: Update `processOne()` to discover skills

**Location:** In `processOne()` function, add after MCP server handling (line ~75, after the `if ev.Tool.MCP != nil` block)

```go
	// 6b. Resolve skill if applicable
	var skillID *uuid.UUID
	if ev.Tool.Category == "skill" && ev.Tool.Name != "" {
		id, err := upsertSkill(ctx, ev.Tool.Name)
		if err != nil {
			log.Printf("ingest: upsert skill %q: %v", ev.Tool.Name, err)
		} else {
			skillID = &id
			// Upsert session-skill bridge
			if err := upsertSessionSkill(ctx, sessionID, id); err != nil {
				log.Printf("ingest: upsert session-skill bridge: %v", err)
			}
		}
	}
```

**Context (showing where this fits in the existing flow):**

```go
func (p *Processor) processOne(ctx context.Context, ev models.ActivityEvent) error {
	// 1. Resolve/create developer
	// 2. Resolve agent product
	// 3. Upsert developer_agents
	// 4. Upsert session
	// 5. Update session counters
	// 6. Resolve MCP server if applicable
	if ev.Tool.MCP != nil && ev.Tool.MCP.Server != "" {
		// ... existing MCP logic
	}
	
	// 6b. Resolve skill if applicable  ← NEW
	var skillID *uuid.UUID
	if ev.Tool.Category == "skill" && ev.Tool.Name != "" {
		// ... new skill logic (code above)
	}
	
	// 7. Write ALL events to activity_events
	// 8-9. Resolve policy and insert policy_action (if applicable)
	return nil
}
```

#### Change 3: Update `insertActivityEvent()` to populate skill fields

**No code change needed.** The function already writes `tool_name` and `tool_category` to `activity_events`, which will now correctly contain `"skill"` and the skill name respectively. The existing schema handles it.

### 6.3 API Endpoint Inventory

#### Skills Endpoints (new)

```
GET  /api/v1/skills                    → list all discovered skills with usage stats
GET  /api/v1/skills/{id}               → skill detail + per-session breakdown
GET  /api/v1/sessions/{id}/skills      → skills used in a specific session
```

#### File: `internal/api/dashboard.go`

#### Change 1: Add ListSkills endpoint

**Location:** After `ListMCPServers()` function

```go
// GET /api/v1/skills — list all discovered skills with usage stats
func ListSkills(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r.URL.Query())
	rows, err := db.Pool.Query(r.Context(), `
		SELECT
			s.skill_id::text, s.skill_name, s.status,
			s.first_seen_at, s.last_seen_at,
			s.total_invocations,
			COUNT(DISTINCT ss.session_id) as session_count
		FROM skills s
		LEFT JOIN session_skills ss ON s.skill_id = ss.skill_id
		GROUP BY s.skill_id, s.skill_name, s.status, s.first_seen_at, s.last_seen_at, s.total_invocations
		ORDER BY s.last_seen_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var (
			skillID, skillName, status string
			firstSeenAt, lastSeenAt    time.Time
			totalInvocations, sessionCount int
		)
		if err := rows.Scan(&skillID, &skillName, &status, &firstSeenAt, &lastSeenAt, &totalInvocations, &sessionCount); err != nil {
			continue
		}
		result = append(result, map[string]any{
			"skillId":          skillID,
			"skillName":        skillName,
			"status":           status,
			"firstSeenAt":      firstSeenAt,
			"lastSeenAt":       lastSeenAt,
			"totalInvocations": totalInvocations,
			"sessionCount":     sessionCount,
		})
	}

	jsonOK(w, result)
}
```

#### Change 2: Add GetSkillDetail endpoint

**Location:** After `ListSkills()` function

```go
// GET /api/v1/skills/{id} — get skill details and per-session invocation breakdown
func GetSkillDetail(w http.ResponseWriter, r *http.Request) {
	skillID := chi.URLParam(r, "id")
	
	// Get skill metadata
	var skillName, status string
	var firstSeenAt, lastSeenAt time.Time
	var totalInvocations int
	
	err := db.Pool.QueryRow(r.Context(), `
		SELECT skill_name, status, first_seen_at, last_seen_at, total_invocations
		FROM skills WHERE skill_id = $1`, skillID).Scan(
		&skillName, &status, &firstSeenAt, &lastSeenAt, &totalInvocations)
	if err != nil {
		jsonError(w, http.StatusNotFound, "skill not found")
		return
	}

	// Get per-session breakdown
	rows, err := db.Pool.Query(r.Context(), `
		SELECT
			s.session_id::text, s.claude_session_id,
			ss.invocation_count,
			ss.first_invoked_at, ss.last_invoked_at,
			d.email, a.agent_name
		FROM session_skills ss
		JOIN ai_sessions s ON ss.session_id = s.session_id
		JOIN developers d ON s.developer_id = d.developer_id
		JOIN ai_coding_agents a ON s.agent_id = a.agent_id
		WHERE ss.skill_id = $1
		ORDER BY ss.last_invoked_at DESC`, skillID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	sessions := []map[string]any{}
	for rows.Next() {
		var (
			sessionID, claudeID, email, agentName string
			invocationCount int
			firstInvokedAt, lastInvokedAt time.Time
		)
		if err := rows.Scan(&sessionID, &claudeID, &invocationCount, &firstInvokedAt, &lastInvokedAt, &email, &agentName); err != nil {
			continue
		}
		sessions = append(sessions, map[string]any{
			"sessionId":       sessionID,
			"claudeId":        claudeID,
			"invocationCount": invocationCount,
			"firstInvokedAt":  firstInvokedAt,
			"lastInvokedAt":   lastInvokedAt,
			"developer":       email,
			"agent":           agentName,
		})
	}

	jsonOK(w, map[string]any{
		"skillId":          skillID,
		"skillName":        skillName,
		"status":           status,
		"firstSeenAt":      firstSeenAt,
		"lastSeenAt":       lastSeenAt,
		"totalInvocations": totalInvocations,
		"sessions":         sessions,
	})
}
```

#### Change 3: Add GetSessionSkills endpoint

**Location:** After `GetSkillDetail()` function

```go
// GET /api/v1/sessions/{id}/skills — get skills used in a specific session
func GetSessionSkills(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	
	rows, err := db.Pool.Query(r.Context(), `
		SELECT
			s.skill_id::text, s.skill_name, s.status,
			ss.invocation_count,
			ss.first_invoked_at, ss.last_invoked_at
		FROM session_skills ss
		JOIN skills s ON ss.skill_id = s.skill_id
		WHERE ss.session_id = $1
		ORDER BY ss.last_invoked_at DESC`, sessionID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	result := []map[string]any{}
	for rows.Next() {
		var (
			skillID, skillName, status string
			invocationCount int
			firstInvokedAt, lastInvokedAt time.Time
		)
		if err := rows.Scan(&skillID, &skillName, &status, &invocationCount, &firstInvokedAt, &lastInvokedAt); err != nil {
			continue
		}
		result = append(result, map[string]any{
			"skillId":          skillID,
			"skillName":        skillName,
			"status":           status,
			"invocationCount":  invocationCount,
			"firstInvokedAt":   firstInvokedAt,
			"lastInvokedAt":    lastInvokedAt,
		})
	}

	jsonOK(w, result)
}
```

#### File: `internal/api/router.go`

#### Change: Register skill endpoints

**Location:** In `NewRouter()` function, add after MCP server routes (line ~57)

```go
		// ── Skills ─────────────────────────────────────────────
		r.Get("/skills", ListSkills)
		r.Get("/skills/{id}", GetSkillDetail)
		r.Get("/sessions/{id}/skills", GetSessionSkills)
```

**Full router context:**

```go
func NewRouter() *chi.Mux {
	r := chi.NewRouter()
	// ... middleware setup ...
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.BearerAuth)
		
		// ... existing routes ...
		
		// ── MCP Servers ────────────────────────────────────────────
		r.Get("/mcp-servers", ListMCPServers)
		r.Get("/mcp-servers/{id}", GetMCPServer)
		
		// ── Skills ─────────────────────────────────────────────  ← NEW
		r.Get("/skills", ListSkills)
		r.Get("/skills/{id}", GetSkillDetail)
		r.Get("/sessions/{id}/skills", GetSessionSkills)
	})
	return r
}
```

### Summary of governance-api Changes
- **1 new migration:** `004_skills.sql` (creates `skills` and `session_skills` tables)
- **2 new ingest functions:** `upsertSkill()`, `upsertSessionSkill()`
- **1 modified function:** `processOne()` to detect and upsert skills (mirrors MCP logic)
- **3 new API endpoints:** `ListSkills`, `GetSkillDetail`, `GetSessionSkills`
- **No changes to:** policy evaluation, activity event schema, existing endpoints

---

## 7. Implementation: governance-ui Changes

### 7.1 Skills Inventory Page

#### File: `app/skills/page.tsx` (new)

**Create this file to display the skills inventory:**

```typescript
'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';

interface Skill {
  skillId: string;
  skillName: string;
  status: string;
  firstSeenAt: string;
  lastSeenAt: string;
  totalInvocations: number;
  sessionCount: number;
}

export default function SkillsPage() {
  const [skills, setSkills] = useState<Skill[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchSkills = async () => {
      try {
        const data = await api.skills();
        setSkills(data);
      } catch (error) {
        console.error('Failed to load skills:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchSkills();
    const interval = setInterval(fetchSkills, 5000); // Refresh every 5s
    return () => clearInterval(interval);
  }, []);

  if (loading) return <div>Loading skills...</div>;

  return (
    <div className="p-8">
      <h1 className="text-3xl font-bold mb-6">Skills Inventory</h1>
      
      {skills.length === 0 ? (
        <p>No skills discovered yet.</p>
      ) : (
        <table className="w-full border-collapse border border-gray-300">
          <thead className="bg-gray-100">
            <tr>
              <th className="border p-3 text-left">Skill Name</th>
              <th className="border p-3 text-left">Status</th>
              <th className="border p-3 text-right">Total Invocations</th>
              <th className="border p-3 text-right">Sessions</th>
              <th className="border p-3 text-left">Last Seen</th>
            </tr>
          </thead>
          <tbody>
            {skills.map((skill) => (
              <tr key={skill.skillId} className="hover:bg-gray-50">
                <td className="border p-3 font-mono">{skill.skillName}</td>
                <td className="border p-3">
                  <span className={`px-2 py-1 rounded text-sm ${
                    skill.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'
                  }`}>
                    {skill.status}
                  </span>
                </td>
                <td className="border p-3 text-right">{skill.totalInvocations}</td>
                <td className="border p-3 text-right">{skill.sessionCount}</td>
                <td className="border p-3 text-sm">
                  {new Date(skill.lastSeenAt).toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
```

### 7.2 API Client

#### File: `lib/api.ts`

#### Change: Add skill API methods

**Location:** Add after `mcp_servers()` method

```typescript
  async skills(): Promise<any[]> {
    const response = await fetch(`${this.apiUrl}/skills`, {
      headers: { Authorization: `Bearer ${this.apiKey}` },
    });
    if (!response.ok) throw new Error(`Skills request failed: ${response.statusText}`);
    return response.json();
  }

  async skillDetail(skillId: string): Promise<any> {
    const response = await fetch(`${this.apiUrl}/skills/${skillId}`, {
      headers: { Authorization: `Bearer ${this.apiKey}` },
    });
    if (!response.ok) throw new Error(`Skill detail request failed: ${response.statusText}`);
    return response.json();
  }

  async sessionSkills(sessionId: string): Promise<any[]> {
    const response = await fetch(`${this.apiUrl}/sessions/${sessionId}/skills`, {
      headers: { Authorization: `Bearer ${this.apiKey}` },
    });
    if (!response.ok) throw new Error(`Session skills request failed: ${response.statusText}`);
    return response.json();
  }
```

### 7.3 Session Detail Integration

#### File: `components/panels/session-panel.tsx`

#### Change: Add Skills section to session detail

**Location:** In the session detail panel (alongside MCP servers section), add a new Skills tab/section:

```typescript
// Inside the SessionPanel component, add a new tab for skills
<tab name="Skills">
  <div className="p-4">
    <h3 className="text-lg font-semibold mb-4">Skills Used ({skills.length})</h3>
    {skills.length === 0 ? (
      <p className="text-gray-500">No skills invoked in this session.</p>
    ) : (
      <ul className="space-y-2">
        {skills.map((skill) => (
          <li key={skill.skillId} className="p-3 bg-blue-50 rounded border border-blue-200">
            <div className="flex justify-between">
              <span className="font-mono">{skill.skillName}</span>
              <span className="text-sm text-gray-600">{skill.invocationCount} invocations</span>
            </div>
            <div className="text-xs text-gray-500 mt-1">
              {new Date(skill.lastInvokedAt).toLocaleString()}
            </div>
          </li>
        ))}
      </ul>
    )}
  </div>
</tab>
```

### Summary of governance-ui Changes
- **1 new page:** `app/skills/page.tsx` (skills inventory list)
- **3 new API methods:** `skills()`, `skillDetail()`, `sessionSkills()`
- **1 UI enhancement:** Session detail panel gets a "Skills" tab showing per-session skill usage
- **No changes to:** existing pages, navigation (though you may want to add a "Skills" link to main nav)

---

## 8. Testing Strategy

### Unit Tests (ast-cli)

**File:** `internal/commands/agenthooks/governance/normalize_test.go`

Add tests for:
```go
TestExtractSkillName_Valid()        // Input: {"skill":"code-review"} → "code-review"
TestExtractSkillName_Invalid()      // Input: {} → ""
TestNormalizeTool_Skill()           // Input: ToolName="Skill", ToolInput={skill:"dataviz"} 
                                     // → ToolInfo{Category:"skill", Name:"dataviz"}
TestNormalizeTool_SkillUnknown()    // Input: ToolName="Skill", ToolInput={} 
                                     // → ToolInfo{Category:"skill", Name:"unknown"}
```

### Integration Tests (governance-api)

**File:** `internal/ingest/ingest_test.go`

Add tests for:
```go
TestUpsertSkill_Create()            // New skill inserted
TestUpsertSkill_Update()            // Existing skill: invocation counter incremented
TestUpsertSessionSkill_Create()     // New session-skill relationship created
TestUpsertSessionSkill_Update()     // Existing session-skill: invocation counter incremented
TestProcessOne_WithSkill()          // Full ingest flow with skill event
```

### API Tests (governance-api)

**File:** `internal/api/dashboard_test.go`

Add tests for:
```go
TestListSkills_Empty()              // No skills yet
TestListSkills_Multiple()           // Multiple skills with correct counts
TestGetSkillDetail_Found()          // Returns skill metadata + session breakdown
TestGetSkillDetail_NotFound()       // 404 for unknown skill
TestGetSessionSkills_Empty()        // Session with no skills
TestGetSessionSkills_Multiple()     // Session with multiple skills and invocation counts
```

### E2E Test Scenario

**Walkthrough:**
1. Create session in Claude Code
2. Invoke `/code-review` skill
3. Verify via API: `GET /api/v1/skills` returns `{"skillName":"code-review", ...}`
4. Verify via API: `GET /api/v1/sessions/{id}/skills` returns the skill with invocation count = 1
5. Invoke `/code-review` again in the same session
6. Verify invocation count = 2
7. Open UI on `/skills` — see "code-review" listed with totalInvocations = 2, sessionCount = 1
8. Create new session, invoke `/dataviz`
9. Verify `GET /api/v1/skills` returns both skills
10. Verify session-specific skills only show what that session invoked

---

## 9. Implementation Order & Phases

### Phase Timeline

### Phase 1: Core Data Model
1. **governance-api:** Create migration `004_skills.sql`
2. **governance-api:** Create DB migration sequentially (already happens on startup)
3. **Verify:** Connect to DB, confirm tables exist

### Phase 2: Event Normalization
4. **ast-cli:** Add `extractSkillName()` function
5. **ast-cli:** Add skill case to `normalizeTool()`
6. **Verify:** Unit tests for skill extraction

### Phase 3: Ingest Logic
7. **governance-api:** Add `upsertSkill()` and `upsertSessionSkill()` functions
8. **governance-api:** Update `processOne()` to call skill upsert (mirrors MCP logic)
9. **Verify:** Integration tests for skill upsert

### Phase 4: API Endpoints
10. **governance-api:** Implement `ListSkills`, `GetSkillDetail`, `GetSessionSkills` handlers
11. **governance-api:** Register routes in `router.go`
12. **Verify:** API tests + manual curl tests

### Phase 5: UI
13. **governance-ui:** Add `lib/api.ts` methods for skills
14. **governance-ui:** Create `app/skills/page.tsx`
15. **governance-ui:** Add Skills tab to session detail panel
16. **Verify:** E2E test walkthrough

### Phase 6: Polish
17. Add navigation link to Skills inventory
18. Add skill status filtering/search in UI (future enhancement)
19. Documentation + changelog

---

## 10. Risk Analysis & Backward Compatibility

### 10.1 No Breaking Changes

| Component | Impact | Mitigation |
|---|---|---|
| ast-cli | New code path only for `toolName == "Skill"` | All other tools flow through existing logic unchanged |
| governance-api | New tables only; existing tables untouched | No schema migrations to existing tables |
| governance-ui | New page + new API methods | Existing pages fully functional |
| Policies | No change to policy evaluation | Skills invocations flow through existing pack.Evaluate() |

### 10.2 Data Migration Strategy

**Existing audit data (before this change):**
- Skill invocations logged as `tool_category: "other"`, `tool_name: "Skill"` (name discarded)
- These records remain unchanged in `activity_events` (no retroactive update needed)

**Going forward (after this change):**
- New skill invocations: `tool_category: "skill"`, `tool_name: "code-review"` (actual skill name extracted)
- Skills discovered and tracked in new `skills` and `session_skills` tables

### 10.3 Fallback Behavior

If a Skill invocation arrives **without** a skill name in the input JSON:
1. `extractSkillName()` returns `""` (empty string)
2. `normalizeTool()` replaces with `"unknown"`
3. Event ingested with `tool_name: "unknown"`
4. DB upsert creates/increments a single "unknown" skill row

This ensures **no null entries** in the DB and **no crashes** due to malformed input.

---

## 11. Future Enhancements (Tier 2+)

These are out of scope for Tier 1 but possible future extensions:

- **Skill Status Management:** API endpoints to mark skills as `deprecated` or `blocked`, then enforce via policies
- **Skill Activity Attribution (Tier 2):** Time-window heuristic linking tool calls to which skill invoked them
- **Skill Approval Workflows:** Require approval before a new skill can be used org-wide
- **Skill Metadata Enrichment:** Store skill description, owner, documentation URL (manually populated or auto-scraped)
- **Skill Dependency Graph:** Track which tools/MCP servers a skill typically uses
- **Skill Cost/Usage Analytics:** Track API calls, token consumption per skill (requires Claude API telemetry integration)

---

## 12. Success Criteria & Verification

### Tier 1 Completion Checklist

**Tier 1 is complete when:**

- ✅ All skill invocations (detected via `toolName == "Skill"`) have their skill name extracted and stored
- ✅ Skills table grows with unique skill names on first invocation, with invocation counters
- ✅ Session-skills bridge tracks per-session skill usage counts
- ✅ UI shows skills inventory with usage stats (total invocations, session count, last seen)
- ✅ Session detail panel shows which skills were invoked in that session
- ✅ API endpoints return accurate, real-time skill data
- ✅ No skill invocations show as `category: "other"` (all properly categorized as `"skill"`)
- ✅ Unit, integration, and E2E tests pass
- ✅ No breaking changes to existing functionality

---

## 13. Implementation Checklist

### Deliverables by Component

### ast-cli
- [ ] `internal/commands/agenthooks/governance/normalize.go` — add `extractSkillName()` + skill case

### governance-api
- [ ] `internal/db/migrations/004_skills.sql` — new migration file
- [ ] `internal/ingest/ingest.go` — add `upsertSkill()`, `upsertSessionSkill()`, update `processOne()`
- [ ] `internal/api/dashboard.go` — add `ListSkills()`, `GetSkillDetail()`, `GetSessionSkills()`
- [ ] `internal/api/router.go` — register skill endpoints
- [ ] Tests: `internal/ingest/ingest_test.go`, `internal/api/dashboard_test.go`

### governance-ui
- [ ] `lib/api.ts` — add `skills()`, `skillDetail()`, `sessionSkills()` methods
- [ ] `app/skills/page.tsx` — new page for skills inventory
- [ ] `components/panels/session-panel.tsx` — add Skills tab to session detail
- [ ] (Optional) Navigation link in main nav

---

## 14. References & Related Documentation

- **Current MCP Implementation:** governance-api `upsertMCPServer()`, `upsertSessionMCPServer()`, `ListMCPServers()` endpoints
- **Hook Event Format:** ast-cx-hooks `claude/types.go` → `PreToolUseEvent` structure
- **Database Conventions:** See `001_initial_schema.sql` for naming/structure patterns
- **API Conventions:** See existing endpoints in `dashboard.go` for response format patterns
- **UI Conventions:** See existing skills/MCP inventory pages for styling/UX patterns

