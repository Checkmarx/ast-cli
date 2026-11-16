# AI Governance Platform - End-to-End Agent Setup Guide

**Designed for: Autonomous coding agents (Claude Code, Cursor, etc.)**

This guide provides complete step-by-step instructions to set up the entire AI Governance platform from scratch.

---

## Phase 1: Prerequisites Verification

```bash
# Operating system
uname -s

# Available disk space (need 5GB+)
df -h | grep -E "/$|/home"

# RAM available
free -h || vm_stat
```

### Verify Required Tools

```bash
# Check Git
git --version || exit 1

# Check Go 1.21+
go version || exit 1

# Check Node.js 18+
node --version || exit 1

# Check PostgreSQL 14+
psql --version || exit 1
```

**Exit if any tool is missing.**

---

## Phase 2: Database Setup

### Start PostgreSQL Service

```bash
# Check if PostgreSQL is running
pg_isready -h localhost -p 5432 || {
  echo "Starting PostgreSQL..."
  # Linux
  sudo service postgresql start
  # macOS
  brew services start postgresql
  # Windows - should auto-start
  
  sleep 3
  pg_isready -h localhost -p 5432
}
```

### Create Database and User

```bash
# Create governance user and database
psql -U postgres << SQL_SCRIPT
CREATE USER governance WITH PASSWORD 'governance';
CREATE DATABASE governance OWNER governance;
GRANT ALL PRIVILEGES ON DATABASE governance TO governance;
\c governance
GRANT ALL ON SCHEMA public TO governance;
SQL_SCRIPT

# Verify
psql -U governance -d governance -c "SELECT version();"
```

**Exit if database creation fails.**

---

## Phase 3: Repository Setup

### Clone Repository

```bash
# Clone the AI Governance repository
cd ~/Documents/MyProjects || mkdir -p ~/Documents/MyProjects && cd ~/Documents/MyProjects

git clone <REPOSITORY_URL>
cd "AI Governance"

# Verify
git status
git branch
```

**Exit if clone fails.**

### Verify Directory Structure

```bash
# Check critical directories exist
test -d governance-api/governance-api || exit 1
test -d governance-ui/governance-ui || exit 1
test -f COMPLIANCE_ARCHITECTURE.md || exit 1

echo "✅ Directory structure verified"
```

---

## Phase 4: Backend Setup (Go)

### Navigate and Install Dependencies

```bash
cd governance-api/governance-api

# Download Go modules
go mod download
go mod verify

echo "✅ Go dependencies installed"
```

**Exit if mod download fails.**

### Apply Database Migrations

```bash
# Run setup script
psql -U governance -d governance -f scripts/setup-db.sql

# Apply schema migrations in order
for migration in internal/db/migrations/001_*.sql \
                internal/db/migrations/002_*.sql \
                internal/db/migrations/003_*.sql \
                internal/db/migrations/004_*.sql \
                internal/db/migrations/005_*.sql \
                internal/db/migrations/006_*.sql \
                internal/db/migrations/007_*.sql \
                internal/db/migrations/008_*.sql; do
  if [ -f "$migration" ]; then
    echo "Applying: $(basename $migration)"
    psql -U governance -d governance -f "$migration"
    if [ $? -ne 0 ]; then
      echo "❌ Migration failed: $migration"
      exit 1
    fi
  fi
done

echo "✅ Database migrations complete"
```

**Exit if any migration fails.**

### Build Backend Binary

```bash
# Build the Go application
go build -o governance-api.exe ./cmd/server

# Verify binary exists
test -f governance-api.exe && echo "✅ Backend binary built successfully" || exit 1
```

**Exit if build fails.**

### Create Backend Environment File

```bash
# Create .env file
cat > .env << ENV_FILE
DATABASE_URL=postgresql://governance:governance@localhost:5432/governance
PORT=8080
HOST=localhost
ENV=development
API_BASE_URL=http://localhost:8080/api/v1
CORS_ORIGINS=http://localhost:3000
LOG_LEVEL=info
ENV_FILE

echo "✅ Backend .env created"
```

### Verify Backend Configuration

```bash
# Check database connection
psql -U governance -d governance -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';" || exit 1

echo "✅ Backend ready for startup"
```

---

## Phase 5: Frontend Setup (Next.js)

### Navigate and Install Dependencies

```bash
cd ../../governance-ui/governance-ui

# Install npm packages
npm install

if [ $? -ne 0 ]; then
  echo "❌ npm install failed"
  exit 1
fi

echo "✅ Frontend dependencies installed"
```

**Exit if npm install fails.**

### Build Frontend

