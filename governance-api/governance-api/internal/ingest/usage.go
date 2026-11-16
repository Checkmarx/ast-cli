package ingest

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/checkmarx/governance-api/internal/db"
	"github.com/checkmarx/governance-api/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	maxUsageRecordsPerEvent = 1000
	maxTokensPerField       = int64(50_000_000_000) // sanity bound; a single request is far below this
)

var validUsageSources = map[string]bool{"transcript": true, "otel": true, "estimate": true}

// processUsage persists a usage.delta event into the token_usage_events ledger.
//
// It deliberately does not go through upsertSession: that path revives ended/expired
// sessions, and spool files flush in arbitrary order, so a late usage.delta must not
// flip a session that has already ended back to active.
func processUsage(ctx context.Context, ev models.ActivityEvent, devID, agentProdID, devAgentID uuid.UUID) error {
	if ev.Usage == nil || len(ev.Usage.Records) == 0 {
		return nil
	}
	source := ev.Usage.Source
	if !validUsageSources[source] {
		return fmt.Errorf("usage: unknown source %q", source)
	}
	if len(ev.Usage.Records) > maxUsageRecordsPerEvent {
		return fmt.Errorf("usage: %d records exceeds limit %d", len(ev.Usage.Records), maxUsageRecordsPerEvent)
	}

	var sessionID uuid.UUID
	err := db.Pool.QueryRow(ctx,
		`SELECT session_id FROM ai_sessions WHERE claude_session_id = $1 AND agent_id = $2`,
		ev.SessionID, agentProdID).Scan(&sessionID)
	if err == pgx.ErrNoRows {
		sessionID, err = upsertSession(ctx, ev.SessionID, devID, agentProdID, devAgentID, ev.Kind)
	}
	if err != nil {
		return fmt.Errorf("usage: resolve session: %w", err)
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	keys := make([]string, 0, len(ev.Usage.Records))
	for _, r := range ev.Usage.Records {
		if r.RequestKey == "" || r.Model == "" {
			continue
		}
		if !tokensInBounds(r) {
			log.Printf("ingest: usage record %s out of bounds, skipped", r.RequestKey)
			continue
		}
		occurred := r.OccurredAt
		if occurred.IsZero() || occurred.After(time.Now().Add(24*time.Hour)) {
			occurred = ev.Timestamp
		}
		// Streaming can deliver the same request twice with growing counts; GREATEST
		// makes the merge order-independent and keeps retries idempotent.
		_, err := tx.Exec(ctx, `
			INSERT INTO token_usage_events (
				session_id, request_key, source, model_name, is_sidechain,
				input_tokens, output_tokens, cache_read_tokens,
				cache_creation_5m_tokens, cache_creation_1h_tokens,
				thinking_tokens, service_tier, client_version, occurred_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
			ON CONFLICT (session_id, request_key, source) DO UPDATE SET
				input_tokens             = GREATEST(token_usage_events.input_tokens,             EXCLUDED.input_tokens),
				output_tokens            = GREATEST(token_usage_events.output_tokens,            EXCLUDED.output_tokens),
				cache_read_tokens        = GREATEST(token_usage_events.cache_read_tokens,        EXCLUDED.cache_read_tokens),
				cache_creation_5m_tokens = GREATEST(token_usage_events.cache_creation_5m_tokens, EXCLUDED.cache_creation_5m_tokens),
				cache_creation_1h_tokens = GREATEST(token_usage_events.cache_creation_1h_tokens, EXCLUDED.cache_creation_1h_tokens),
				thinking_tokens          = GREATEST(token_usage_events.thinking_tokens,          EXCLUDED.thinking_tokens)`,
			sessionID, r.RequestKey, source, r.Model, r.IsSidechain,
			r.InputTokens, r.OutputTokens, r.CacheReadTokens,
			r.CacheCreation5mTokens, r.CacheCreation1hTokens,
			r.ThinkingTokens, strPtrOrNil(r.ServiceTier), strPtrOrNil(ev.Usage.ClientVersion), occurred,
		)
		if err != nil {
			return fmt.Errorf("usage: insert %s: %w", r.RequestKey, err)
		}
		keys = append(keys, r.RequestKey)
	}
	if _, err := tx.Exec(ctx, repriceStmt(`u2.session_id = $1 AND u2.request_key = ANY($2)`), sessionID, keys); err != nil {
		return fmt.Errorf("usage: price: %w", err)
	}
	return tx.Commit(ctx)
}

func tokensInBounds(r models.UsageRecord) bool {
	for _, v := range []int64{r.InputTokens, r.OutputTokens, r.CacheReadTokens,
		r.CacheCreation5mTokens, r.CacheCreation1hTokens, r.ThinkingTokens} {
		if v < 0 || v > maxTokensPerField {
			return false
		}
	}
	return true
}

// repriceStmt recomputes cost_usd/price_id from model_prices (longest matching prefix,
// latest effective_from not after the request date) for rows matching scope, a
// condition on alias u2. Unmatched models resolve to NULL cost — unknown, not zero.
func repriceStmt(scope string) string {
	return repriceHead + scope + repriceTail
}

const repriceHead = `
	UPDATE token_usage_events u
	SET cost_usd = c.cost, price_id = c.price_id
	FROM (
		SELECT u2.usage_id, p.price_id,
		       (u2.input_tokens            * p.input_per_mtok
		      + u2.output_tokens           * p.output_per_mtok
		      + u2.cache_read_tokens       * p.cache_read_per_mtok
		      + u2.cache_creation_5m_tokens * p.cache_write_5m_per_mtok
		      + u2.cache_creation_1h_tokens * p.cache_write_1h_per_mtok) / 1000000.0 AS cost
		FROM token_usage_events u2
		LEFT JOIN LATERAL (
			SELECT * FROM model_prices mp
			WHERE starts_with(u2.model_name, mp.model_prefix)
			  AND mp.effective_from <= u2.occurred_at::date
			ORDER BY length(mp.model_prefix) DESC, mp.effective_from DESC
			LIMIT 1
		) p ON TRUE
		WHERE `
const repriceTail = `
	) c WHERE u.usage_id = c.usage_id`

// RepriceAll recomputes cost for every ledger row (after model_prices changed).
func RepriceAll(ctx context.Context) (int64, error) {
	tag, err := db.Pool.Exec(ctx, repriceStmt(`TRUE`))
	return tag.RowsAffected(), err
}
