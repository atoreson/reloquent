# CLAUDE.md

## Project Overview

This is **Reloquent** — an open source (BSD 3-Clause) migration tool that automates offline migrations from relational databases (Oracle, PostgreSQL) to MongoDB using Apache Spark (AWS EMR / Glue).

**Read PLAN.md before starting any work.** It contains the full architecture, migration pipeline (9 phases), CLI design (interactive wizard + web UI + subcommands), development phases with checklists, testing strategy, and all resolved technical decisions.

## Development Approach

- Work through **Phase 1: Foundation (MVP)** checklist items first, in order.
- Do not move to Phase 2 until all Phase 1 items are complete and tested.
- Ask before moving to the next phase.
- Write tests alongside each feature, not after. Each phase in PLAN.md specifies its testing requirements.
- Follow the repository structure defined in PLAN.md under "License & Distribution."

## Tech Stack

- **Backend + CLI:** Go (cobra for CLI, bubbletea + lipgloss for terminal UI)
- **Web UI:** React (embedded in Go binary via `embed` package), react-flow for schema designer
- **API:** REST (chi or net/http) + WebSocket for live updates
- **Generated output:** PySpark scripts targeting latest stable Spark + MongoDB Spark Connector
- **Config:** YAML with `version: 1` schema versioning
- **Distribution:** goreleaser, Homebrew tap, Docker image, `go install`

## Key Constraints

- All three interfaces (web UI, CLI wizard, CLI subcommands) must share the same core engine. No duplicated business logic.
- Wizard state persists to `~/.reloquent/state.yaml` and is shared across all interfaces.
- The web UI is a full wizard — not just a helper for the schema designer. Every step from connection through production readiness.
- MongoDB writes must be configured for maximum bulk throughput: `w:1, j:false`, unordered, max batch size, zstd compression.
- Oracle JDBC driver cannot be bundled. The tool must detect, guide, and handle this gracefully.
- JDBC reads from source DBs must always specify explicit partitioning (`partitionColumn`, `lowerBound`, `upperBound`, `numPartitions`). Never generate a single-partition read.
- 16MB BSON document limit must be estimated and surfaced during the denormalization design phase, not after migration.

## Code Style

- Go: Follow standard Go conventions (`gofmt`, `go vet`, `golint`). Use `internal/` for non-exported packages.
- React: Functional components with hooks. Tailwind for styling. TypeScript preferred.
- Tests: Table-driven tests in Go. React Testing Library for frontend. Integration tests use Docker Compose.
- Commit messages: Conventional Commits format (`feat:`, `fix:`, `test:`, `docs:`).

## File Locations

- `PLAN.md` — Full project plan (architecture, pipeline, CLI design, development phases, testing strategy, all decisions)
- `cmd/` — CLI entry points
- `internal/` — Core engine packages
- `web/` — React app source
- `docs/` — Documentation site source
- `test/` — Integration and E2E test fixtures
