-- Third-party decisions now carry the deciding tool's own words.
--
-- The reason and the hook event are read from the agent's session transcript,
-- which records every hook invocation with its command and its verdict. That
-- makes attribution a quotation rather than an inference, so 'transcript' joins
-- the detection_method vocabulary as the strongest provenance we have.

ALTER TABLE activity_events
  ADD COLUMN IF NOT EXISTS third_party_reason     TEXT,
  ADD COLUMN IF NOT EXISTS third_party_hook_event TEXT;

-- Widen detection_method to admit 'transcript'. The constraint is recreated
-- rather than altered because Postgres has no ALTER CONSTRAINT for CHECK.
ALTER TABLE activity_events
  DROP CONSTRAINT IF EXISTS activity_events_detection_method_check;

ALTER TABLE activity_events
  ADD CONSTRAINT activity_events_detection_method_check
  CHECK (detection_method IN ('telemetry', 'hook_correlation', 'transcript'));

-- third_party_action was block/warn only; it now also carries allow, ask and
-- error, so a third-party hook that approved with a note, escalated to the
-- user, or failed outright is all visible. No constraint existed on the column,
-- so nothing needs widening — this comment records the vocabulary change.
COMMENT ON COLUMN activity_events.third_party_action IS
  'What the third-party tool decided: block | warn | allow | ask | error';
COMMENT ON COLUMN activity_events.third_party_reason IS
  'Verbatim reason text the third-party tool returned, read from the agent transcript';
COMMENT ON COLUMN activity_events.third_party_hook_event IS
  'The hook that produced the decision, e.g. "PreToolUse:Bash"';
