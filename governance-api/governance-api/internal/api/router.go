package api

import (
	"net/http"

	"github.com/checkmarx/governance-api/pkg/middleware"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// NewRouter builds and returns the full API router.
// All routes require Bearer authentication.
func NewRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)
	r.Use(corsHeaders)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.BearerAuth)

		// Overview (live stats for dashboard)
		r.Get("/overview", GetOverview)

		// ── Agent-facing endpoints (called by cx CLI) ──────────────────
		r.Get("/agent/policies", GetAgentPolicies)
		// HEAD and GET both registered so cx CLI and monitoring tools can use either method
		r.Head("/agent/policies/version", HeadAgentPoliciesVersion)
		r.Get("/agent/policies/version", HeadAgentPoliciesVersion)
		r.Post("/audit/events", IngestAuditEvents)
		r.Get("/developers", GetDeveloperByUsername) // ?username=
		r.Post("/agents/register", RegisterAgent)

		// ── Policies ───────────────────────────────────────────────────
		r.Get("/policies", ListPolicies)
		r.Post("/policies", CreatePolicy)
		r.Put("/policies/{id}", UpdatePolicy)
		r.Put("/policies/{id}/activate", ActivatePolicy)
		r.Put("/policies/{id}/disable", DisablePolicy)
		r.Put("/policies/{id}/enable", EnablePolicy)
		r.Delete("/policies/{id}", DeletePolicy)
		r.Get("/policies/{id}/history", GetPolicyHistory)

		// ── Sessions ───────────────────────────────────────────────────
		r.Get("/sessions", ListSessions)
		r.Get("/sessions/{id}/timeline", GetSessionTimeline)
		r.Get("/sessions/{id}/hooks", GetSessionHooks)
		r.Get("/sessions/{id}/usage", GetSessionUsage)
		r.Get("/sessions/{id}/timeline/export", ExportSessionTimeline)

		// ── Violations ─────────────────────────────────────────────────
		r.Get("/violations", ListViolations)
		r.Get("/violations/export", ExportViolations)
		r.Post("/violations/{id}/review", ReviewViolation)

		// ── Compliance ─────────────────────────────────────────────────
		r.Get("/compliance/frameworks", ListComplianceFrameworks)
		r.Get("/compliance/controls", ListControls)
		r.Get("/compliance/frameworks/{id}/controls", GetFrameworkControls)
		r.Get("/compliance/frameworks/{id}/status", GetFrameworkStatus)
		r.Get("/compliance/controls/{id}/evidence", GetControlEvidence)
		r.Get("/policies/{id}/control-mappings", ListPolicyControlMappings)
		r.Post("/policies/{id}/control-mappings", CreatePolicyControlMapping)
		r.Delete("/policies/{id}/control-mappings/{mappingId}", DeletePolicyControlMapping)

		// ── Developers ─────────────────────────────────────────────────
		r.Get("/developers/list", ListDevelopers) // /developers?username= is already used by cx

		// ── Agents ─────────────────────────────────────────────────────
		r.Get("/agents", ListAgents)

		// ── MCP Servers ────────────────────────────────────────────────
		r.Get("/mcp-servers", ListMCPServers)
		r.Get("/mcp-servers/{id}", GetMCPServer)

		// ── Skills ─────────────────────────────────────────────────────
		r.Get("/skills", ListSkills)
		r.Get("/skills/{id}", GetSkillDetail)
		r.Get("/sessions/{id}/skills", GetSessionSkills)

		// ── Models ─────────────────────────────────────────────────────
		r.Get("/models", ListModels)
		r.Get("/models/{modelId}", GetModel)
		r.Get("/analytics/model-usage", GetModelUsageStats)
		r.Get("/analytics/model-detection-issues", GetModelDetectionIssues)

		// ── Token usage ────────────────────────────────────────────────
		r.Get("/analytics/token-usage", GetTokenUsageAnalytics)
		r.Get("/token-usage/prices", ListModelPrices)
		r.Post("/token-usage/reprice", RepriceUsage)

		// ── Audit Log ──────────────────────────────────────────────────
		r.Get("/audit", ListAuditEvents)
	})

	return r
}

// corsHeaders adds permissive CORS headers for dashboard development.
// In production, restrict to the actual dashboard domain.
func corsHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
