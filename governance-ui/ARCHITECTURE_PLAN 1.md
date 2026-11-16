# AI Governance Auditing Platform — Architecture Plan
**Role:** Senior Architect, Checkmarx  
**Reference:** endorlabs.com/ai-coding-agent-governance  
**Scope:** Full stack — Database · Backend API · Hooks · Policy Sync · Dashboard UI

---

## 1. Executive Summary

This is a **new, standalone governance platform** for AI coding agents. It uses the `ast-cx-hooks` framework purely as a plumbing layer — for stdin/stdout hook dispatch and per-agent config patching. Everything else is built from scratch: its own hooks, its own policy format, its own audit pipeline, its own backend, its own dashboard.

**What we take from existing repos (plumbing only — no logic reuse):**
- `ast-cx-hooks` framework: stdin/stdout hook dispatch, `BeforeToolCall` / `BeforePrompt` / `WhenAgentIdle` registration API
- Per-agent config patchers (`InstallClaude`, `InstallCursor`) — to write hook commands into agent settings files

**What we are building (entirely new):**
- `cx hooks governance install` — installs governance hook routes into Claude and Cursor settings
- `~/.checkmarx/governance/policy-pack.json` — the governance policy file, authored in the dashboard and synced to the machine
- Local pack evaluator — `ALLOW / ALERT / BLOCK` decisions per tool category, zero network on hot path
- Audit spool (`audit-queue/`) → async `POST /api/v1/audit/events`
- Version-check sync — `HEAD /api/v1/agent/policies/version` on every hook call; full pack fetched only when version changes
- `cx policy sync` — manual / scheduled pull
- Policy API Server (Go + PostgreSQL) — policy CRUD, versioning, audit ingest
- Dashboard UI (Next.js/React) — policy editor, sessions, violations, MCP inventory, developer inventory

**What we do NOT use:**
- `policyhooks.json` — product guardrails file, not part of this platform
- `RegisterGuardrails()` / `HooksPolicy` — existing enforcement engine, not used
- Shell, prompt, file guardrails from `ast-hook-cli` / `ast-cli` — not used

---

## 2. Current State Analysis

### 2.1 What We Take from Existing Repos (plumbing only)

| Component | Location | Why we use it |
|---|---|---|
| Hook dispatch framework | `ast-cx-hooks` package | Handles stdin/stdout JSON protocol with Claude and Cursor |
| `BeforeToolCall` / `BeforePrompt` / `WhenAgentIdle` | `ast-cx-hooks/agenthooks` | Registration API for our governance handlers |
| `InstallClaude` | `ast-cx-hooks/install` | Patches `~/.claude/settings.json` with our hook commands |
| `InstallCursor` | `ast-cx-hooks/install` | Patches `~/.cursor/hooks.json` with our hook commands |

Everything else — guardrails logic, `HooksPolicy`, secret scanning, shell enforcement, MCP tools — is ignored. We build our own.

### 2.2 Gaps to Build

| # | Gap | What needs to be built |
|---|---|---|
| G1 | No governance hook handlers | New `governance/hooks.go` — `ALLOW / ALERT / BLOCK` based on governance pack |
| G2 | No `RuntimeEvent` / `ActivityEvent` structs | New `governance/normalize.go` — standard event schema |
| G3 | No audit spool | New `governance/spool.go` — atomic disk write + background HTTP flush |
| G4 | No `machineId` computation | New `governance/identity.go` — OS MachineGuid / machine-id / yaml fallback |
| G5 | No `agentId` computation | New `governance/identity.go` — SHA256(userId + agent.type + machineId) |
| G6 | No `developer_id` resolution | New `governance/identity.go` — resolve OS username → backend UUID |
| G7 | No policy version check | New `governance/sync.go` — `HEAD` version check after every hook call; fetch full pack only when changed |
| G8 | No `cx policy sync` command | New Cobra command calling `governance/sync.go` |
| G9 | No tool category normalizer | New `governance/normalize.go` — maps Claude/Cursor tool names → categories |
| G10 | No governance install command | New `governance/install.go` + `cx hooks governance install` |
| G11 | No MCP server inventory | ActivityEvent MCP fields + backend upsert on ingest |
| G12 | No session lifecycle tracking | Session start/end hooks → DB upsert |
| G13 | No Policy API Server | Phase 1 — Go server with full endpoint set |
| G14 | No Dashboard UI | Phase 5 — Next.js |
| G15 | Windows hook env injection | Governance config read from `checkmarxcli.yaml`, not env vars |
| G16 | No session-start stale check | `SyncIfStale()` called in session-start handler |

---

## 3. Architecture Overview

```
┌──────────────────────────────────────────────────────────────────────┐
│                      Governance Platform                              │
│                                                                       │
│  ┌──────────────────┐    ┌───────────────────────────────────────┐   │
│  │   Dashboard UI   │    │       Policy API Server (Go)           │   │
│  │  (Next.js/React) │◄──►│                                       │   │
│  │                  │    │  POST /api/v1/policies                 │   │
│  │  • Policy CRUD   │    │  PUT  /api/v1/policies/:id/activate    │   │
│  │  • Sessions      │    │  GET  /api/v1/agent/policies           │   │
│  │  • Violations    │    │  HEAD /api/v1/agent/policies/version   │   │
│  │  • MCP inventory │    │  POST /api/v1/audit/events             │   │
│  │  • Dev inventory │    │  GET  /api/v1/audit                    │   │
│  └──────────────────┘    └───────────────┬───────────────────────┘   │
└─────────────────────────────────────────┬┘                            │
                                          │ HTTPS                        │
                    ┌─────────────────────▼────────────────────────┐
                    │              Developer Machine                  │
                    │                                               │
                    │  cx governance CLI (NEW)                      │
                    │  ├── cx hooks governance install              │
                    │  │   ├── writes ~/.claude/settings.json       │
                    │  │   └── writes ~/.cursor/hooks.json          │
                    │  ├── cx policy sync                           │
                    │  └── version check (async, after each hook)   │
                    │                                               │
                    │  ~/.checkmarx/governance/                     │
                    │  ├── policy-pack.json   ← governance policy   │
                    │  ├── audit-queue/       ← event spool         │
                    │  ├── logs/              ← local session logs   │
                    │  │   └── session-<id>.log  (PII-free NDJSON)  │
                    │  └── agents/<type>/agent.json ← identity      │
                    └──────────────────────────────────────────────┘
                                          │
                    ┌─────────────────────▼────────────────────────┐
                    │          AI Coding Agent                       │
                    │   Claude Code / Cursor                         │
                    │   fires hook → cx hooks <route>               │
                    │   ├── governance pack eval (local, ~1–4ms)    │
                    │   │   └── ALLOW / ALERT / BLOCK               │
                    │   └── verdict returned to agent               │
                    │       audit event spooled async               │
                    └──────────────────────────────────────────────┘
```

**Cardinal rule:** Never `Hook → Backend → DB → decision` on a tool call. Local pack only on the hot path.

---

## 4. Hook Design

### 4.1 New Package Structure (in ast-cli, new package — no existing files touched)

```
internal/commands/agenthooks/
└── governance/                  ← NEW — entire package
    ├── cmd.go                   ← Cobra commands: install, sync
    ├── hooks.go                 ← hook handlers: session, tool, prompt
    ├── install.go               ← writes governance routes into agent configs
    ├── normalize.go             ← RuntimeEvent builder + tool category normalizer
    ├── pack.go                  ← policy-pack.json loader + Evaluate()
    ├── identity.go              ← machineId, agentId, developer_id resolution
    ├── spool.go                 ← audit-queue/ writer + background flusher
    ├── sync.go                  ← SyncOnSessionStart, CheckAndSyncVersion, SyncOnce
    └── log.go                   ← local session log writer (PII-free)
```

### 4.2 Hook Handlers (hooks.go)

```go
func RegisterGovernanceHooks(cfg GovernanceConfig) {
    agenthooks.WhenAgentIdle(governanceSessionStart(cfg))
    agenthooks.BeforeToolCall(governancePreToolUse(cfg))
    agenthooks.AfterToolCall(governancePostToolUse(cfg))   // delivers ALERT messages to agent
    agenthooks.BeforePrompt(governancePrePrompt(cfg))
}

func governancePreToolUse(cfg GovernanceConfig) func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
    return func(ev agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
        pack := pack.Load()                            // reads policy-pack.json, ~0ms
        if pack == nil {
            return agenthooks.Allow()                  // fail-open: no pack = allow all
        }
        event    := normalize.BuildRuntimeEvent(ev, cfg)  // tool category + match fields
        decision := pack.Evaluate(event)               // local regex eval, no network

        go spool.Write(event.WithDecision(decision))                   // async audit spool
        go sync.CheckAndSyncVersion(cfg.ServerURL, cfg.Token)          // async version check

        switch decision.Action {
        case "BLOCK":
            return agenthooks.Deny(decision.Message())  // agentMessage shown in agent; falls back to policyName
        case "ALERT":
            pendingAlerts.Set(ev.EventID, decision)     // picked up by PostToolUse to show warning
            return agenthooks.Allow()
        default:
            return agenthooks.Allow()
        }
    }
}

// governancePostToolUse fires after the tool completes.
// For ALERT decisions, it injects the agentMessage as a warning note visible in the agent.
func governancePostToolUse(cfg GovernanceConfig) func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
    return func(ev agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
        if alert := pendingAlerts.Pop(ev.EventID); alert != nil {
            return agenthooks.AllowWithNote("⚠ " + alert.Message())
        }
        return agenthooks.Allow()
    }
}
```

### 4.2.1 How ALERT Message Appears in the Agent

For `BLOCK`, the flow is straightforward — `Deny(message)` stops the tool call and Claude or Cursor surfaces the message directly to the developer.

For `ALERT`, the tool call is **allowed through** but the message needs to reach the developer afterward. This is a two-hook sequence:

```
PreToolUse fires
  → policy evaluates → ALERT
  → decision stored in pendingAlerts keyed by eventId
  → return Allow()        ← tool proceeds normally

[tool executes — Read, Bash, MCP call, etc.]

PostToolUse fires (same eventId)
  → pendingAlerts.Pop(eventId) → retrieves stored decision
  → return AllowWithNote("⚠ " + agentMessage)
```

`AllowWithNote` maps to the Claude hook stdout format:

```json
{
  "continue": true,
  "stopReason": null,
  "suppressOutput": false,
  "note": "⚠ Destructive commands are monitored. This action has been logged for security review."
}
```

Claude injects the `note` into its context and surfaces it in the conversation after the tool result:

