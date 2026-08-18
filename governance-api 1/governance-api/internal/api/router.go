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
		r.Put("/policies/{id}/activate", ActivatePolicy)
		r.Get("/policies/{id}/history", GetPolicyHistory)

		// ── Sessions ───────────────────────────────────────────────────
		r.Get("/sessions", ListSessions)
		r.Get("/sessions/{id}/timeline", GetSessionTimeline)
		r.Get("/sessions/{id}/timeline/export", ExportSessionTimeline)

		// ── Violations ─────────────────────────────────────────────────
		r.Get("/violations", ListViolations)
		r.Get("/violations/export", ExportViolations)

		// ── Developers ─────────────────────────────────────────────────
		r.Get("/developers/list", ListDevelopers) // /developers?username= is already used by cx

		// ── Agents ─────────────────────────────────────────────────────
		r.Get("/agents", ListAgents)

		// ── MCP Servers ────────────────────────────────────────────────
		r.Get("/mcp-servers", ListMCPServers)
		r.Get("/mcp-servers/{id}", GetMCPServer)

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
