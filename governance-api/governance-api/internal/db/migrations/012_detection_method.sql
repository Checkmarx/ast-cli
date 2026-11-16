-- Third-party enforcement provenance: how a block/warn verdict was established.
--
--   'telemetry'        — OS-level ground truth (process table / file-state observation)
--   'hook_correlation' — marker-timing heuristic, or a cooperating hook's own verdict
--
-- Rows written before this column existed predate telemetry detection entirely,
-- so a NULL reads as "hook_correlation" for display purposes.
ALTER TABLE activity_events
  ADD COLUMN IF NOT EXISTS detection_method TEXT
    CHECK (detection_method IN ('telemetry', 'hook_correlation'));

-- Third-party events are a small slice of the table; index only those rows so
-- provenance filtering on the violations view stays cheap.
CREATE INDEX IF NOT EXISTS idx_activity_events_detection_method
  ON activity_events (detection_method)
  WHERE detection_method IS NOT NULL;