```
> [Bash: rm -rf ./tmp completed]
⚠ Destructive commands are monitored. This action has been
  logged for security review.
```

**Cursor** uses the same PostToolUse hook mechanism — the `note` field is rendered inline after the tool output in the agent panel.

**pendingAlerts** is an in-process thread-safe map scoped to the subprocess lifetime:

```go
// keyed by eventId — lives only for the duration of the hook subprocess
var pendingAlerts = &sync.Map{}

func (m *sync.Map) Set(eventID string, d Decision) {
    m.Store(eventID, d)
}
func (m *sync.Map) Pop(eventID string) *Decision {
    if v, ok := m.LoadAndDelete(eventID); ok {
        d := v.(Decision)
        return &d
    }
    return nil
}
```

Note: `PreToolUse` and `PostToolUse` for the same tool call run in the **same subprocess** — the ast-cx-hooks framework dispatches both hooks within a single `cx hooks` process invocation for that event, so `pendingAlerts` is shared in memory between the two handlers.

```go

func governanceSessionStart(cfg GovernanceConfig) func(agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
    return func(ev agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
        sync.SyncOnSessionStart(cfg.ServerURL, cfg.Token)  // synchronous — session start can afford ~500ms
        go identity.EnsureResolved(cfg)                    // resolve developer_id if not cached
        go spool.StartFlusher(cfg.ServerURL, cfg.Token)    // drain any queued audit events
        return agenthooks.Resume()
    }
}
```

### 4.3 Tool Category Normalizer (normalize.go — fills G9)

Maps Claude and Cursor tool names to governance categories and extracts the match field used for policy evaluation and sent in every `ActivityEvent`.

```go
func normalizeTool(toolName string, input map[string]any) ToolInfo {
    switch {
    case isFilesystemTool(toolName):              // Read, Write, Edit, MultiEdit, Glob, Grep, LS
        return ToolInfo{Category: "filesystem", Name: toolName, Paths: extractPaths(input)}
    case toolName == "Bash":
        return ToolInfo{Category: "shell", Name: "Bash", Command: extractCommand(input)}
    case strings.HasPrefix(toolName, "mcp__"):    // mcp__github__create_issue
        server, tool := parseMCPName(toolName)
        return ToolInfo{Category: "mcp", Name: tool, RawName: toolName,
            MCP: &MCPInfo{Server: server, Name: tool}}
    case toolName == "WebSearch" || toolName == "WebFetch":
        return ToolInfo{Category: "web_search", Name: toolName, Query: extractQuery(input)}
    default:
        return ToolInfo{Category: "other", RawName: toolName}
    }
}
```

| Claude / Cursor `tool_name` | Category | Match field sent in ActivityEvent |
|---|---|---|
| Read, Write, Edit, MultiEdit, Glob, Grep, LS | `filesystem` | `tool.paths[]` |
| Bash | `shell` | `tool.command` |
| `mcp__{server}__{tool}` | `mcp` | `mcp.server` + `mcp.name` |
| WebSearch, WebFetch | `web_search` | `tool.query` |
| UserPromptSubmit | `prompt` | `prompt.length` (content redacted) |
| unknown | `other` | `tool.rawName` |

### 4.4 Identity Resolution (identity.go — fills G4, G5, G6)

```go
func MachineID() string {
    // 1. Windows: HKLM\SOFTWARE\Microsoft\Cryptography\MachineGuid (registry)
    // 2. Linux:   /etc/machine-id
    // 3. Mac:     ioreg -rd1 -c IOPlatformExpertDevice | grep IOPlatformUUID
    // 4. Fallback: generate UUID once, store as cx_machine_id in checkmarxcli.yaml
}

func AgentID(agentType string) string {
    if cached := readAgentJSON(agentType).AgentID; cached != "" {
        return cached
    }
    id := sha256hex(sha256hex(CurrentUserID()) + "|" + sha256hex(agentType) + "|" + MachineID())
    writeAgentJSON(agentType, AgentJSON{AgentID: id})
    go registerWithBackend(id, agentType)   // POST /api/v1/agents/register (non-blocking)
    return id
}

func EnsureResolved(cfg GovernanceConfig) {
    // GET /api/v1/developers?username={os.user.Current().Username}
    // Caches developer_id in agents/<type>/agent.json
    // All subsequent ActivityEvents carry this developer_id as the DB FK
}
```

### 4.5 Local Pack Evaluator (pack.go)

```go
type GovernancePack struct {
    PackVersion int          `json:"packVersion"`
    Policies    []PackPolicy `json:"policies"`
    UpdatedAt   time.Time    `json:"updatedAt"`
}

type PackPolicy struct {
    PolicyID     string `json:"policyId"`
    PolicyName   string `json:"policyName"`
    Version      int    `json:"policyVersion"`
    Type         string `json:"type"`         // filesystem | shell | mcp | web_search | prompt
    Action       string `json:"action"`       // BLOCK | ALERT
    MatchTarget  string `json:"matchTarget"`  // path | command | mcp_tool | query | prompt
    Pattern      string `json:"pattern"`      // regex
    AgentMessage string `json:"agentMessage,omitempty"` // optional: custom message shown in agent on match
}

type Decision struct {
    Action        string
    PolicyID      string
    PolicyName    string
    AgentMessage  string  // from PackPolicy.AgentMessage
    PolicyVersion int
    PackVersion   int
    MatchedOn     string
    MatchedValue  string
}

// Message returns the text shown to the developer in the agent.
// Uses AgentMessage if the admin set one; falls back to PolicyName.
func (d Decision) Message() string {
    if d.AgentMessage != "" {
        return d.AgentMessage
    }
    return d.PolicyName
}

func (p *GovernancePack) Evaluate(ev RuntimeEvent) Decision {
    for _, policy := range p.Policies {
        if policy.Type != ev.Tool.Category {
            continue
        }
        if matched, value := matchPolicy(policy, ev); matched {
            return Decision{
                Action:        policy.Action,
                PolicyID:      policy.PolicyID,
                PolicyName:    policy.PolicyName,
                AgentMessage:  policy.AgentMessage,
                PolicyVersion: policy.Version,
                PackVersion:   p.PackVersion,
                MatchedOn:     policy.MatchTarget,
                MatchedValue:  value,
            }
        }
    }
    return Decision{Action: "ALLOW"}
}
```

### 4.6 Audit Spool (spool.go — fills G3)

The flusher uses a **channel-based signal** instead of a fixed sleep interval. `Write()` signals the flusher the moment a file lands on disk — reducing spool-to-backend lag from 5–10 seconds to ~100ms (one network round-trip). The 10-second fallback ticker catches any files missed if the signal was dropped.

```go
const SpoolDir = "~/.checkmarx/governance/audit-queue"

// flushSignal wakes the flusher immediately when a new event is written.
// Buffered capacity 1: if a flush is already pending, extra signals are dropped (no-op).
var flushSignal = make(chan struct{}, 1)

func Write(ev ActivityEvent) {
    data, _ := json.Marshal(ev)
    tmp  := filepath.Join(spoolDir(), "evt_"+ev.EventID+".tmp")
    dest := filepath.Join(spoolDir(), "evt_"+ev.EventID+".json")
    os.WriteFile(tmp, data, 0o600)
    os.Rename(tmp, dest)   // atomic: never a partial file

    // signal flusher to wake immediately — non-blocking
    select {
    case flushSignal <- struct{}{}:
    default:   // flush already queued, skip
    }
}

func StartFlusher(serverURL, token string) {
    go func() {
        for {
            select {
            case <-flushSignal:                  // woken immediately by Write()
            case <-time.After(10 * time.Second): // fallback: catch any missed files
            }

            files, _ := filepath.Glob(filepath.Join(spoolDir(), "*.json"))
            if len(files) == 0 { continue }

            batch := readBatch(files, 50)
            err   := postBatch(serverURL, token, batch)
            if err == nil {
                deleteFiles(batch)                // ACK: remove from queue
            }
            // 4xx → move to audit-queue/dead/ (bad payload, stop retrying)
            // 5xx / network error → leave files, retry on next signal or tick
        }
    }()
}

### 4.7 Local Session Log (log.go)

Every hook event is appended to a local NDJSON log file scoped to the session. The log is written on the developer's machine only — it is never uploaded to the backend. It contains no personal information.

**File structure:**

```
~/.checkmarx/governance/
└── logs/
    ├── session-2026-08-12-sess_abc.log    ← one file per session
    ├── session-2026-08-12-sess_def.log
    ├── session-2026-08-11-sess_ghi.log
    └── session-2026-07-10-sess_jkl.log   ← auto-purged after 30 days
```

**File naming convention:**
```
session-<YYYY-MM-DD>-<sessionId>.log
```
Date prefix makes retention purge efficient — `purgeStaleLogs()` can identify old files without reading their contents.

**File format:** NDJSON — one JSON object per line, newline-delimited. Each line is one hook event. File is append-only; never overwritten or truncated.

```
<entry>\n
<entry>\n
<entry>\n
...
```

**Retention:** Files older than 30 days are deleted on session start by `purgeStaleLogs()`. No manual cleanup required.

**What is logged — PII-free fields only:**

```go
// governance/log.go

type SessionLogEntry struct {
    Timestamp     time.Time `json:"ts"`
    EventID       string    `json:"eventId"`
    SessionID     string    `json:"sessionId"`
    Kind          string    `json:"kind"`          // session.start | tool.pre | prompt.submit | session.end
    AgentType     string    `json:"agentType"`     // claude-code | cursor
    Category      string    `json:"category"`      // filesystem | shell | mcp | web_search | prompt | ""
    ToolName      string    `json:"toolName"`      // Read | Write | Bash | mcp__github__create_issue | ""
    Action        string    `json:"action"`        // ALLOW | ALERT | BLOCK | ""
    PolicyName    string    `json:"policyName"`    // policy that fired, "" if ALLOW
    PolicyVersion int       `json:"policyVersion"` // 0 if ALLOW
    MatchedOn     string    `json:"matchedOn"`     // path | command | mcp_tool | "" — never prompt/query
    MatchedValue  string    `json:"matchedValue"`  // see PII rules below
}
```

**PII exclusion rules — what is never written to the log:**

| Field | Why excluded |
|---|---|
| `userId` / `userName` | OS username — personal identity |
| `machineId` | Device identifier — personal identity |
| `developer_id` | Backend UUID — personal identity |
| `agentId` | Derived from userId + machineId — personal identity |
| `email` | Personal identity |
| Prompt content | May contain personal data, secrets, or sensitive context |
| `matchedValue` when `kind = prompt.submit` | Prompt match content is personal data |
| `matchedValue` when `matchedOn = query` | Web search query may be personal |
| File contents | Never read or logged |

**`matchedValue` is logged only for:** `matchedOn = path` / `command` / `mcp_tool` — these are tool names and file paths, not personal data.

```go
func WriteSessionLog(sessionID string, ev RuntimeEvent, decision Decision) {
    entry := SessionLogEntry{
        Timestamp:  ev.Timestamp,
        EventID:    ev.EventID,
        SessionID:  sessionID,
        Kind:       ev.Kind,
        AgentType:  ev.Agent.Type,
        Category:   ev.Tool.Category,
        ToolName:   ev.Tool.Name,
        Action:     decision.Action,
        PolicyName: decision.PolicyName,
        PolicyVersion: decision.PolicyVersion,
        MatchedOn:  safeMatchedOn(ev, decision),
        MatchedValue: safeMatchedValue(ev, decision),
    }
    appendToLog(sessionLogPath(sessionID), entry)
}

