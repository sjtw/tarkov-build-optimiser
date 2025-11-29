# Tarkov Build Optimiser

A Go-based tool for pre-computing and serving optimal weapon builds in Escape from Tarkov. The system analyzes all possible weapon configurations with different trader level constraints to find builds that minimize recoil or maximise ergonomics (todo).

## Overview

The project consists of three main components:

1. **Importer** - Fetches weapon and modification data from the [tarkov.dev](https://tarkov.dev) GraphQL API and stores it in PostgreSQL
2. **Evaluator** - Pre-computes optimal weapon builds for all weapons across various trader level combinations using a candidate tree algorithm
3. **API** - REST API that serves pre-computed optimal builds based on user constraints

Additionally, an Next.js UI for visualising evaluated builds can be found [here](https://github.com/sjtw/tarkov-build-optimiser-ui).

## Architecture

The system follows a batch processing architecture with three main services:

```
┌──────────────┐
│  tarkov.dev  │
│   GraphQL    │
└──────┬───────┘
       │
       │ fetch weapons/mods
       ▼
┌──────────────┐         ┌──────────────┐         ┌──────────────┐
│   Importer   │────────▶│  PostgreSQL  │◀────────│  Evaluator   │
│              │ write   │              │ read/   │              │
│              │ data    │  - weapons   │ write   │              │
└──────────────┘         │  - mods      │ builds  └──────────────┘
                         │  - traders   │
                         │  - conflicts │
                         │  - builds    │
                         └──────┬───────┘
                                │
                                │ read builds
                                ▼
                         ┌──────────────┐
                         │     API      │
                         │   (REST)     │
                         └──────────────┘
```

- **Importer**: Fetches weapon and mod data from tarkov.dev GraphQL API, transforms it, and writes to the database. Optionally caches responses to disk as JSON (primarly for development purposes).
- **Evaluator**: Reads weapon/mod data from the database, computes optimal builds for all weapons across different trader level combinations, and writes results back to the database. Runs as a batch job.
- **API**: REST service that queries pre-computed builds from the database based on user-specified constraints (weapon ID, trader levels).
- **PostgreSQL**: Central data store holding weapon definitions, mods, trader offers, item conflicts, and computed optimal builds.

## Evaluation Process

Each weapon has modification slots (e.g., "Handguard", "Muzzle") that accept compatible items. Items can have their own nested slots, forming a tree structure. The goal is to find the combination that minimizes recoil (or maximizes ergonomics - TODO).

Checking every possible combination would be intractable for complex weapons. Instead, the evaluator uses recursive search with several optimizations:

**Branch-and-bound pruning** — Before exploring a branch, calculate the theoretical best possible outcome for remaining slots. For recoil optimization, this means computing the lowest achievable recoil by using each slot's best recoil modifier. If even this best-case scenario can't beat the current solution, skip the entire branch.

**Conflict handling** — Some items are incompatible (e.g., certain stocks conflict with certain grips). When an item is selected, all incompatible items are added to an exclusion list for that branch. The evaluator also tries leaving slots empty, since avoiding a conflicting item in one slot may enable better items in other slots.

**Conflict-free caching** — Items without conflicts always produce the same optimal subtree. When such an item is encountered, its previously computed result (if cached) can be reused. This also enables additional pruning: if the cached subtree's stats can't improve the current best, skip evaluating that entire subtree.

**Useless item pruning** — Before evaluation starts, each item's potential value (its own modifier plus the best possible contribution from its nested slots) is calculated. Items whose best-case subtree cannot improve the target stat are filtered out (e.g., an item with a minimum achievable recoil of +5 when minimizing recoil).

The algorithm explores all viable branches and is guaranteed to find the globally optimal build, but pruning eliminates the vast majority of the search space.

## Prerequisites

**Required:**
- `docker` & `docker-compose`
- `go` 1.22+
- [Task](https://taskfile.dev/) (task runner)

**Optional:**
- `nodejs` (only needed for updating the tarkov.dev GraphQL schema)

## Getting Started

### Initial Setup

Set up the development environment (installs dependencies, starts Docker containers, runs migrations):

```bash
task init
```

Or if you don't have Node.js installed:

```bash
task init:go-only
```

### Running the System

1. **Import weapon data:**

```bash
task importer:start
```

The importer caches data to `file-caches/*.json` by default. To use cached data instead of fetching from tarkov.dev:

```bash
task importer:start:use-cache
```

To only cache without updating the database:

```bash
task importer:start:cache-only
```

2. **Compute optimal builds:**

```bash
task evaluator:start
```

For faster testing with limited weapons:

```bash
task evaluator:start:test-mode
```

3. **Start the API:**

```bash
task api:start
```

The API will be available at `http://localhost:8080`.

### Using Docker Compose

Start all services (database, migrations, importer, evaluator, API):

```bash
task compose:up
```

Stop all services:

```bash
task compose:down
```

## API Endpoints

### `GET /api/items/weapons`
Returns a list of all weapons in the database.

### `GET /api/items/weapons/:item_id/calculate`
Returns the pre-computed optimal build for a weapon.

**Query Parameters:**
- `build_type` - Type of optimization (currently only `recoil` is supported)
- `jaeger_level`, `prapor_level`, `skier_level`, `peacekeeper_level`, `mechanic_level` - Trader levels (1-4, defaults to 4)

**Example:**
```bash
curl "http://localhost:8080/api/items/weapons/5447a9cd4bdc2dbd208b4567/calculate?build_type=recoil&prapor_level=2&mechanic_level=3"
```

## Development

### Running Tests

All tests:
```bash
task test
```

Unit tests only (no database required):
```bash
task test:unit
```

Integration tests (requires database):
```bash
task test:integration
```

### Database Migrations

Apply migrations:
```bash
task migrate:up
```

Rollback migrations:
```bash
task migrate:down
```

Create a new migration:
```bash
task migrate:create -- migration_name
```

### Updating tarkov.dev Schema

The GraphQL queries are defined in `internal/tarkovdev/schemas/queries.graphql`. The client code is auto-generated using:
- `graphql-inspector` - Introspects the tarkov.dev API to generate `schema.graphql`
- `genqlient` - Generates Go functions and types from the schema and queries

Update both schema and generated code:
```bash
task tarkovdev
```

Only fetch the latest schema:
```bash
task tarkovdev:get-schema
```

Only regenerate Go code from existing schema:
```bash
task tarkovdev:regenerate
```

### Available Tasks

View all available tasks:
```bash
task --list-all
```

## Project Structure

```
.
├── cmd/                    # Entry points for binaries
│   ├── api/               # REST API server
│   ├── evaluator/         # Build optimization engine
│   ├── importer/          # Data import from tarkov.dev
│   └── migrations/        # Database migration runner
├── internal/
│   ├── candidate_tree/    # Core optimization algorithm
│   ├── evaluator/         # Build evaluation logic
│   ├── models/            # Database models
│   ├── router/            # API routes and handlers
│   ├── tarkovdev/         # GraphQL client for tarkov.dev
│   ├── db/                # Database connection utilities
│   ├── cache/             # Caching implementations
│   └── importers/         # Import logic for weapons/mods
├── migrations/            # Database migrations (goose)
├── docker/                # Dockerfiles for each service
└── file-caches/           # JSON caches for tarkov.dev data
```

## License

Mozilla Public License Version 2.0 - See [LICENSE](LICENSE) for details.

## External Resources

- [tarkov.dev API Playground](https://api.tarkov.dev) - Source of weapon and modification data
- [tarkov.dev](https://tarkov.dev) - Community-maintained Escape from Tarkov database
- [genqlient](https://github.com/Khan/genqlient) - A type-safe GraphQL client for Go

