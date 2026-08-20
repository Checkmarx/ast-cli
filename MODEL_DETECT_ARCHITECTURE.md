# Claude Code Model Detection Architecture — POC Design

**Role:** Senior Architect, Checkmarx  
**Reference:** Model tracking analysis · AI governance compliance  
**Scope:** POC only — Claude Code model detection at session start. Focus: detect which Claude model version is in use, persist it, and make it queryable for cost/compliance analysis.

---

## 1. Executive Summary

**The one thing this POC proves:** the model a developer is using can be detected at session start and persisted so compliance/cost analysis can group activities by model.

**Reused (no changes):**
- `ai_sessions` table with `model_id` FK
- `activity_events` — hook event stream
- Hook framework — session lifecycle events

**Built new for POC:**
- `model_detect.go` — 4-layer fallback detection (Claude Code only)
- Model detection fields in `activity_events`
- `/api/v1/models` endpoints for querying
- Model usage dashboard

---

## 2. Detection Strategy

**Four layers, priority order (first success wins):**

| Layer | Source | Reliability | Speed |
|---|---|---|---|
| 1 | `~/.claude/settings.json` | 95% | ~10ms |
| 2 | `CLAUDE_MODEL` env var | 80% | instant |
| 3 | Historical (prior session) | 60% | ~50ms |
| 4 | Default fallback | 0% (flags monitoring) | instant |

**Code structure:**
```go
func DetectModel(ctx DetectionContext) ModelDetectionResult {
  // Layer 1: Settings file
  if model, path, err := detectFromSettingsFile(); err == nil {
    return success(model, "settings_file", 0.95, path)
  }
  // Layer 2: Env var
  if model := detectFromEnv(); model != "" {
    return success(model, "env_var", 0.80, "")
  }
  // Layer 3: History
  if ctx.DBClient != nil {
    if model, err := inferFromHistory(ctx); err == nil {
      return success(model, "history", 0.60, "")
    }
  }
  // Layer 4: Default
  return success("claude-opus-5", "default", 0.0, "")
}
```

---

## 3. Data Model

**New field in `ActivityEvent`:**
```go
type ActivityEvent struct {
  // ...existing fields...
  ModelInfo *ModelDetectionInfo `json:"modelInfo,omitempty"`
}

type ModelDetectionInfo struct {
  Model      string    `json:"model"`
  Vendor     string    `json:"vendor"`
  Confidence float64   `json:"confidence"`
  Source     string    `json:"source"`
  SourcePath string    `json:"sourcePath,omitempty"`
  DetectedAt time.Time `json:"detectedAt"`
  Errors     []string  `json:"errors,omitempty"`
}
```

**New columns in `activity_events` table:**
```sql
ALTER TABLE activity_events ADD COLUMN IF NOT EXISTS (
  model_detected      VARCHAR(255),
  model_vendor        VARCHAR(100),
  model_confidence    DECIMAL(3,2),
  model_source        VARCHAR(50),
  model_source_path   TEXT,
  model_detected_at   TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_activity_events_model_detected ON activity_events(model_detected);
```

**Seed `ai_models` table (Claude variants only):**
```sql
INSERT INTO ai_models (model_id, vendor, family, version, type, context_length, cost_per_1m_input, cost_per_1m_output)
VALUES
  ('claude-opus-5', 'anthropic', 'claude', '5', 'frontier', 200000, 15.00, 75.00),
  ('claude-sonnet-5', 'anthropic', 'claude', '5', 'mid-tier', 200000, 3.00, 15.00),
  ('claude-haiku-4', 'anthropic', 'claude', '4', 'lightweight', 200000, 0.80, 4.00)
ON CONFLICT (model_id) DO NOTHING;
```

---

## 4. Hook Integration

**File:** `ast-cli/internal/commands/agenthooks/governance/model_detect.go` (NEW)

**File:** `ast-cli/internal/commands/agenthooks/governance/hooks.go` (MODIFIED)

In `handleClaudeSessionStart()`, add model detection:
```go
modelResult := DetectModel(DetectionContext{
  UserID:   CurrentUserName(),
  AgentID:  AgentID(cfg.AgentType),
  DBClient: nil,
})

event := buildSessionEvent(ev.SessionID, "session.start", cfg)
event.ModelInfo = modelResult  // ← Enrich event

Write(event.WithDecision(Decision{Action: ""}))
```