// safeMatchedValue returns the matched value only when it is not personal data.
func safeMatchedValue(ev RuntimeEvent, d Decision) string {
    if ev.Kind == "prompt.submit" { return "" }
    if d.MatchedOn == "query"     { return "" }
    return d.MatchedValue
}

func purgeStaleLogs(maxAge time.Duration) {
    files, _ := filepath.Glob(logDir() + "/session-*.log")
    for _, f := range files {
        info, _ := os.Stat(f)
        if info != nil && time.Since(info.ModTime()) > maxAge {
            os.Remove(f)
        }
    }
}
```

**Example log file** (`session-sess_xyz.log`):

```json
{"ts":"2026-08-12T10:00:01Z","eventId":"evt_001","sessionId":"sess_xyz","kind":"session.start","agentType":"claude-code","category":"","toolName":"","action":"","policyName":"","policyVersion":0,"matchedOn":"","matchedValue":""}
{"ts":"2026-08-12T10:02:14Z","eventId":"evt_002","sessionId":"sess_xyz","kind":"tool.pre","agentType":"claude-code","category":"filesystem","toolName":"Read","action":"ALLOW","policyName":"","policyVersion":0,"matchedOn":"","matchedValue":""}
{"ts":"2026-08-12T10:12:47Z","eventId":"evt_003","sessionId":"sess_xyz","kind":"tool.pre","agentType":"claude-code","category":"filesystem","toolName":"Read","action":"BLOCK","policyName":"Block credential files","policyVersion":3,"matchedOn":"path","matchedValue":"C:/proj/.env"}
{"ts":"2026-08-12T10:23:05Z","eventId":"evt_004","sessionId":"sess_xyz","kind":"prompt.submit","agentType":"claude-code","category":"prompt","toolName":"","action":"ALERT","policyName":"Alert on prompt patterns","policyVersion":1,"matchedOn":"","matchedValue":""}
{"ts":"2026-08-12T10:47:00Z","eventId":"evt_005","sessionId":"sess_xyz","kind":"session.end","agentType":"claude-code","category":"","toolName":"","action":"","policyName":"","policyVersion":0,"matchedOn":"","matchedValue":""}
```

Notice: `prompt.submit` entry has empty `matchedOn` and `matchedValue` — prompt content is never written.

**Hooks.go — log call added after every decision:**

```go
func governancePreToolUse(cfg GovernanceConfig) func(agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
    return func(ev agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
        pack := pack.Load()
        if pack == nil { return agenthooks.Allow() }

        event    := normalize.BuildRuntimeEvent(ev, cfg)
        decision := pack.Evaluate(event)

        go spool.Write(event.WithDecision(decision))
        go sync.CheckAndSyncVersion(cfg.ServerURL, cfg.Token)
        log.WriteSessionLog(ev.SessionID, event, decision)   // sync, local disk, ~microseconds

        switch decision.Action {
        case "BLOCK":
            return agenthooks.Deny(decision.Message())
        case "ALERT":
            pendingAlerts.Set(ev.EventID, decision)
            return agenthooks.Allow()
        default:
            return agenthooks.Allow()
        }
    }
}
```

`log.WriteSessionLog` is called synchronously (not in a goroutine) — it is a local disk append, takes microseconds, and must complete before the subprocess exits.

---

### 4.8 Policy Version Check + Sync (sync.go — fills G7, G8, G16)

`cx hooks <route>` is a short-lived subprocess — it starts, returns a verdict, and exits. There is no persistent process to hold an SSE connection between hook calls. Instead, a lightweight version check runs asynchronously after every verdict is returned, and a synchronous check runs at session start where latency is acceptable.

```go
// SyncOnSessionStart — called synchronously from session-start hook.
// Session start is the one place a small network delay is acceptable.
func SyncOnSessionStart(serverURL, token string) {
    remote := fetchPackVersion(serverURL, token, 500*time.Millisecond)
    if remote == -1 || remote <= readLocalPackVersion() {
        return   // unreachable or no change
    }
    fetchAndWritePack(serverURL, token)
}

// CheckAndSyncVersion — called in a goroutine after every hook verdict.
// Returns immediately; fetches full pack in background only if version changed.
func CheckAndSyncVersion(serverURL, token string) {
    remote := fetchPackVersion(serverURL, token, 200*time.Millisecond)
    if remote == -1 || remote <= readLocalPackVersion() {
        return   // fail-open or no change
    }
    fetchAndWritePack(serverURL, token)
}

// fetchPackVersion — HEAD request, returns X-Pack-Version header value.
// Returns -1 on error or timeout so callers always fail-open.
func fetchPackVersion(serverURL, token string, timeout time.Duration) int {
    resp, err := headWithTimeout(serverURL+"/api/v1/agent/policies/version", token, timeout)
    if err != nil || resp.StatusCode != 200 {
        return -1
    }
    return parseIntHeader(resp.Header.Get("X-Pack-Version"))
}

// fetchAndWritePack — downloads full pack and atomically replaces local file.
func fetchAndWritePack(serverURL, token string) {
    resp, err := getWithAuth(serverURL+"/api/v1/agent/policies", token)
    if err != nil { return }   // fail-open: keep existing pack
    atomicWritePack(resp.Body) // write to .tmp → rename → policy-pack.json
}

// SyncOnce — used by `cx policy sync` manual command.
func SyncOnce(serverURL, token string) error {
    resp, err := getWithAuth(serverURL+"/api/v1/agent/policies", token)
    if err != nil { return err }
    return atomicWritePack(resp.Body)
}
```

**Policy update propagation timing:**

| Scenario | Lag |
|---|---|
| Developer is actively using Claude / Cursor | Next tool call after admin saves — seconds |
| Developer starts a new session | Session-start hook syncs synchronously — always fresh |
| Machine offline when policy changed | Next hook call after reconnect picks it up |

### 4.8 Governance Install (install.go — fills G10)

Writes governance hook routes into the agent's config. Governance installs independently — it does not read or modify any existing entries in the config file.

```go
var GovernanceRoutes = map[string][]string{
    "claude": {
        "claude-session-start",
        "claude-pre-tool-use",
        "claude-user-prompt-submit",
        "claude-post-tool-use",
        "claude-session-end",
    },
    "cursor": {
        "cursor-session-start",
        "cursor-before-shell",
        "cursor-before-submit-prompt",
        "cursor-before-mcp",
        "cursor-before-file-read",
        "cursor-session-end",
    },
}

// Resulting ~/.claude/settings.json (governance only):
// {
//   "hooks": [
//     {
//       "matcher": ".*",
//       "hooks": [
//         { "type": "command", "command": "cx hooks claude-pre-tool-use" },
//         { "type": "command", "command": "cx hooks claude-user-prompt-submit" },
//         { "type": "command", "command": "cx hooks claude-session-start" }
//       ]
//     }
//   ]
// }
```

**Windows note:** No `VAR=value` env injection in hook commands. Server URL and token are read from `~/.checkmarx/checkmarxcli.yaml` by the `cx` process at hook runtime.

---

## 5. Database Design

**Database:** PostgreSQL 16  
**ORM / query layer:** pgx direct or sqlc  
**Migrations:** golang-migrate, run in order

### 5.0 Design Principles

- No organisation layer — scoped to developers only
- Policy versions are never overwritten; a new version row is always created on change
- `policies.current_version_id` points to the active version
- `audit_events` is append-only — no UPDATE or DELETE ever
- Every policy action stores the exact `policy_version_id` that produced the decision for permanent auditability

### 5.0.1 Entity Relationship Overview

```
Developer
    │
    ├── developer_agents (bridge: developer + machine + agent product)
    │
    ├── AI Sessions
    │       │
    │       ├── AI Coding Agent  (product catalog)
    │       ├── AI Model
    │       └── Session MCP Servers ──► MCP Servers
    │
    └── Policy Actions
            │
            ├── Policy ──► Policy Versions (current_version_id)
            └── Policy Version (snapshot of decision)

Audit Events (append-only log of all system mutations — server-side only)
```

---

### 5.1 Policies & Versions

#### `policies`

| Column | Type | Notes |
|---|---|---|
| `policy_id` | UUID PK | |
| `policy_name` | VARCHAR | e.g. "Block credential files" |
| `description` | TEXT | |
| `current_version_id` | UUID FK → policy_versions | Points to the active version |
| `status` | ENUM | `active`, `inactive` |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |

#### `policy_versions`

| Column | Type | Notes |
|---|---|---|
| `policy_version_id` | UUID PK | |
| `policy_id` | UUID FK → policies | |
| `version_number` | INTEGER | Auto-incremented per policy |
| `definition` | JSONB | Full `GovernancePack` JSON snapshot (policy-pack.json format, includes `agentMessage` per policy) — write-once, never mutated |
| `enforcement_mode` | ENUM | `alert`, `block` |
| `created_by` | UUID FK → developers | Who authored this version |
| `created_at` | TIMESTAMP | |
| `change_summary` | TEXT | Human-readable note e.g. "Escalated to BLOCK" |

**Key rule:** `definition` is write-once. Changing a policy always inserts a new `policy_versions` row and updates `policies.current_version_id`.

Version history example:
```
Policy: Block credential files
  Version 1 → enforcement_mode: alert
  Version 2 → enforcement_mode: block   ← current_version_id
