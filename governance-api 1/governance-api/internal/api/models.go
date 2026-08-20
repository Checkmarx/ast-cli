package api

import (
	"net/http"
	"time"

	"github.com/checkmarx/governance-api/internal/db"
	"github.com/go-chi/chi/v5"
)

// ModelInfo represents an AI model in the system
type ModelInfo struct {
	ModelID            string  `json:"modelId"`
	Vendor             string  `json:"vendor"`
	Family             string  `json:"family"`
	Version            string  `json:"version"`
	Type               string  `json:"type"`
	ContextLength      int     `json:"contextLength"`
	CostPerMilInput    float64 `json:"costPerMilInput"`
	CostPerMilOutput   float64 `json:"costPerMilOutput"`
}

// ModelUsageStats represents usage statistics for a model
type ModelUsageStats struct {
	Model               string  `json:"model"`
	Vendor              string  `json:"vendor"`
	SessionCount        int     `json:"sessionCount"`
	UniqueDevelopers    int     `json:"uniqueDevelopers"`
	AverageDuration     int     `json:"averageDuration"`
	AverageConfidence   float64 `json:"averageConfidence"`
	EstimatedCostUSD    float64 `json:"estimatedCostUSD"`
}

// ModelDetectionIssue represents a session with detection problems
type ModelDetectionIssue struct {
	SessionID         string    `json:"sessionId"`
	DeveloperID       string    `json:"developerId"`
	DetectedModel     string    `json:"detectedModel"`
	Confidence        float64   `json:"confidence"`
	Source            string    `json:"source"`
	DetectionErrors   []string  `json:"detectionErrors"`
	StartedAt         time.Time `json:"startedAt"`
}

// ListModels returns all known AI models
func ListModels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := db.Pool.Query(ctx, `
		SELECT model_id, vendor, family, version, type, context_length,
		       cost_per_1m_input, cost_per_1m_output
		FROM ai_models
		ORDER BY vendor, family, version DESC
	`)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var models []ModelInfo
	for rows.Next() {
		var m ModelInfo
		if err := rows.Scan(
			&m.ModelID, &m.Vendor, &m.Family, &m.Version, &m.Type,
			&m.ContextLength, &m.CostPerMilInput, &m.CostPerMilOutput,
		); err != nil {
			continue
		}
		models = append(models, m)
	}

	jsonOK(w,models)
}

// GetModel returns details for a specific model
func GetModel(w http.ResponseWriter, r *http.Request) {
	modelID := chi.URLParam(r, "modelId")
	ctx := r.Context()

	var m ModelInfo
	err := db.Pool.QueryRow(ctx, `
		SELECT model_id, vendor, family, version, type, context_length,
		       cost_per_1m_input, cost_per_1m_output
		FROM ai_models
		WHERE model_id = $1
	`, modelID).Scan(
		&m.ModelID, &m.Vendor, &m.Family, &m.Version, &m.Type,
		&m.ContextLength, &m.CostPerMilInput, &m.CostPerMilOutput,
	)

	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}

	jsonOK(w,m)
}

// GetModelUsageStats returns usage statistics for models (last 30 days)
func GetModelUsageStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	lookback := time.Now().AddDate(0, 0, -30)

	rows, err := db.Pool.Query(ctx, `
		SELECT
			COALESCE(ae.model_detected, 'unknown') as model,
			COALESCE(ae.model_vendor, 'unknown') as vendor,
			COUNT(DISTINCT ae.session_id) as session_count,
			COUNT(DISTINCT dev.developer_id) as unique_developers,
			COALESCE(AVG(EXTRACT(EPOCH FROM (s.ended_at - s.started_at)))::INT, 0) as avg_duration,
			COALESCE(AVG(ae.model_confidence), 0) as avg_confidence,
			COALESCE(SUM((
				CASE
					WHEN ae.model_detected = 'claude-opus-5' THEN 15.00 * (s.request_count / 1000000.0)
					WHEN ae.model_detected = 'claude-sonnet-5' THEN 3.00 * (s.request_count / 1000000.0)
					WHEN ae.model_detected = 'claude-haiku-4' THEN 0.80 * (s.request_count / 1000000.0)
					ELSE 0
				END
			)), 0) as estimated_cost
		FROM activity_events ae
		JOIN ai_sessions s ON ae.session_id = s.session_id
		JOIN developers dev ON s.developer_id = dev.developer_id
		WHERE ae.kind = 'session.start'
			AND ae.occurred_at >= $1
			AND ae.model_detected IS NOT NULL
		GROUP BY ae.model_detected, ae.model_vendor
		ORDER BY session_count DESC
	`, lookback)

	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var stats []ModelUsageStats
	for rows.Next() {
		var s ModelUsageStats
		if err := rows.Scan(
			&s.Model, &s.Vendor, &s.SessionCount, &s.UniqueDevelopers,
			&s.AverageDuration, &s.AverageConfidence, &s.EstimatedCostUSD,
		); err != nil {
			continue
		}
		stats = append(stats, s)
	}

	jsonOK(w,map[string]interface{}{
		"models": stats,
		"window": map[string]string{
			"from": lookback.Format(time.RFC3339),
			"to":   time.Now().Format(time.RFC3339),
		},
	})
}

// GetModelDetectionIssues returns sessions with low confidence or detection failures
func GetModelDetectionIssues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	lookback := time.Now().AddDate(0, 0, -30)
	confidenceThreshold := 0.80

	rows, err := db.Pool.Query(ctx, `
		SELECT DISTINCT ON (ae.session_id)
			s.claude_session_id,
			dev.developer_id::TEXT,
			COALESCE(ae.model_detected, 'unknown'),
			COALESCE(ae.model_confidence, 0),
			COALESCE(ae.model_source, 'unknown'),
			COALESCE(ae.model_source_path, ''),
			ae.occurred_at
		FROM activity_events ae
		JOIN ai_sessions s ON ae.session_id = s.session_id
		JOIN developers dev ON s.developer_id = dev.developer_id
		WHERE ae.kind = 'session.start'
			AND ae.occurred_at >= $1
			AND (ae.model_confidence IS NULL OR ae.model_confidence < $2)
		ORDER BY ae.session_id, ae.occurred_at DESC
		LIMIT 100
	`, lookback, confidenceThreshold)

	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var issues []ModelDetectionIssue
	for rows.Next() {
		var issue ModelDetectionIssue
		var sourcePath string
		if err := rows.Scan(
			&issue.SessionID, &issue.DeveloperID, &issue.DetectedModel,
			&issue.Confidence, &issue.Source, &sourcePath, &issue.StartedAt,
		); err != nil {
			continue
		}
		issues = append(issues, issue)
	}

	jsonOK(w,map[string]interface{}{
		"issues": issues,
		"total":  len(issues),
		"window": "last 30 days",
	})
}