---

## 5. Ingest Handler

**File:** `governance-api/internal/api/ingest.go` (MODIFIED)

When processing `ActivityEvent`, extract model info:
```go
if ae.ModelInfo != nil && ae.Kind == "session.start" {
  // Update activity_events.model_* columns
  s.db.Exec(ctx, `
    UPDATE activity_events
    SET model_detected = $1, model_vendor = $2, model_confidence = $3,
        model_source = $4, model_source_path = $5, model_detected_at = $6
    WHERE event_id = $7`,
    ae.ModelInfo.Model, ae.ModelInfo.Vendor, ae.ModelInfo.Confidence,
    ae.ModelInfo.Source, ae.ModelInfo.SourcePath, ae.ModelInfo.DetectedAt, ae.EventID)

  // Update ai_sessions.model_id if confidence >= 0.80
  if ae.ModelInfo.Confidence >= 0.80 {
    modelID, _ := s.db.LookupModelID(ctx, ae.ModelInfo.Model)
    if modelID != nil {
      s.db.Exec(ctx, `UPDATE ai_sessions SET model_id = $1 WHERE claude_session_id = $2`,
        modelID, ae.SessionID)
    }
  }
}
```

---

## 6. API Endpoints

**File:** `governance-api/internal/api/models.go` (NEW)

```
GET /api/v1/models
  → List Claude models (metadata, pricing)

GET /api/v1/models/{modelId}
  → Get single model details

GET /api/v1/sessions?model=claude-opus-5
  → Filter sessions by detected model

GET /api/v1/analytics/model-usage
  → Usage stats: session count, avg duration, estimated cost, confidence

GET /api/v1/analytics/model-detection-issues
  → Sessions with low confidence or detection failures
```

---

## 7. Database Migration

**File:** `governance-api/internal/db/migrations/011_model_detection.sql`

```sql
ALTER TABLE activity_events ADD COLUMN IF NOT EXISTS (
  model_detected      VARCHAR(255),
  model_vendor        VARCHAR(100),
  model_confidence    DECIMAL(3,2),
  model_source        VARCHAR(50),
  model_source_path   TEXT,
  model_detected_at   TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_activity_events_model_detected 
ON activity_events(model_detected);

INSERT INTO ai_models (model_id, vendor, family, version, type, context_length, cost_per_1m_input, cost_per_1m_output)
VALUES
  ('claude-opus-5', 'anthropic', 'claude', '5', 'frontier', 200000, 15.00, 75.00),
  ('claude-sonnet-5', 'anthropic', 'claude', '5', 'mid-tier', 200000, 3.00, 15.00),
  ('claude-haiku-4', 'anthropic', 'claude', '4', 'lightweight', 200000, 0.80, 4.00)
ON CONFLICT (model_id) DO NOTHING;
```

---

## 8. Implementation Checklist

- [ ] Create `model_detect.go` with 4-layer detection
- [ ] Modify `handleClaudeSessionStart()` to call `DetectModel()`
- [ ] Update `ActivityEvent` model with `ModelInfo` field
- [ ] Create migration 011 with schema changes
- [ ] Modify ingest handler to parse and store model info
- [ ] Create `/api/v1/models*` endpoints
- [ ] Seed `ai_models` table with Claude variants
- [ ] Add model usage dashboard page
- [ ] Add alerting for low-confidence detections

---

## 9. Success Criteria

| Metric | Target |
|---|---|
| Detection success rate (confidence ≥ 0.80) | >85% |
| Default fallback rate | <5% |
| Average confidence score | >0.88 |
| Sessions with model_id populated | >85% |

---

## 10. Key Constraints

1. Detection is best-effort; never blocks session or governance decisions
2. One detection per session at start time
3. Fallback to `claude-opus-5` (not unknown/null)
4. Claude Code only; multi-agent support deferred
5. Confidence scores drive alerting and monitoring, not UI hints
6. All detection errors logged for debugging

---

**Document Version:** 1.0  
**Status:** Ready for Implementation