```

#### `audit_events`

| Column | Type | Notes |
|---|---|---|
| `audit_event_id` | UUID PK | |
| `actor_id` | UUID FK → developers | Who performed the action |
| `actor_type` | ENUM | `developer`, `system` |
| `event_type` | VARCHAR | See event types below |
| `resource_type` | VARCHAR | e.g. `policy`, `developer` |
| `resource_id` | UUID | ID of the affected resource |
| `old_value` | JSONB | State before change (nullable) |
| `new_value` | JSONB | State after change (nullable) |
| `timestamp` | TIMESTAMP | |
| `metadata` | JSONB | Extra context (IP, agent version, etc.) |

**Event types:**
```
POLICY_CREATED        POLICY_VERSION_CREATED
POLICY_ACTIVATED      POLICY_DEACTIVATED
POLICY_ROLLED_BACK    POLICY_UPDATED
DEVELOPER_ADDED       DEVELOPER_DISABLED
AGENT_REGISTERED      AGENT_DISABLED
```

**Rules:** Append-only — no UPDATE or DELETE. `old_value` / `new_value` store full JSON snapshots, not diffs. Written by backend API server only — `cx` never writes here directly.

---

### 5.2 Developer & Agent Inventory

#### `developers`

| Column | Type | Notes |
|---|---|---|
| `developer_id` | UUID PK | |
| `email` | VARCHAR UNIQUE | |
| `display_name` | VARCHAR | |
| `team` | VARCHAR | |
| `department` | VARCHAR | |
| `role` | VARCHAR | e.g. `admin`, `developer` |
| `status` | ENUM | `active`, `disabled` |
| `first_seen_at` | TIMESTAMP | |
| `last_seen_at` | TIMESTAMP | |

#### `ai_coding_agents` (product catalog)

One row per agent product globally. Not per developer.

| Column | Type | Notes |
|---|---|---|
| `agent_id` | UUID PK | |
| `agent_name` | VARCHAR | e.g. "Claude Code", "Cursor" |
| `vendor` | VARCHAR | e.g. "Anthropic", "Anysphere" |
| `agent_type` | ENUM | `ide_extension`, `cli`, `standalone` |
| `version` | VARCHAR | |
| `environment` | VARCHAR | e.g. `vscode`, `terminal` |
| `status` | ENUM | `active`, `disabled` |
| `first_seen_at` | TIMESTAMP | |
| `last_seen_at` | TIMESTAMP | |
| `metadata` | JSONB | |

Known agents: Claude Code, Cursor, GitHub Copilot, Windsurf, Amazon Q, Gemini CLI

#### `developer_agents` (bridge — per developer × machine × agent)

Bridges the `ai_coding_agents` product catalog with per-developer, per-machine instances. One row per unique combination of developer + agent product + machine.

| Column | Type | Notes |
|---|---|---|
| `developer_agent_id` | UUID PK | |
| `developer_id` | UUID FK → developers | |
| `agent_product_id` | UUID FK → ai_coding_agents | Global product catalog row (e.g. "Claude Code") |
| `machine_id` | VARCHAR | OS machine GUID (Windows) / `/etc/machine-id` (Linux) / platform UUID (Mac) |
| `governance_agent_id` | VARCHAR UNIQUE | SHA256 hex — deterministic instance id: `SHA256(SHA256(userId) + "\|" + SHA256(agent.type) + "\|" + machineId)` |
| `first_seen_at` | TIMESTAMP | |
| `last_seen_at` | TIMESTAMP | |

#### `ai_models`

| Column | Type | Notes |
|---|---|---|
| `model_id` | UUID PK | |
| `provider` | VARCHAR | e.g. "Anthropic", "OpenAI", "Google" |
| `model_name` | VARCHAR | e.g. "claude-sonnet-4-6", "gpt-4o" |
| `model_version` | VARCHAR | |
| `model_family` | VARCHAR | e.g. "Claude", "GPT", "Gemini" |
| `status` | ENUM | `active`, `deprecated` |
| `metadata` | JSONB | |

---

### 5.3 Session Tracking

#### `ai_sessions`

| Column | Type | Notes |
|---|---|---|
| `session_id` | UUID PK | Generated by backend on first event for this Claude/Cursor session |
| `claude_session_id` | VARCHAR | Raw session string from Claude/Cursor stdin — used for lookup |
| `developer_id` | UUID FK → developers | |
| `agent_id` | UUID FK → ai_coding_agents | Product catalog row |
| `model_id` | UUID FK → ai_models | Nullable |
| `started_at` | TIMESTAMP | |
| `ended_at` | TIMESTAMP | Nullable — ongoing session |
| `duration` | INTEGER | Seconds — computed on session-end and stored |
| `request_count` | INTEGER | Total prompts — incremented on each prompt.submit event |
| `tool_call_count` | INTEGER | Incremented on each tool.pre event |
| `status` | ENUM | `active`, `ended`, `expired` |
| `metadata` | JSONB | |

Example query — all sessions for developer X using Cursor in last 30 days:
```sql
SELECT s.*
FROM ai_sessions s
JOIN ai_coding_agents a ON s.agent_id = a.agent_id
WHERE s.developer_id = :developer_id
  AND a.agent_name = 'Cursor'
  AND s.started_at >= NOW() - INTERVAL '30 days';
```

---

### 5.4 MCP Governance

#### `mcp_servers`

| Column | Type | Notes |
|---|---|---|
| `mcp_server_id` | UUID PK | |
| `server_name` | VARCHAR | e.g. "github", "stripe" |
| `server_url` | VARCHAR | |
| `vendor` | VARCHAR | |
| `server_version` | VARCHAR | |
| `status` | ENUM | `active`, `disabled` |
| `first_seen_at` | TIMESTAMP | Auto-populated on first ActivityEvent |
| `last_seen_at` | TIMESTAMP | |
| `metadata` | JSONB | |

Discovery only — no approval flag. Block/warn decisions are handled via policies (type=`mcp`) in the Policy dashboard.

#### `session_mcp_servers` (bridge)

| Column | Type | Notes |
|---|---|---|
| `session_id` | UUID PK/FK → ai_sessions | |
| `mcp_server_id` | UUID PK/FK → mcp_servers | |
| `first_used_at` | TIMESTAMP | |
| `last_used_at` | TIMESTAMP | |
| `tool_call_count` | INTEGER | |
| `blocked_call_count` | INTEGER | |

---

### 5.5 Policy Actions (Enforcement Decisions)

#### `policy_actions`

| Column | Type | Notes |
|---|---|---|
| `policy_action_id` | UUID PK | |
| `policy_id` | UUID FK → policies | |
| `policy_version_id` | UUID FK → policy_versions | Exact version that produced this decision |
| `session_id` | UUID FK → ai_sessions | |
| `developer_id` | UUID FK → developers | |
| `agent_id` | UUID FK → ai_coding_agents | |
| `mcp_server_id` | UUID FK → mcp_servers | Nullable |
| `action_type` | ENUM | `ALLOW`, `ALERT`, `BLOCK` |
| `violation_type` | VARCHAR | `path` / `command` / `mcp_tool` / `query` / `prompt` — from `decision.matchedOn` |
| `severity` | ENUM | `low` (ALLOW), `medium` (ALERT), `high` (BLOCK) — derived from `action_type` |
| `resource_type` | VARCHAR | `filesystem` / `shell` / `mcp` / `web_search` / `prompt` — from `tool.category` |
| `resource_id` | VARCHAR | Matched value — file path / command / mcp tool / query |
| `reason` | TEXT | `policyName` from the decision |
| `matched_rule` | VARCHAR | `policyId` from the decision |
| `occurred_at` | TIMESTAMP | |
| `resolution_status` | ENUM | `open`, `resolved`, `suppressed` |
| `resolved_at` | TIMESTAMP | Nullable |
| `metadata` | JSONB | `latencyMs`, `packVersion` |

**Auditability:** `policy_version_id` is a permanent forensic link. Six months later, even after ten policy upgrades, you can still answer "which exact rule produced this block" because the action points to the version's `definition` JSONB.

---

### 5.6 ActivityEvent → DB Field Mapping

How `ActivityEvent` fields emitted by `cx` map to DB columns on ingest:

| ActivityEvent field | DB table | DB column |
|---|---|---|
| `decision.policyId` | `policies` | `policy_id` |
| `decision.policyVersion` | `policy_versions` | `version_number` |
| `decision.action` | `policy_actions` | `action_type` |
| `decision.matchedOn` | `policy_actions` | `violation_type` |
| `decision.matchedValue` | `policy_actions` | `resource_id` |
| `tool.category` | `policy_actions` | `resource_type` |
| `sessionId` (Claude string) | `ai_sessions` | `claude_session_id` → resolved to `session_id` UUID |
| `userId` → resolved `developer_id` | `policy_actions` | `developer_id` |
| `agentId` → resolved `developer_agent_id` | `policy_actions` | `agent_id` (product catalog UUID) |
| `mcp.server` → resolved `mcp_server_id` | `policy_actions` | `mcp_server_id` |
| `decision.latencyMs` | `policy_actions` | `metadata` (JSONB) |
| `decision.packVersion` | `policy_actions` | `metadata` (JSONB) |
| `timestamp` | `policy_actions` | `occurred_at` |

**Version resolution:** `decision.policyVersion` (INTEGER) + `decision.policyId` (string) → look up `policy_version_id` (UUID) via `version_number` + `policy_id` join.

**Severity derivation:** computed from `action_type` — `ALLOW` → `low`, `ALERT` → `medium`, `BLOCK` → `high`. Never sent by `cx`; always set by backend on ingest.

---

### 5.7 Developer & Agent Identity Resolution

`cx` knows the OS username (`userId` from `os/user.Current()`). The DB identifies developers by UUID. Resolution on first hook fire:

```
cx session-start hook fires
  →  GET /api/v1/developers?username={userId}
  →  backend returns developer_id (UUID)
  →  cached in ~/.checkmarx/governance/agents/<type>/agent.json