```bash
# Build Next.js application
npm run build

# Verify build succeeded
test -d .next && echo "✅ Frontend built successfully" || {
  echo "❌ Frontend build failed"
  exit 1
}
```

**Exit if build fails.**

### Create Frontend Environment File

```bash
# Create .env.local file
cat > .env.local << ENV_LOCAL
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
ENV_LOCAL

echo "✅ Frontend .env.local created"
```

---

## Phase 6: Verification & Testing

### Verify All Components

```bash
echo "📋 Verification Report"
echo "===================="

# Check backend binary
test -f governance-api/governance-api/governance-api.exe && echo "✅ Backend binary exists" || echo "❌ Backend binary missing"

# Check frontend build
test -d governance-ui/governance-ui/.next && echo "✅ Frontend build exists" || echo "❌ Frontend build missing"

# Check environment files
test -f governance-api/governance-api/.env && echo "✅ Backend .env exists" || echo "❌ Backend .env missing"
test -f governance-ui/governance-ui/.env.local && echo "✅ Frontend .env.local exists" || echo "❌ Frontend .env.local missing"

# Check database
psql -U governance -d governance -c "SELECT COUNT(*) as table_count FROM information_schema.tables WHERE table_schema='public';" && echo "✅ Database connected" || echo "❌ Database unreachable"

echo "✅ Verification complete"
```

---

## Phase 7: Load Test Data

### Create Test Session

```bash
cd governance-api/governance-api

# Create dummy session with aegis block
psql -U governance -d governance -f scripts/create-dummy-session.sql

echo "✅ Test data loaded"
```

---

## Phase 8: Startup Instructions

### Terminal 1 - Start Backend

```bash
cd governance-api/governance-api

# Option A: Direct execution
./governance-api.exe

# Option B: Go run
# go run ./cmd/server

# Expected output:
# [INFO] Starting AI Governance API server
# [INFO] Connected to PostgreSQL at localhost:5432
# [INFO] Server running on http://localhost:8080
```

**Wait for "Server running" message before proceeding.**

### Terminal 2 - Start Frontend

```bash
cd governance-ui/governance-ui

# Run in development mode
npm run dev

# Expected output:
# > ready - started server on 0.0.0.0:3000, url: http://localhost:3000
```

**Wait for "ready" message.**

### Terminal 3 - Verify Both Services

```bash
# Verify backend
curl -s http://localhost:8080/api/v1/health | grep -q "ok" && echo "✅ Backend healthy" || echo "❌ Backend unhealthy"

# Verify frontend
curl -s http://localhost:3000 | grep -q "html" && echo "✅ Frontend healthy" || echo "❌ Frontend unhealthy"
```

---

## Phase 9: Dashboard Access

### Verify All Systems Running

Dashboard should be accessible at: **http://localhost:3000**

Check for:
- ✅ Sessions page populated
- ✅ Violations showing aegis block
- ✅ Compliance dashboard
- ✅ Policies list
- ✅ Developers inventory
- ✅ Audit logs

---

## Phase 10: Automated Startup Script (Optional)

### Create reusable startup script

**File: `start-all.sh`**

```bash
#!/bin/bash

echo "Starting AI Governance Platform..."

# Start backend in background
cd governance-api/governance-api
./governance-api.exe &
BACKEND_PID=$!
echo "✅ Backend started (PID: $BACKEND_PID)"

# Wait for backend to be ready
sleep 3
curl -s http://localhost:8080/api/v1/health > /dev/null || {
  echo "❌ Backend failed to start"
  kill $BACKEND_PID
  exit 1
}

# Start frontend
cd ../../governance-ui/governance-ui
npm run dev &
FRONTEND_PID=$!
echo "✅ Frontend started (PID: $FRONTEND_PID)"

# Wait for frontend
sleep 5

echo "✅ AI Governance running:"
echo "   Dashboard: http://localhost:3000"
echo "   API: http://localhost:8080"
echo ""
echo "To stop: kill $BACKEND_PID $FRONTEND_PID"

wait
```

**Make executable:**
```bash
chmod +x start-all.sh
./start-all.sh
```

---

## Phase 11: Troubleshooting Guide

### Issue: Database Connection Error

```bash
# Check PostgreSQL service
pg_isready -h localhost -p 5432

# Restart PostgreSQL
sudo service postgresql restart  # Linux
brew services restart postgresql  # macOS

# Verify credentials
psql -U governance -d governance -c "SELECT 1;"
```

### Issue: Port 8080 Already in Use

```bash
# Find process using port
lsof -i :8080  # macOS/Linux
netstat -ano | findstr :8080  # Windows

# Kill the process
kill -9 <PID>  # macOS/Linux
taskkill /PID <PID> /F  # Windows

# Or change PORT in governance-api/governance-api/.env to 8081
```

