// Command server runs the Checkmarx AI Governance Policy API Server.
// Usage: server [--port PORT] [--migrations-dir DIR]
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/checkmarx/governance-api/internal/api"
	"github.com/checkmarx/governance-api/internal/db"
)

func main() {
	port := flag.String("port", envOr("PORT", "8080"), "HTTP listen port")
	migrationsDir := flag.String("migrations-dir", defaultMigrationsDir(), "path to SQL migration files")
	flag.Parse()

	ctx := context.Background()

	// Connect to PostgreSQL
	if err := db.Connect(ctx); err != nil {
		log.Fatalf("governance-api: database connect: %v", err)
	}
	log.Println("governance-api: connected to PostgreSQL")

	// Run migrations
	if err := db.RunMigrations(ctx, *migrationsDir); err != nil {
		log.Fatalf("governance-api: migrations: %v", err)
	}
	log.Println("governance-api: migrations applied")

	// Build router
	router := api.NewRouter()

	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("governance-api: listening on :%s", *port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("governance-api: listen: %v", err)
		}
	}()

	<-stop
	log.Println("governance-api: shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("governance-api: shutdown: %v", err)
	}
	db.Pool.Close()
	log.Println("governance-api: stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// defaultMigrationsDir returns the migrations directory relative to the binary.
// Supports running from the project root during development.
func defaultMigrationsDir() string {
	if d := os.Getenv("MIGRATIONS_DIR"); d != "" {
		return d
	}
	// Try relative to executable (production)
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "migrations")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	// Development fallback — find project root via runtime caller
	_, file, _, ok := runtime.Caller(0)
	if ok {
		// file is .../cmd/server/main.go → go up 3 dirs to project root
		root := filepath.Join(filepath.Dir(file), "..", "..", "internal", "db", "migrations")
		if _, err := os.Stat(root); err == nil {
			return root
		}
	}
	return "internal/db/migrations"
}