All subsequent ActivityEvents carry developer_id as the FK
```

**Two distinct agent ID concepts — never conflate:**

| Concept | Type | Meaning |
|---|---|---|
| `ai_coding_agents.agent_id` | UUID (DB PK) | Global product catalog — one row for "Claude Code", one for "Cursor" |
| `governance_agent_id` in `developer_agents` | SHA256 hex | Per-developer + per-machine + per-agent-type instance identifier |

```
governance_agent_id = SHA256(SHA256(userId) + "|" + SHA256(agent.type) + "|" + machineId)
```

On first ActivityEvent from a new `governance_agent_id`, the backend upserts a `developer_agents` bridge row.

---

### 5.8 Key Query Filters

| Entity | Primary filters |
|---|---|
| Policies | `policy_id`, `status`, `enforcement_mode` |
| Developers | `developer_id`, `email`, `team`, `department`, `role`, `status` |
| Agents | `agent_id`, `agent_name`, `vendor` |
| Models | `model_id`, `provider`, `model_family` |
| Sessions | `developer_id`, `machine_id`, `agent_id`, `started_at` range |
| MCP Servers | `server_name`, `vendor`, `status`, `first_seen_at` |
| Policy Actions | `developer_id`, `machine_id`, `agent_id`, `policy_version_id`, `action_type`, `severity`, `occurred_at` range |
| Audit Events | `actor_id`, `event_type`, `resource_type`, `timestamp` range |

---

### 5.9 Governance Pack Format (`policy_versions.definition`)

The `definition` JSONB stores the governance pack exactly as `cx` receives and writes it to `policy-pack.json`:

The `definition` JSONB in `policy_versions` stores the governance pack exactly as `cx` receives and writes it to `policy-pack.json`:

```json
{
  "packVersion": 13,
  "policies": [
    {
      "policyId": "pol_abc",
      "policyName": "Block credential files",
      "policyVersion": 3,
      "type": "filesystem",
      "action": "BLOCK",
      "matchTarget": "path",
      "pattern": "\\.env$|\\.pem$|\\.key$",
      "agentMessage": "Production credential files are restricted. Use the secrets manager at vault.company.com instead of reading credential files directly."
    },
    {
      "policyId": "pol_def",
      "policyName": "Alert on destructive shell",
      "policyVersion": 1,
      "type": "shell",
      "action": "ALERT",
      "matchTarget": "command",
      "pattern": "rm\\s+-rf",
      "agentMessage": "Destructive commands are monitored. This action has been logged for security review."
    },
    {
      "policyId": "pol_ghi",
      "policyName": "Block GitHub MCP tools",
      "policyVersion": 2,
      "type": "mcp",
      "action": "BLOCK",
      "matchTarget": "mcp_tool",
      "pattern": "^github/.*"
    }
  ]
}
```

---

## 6. Backend API Server

**Language:** Go  
**DB:** PostgreSQL 16 (full schema in section 5)  
**Transport:** HTTP/1.1

### 6.1 Endpoint Inventory

```
# cx CLI — agent-facing
GET  /api/v1/agent/policies              → serve current pack (returns full policy-pack.json)
HEAD /api/v1/agent/policies/version      → returns X-Pack-Version header only; no body (used by CheckAndSyncVersion)
POST /api/v1/audit/events                → ingest ActivityEvent batch from spool
GET  /api/v1/developers?username={u}     → resolve OS userId → developer_id (UUID)
POST /api/v1/agents/register             → upsert developer_agent row on first hook

# Dashboard-facing
POST /api/v1/policies                    → create policy (inserts new policy_versions row)
PUT  /api/v1/policies/:id/activate       → activate version (updates current_version_id, increments packVersion)
GET  /api/v1/policies/:id/history        → version history for rollback UI
GET  /api/v1/sessions                         → session list with filters (developer, machineId, agent, date range)
GET  /api/v1/sessions/:id/timeline            → ordered events in a session (for timeline view)
GET  /api/v1/sessions/:id/timeline/export     → download session timeline as CSV or Excel (?format=csv|xlsx)
GET  /api/v1/violations                       → policy_actions query (filter by dev/machineId/agent/policy/date)
GET  /api/v1/violations/export                → download filtered violations as CSV or Excel (?format=csv|xlsx + same filters)
GET  /api/v1/developers                       → developer inventory
GET  /api/v1/developers/:id/export            → download developer sessions + violations as CSV or Excel
GET  /api/v1/agents                           → AI coding agent inventory
GET  /api/v1/mcp-servers                      → MCP server inventory
GET  /api/v1/mcp-servers/:id                  → MCP server detail (sessions, tool call count, first/last seen)
GET  /api/v1/machines/:machine_id/export      → download sessions + violations for a machine as CSV or Excel
GET  /api/v1/audit                            → audit_events log (system mutations only)
```

### 6.2 Audit Event Ingest

```go
// POST /api/v1/audit/events — called by cx spool flusher
// Input: { "events": [ ActivityEvent, ... ] }

func ingestAuditEvents(events []ActivityEvent) error {
    for _, ev := range events {
        devID       := resolveOrCreateDeveloper(ev.UserID)
        agentProdID := resolveAgentProduct(ev.Agent.Type)    // from ai_coding_agents catalog
        devAgentID  := upsertDeveloperAgent(devID, agentProdID, ev.MachineID, ev.AgentID)
        sessionID   := upsertSession(ev.SessionID, devID, devAgentID)
        mcpServerID := resolveMCPServer(ev.MCP)              // nullable

        pvID := resolvePolicyVersionID(ev.Decision.PolicyID, ev.Decision.PolicyVersion)

        db.Insert("policy_actions", PolicyAction{
            PolicyID:        ev.Decision.PolicyID,
            PolicyVersionID: pvID,
            SessionID:       sessionID,
            DeveloperID:     devID,
            AgentID:         devAgentID,
            MCPServerID:     mcpServerID,
            ActionType:      ev.Decision.Action,       // ALLOW | ALERT | BLOCK
            ViolationType:   ev.Decision.MatchedOn,    // path | command | mcp_tool | query | prompt
            ResourceType:    ev.Tool.Category,         // filesystem | shell | mcp | web_search | prompt
            ResourceID:      ev.Decision.MatchedValue,
            Reason:          ev.Decision.PolicyName,
            MatchedRule:     ev.Decision.PolicyID,
            OccurredAt:      ev.Timestamp,
            Metadata:        map[string]any{"latencyMs": ev.Decision.LatencyMs, "packVersion": ev.Decision.PackVersion},
        })

        if ev.Tool.Category == "mcp" {
            upsertSessionMCPServer(sessionID, mcpServerID)
        }
    }
}
```

### 6.3 Policy Activation — packVersion Increment

When `PUT /api/v1/policies/:id/activate` is called:

```go
// 1. UPDATE policies SET current_version_id = :version_id
// 2. INCREMENT pack_version counter (stored in policies or a global sequence)
// 3. INSERT INTO audit_events: POLICY_VERSION_CREATED

// The new packVersion is returned in the response and also served
// via HEAD /api/v1/agent/policies/version as X-Pack-Version header.
// cx detects the change on the next hook call and pulls the full pack.

resp := ActivateResponse{
    PolicyID:      policyID,
    PolicyVersion: newVersionNumber,
    PackVersion:   newPackVersion,   // incremented global counter
}
```

No broadcast or push required. `cx` discovers the new version on the next hook call via the `HEAD` check.

---

## 7. Policy Sync Strategy

No daemon, no SSE, no persistent process. Policy freshness is driven entirely by version checks inside the short-lived hook subprocess.

```
Session-start hook (synchronous — every new session)
  └── HEAD /api/v1/agent/policies/version  (500ms timeout)
  └── if packVersion changed → GET /api/v1/agent/policies → atomic write policy-pack.json
  └── policy guaranteed fresh at start of every session

Every hook call — after verdict returned (async goroutine)
  └── HEAD /api/v1/agent/policies/version  (200ms timeout)
  └── if packVersion changed → GET /api/v1/agent/policies → atomic write policy-pack.json
  └── next hook call picks up new pack immediately
  └── policy lag = at most 1 tool call (seconds, not minutes)

Manual
  └── cx policy sync  (force refresh, full pack download)
```

**Why this fits the subprocess model:** `cx hooks <route>` is a short-lived OS process — it starts, returns a verdict, and exits. There is no persistent process between hook calls to hold a push connection. The `HEAD` check is a single HTTP round-trip (~5–50ms) that happens in a background goroutine after the verdict is already returned, so developer latency is zero.

---

## 8. Claude & Cursor Hook Routes

### 8.1 Claude

| Claude event | Route | Kind | Governance decision? |
|---|---|---|---|
| SessionStart | `claude-session-start` | `session.start` | No — identity resolve + soft sync |
| UserPromptSubmit | `claude-user-prompt-submit` | `prompt.submit` | **Yes** |
| PreToolUse (`.*`) | `claude-pre-tool-use` | `tool.pre` | **Yes** |
| PostToolUse | `claude-post-tool-use` | `tool.post` | No — activity only |
| Stop | `claude-session-end` | `session.end` | No |

### 8.2 Cursor

| Cursor event | Route | Kind | Governance decision? |
|---|---|---|---|
| Session start | `cursor-session-start` | `session.start` | No — identity resolve + soft sync |
| Prompt submit | `cursor-before-submit-prompt` | `prompt.submit` | **Yes** |
| Shell command | `cursor-before-shell` | `tool.pre` | **Yes** |
| MCP call | `cursor-before-mcp` | `tool.pre` | **Yes** |
| File read | `cursor-before-file-read` | `tool.pre` | **Yes** |
| Stop | `cursor-session-end` | `session.end` | No |

### 8.3 `~/.claude/settings.json` (governance only)

```json
{
  "hooks": [
    {
      "matcher": ".*",
      "hooks": [
        { "type": "command", "command": "cx hooks claude-pre-tool-use" },
        { "type": "command", "command": "cx hooks claude-user-prompt-submit" },
        { "type": "command", "command": "cx hooks claude-session-start" },
        { "type": "command", "command": "cx hooks claude-session-end" }
      ]
    }
  ]
}
```

### 8.4 `~/.cursor/hooks.json` (governance only)

```json
{
  "hooks": [
    { "event": "before-shell",         "command": "cx hooks cursor-before-shell" },
    { "event": "before-submit-prompt", "command": "cx hooks cursor-before-submit-prompt" },
    { "event": "before-mcp",           "command": "cx hooks cursor-before-mcp" },
    { "event": "session-start",        "command": "cx hooks cursor-session-start" },
    { "event": "session-end",          "command": "cx hooks cursor-session-end" }
  ]
}
```

---

## 9. Dashboard UI

**Stack:** Next.js + React + TanStack Query  
**Auth:** JWT (same token from `cx configure`)

### 9.1 Detail Panel Pattern

Every clickable row in the dashboard opens a **full-screen detail panel** that overlays the current page — no navigation to a new URL, no new browser tab. The existing dashboard is hidden behind the panel. A **✕ Close** button in the top-right corner dismisses the panel and restores the previous view exactly as it was.

```
┌─────────────────────────────────────────────────────────────┐
│  Dashboard (hidden behind panel when panel is open)          │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Detail Panel                              ✕ Close    │  │
│  │                                                       │  │
│  │  [detail content — session / developer /              │  │
│  │   machine / violation]                                │  │
│  │                                                       │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

