package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/checkmarx/governance-api/internal/db"
	"github.com/checkmarx/governance-api/internal/ingest"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// All token totals read token_usage_effective (one row per real request, otel >
// transcript > estimate) — never token_usage_events directly, or a request seen by
// two sources would be counted twice.
//
// Cost semantics: cost_usd is NULL for models with no price row. Sums skip NULLs and
// the response carries unpricedRequests so the UI can say "partial" instead of
// presenting an understated total as complete.

// usageTotalsSQL is the aggregate column list shared by every usage query.
const usageTotalsSQL = `
	COUNT(*)::bigint,
	COALESCE(SUM(input_tokens),0)::bigint,
	COALESCE(SUM(output_tokens),0)::bigint,
	COALESCE(SUM(cache_read_tokens),0)::bigint,
	COALESCE(SUM(cache_creation_tokens),0)::bigint,
	SUM(cost_usd)::float8,
	(COUNT(*) FILTER (WHERE cost_usd IS NULL))::bigint`

type usageTotals struct {
	Requests         int64    `json:"requests"`
	InputTokens      int64    `json:"inputTokens"`
	OutputTokens     int64    `json:"outputTokens"`
	CacheReadTokens  int64    `json:"cacheReadTokens"`
	CacheWriteTokens int64    `json:"cacheCreationTokens"`
	TotalTokens      int64    `json:"totalTokens"`
	CostUSD          *float64 `json:"costUsd"` // nil = nothing priced
	UnpricedRequests int64    `json:"unpricedRequests"`
}

type rowScanner interface{ Scan(dest ...any) error }

func (u *usageTotals) scanFrom(s rowScanner, extra ...any) error {
	dest := append(extra, &u.Requests, &u.InputTokens, &u.OutputTokens, &u.CacheReadTokens,
		&u.CacheWriteTokens, &u.CostUSD, &u.UnpricedRequests)
	if err := s.Scan(dest...); err != nil {
		return err
	}
	u.TotalTokens = u.InputTokens + u.OutputTokens + u.CacheReadTokens + u.CacheWriteTokens
	return nil
}

// ============================================================
// GET /api/v1/sessions/{id}/usage
// ============================================================

func GetSessionUsage(w http.ResponseWriter, r *http.Request) {
	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid session id")
		return
	}
	ctx := r.Context()

	var totals usageTotals
	if err := totals.scanFrom(db.Pool.QueryRow(ctx,
		`SELECT `+usageTotalsSQL+` FROM token_usage_effective WHERE session_id = $1`, sessionID)); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if totals.Requests == 0 {
		jsonOK(w, map[string]any{"hasData": false})
		return
	}

	type modelRow struct {
		Model string `json:"model"`
		usageTotals
	}
	byModel := []modelRow{}
	rows, err := db.Pool.Query(ctx, `
		SELECT model_name, `+usageTotalsSQL+`
		FROM token_usage_effective WHERE session_id = $1
		GROUP BY model_name ORDER BY SUM(input_tokens + output_tokens + cache_read_tokens + cache_creation_tokens) DESC`, sessionID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for rows.Next() {
		var m modelRow
		if err := m.scanFrom(rows, &m.Model); err != nil {
			log.Printf("api: GetSessionUsage model scan: %v", err)
			continue
		}
		byModel = append(byModel, m)
	}
	rows.Close()

	// Sub-agent share of the session.
	var side usageTotals
	_ = side.scanFrom(db.Pool.QueryRow(ctx,
		`SELECT `+usageTotalsSQL+` FROM token_usage_effective WHERE session_id = $1 AND is_sidechain`, sessionID))

	// Cumulative series, one point per request (bounded so a huge session stays cheap).
	type point struct {
		At               time.Time `json:"at"`
		Model            string    `json:"model"`
		Tokens           int64     `json:"tokens"`
		CumulativeTokens int64     `json:"cumulativeTokens"`
		CumulativeCost   *float64  `json:"cumulativeCostUsd"`
	}
	series := []point{}
	prows, err := db.Pool.Query(ctx, `
		SELECT occurred_at, model_name,
		       (input_tokens + output_tokens + cache_read_tokens + cache_creation_tokens)::bigint,
		       SUM(input_tokens + output_tokens + cache_read_tokens + cache_creation_tokens) OVER w,
		       SUM(cost_usd) OVER w
		FROM token_usage_effective WHERE session_id = $1
		WINDOW w AS (ORDER BY occurred_at, request_key)
		ORDER BY occurred_at, request_key
		LIMIT 2000`, sessionID)
	if err == nil {
		for prows.Next() {
			var p point
			var cumTok int64
			var cumCost *float64
			if err := prows.Scan(&p.At, &p.Model, &p.Tokens, &cumTok, &cumCost); err != nil {
				continue
			}
			p.CumulativeTokens, p.CumulativeCost = cumTok, cumCost
			series = append(series, p)
		}
		prows.Close()
	}

	jsonOK(w, map[string]any{
		"hasData":   true,
		"totals":    totals,
		"byModel":   byModel,
		"subagents": side,
		"series":    series,
	})
}

// ============================================================
// GET /api/v1/analytics/token-usage?days=30
// ============================================================

func GetTokenUsageAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &days); err != nil || days < 1 {
			days = 30
		}
	}
	if days > 365 {
		days = 365
	}
	since := time.Now().AddDate(0, 0, -days)

	var totals usageTotals
	var sessions int64
	err := db.Pool.QueryRow(ctx, `
		SELECT `+usageTotalsSQL+`, COUNT(DISTINCT session_id)::bigint
		FROM token_usage_effective WHERE occurred_at >= $1`, since).Scan(
		&totals.Requests, &totals.InputTokens, &totals.OutputTokens, &totals.CacheReadTokens,
		&totals.CacheWriteTokens, &totals.CostUSD, &totals.UnpricedRequests, &sessions)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	totals.TotalTokens = totals.InputTokens + totals.OutputTokens + totals.CacheReadTokens + totals.CacheWriteTokens

	type dayRow struct {
		Date string `json:"date"`
		usageTotals
	}
	daily := []dayRow{}
	if rows, err := db.Pool.Query(ctx, `
		SELECT to_char(date_trunc('day', occurred_at), 'YYYY-MM-DD'), `+usageTotalsSQL+`
		FROM token_usage_effective WHERE occurred_at >= $1
		GROUP BY 1 ORDER BY 1`, since); err == nil {
		for rows.Next() {
			var d dayRow
			if d.scanFrom(rows, &d.Date) == nil {
				daily = append(daily, d)
			}
		}
		rows.Close()
	}

	type modelRow struct {
		Model string `json:"model"`
		usageTotals
	}
	byModel := []modelRow{}
	if rows, err := db.Pool.Query(ctx, `
		SELECT model_name, `+usageTotalsSQL+`
		FROM token_usage_effective WHERE occurred_at >= $1
		GROUP BY model_name
		ORDER BY SUM(input_tokens + output_tokens + cache_read_tokens + cache_creation_tokens) DESC`, since); err == nil {
		for rows.Next() {
			var m modelRow
			if m.scanFrom(rows, &m.Model) == nil {
				byModel = append(byModel, m)
			}
		}
		rows.Close()
	}

	type devRow struct {
		Email    string `json:"email"`
		Sessions int64  `json:"sessions"`
		usageTotals
	}
	byDev := []devRow{}
	if rows, err := db.Pool.Query(ctx, `
		SELECT d.email, COUNT(DISTINCT u.session_id)::bigint, `+usageTotalsSQL+`
		FROM token_usage_effective u
		JOIN ai_sessions s ON s.session_id = u.session_id
		JOIN developers d  ON d.developer_id = s.developer_id
		WHERE u.occurred_at >= $1
		GROUP BY d.email
		ORDER BY SUM(u.input_tokens + u.output_tokens + u.cache_read_tokens + u.cache_creation_tokens) DESC
		LIMIT 20`, since); err == nil {
		for rows.Next() {
			var d devRow
			if d.scanFrom(rows, &d.Email, &d.Sessions) == nil {
				byDev = append(byDev, d)
			}
		}
		rows.Close()
	}

	type sessRow struct {
		SessionID string    `json:"sessionId"`
		Email     string    `json:"email"`
		StartedAt time.Time `json:"startedAt"`
		usageTotals
	}
	top := []sessRow{}
	if rows, err := db.Pool.Query(ctx, `
		SELECT s.session_id::text, d.email, s.started_at, `+usageTotalsSQL+`
		FROM token_usage_effective u
		JOIN ai_sessions s ON s.session_id = u.session_id
		JOIN developers d  ON d.developer_id = s.developer_id
		WHERE u.occurred_at >= $1
		GROUP BY s.session_id, d.email, s.started_at
		ORDER BY SUM(u.input_tokens + u.output_tokens + u.cache_read_tokens + u.cache_creation_tokens) DESC
		LIMIT 10`, since); err == nil {
		for rows.Next() {
			var s sessRow
			if s.scanFrom(rows, &s.SessionID, &s.Email, &s.StartedAt) == nil {
				top = append(top, s)
			}
		}
		rows.Close()
	}

	jsonOK(w, map[string]any{
		"days":        days,
		"sessions":    sessions,
		"totals":      totals,
		"daily":       daily,
		"byModel":     byModel,
		"byDeveloper": byDev,
		"topSessions": top,
	})
}

// ============================================================
// GET /api/v1/token-usage/prices, POST /api/v1/token-usage/reprice
// ============================================================

// ListModelPrices exposes the price table so the UI can show which rates were used.
func ListModelPrices(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Pool.Query(r.Context(), `
		SELECT model_prefix, effective_from::text, input_per_mtok::float8, output_per_mtok::float8,
		       cache_read_per_mtok::float8, cache_write_5m_per_mtok::float8, cache_write_1h_per_mtok::float8,
		       COALESCE(note,'')
		FROM model_prices ORDER BY model_prefix, effective_from DESC`)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var prefix, from, note string
		var in, outp, cr, w5, w1 float64
		if rows.Scan(&prefix, &from, &in, &outp, &cr, &w5, &w1, &note) != nil {
			continue
		}
		out = append(out, map[string]any{
			"modelPrefix": prefix, "effectiveFrom": from, "inputPerMtok": in, "outputPerMtok": outp,
			"cacheReadPerMtok": cr, "cacheWrite5mPerMtok": w5, "cacheWrite1hPerMtok": w1, "note": note,
		})
	}
	jsonOK(w, out)
}

// RepriceUsage recomputes cost for the whole ledger after model_prices changed.
func RepriceUsage(w http.ResponseWriter, r *http.Request) {
	n, err := ingest.RepriceAll(r.Context())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOK(w, map[string]any{"repriced": n})
}
