@echo off
REM Start the governance-api server in development mode.
REM Run this from the governance-api directory after setting up the database.

set DATABASE_URL=postgres://governance:governance@localhost:5432/governance?sslmode=disable
set PORT=8080
set JWT_SECRET=dev-secret-change-in-production
set API_KEY=cx-dev-api-key
set MIGRATIONS_DIR=internal\db\migrations

go run ./cmd/server
