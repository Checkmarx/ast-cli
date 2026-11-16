-- Add model detection fields to activity_events table
ALTER TABLE activity_events ADD COLUMN IF NOT EXISTS model_detected VARCHAR(255);
ALTER TABLE activity_events ADD COLUMN IF NOT EXISTS model_vendor VARCHAR(100);
ALTER TABLE activity_events ADD COLUMN IF NOT EXISTS model_confidence DECIMAL(3,2);
ALTER TABLE activity_events ADD COLUMN IF NOT EXISTS model_source VARCHAR(50);
ALTER TABLE activity_events ADD COLUMN IF NOT EXISTS model_source_path TEXT;
ALTER TABLE activity_events ADD COLUMN IF NOT EXISTS model_detected_at TIMESTAMP WITH TIME ZONE;

-- Index for fast filtering by detected model
CREATE INDEX IF NOT EXISTS idx_activity_events_model_detected ON activity_events(model_detected);

-- Seed ai_models with Claude variants
INSERT INTO ai_models (model_id, provider, model_name, model_version, model_family, status, metadata)
VALUES
  (gen_random_uuid(), 'anthropic', 'claude-opus-5', '5', 'claude', 'active',
   jsonb_build_object('type', 'frontier', 'context_length', 200000, 'cost_per_1m_input', 15.00, 'cost_per_1m_output', 75.00)),
  (gen_random_uuid(), 'anthropic', 'claude-sonnet-5', '5', 'claude', 'active',
   jsonb_build_object('type', 'mid-tier', 'context_length', 200000, 'cost_per_1m_input', 3.00, 'cost_per_1m_output', 15.00)),
  (gen_random_uuid(), 'anthropic', 'claude-haiku-4', '4', 'claude', 'active',
   jsonb_build_object('type', 'lightweight', 'context_length', 200000, 'cost_per_1m_input', 0.80, 'cost_per_1m_output', 4.00))
ON CONFLICT (provider, model_name, model_version) DO NOTHING;
