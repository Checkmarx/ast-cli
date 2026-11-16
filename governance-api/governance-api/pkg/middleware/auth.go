// Package middleware provides HTTP middleware for the governance API.
package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const ctxKeyActorID contextKey = "actorId"

// BearerAuth validates the Authorization: Bearer <token> header.
// Supports two modes:
//   - JWT: validated against JWT_SECRET env var; actor_id extracted from sub claim.
//   - API key: compared against API_KEY env var (for cx CLI agent calls).
//
// Requests without a valid token receive 401.
func BearerAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearer(r)
		if token == "" {
			http.Error(w, `{"error":"missing authorization"}`, http.StatusUnauthorized)
			return
		}

		// API key path (used by cx CLI agent hooks and spool flusher)
		if apiKey := os.Getenv("API_KEY"); apiKey != "" && token == apiKey {
			ctx := context.WithValue(r.Context(), ctxKeyActorID, "system")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// JWT path (used by dashboard)
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			// No JWT secret configured — allow all (development mode)
			ctx := context.WithValue(r.Context(), ctxKeyActorID, "dev")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(token, claims, func(_ *jwt.Token) (any, error) {
			return []byte(secret), nil
		})
		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		actorID, _ := claims["sub"].(string)
		ctx := context.WithValue(r.Context(), ctxKeyActorID, actorID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ActorID retrieves the actor identifier from the request context.
func ActorID(r *http.Request) string {
	v, _ := r.Context().Value(ctxKeyActorID).(string)
	return v
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
