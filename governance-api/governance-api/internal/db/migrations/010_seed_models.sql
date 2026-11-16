-- Migration 010: Seed AI Models Catalog
-- Note: Model seeding is now handled by migration 011_model_detection.sql
-- This migration is kept for backward compatibility but is intentionally empty

-- ==========================================
-- VERIFICATION QUERIES (commented out for migration compatibility)
-- ==========================================

-- -- Show summary by provider
-- SELECT
--   provider,
--   COUNT(*) as model_count,
--   COUNT(*) FILTER (WHERE status = 'active') as active_count,
--   COUNT(*) FILTER (WHERE status = 'deprecated') as deprecated_count
-- FROM ai_models
-- GROUP BY provider
-- ORDER BY model_count DESC;

-- -- Show all active models
-- SELECT
--   provider,
--   model_name,
--   model_version,
--   status,
--   (metadata->'pricing'->>'input_price_per_mtok')::decimal as input_price_mtok,
--   (metadata->'pricing'->>'output_price_per_mtok')::decimal as output_price_mtok
-- FROM ai_models
-- WHERE status = 'active'
-- ORDER BY provider, model_name;

-- -- Show deprecated models with sunset dates
-- SELECT
--   provider,
--   model_name,
--   metadata->>'sunset_date' as sunset_date,
--   metadata->>'replacement_model_id' as replacement
-- FROM ai_models
-- WHERE status = 'deprecated'
-- ORDER BY metadata->>'sunset_date';