Panels stack — clicking a linked item inside a panel opens a nested panel on top, each with its own ✕ Close. Closing the top panel returns to the panel below it; closing all panels returns to the dashboard.

The URL does **not** change when a panel opens. The panel state is held in React context only, so refreshing the page returns to the base dashboard view.

---

### 9.2 Pages

| Page | Key elements |
|---|---|
| **Overview** | Developer count, active sessions, violations today, MCP servers discovered (live stats) |
| **Policies** | Create / edit (type + action + matchTarget + regex pattern), version history, activate, rollback |
| **Sessions** | Filterable table (developer, machineId, agent type, date range); click row → Session Detail panel; Download button exports filtered list as CSV or Excel |
| **Violations** | Filter by developer / machineId / agent / policy / severity / date; click row → Violation Detail panel; Download button exports filtered violations as CSV or Excel |
| **MCP Servers** | Inventory of all discovered servers — server name, first seen, last seen, sessions used in, tool call count. No decisions here — block/warn policies are managed in the Policies page. |
| **Developers** | Inventory; last active; agents used; violation count; click row → Developer Detail panel |
| **Agents** | Claude Code / Cursor instances, versions, machines; click machine → Machine Detail panel |
| **Audit Log** | Append-only log of system mutations (policy changes, developer changes) |

---

### 9.3 Detail Panels

#### Session Detail Panel

Opened by clicking any session row on the Sessions page. Full-screen overlay — splits into two panels side by side.

