-- Fourth detection method: pattern-matching a failed tool call's own error
-- text against known security-product wording (Windows Defender, CrowdStrike,
-- corporate proxy block pages, etc).
--
-- This covers a real gap: the transcript only sees blocks that go through
-- Claude Code's hook system. Some blockers act below that — antivirus killing
-- a spawned process, a firewall dropping a connection — and never register a
-- hook at all. Those calls still "complete" (with a failure), so PostToolUse
-- fires normally and its response carries the evidence. Weaker than both
-- other sources: it is text matching, not a structured verdict or a direct OS
-- observation, which is why the app only assigns 'candidate' confidence to
-- generic phrase matches and reserves 'definitive' for a named product.

ALTER TABLE activity_events
  DROP CONSTRAINT IF EXISTS activity_events_detection_method_check;

ALTER TABLE activity_events
  ADD CONSTRAINT activity_events_detection_method_check
  CHECK (detection_method IN ('telemetry', 'hook_correlation', 'transcript', 'response_signature'));
