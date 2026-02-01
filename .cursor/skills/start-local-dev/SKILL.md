---
name: start-local-dev
description: Start the local development environment with all services (database, API, importer, evaluator)
---

# Start Local Dev

Start and manage the Tarkov Build Optimiser services for local development.

## Quick Start

### Start Everything

```bash
# Start database and all Docker services
task compose:up

# In separate terminals, start each service:
task api:start        # Terminal 1
task importer:start   # Terminal 2 (optional)
task evaluator:start  # Terminal 3 (optional)
```

### Start Just the API

```bash
# Recommended for DevContainer (assuming database is running)
task api:start

# Use if database needs to be started via Docker Compose
task api:start:docker
```

This automatically:
1. Builds the API binary
2. Runs the API server
3. (`:docker` version only) Starts Docker services (database, etc.)

## Services Overview

### Infrastructure (Docker Compose)

**PostgreSQL Database** - Required for all services. 
*Note: In a DevContainer, the database is usually already running.*

```bash
# Start only the database
task compose:postgres:up

# Start all Docker services
task compose:up

# Stop all services
task compose:down

# Restart services
task compose:restart
```

**Check status:**
```bash
docker compose ps
```

### Application Services

#### 1. API Server

The main HTTP API for querying optimal weapon builds.

```bash
task api:start
```

**What it does:**
- Builds from `cmd/api/main.go`
- Serves REST endpoints (check `internal/router/`)
- Depends on: PostgreSQL

**Endpoints:**
- Health check: `GET /health`
- Items: `GET /items/*` (see `internal/router/items/`)

**Configuration:** Uses `.env` file for database connection and other settings

#### 2. Importer

Fetches weapon and mod data from tarkov.dev GraphQL API and populates the database.

```bash
# Normal mode: fetch fresh data from API
task importer:start

# Use cached data (faster for development)
task importer:start:use-cache

# Only cache data, don't import to database
task importer:start:cache-only
```

**What it does:**
- Fetches data from tarkov.dev GraphQL API
- Imports weapons, mods, trader offers, etc.
- Populates database tables

**When to run:**
- After initializing a fresh database
- When tarkov.dev data has been updated
- When you want to refresh local data

**Cache files:** Stored in `file-caches/*.json` (see `internal/cache/`)

#### 3. Evaluator

Computes optimal weapon builds and caches results.

```bash
# Normal mode: evaluate all weapons
task evaluator:start

# Test mode: limited weapons and trader levels (faster)
task evaluator:start:test-mode
```

**What it does:**
- Evaluates all possible weapon configurations
- Finds optimal builds for each weapon
- Stores results in `optimum_builds` table
- Uses precomputed subtrees for efficiency

**When to run:**
- After importing fresh data
- When build optimization algorithm changes
- To regenerate cached optimal builds

**Performance:** Test mode is much faster for development/testing

## Typical Development Workflows

### Working on API Endpoints

```bash
# Terminal 1: Start database
task compose:up

# Terminal 2: Start API with auto-rebuild
task api:start
# Or use air for live reload (if configured)
```

### Working on Data Import

```bash
# Terminal 1: Ensure database is running
task compose:postgres:up

# Terminal 2: Run importer with cache
task importer:start:use-cache

# If you need fresh data:
task importer:start
```

### Working on Build Evaluation

```bash
# Terminal 1: Database
task compose:up

# Terminal 2: Import data first
task importer:start:use-cache

# Terminal 3: Run evaluator in test mode
task evaluator:start:test-mode
```

### Full Stack Testing

```bash
# Terminal 1: All infrastructure
task compose:up

# Terminal 2: API
task api:start

# Terminal 3 (optional): Re-import data
task importer:start:use-cache

# Terminal 4 (optional): Re-evaluate builds
task evaluator:start:test-mode
```

## Service Dependencies

```
PostgreSQL (Docker)
    ├── API Server (reads optimal builds)
    ├── Importer (writes weapon/mod data)
    └── Evaluator (reads weapon data, writes optimal builds)
```

**Startup order:**
1. PostgreSQL (`task compose:up`)
2. Migrations (`task migrate:up` - if not already applied)
3. Importer (`task importer:start` - if database is empty)
4. Evaluator (`task evaluator:start` - to compute builds)
5. API (`task api:start` - to serve queries)

## Configuration

All services use environment variables from `.env`:

```bash
# Database connection
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=tarkov_build_optimiser

# API configuration
API_PORT=8080

# Other settings...
```

## Troubleshooting

### Database Connection Failures

**Symptoms:** Services can't connect to PostgreSQL

**Solutions:**
```bash
# Check if database is running
docker compose ps

# Check database logs
docker compose logs postgres

# Verify .env settings match docker-compose.yml
cat .env
cat docker-compose.yml

# Restart database
task compose:down
task compose:up
```

### API Won't Start

**Check:**
- Database is running: `docker compose ps`
- Migrations applied: `task migrate:up`
- Port 8080 not in use: `lsof -i :8080`
- Build errors: `task api:build`

### Importer Fails

**Check:**
- Database is running
- Migrations applied
- GraphQL API is accessible: `curl https://api.tarkov.dev/graphql`
- Cache files if using `use-cache` mode: `ls file-caches/`

### Evaluator Takes Too Long

**Solutions:**
- Use test mode: `task evaluator:start:test-mode`
- Check if data import completed
- Monitor with logs to see progress

### Port Already in Use

**Find and kill the process:**
```bash
# Find process using port 8080
lsof -i :8080

# Kill it
kill -9 <PID>
```

### Stale Data

**Fresh start:**
```bash
# Stop everything
task compose:down

# Remove volumes (deletes all data)
docker compose down -v

# Rebuild and restart
task compose:up
task migrate:up
task importer:start
```

## Monitoring

### View Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f postgres

# API (if running in terminal, just watch the output)
```

### Database Inspection

```bash
# Connect to database
docker compose exec postgres psql -U $POSTGRES_USER -d $POSTGRES_DB

# List tables
\dt

# Check imported data
SELECT COUNT(*) FROM weapons;
SELECT COUNT(*) FROM weapon_mods;
SELECT COUNT(*) FROM optimum_builds;

# Exit
\q
```

## Performance Tips

- Use `importer:start:use-cache` to avoid API calls during development
- Use `evaluator:start:test-mode` for faster iteration
- Keep Docker running between sessions to avoid startup time
- Run only the services you need (not always all 3)