**Layout:**

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│  Observe / Activity / fc82f041-b5ed-5a09-87fa-458c9bb6a929        Auto-refresh ○   │
├─────────────────────────────────────────────────────────────────────────────────────┤
│  Session ID                          Client                                         │
│  fc82f041-b5ed-5a09-87fa-458c9bb6a929   [● Claude Code]   ⬇ CSV  ⬇ Excel  ✕ Close │
│  Session Duration  10m                                                              │
├──────────────────────────────────────┬──────────────────────────────────────────────┤
│  LEFT PANEL (50%)                    │  RIGHT PANEL (50%)                           │
│                                      │                                              │
│  Session Overview        Clear filter│  ● Session Activity  14 actions  Clear filter│
│  ┌────────────────────────────────┐  │  ──────────────────────────────────────────  │
│  │         [macbookpro.lan]       │  │  Action                                      │
│  │              user              │  │                                              │
│  │  ┌─────────────────────────┐  │  │  ⚡ Stop  {"content":"```bash\n# App\n...    │
│  │  │   Developer Machine     │  │  │                                              │
│  │  │  [User Prompts]         │  │  │  ○ User Prompt                               │
│  │  │   3 prompts    [Files   │  │  │    yes                               ↗       │
│  │  │                 Read]   │  │  │                                              │
│  │  │  [Claude Code]  0 files │  │  │  ✎ edit_file {"file_path":"/Users/.../.env  │
│  │  │   agent        [Files   │  │  │  ┌──────────────────────────────────────┐   │
│  │  │                Written] │  │  │  │ {"type":"create","filePath":"/Users/ │   │
│  │  │  [Bash]         1 file  │  │  │  │  .../project/.env.example","content" │   │
│  │  │   echo                  │  │  │  └──────────────────────────────────────┘   │
│  │  └─────────────────────────┘  │  │                                              │
│  └────────────────────────────┘  │  │  ⚡ Stop  {"content":"Written to `.env.     │
│                                  │  │                                              │
│  Findings  [3 findings] Clear fil│  │  ○ User Prompt                               │
│  ──────────────────────────────  │  │    Echo this test record so I can confirm... │
│  Findings               Time     │  │                                              │
│  Potential PII detected in       │  │  ...                                         │
│  execute_shell co...   10h ago → │  │                                              │
└──────────────────────────────────┴──────────────────────────────────────────────────┘
```

---

**Left panel — Session Overview (React Flow node graph):**

Built with `@xyflow/react`. Nodes are derived from the session's `ActivityEvent` list — grouped by `tool.category`. Edges connect the agent node to each category that appeared in the session.

| Node | Icon | Populated from |
|---|---|---|
| `macbookpro.lan` / machine hostname | User avatar | `machineId` |
| `Developer Machine` | Container box | wraps all child nodes |
| `User Prompts` | Chat bubble | count of `kind = prompt.submit` events |
| `Files Read` | File icon | count of `category = filesystem` + `name = Read` |
| `Files Written` | Edit icon | count of `category = filesystem` + `name = Write/Edit` |
| `Claude Code` / agent | Agent icon | `agent.type` |
| `Bash` | Terminal `>_` | last `category = shell` command |
| `MCP` | Cube icon | appears if any `category = mcp` events exist |

Node color coding: neutral = no violations, yellow border = ALERT, red border = BLOCK occurred in that category.

```tsx
// Build nodes from ActivityEvent list
function buildSessionGraph(events: ActivityEvent[]): { nodes: Node[], edges: Edge[] } {
  const categories = groupBy(events, e => e.tool?.category)
  const nodes = [
    { id: 'user',    data: { label: machineHostname, sub: 'user' } },
    { id: 'agent',   data: { label: agentType, sub: 'agent' } },
    ...Object.entries(categories).map(([cat, evs]) => ({
      id: cat,
      data: {
        label: categoryLabel(cat),
        sub: `${evs.length} ${cat === 'filesystem' ? 'files' : 'calls'}`,
        hasBlock: evs.some(e => e.decision?.action === 'BLOCK'),
        hasAlert: evs.some(e => e.decision?.action === 'ALERT'),
      }
    }))
  ]
  const edges = Object.keys(categories).map(cat => ({
    id: `agent-${cat}`, source: 'agent', target: cat
  }))
  return { nodes, edges }
}
```

---

**Left panel — Findings section (bottom):**

Shows only BLOCK and ALERT events as findings. Each row has a navigate arrow (→) that opens the Violation Detail panel.

```
Findings  [N findings]
─────────────────────────────────────
Findings                        Time
Potential PII detected in ...   10h ago  →
Credential file accessed  ...   10h ago  →
```

Auto-refresh toggle (top-right header) polls `GET /api/v1/sessions/:id/timeline` every 5 seconds when the session is active (`status = active`).

---

**Right panel — Session Activity log:**

Scrollable list of every `ActivityEvent` in chronological order. Each row has:
- **Icon** by action type: `⚡` Stop/session, `○` User Prompt, `✎` file edit, `>_` shell, `⬡` MCP
- **Action name** (bold) — tool name or kind
- **Content preview** — truncated first line of the payload
- **↗ expand button** — opens the full JSON payload inline below the row
- BLOCK rows have a red left border, ALERT rows have a yellow left border

```tsx
const actionIcon: Record<string, string> = {
  'session.start':  '⚡',
  'session.end':    '⚡',
  'prompt.submit':  '○',
  'filesystem':     '✎',
  'shell':          '>_',
  'mcp':            '⬡',
  'web_search':     '🔍',
}

function ActivityRow({ event }: { event: ActivityEvent }) {
  const [expanded, setExpanded] = useState(false)
  const borderClass =
    event.decision?.action === 'BLOCK' ? 'border-l-2 border-red-500 bg-red-950/20' :
    event.decision?.action === 'ALERT' ? 'border-l-2 border-yellow-500 bg-yellow-950/10' : ''

  return (
    <div className={`px-4 py-3 ${borderClass} hover:bg-gray-800`}>
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-gray-400 shrink-0">
            {actionIcon[event.tool?.category ?? event.kind] ?? '○'}
          </span>
          <span className="font-mono text-sm font-semibold shrink-0">
            {event.tool?.name ?? event.kind}
          </span>
          <span className="text-gray-400 text-xs truncate">
            {event.tool?.command ?? event.tool?.paths?.[0] ?? event.tool?.query ?? ''}
          </span>
        </div>
        <button onClick={() => setExpanded(!expanded)} className="text-gray-500 shrink-0">↗</button>
      </div>
      {expanded && (
        <pre className="mt-2 text-xs bg-gray-900 rounded p-3 overflow-x-auto max-h-48">
          {JSON.stringify(event, null, 2)}
        </pre>
      )}
    </div>
  )
}
```

---

**Implementation libraries:**

```json
{
  "@xyflow/react": "^12.x",
  "@tanstack/react-query": "^5.x",
  "lucide-react": "latest",
  "xlsx": "^0.18.x"
}
```

**Download** — exports all events in the timeline for this session. Calls `GET /api/v1/sessions/:id/timeline/export?format=csv` or `?format=xlsx`.

---

#### Developer Detail Panel

Opened by clicking any developer row on the Developers page, or a developer link inside another panel. Full-screen overlay — splits into two panels side by side.

**Layout:**

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│  Observe / Developers / apoorvas                                                    │
├─────────────────────────────────────────────────────────────────────────────────────┤
│  apoorvas                    [platform]  [admin]    ⬇ CSV  ⬇ Excel  ✕ Close        │
│  apoorvas@checkmarx.com  ·  First seen: 2026-07-01  ·  Last active: 2026-08-12     │
├──────────────────────────────────────┬──────────────────────────────────────────────┤
│  LEFT PANEL (50%)                    │  RIGHT PANEL (50%)                           │
│                                      │                                              │
│  Developer Overview      Clear filter│  ● Recent Sessions  24 sessions  Clear filter│
│  ┌────────────────────────────────┐  │  ──────────────────────────────────────────  │
│  │       [apoorvas]               │  │  Session           Duration  Violations      │
│  │        developer               │  │                                              │
│  │           │                    │  │  2026-08-12 10:00  47m                      │
│  │  ┌────────┴────────┐           │  │  Claude Code          3 violations  ●  →    │
│  │  ▼                ▼            │  │                                              │
│  │ [win-abc123]  [mac-def456]     │  │  2026-08-12 08:14  23m                      │
│  │  Windows 11   macOS            │  │  Cursor               0 violations      →    │
│  │     │               │          │  │                                              │
│  │  [Claude]      [Cursor]        │  │  2026-08-11 14:22  1h 12m                   │
│  │   v1.2.3        v0.42.0        │  │  Claude Code          1 violation  ▲   →    │
│  └────────────────────────────────┘  │                                              │
│                                      │  ...                                         │
│  Violation Summary                   │                                              │
│  ──────────────────────────────────  │                                              │
│  Total   BLOCK   ALERT               │                                              │
│   12       8       4                 │                                              │
│                                      │                                              │
│  By category:                        │                                              │
│  filesystem  ████████  7             │                                              │
│  shell       ████      3             │                                              │
│  prompt      ██        2             │                                              │
└──────────────────────────────────────┴──────────────────────────────────────────────┘
```

**Left panel — Developer graph (React Flow):**

Nodes: developer → machines → agents on each machine. Each machine node is clickable → Machine Detail panel.

| Node | Data shown |
|---|---|
| Developer | Username, role badge |
| Machine | Machine ID, OS (Windows/Mac/Linux icon) |
| Agent | Agent type + version, last seen |

Node color: red border if machine has BLOCK violations, yellow if ALERT only, neutral if clean.

**Violation summary bar chart** below the graph — category breakdown (filesystem / shell / mcp / prompt) showing how many violations per category using a simple inline bar.

**Right panel — Recent sessions list:**

Each session row shows: date, time, duration, agent type, violation count with colored dot (● red = BLOCK, ▲ yellow = ALERT). Click row → Session Detail panel.

```tsx
function DeveloperSessionRow({ session }: { session: Session }) {
  return (
    <div className="px-4 py-3 hover:bg-gray-800 cursor-pointer border-b border-gray-800"
         onClick={() => openPanel('session', session.sessionId)}>
      <div className="flex justify-between items-start">
        <div>
          <div className="text-sm text-gray-300">{formatDate(session.startedAt)}</div>
          <div className="flex items-center gap-2 mt-1">
            <AgentBadge type={session.agentType} />
            <span className="text-xs text-gray-500">{session.duration}</span>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <ViolationBadge count={session.violationCount} action={session.worstAction} />
          <span className="text-gray-500">→</span>
        </div>
      </div>
    </div>
  )
}
```

**Download** — exports all sessions and violations for this developer. Calls `GET /api/v1/developers/:id/export?format=csv` or `?format=xlsx`.

---

#### Machine Detail Panel

Opened by clicking a machine ID anywhere in the dashboard. Full-screen overlay — splits into two panels side by side.

**Layout:**

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│  Observe / Machines / win-abc123                                                    │
├─────────────────────────────────────────────────────────────────────────────────────┤
│  win-abc123  [Windows 11]     Developer: apoorvas →    ⬇ CSV  ⬇ Excel  ✕ Close     │
│  governance_agent_id: a3f1...sha256hex  ·  First seen: 2026-07-01                  │
├──────────────────────────────────────┬──────────────────────────────────────────────┤
│  LEFT PANEL (50%)                    │  RIGHT PANEL (50%)                           │
│                                      │                                              │
│  Machine Overview        Clear filter│  ● Sessions from this machine  Clear filter  │
│  ┌────────────────────────────────┐  │  ──────────────────────────────────────────  │
│  │       [win-abc123]             │  │  Session           Duration  Violations      │
│  │        Windows 11              │  │                                              │
│  │        ┌────┴────┐             │  │  2026-08-12 10:00  47m                      │
│  │        ▼         ▼             │  │  Claude Code  3 violations  ●           →   │
│  │   [Claude Code]  [Cursor]      │  │                                              │
│  │    v1.2.3         v0.42.0      │  │  2026-08-12 08:14  23m                      │
│  │    First seen:    First seen:  │  │  Cursor       0 violations              →   │
│  │    2026-07-01     2026-07-15   │  │                                              │
│  └────────────────────────────────┘  │  2026-08-11 14:22  1h 12m                   │
│                                      │  Claude Code  1 violation   ▲           →   │
│  Machine Stats                       │                                              │
│  ──────────────────────────────────  │  ...                                         │
│  Total sessions:    38               │                                              │
│  Total tool calls:  412              │                                              │
│  BLOCK violations:   8               │                                              │
│  ALERT violations:   4               │                                              │
│  MCP servers seen:   3               │                                              │
│                                      │                                              │
│  MCP Servers used                    │                                              │
│  ──────────────────────────────────  │                                              │
│  github    142 calls                 │                                              │
│  stripe      8 calls                 │                                              │
└──────────────────────────────────────┴──────────────────────────────────────────────┘
```

**Left panel — Machine graph (React Flow):**

Nodes: machine → agents installed on it. Each agent node shows version + first seen date.

| Node | Data shown |
|---|---|
| Machine | Machine ID, OS badge |
| Agent | Agent type (Claude Code / Cursor), version, first seen |

Below the graph: machine stats summary (session count, tool calls, violation counts, MCP servers seen) and MCP servers list with call counts.

**Right panel — Sessions from this machine:**

Same session row design as Developer Detail panel. Each row clickable → Session Detail panel.

**Download** — exports all sessions and violations for this machine. Calls `GET /api/v1/machines/:machine_id/export?format=csv` or `?format=xlsx`.

---

#### Policy Detail Panel

Opened by clicking any policy row on the Policies page. Full-screen overlay — splits into two panels side by side.

**Layout:**

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│  Observe / Policies / Block credential files                                        │
├─────────────────────────────────────────────────────────────────────────────────────┤
│  Block credential files   [BLOCK]  [filesystem]   v3 (active)    ✕ Close           │
├──────────────────────────────────────┬──────────────────────────────────────────────┤
│  LEFT PANEL (50%)                    │  RIGHT PANEL (50%)                           │
│                                      │                                              │
│  Policy Configuration                │  ● Recent Violations  47 total  Clear filter │
│  ──────────────────────────────────  │  ──────────────────────────────────────────  │
│  Type:         filesystem            │  Developer    Machine     Time      Action   │
│  Action:       BLOCK                 │                                              │
│  Match target: path                  │  apoorvas     win-abc123  10m ago   BLOCK ● │
│  Pattern:      \.env$|\.pem$|\.key$  │    C:/proj/.env                         →   │
│                                      │                                              │
│  Message to developer:               │  jsmith       mac-def456  2h ago    BLOCK ● │
│  ┌────────────────────────────────┐  │    /home/proj/.env                      →   │
│  │ Production credential files    │  │                                              │
│  │ are restricted. Use the        │  │  apoorvas     win-abc123  1d ago    ALERT ▲ │
│  │ secrets manager at vault...    │  │    C:/proj/.pem                         →   │
│  └────────────────────────────────┘  │                                              │
│                                      │  ...                                         │
│  Version History                     │                                              │
│  ──────────────────────────────────  │                                              │
│  v3  2026-08-10  (active)      ←     │                                              │
│      "Escalated to BLOCK"            │                                              │
│  v2  2026-08-01                      │                                              │
│      "Added .key extension"          │                                              │
│  v1  2026-07-15  (initial)           │                                              │
│                                      │                                              │
│  [Edit Policy]  [Rollback to v2]     │                                              │
└──────────────────────────────────────┴──────────────────────────────────────────────┘
```

**Left panel — Policy configuration:**

- Policy name, type badge, action badge (`BLOCK` red / `ALERT` yellow), active version badge
- Pattern field with monospace display
- `agentMessage` in a text box (read-only view; edit opens inline form)
- Version history list — each version shows version number, date, `change_summary` from `policy_versions`. Active version marked with `←`. Click any version to preview its definition.
- Action buttons: `Edit Policy` (opens inline form replacing left panel), `Rollback to vN` (activates selected version)

**Right panel — Recent violations triggered by this policy:**

Each row shows: developer name, machine ID, time ago, action (`BLOCK ●` / `ALERT ▲`), matched value (file path / command / mcp tool). Click row → Violation Detail panel.

```tsx
function PolicyViolationRow({ violation }: { violation: PolicyAction }) {
  return (
    <div className="px-4 py-3 hover:bg-gray-800 cursor-pointer border-b border-gray-800"
         onClick={() => openPanel('violation', violation.policyActionId)}>
      <div className="flex justify-between items-start">
        <div>
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">{violation.developerEmail}</span>
            <span className="text-xs text-gray-500">{violation.machineId}</span>
          </div>
          <div className="text-xs text-gray-400 font-mono mt-1 truncate max-w-xs">
            {violation.matchedValue}
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <span className="text-xs text-gray-500">{timeAgo(violation.occurredAt)}</span>
          <ActionBadge action={violation.actionType} />
          <span className="text-gray-500">→</span>
        </div>
      </div>
    </div>
  )
}
```

---

#### Violation Detail Panel

Opened by clicking any violation row on the Violations page, a BLOCK/ALERT event row inside a Session Detail panel, or a violation row inside a Policy Detail panel. Full-screen overlay — splits into two panels side by side.

**Layout:**

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│  Observe / Violations / evt_abc123                                                  │
├─────────────────────────────────────────────────────────────────────────────────────┤
│  [● BLOCK]   2026-08-12  10:12:47                                      ✕ Close      │
├──────────────────────────────────────┬──────────────────────────────────────────────┤
│  LEFT PANEL (50%)                    │  RIGHT PANEL (50%)                           │
│                                      │                                              │
│  Context                             │  Full Event Payload                          │
│  ──────────────────────────────────  │  ──────────────────────────────────────────  │
│  Developer:  apoorvas           →    │  {                                           │
│  Machine:    win-abc123         →    │    "eventId": "evt_abc123",                  │
│  Agent:      Claude Code 1.2.3       │    "timestamp": "2026-08-12T10:12:47Z",      │
│  Session:    fc82f041...        →    │    "kind": "tool.pre",                       │
│                                      │    "tool": {                                 │
│  What was matched                    │      "category": "filesystem",               │
│  ──────────────────────────────────  │      "name": "Read",                         │
│  Category:     filesystem            │      "paths": ["C:/proj/.env"]               │
│  Tool:         Read                  │    },                                        │
│  Matched on:   path                  │    "decision": {                             │
│  Matched value:                      │      "action": "BLOCK",                      │
│  C:/proj/.env                        │      "policyId": "pol_abc",                  │
│                                      │      "policyName": "Block credential files", │
│  Policy that fired                   │      "policyVersion": 3,                     │
│  ──────────────────────────────────  │      "packVersion": 13,                      │
│  Name:     Block credential files →  │      "matchedOn": "path",                    │
│  Version:  v3                        │      "matchedValue": "C:/proj/.env"          │
│  Pattern:  \.env$|\.pem$|\.key$      │    }                                         │
│                                      │  }                                           │
│  Message shown to developer:         │                                              │
│  ┌────────────────────────────────┐  │                                              │
│  │ Production credential files    │  │                                              │
│  │ are restricted. Use the        │  │                                              │
│  │ secrets manager at vault...    │  │                                              │
│  └────────────────────────────────┘  │                                              │
└──────────────────────────────────────┴──────────────────────────────────────────────┘
```

**Left panel — Violation context:**

- Action badge (`BLOCK` red / `ALERT` yellow) + timestamp
- Context links: developer → Developer Detail panel, machine → Machine Detail panel, session → Session Detail panel
- What was matched: category, tool name, matchedOn, matchedValue (monospace)
- Policy that fired: policy name (clickable → Policy Detail panel), version, pattern
- `agentMessage` displayed in a read-only text box — exactly what the developer saw in their agent

**Right panel — Full event payload:**

Raw `ActivityEvent` JSON with syntax highlighting. Copyable. This is the complete record as received by the backend from the audit spool — developers and security teams can use this for forensic review.

```tsx
function ViolationPayload({ event }: { event: ActivityEvent }) {
  return (
    <div className="h-full flex flex-col">
      <div className="flex items-center justify-between px-4 py-2 border-b border-gray-700">
        <span className="text-sm font-medium">Full Event Payload</span>
        <button onClick={() => navigator.clipboard.writeText(JSON.stringify(event, null, 2))}
                className="text-xs text-gray-400 hover:text-white">
          Copy
        </button>
      </div>
      <pre className="flex-1 overflow-auto p-4 text-xs font-mono text-green-300 bg-gray-950">
        {JSON.stringify(event, null, 2)}
      </pre>
    </div>
  )
}

---

### 9.4 Policy Editor — Form → Pack Mapping

| UI field | `PackPolicy` field | Example values |
|---|---|---|
| Policy name | `policyName` | "Block credential files" |
| Type | `type` | `filesystem` / `shell` / `mcp` / `web_search` / `prompt` |
| Action | `action` | `BLOCK` / `ALERT` |
| Match target | `matchTarget` | `path` / `command` / `mcp_tool` / `query` / `prompt` |
| Pattern (regex) | `pattern` | `\.env$` / `rm\s+-rf` / `^github/.*` |
| Message to show in agent | `agentMessage` | Optional. Shown to developer inside Claude / Cursor when policy fires. If empty, `policyName` is shown instead. |

On save → `POST /api/v1/policies` → new `policy_versions` row → `PUT /:id/activate` → `packVersion` incremented → `cx` detects change on next hook call via `HEAD` check → atomically updates `policy-pack.json`.

---

### 9.5 Export / Download Specification

All export endpoints accept `?format=csv` or `?format=xlsx`. The backend streams the file directly — no intermediate storage. The browser triggers a file download.

**Session Timeline export** (`GET /api/v1/sessions/:id/timeline/export`)

| Column | Source |
|---|---|
| Timestamp | `ActivityEvent.timestamp` |
| Kind | `ActivityEvent.kind` (session.start / tool.pre / prompt.submit / session.end) |
| Tool Name | `ActivityEvent.tool.name` |
| Category | `ActivityEvent.tool.category` |
| Command / Path / Query | `tool.command` / `tool.paths` / `tool.query` |
| MCP Server | `ActivityEvent.mcp.server` |
| MCP Tool | `ActivityEvent.mcp.name` |
| Decision | `decision.action` (ALLOW / ALERT / BLOCK) |
| Policy Name | `decision.policyName` |
| Policy Version | `decision.policyVersion` |
| Matched On | `decision.matchedOn` |
| Matched Value | `decision.matchedValue` |
| Agent Message | `decision.agentMessage` |

**Violations export** (`GET /api/v1/violations/export`)

Respects all active filters (developer, machineId, agent, policy, date range) applied on the Violations page.

| Column | Source |
|---|---|
| Timestamp | `policy_actions.occurred_at` |
| Developer | `developers.email` |
| Machine ID | `developer_agents.machine_id` |
| Agent Type | `ai_coding_agents.agent_name` |
| Agent Version | `ai_coding_agents.version` |
| Session ID | `policy_actions.session_id` |
| Action | `policy_actions.action_type` |
| Policy Name | `policies.policy_name` |
| Policy Version | `policy_versions.version_number` |
| Resource Type | `policy_actions.resource_type` |
| Matched On | `policy_actions.violation_type` |
| Matched Value | `policy_actions.resource_id` |
| Agent Message | from `policy_versions.definition` |

**Developer export** (`GET /api/v1/developers/:id/export`) — one sheet/section per developer containing their sessions list and violations list with the same columns as above.

**Machine export** (`GET /api/v1/machines/:machine_id/export`) — sessions and violations scoped to that machine, same columns as above.

---

## 10. Implementation Phases

### Phase 1 — Policy Backend (Week 1–2)
- [ ] Go API server + PostgreSQL schema (section 5 — all tables: policies, policy_versions, developers, ai_coding_agents, developer_agents, ai_models, ai_sessions, mcp_servers, session_mcp_servers, policy_actions, audit_events)
- [ ] `POST /api/v1/policies`, `PUT /:id/activate` (increments packVersion), `GET /api/v1/agent/policies`
- [ ] `HEAD /api/v1/agent/policies/version` — returns `X-Pack-Version` header, no body
- [ ] `POST /api/v1/audit/events` ingest with full identity resolution
- [ ] `GET /api/v1/developers?username=` resolver
- [ ] `POST /api/v1/agents/register`
- [ ] JWT auth middleware

### Phase 2 — Governance Hooks (Week 2–3)
- [ ] New package `governance/` in ast-cli
- [ ] `identity.go` — machineId, agentId, EnsureResolved (fills G4, G5, G6)
- [ ] `normalize.go` — RuntimeEvent + tool category normalizer (fills G2, G9)
- [ ] `pack.go` — GovernancePack loader + Evaluate() (fills G1)
- [ ] `spool.go` — atomic write + StartFlusher (fills G3)
- [ ] `log.go` — local session log writer, PII-free NDJSON, 30-day auto-purge
- [ ] `hooks.go` — governance session/tool/prompt handlers (fills G1)
- [ ] `install.go` — writes governance routes into Claude + Cursor configs (fills G10)
- [ ] `sync.go` — `SyncOnSessionStart`, `CheckAndSyncVersion`, `SyncOnce` (fills G7, G8, G16)
- [ ] `cmd.go` — `cx hooks governance install` + `cx policy sync` Cobra commands

### Phase 3 — Audit & Session Tracking (Week 3–4)
- [ ] Session upsert on session-start/end hooks (fills G12)
- [ ] ActivityEvent full spool → batch flush → `/api/v1/audit/events`
- [ ] MCP server upsert on first `tool.category = "mcp"` event (fills G11)
- [ ] `session_mcp_servers` bridge population
- [ ] Dead-letter queue (`audit-queue/dead/`) for non-retryable 4xx

### Phase 4 — MCP Governance (Week 4)
- [ ] Upsert `mcp_servers` row on first-seen MCP tool call (server name, first/last seen, call counts)
- [ ] `session_mcp_servers` bridge population per session
- [ ] MCP inventory page in dashboard (discovery only)
- [ ] Admin creates BLOCK/ALERT policies for MCP servers via Policy page (same flow as any other policy)

### Phase 5 — Dashboard UI (Week 5–6)
- [ ] Next.js project + JWT auth
- [ ] Overview (live stats)
- [ ] Policy CRUD + version history + activate/rollback
- [ ] Sessions + timeline (paths / command / mcp / query shown on BLOCK/ALERT)
- [ ] Violations table (filterable + exportable)
- [ ] MCP server inventory (discovery only — server name, sessions, call counts, first/last seen)
- [ ] Developer + agent inventory
- [ ] Audit log view

---

## 11. Critical Design Principles

**Fail-open everywhere:**
- Missing `policy-pack.json` → all decisions: `ALLOW`
- Malformed pack JSON → `ALLOW`
- Server unreachable during sync → retain existing pack, continue enforcing
- Spool write fails → verdict already returned; log error and continue
- Backend ingest 5xx → retain spool file, retry next flush cycle
- Identity resolution fails → omit `developer_id` from ActivityEvent (nullable FK)

**Default mode is ALERT not BLOCK:**
- All policies created in the dashboard default to `ALERT`
- Admin explicitly sets `BLOCK` — never forced on developers by default
- `ALERT` passes the tool call through but spools the event for the audit log

**Immutable policy versions:**
- Every policy change = new `policy_versions` row + new `packVersion` increment
- `policy_actions.policy_version_id` is a permanent forensic link to the exact rule that fired
- Rollback = activate an older version (never a delete)

**Local session logs contain no personal information:**
- Never written: `userId`, `machineId`, `developer_id`, `agentId`, `email`
- Never written: prompt content or web search query content
- `matchedValue` written only for `path` / `command` / `mcp_tool` — not for `prompt` or `query`
- Log files are local only — never uploaded to backend
- Auto-purged after 30 days on session start

**Windows compatibility:**
- No env var injection in hook commands (`VAR=value cx hooks ...` not supported on Windows)
- Server URL and token always read from `~/.checkmarx/checkmarxcli.yaml` at hook runtime

---

## 12. Gap Resolution Summary

| Gap | Resolution | Phase |
|---|---|---|
| G1 — no governance hook handlers | `governance/hooks.go` — ALLOW / ALERT / BLOCK | Phase 2 |
| G2 — no RuntimeEvent / ActivityEvent | `governance/normalize.go` | Phase 2 |
| G3 — no audit spool | `governance/spool.go` — atomic write + background flush | Phase 2 |
| G4 — no machineId | `governance/identity.go` — OS + yaml fallback | Phase 2 |
| G5 — no agentId | `governance/identity.go` — SHA256(userId + agent.type + machineId) | Phase 2 |
| G6 — no developer_id resolution | `governance/identity.go` — GET /api/v1/developers?username= + cache | Phase 2 |
| G7 — no policy version check | `governance/sync.go` — `CheckAndSyncVersion()` after each hook + `SyncOnSessionStart()` | Phase 2 |
| G8 — no cx policy sync | `governance/sync.go` + Cobra command | Phase 2 |
| G9 — no tool normalizer | `governance/normalize.go` — category + match field extraction | Phase 2 |
| G10 — no governance install | `governance/install.go` + `cx hooks governance install` | Phase 2 |
| G11 — no MCP inventory | ActivityEvent MCP fields + upsert on ingest | Phase 3–4 |
| G12 — no session lifecycle | session-start/end hooks → DB upsert | Phase 3 |
| G13 — no Policy API Server | Phase 1 deliverable | Phase 1 |
| G14 — no Dashboard UI | Phase 5 deliverable | Phase 5 |
| G15 — Windows hook env | Config read from checkmarxcli.yaml, not env vars | Phase 2 |
| G16 — no stale check on session start | SyncIfStale() in session-start handler | Phase 2 |