### Issue: npm Install Fails

```bash
# Clear npm cache
npm cache clean --force

# Delete lock file and reinstall
rm package-lock.json
npm install
```

### Issue: Go Build Fails

```bash
# Check Go installation
go version

# Get latest dependencies
go get -u ./...

# Try build again
go build -o governance-api.exe ./cmd/server
```

### Issue: Frontend Can't Connect to Backend

```bash
# Verify backend running
curl -v http://localhost:8080/api/v1/health

# Check .env.local in governance-ui/governance-ui
cat .env.local

# Should show: NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1

# Rebuild and restart frontend
npm run build && npm run dev
```

### Issue: Database Migrations Fail

```bash
# Check migration file syntax
psql -U governance -d governance -f internal/db/migrations/001_initial_schema.sql

# View database state
psql -U governance -d governance -c "\dt"

# Reset and retry (CAUTION: loses data)
psql -U postgres -c "DROP DATABASE governance;"
psql -U postgres -c "CREATE DATABASE governance OWNER governance;"
# Re-run all migrations
```

---

## Phase 12: Success Criteria

✅ Setup is complete when:

- [ ] PostgreSQL database created and all 8 migrations applied
- [ ] Backend binary built successfully at `governance-api/governance-api/governance-api.exe`
- [ ] Frontend built successfully with `.next` directory present
- [ ] Both environment files created (`.env` and `.env.local`)
- [ ] Backend running on http://localhost:8080
- [ ] Frontend running on http://localhost:3000
- [ ] Dashboard accessible and displaying test data
- [ ] Violations showing aegis session block with npm install lodash
- [ ] No console errors in either service
- [ ] All verification checks passing

---

## Summary of Components

### Governance API (Go Backend)
- Location: `governance-api/governance-api`
- Port: 8080
- Function: Policy enforcement, event processing, compliance calculation
- Database: PostgreSQL (governance)

### Governance Dashboard (Next.js Frontend)
- Location: `governance-ui/governance-ui`
- Port: 3000
- Function: Session monitoring, violation review, compliance reporting
- Connected to: Backend API at :8080

### PostgreSQL Database
- Database: `governance`
- User: `governance`
- Port: 5432
- Function: Event storage, audit logs, policy definitions, session data

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────┐
│  Governance Dashboard (Next.js)                  │
│  http://localhost:3000                           │
│  - Sessions, Violations, Policies                │
│  - Compliance Status, Audit Logs                 │
└────────────────┬────────────────────────────────┘
                 │
                 │ HTTP/REST
                 │
┌────────────────┴────────────────────────────────┐
│  Governance API (Go)                            │
│  http://localhost:8080                          │
│  - Event Ingestion                              │
│  - Policy Enforcement                           │
│  - Compliance Calculation                       │
└────────────────┬────────────────────────────────┘
                 │
                 │ SQL
                 │
┌────────────────┴────────────────────────────────┐
│  PostgreSQL Database                            │
│  localhost:5432/governance                      │
│  - Sessions, Events, Policies                   │
│  - Audit Logs, Compliance Records               │
└─────────────────────────────────────────────────┘
```

---

## Reference URLs

- **Dashboard**: http://localhost:3000
- **API**: http://localhost:8080/api/v1
- **Health Check**: http://localhost:8080/api/v1/health
- **Architecture Doc**: `COMPLIANCE_ARCHITECTURE.md`

---

## Time Estimates

- Prerequisites verification: 2 minutes
- Database setup: 3 minutes
- Repository clone: 2 minutes
- Backend setup: 5 minutes
- Frontend setup: 3 minutes
- Startup & verification: 2 minutes
- **Total: ~15-20 minutes**

---

## Important Notes

1. **First Run**: Initial build may take longer (Go compilation, npm install)
2. **Database**: First migration run creates all tables and indexes
3. **Port Conflicts**: If ports 3000 or 8080 in use, change in respective `.env` files
4. **PostgreSQL**: Must be running and accessible before starting backend
5. **Node Version**: Must be 18+ for frontend to build

---

## Support

For detailed architecture information, refer to: `COMPLIANCE_ARCHITECTURE.md`

For specific component docs:
- Backend: `governance-api/governance-api/README.md`
- Frontend: `governance-ui/governance-ui/README.md`

---

**Document Version**: 1.0  
**Last Updated**: 2026-10-29  
**Target Audience**: Autonomous Coding Agents  
**Estimated Setup Time**: 15-20 minutes  
**Status**: Production Ready
