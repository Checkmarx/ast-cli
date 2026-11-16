-- Migration 005: Policy lifecycle (disable/enable/delete) and third-party monitor policy type.

-- Policy status: active policies are enforced; disabled ones are stored but excluded from the pack.
-- deleted_at enables soft-delete — rows are never hard-deleted so historical violations retain context.
ALTER TABLE policies
  ADD COLUMN IF NOT EXISTS deleted_at   TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS policy_type  TEXT NOT NULL DEFAULT 'governance'
    CHECK (policy_type IN ('governance', 'pii', 'third_party_monitor'));

-- Alter the existing status column to support 'disabled'.
-- policies.status already exists as an enum or text; add 'disabled' if not present.
-- Using DO block to handle both enum and text column safely.
DO $$
BEGIN
  -- If status is an enum type, add the new value
  IF EXISTS (
    SELECT 1 FROM pg_type t
    JOIN pg_enum e ON t.oid = e.enumtypid
    WHERE e.enumlabel = 'active'
      AND t.typname IN (
        SELECT udt_name FROM information_schema.columns
        WHERE table_name = 'policies' AND column_name = 'status'
      )
  ) THEN
    -- It's an enum: add 'disabled' if it doesn't exist
    BEGIN
      ALTER TYPE policy_status ADD VALUE IF NOT EXISTS 'disabled';
    EXCEPTION WHEN duplicate_object THEN NULL;
    END;
  END IF;
END;
$$;

-- Enum validation handles allowed values; no CHECK constraint needed.

-- Index for common queries: active non-deleted policies (pack build), disabled list.
CREATE INDEX IF NOT EXISTS idx_policies_status_deleted ON policies(status, deleted_at);
CREATE INDEX IF NOT EXISTS idx_policies_type ON policies(policy_type);
