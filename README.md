# Reloquent

**Automate offline migrations from Oracle/PostgreSQL to MongoDB using Apache Spark.**

[![License](https://img.shields.io/badge/License-BSD_3--Clause-blue.svg)](LICENSE)

Reloquent is an open source migration tool that guides you through every step of moving data from relational databases (Oracle, PostgreSQL) to MongoDB. It generates PySpark scripts targeting the MongoDB Spark Connector and runs them on AWS EMR or Glue, handling schema discovery, denormalization design, cost estimation, provisioning, migration execution, and post-migration validation.

<!-- TODO: Add screenshot -->

## Feature Highlights

- **12-step interactive wizard** available as a terminal UI (bubbletea) and a full web UI
- **Visual schema designer** with drag-and-drop denormalization of FK relationships
- **PySpark code generation** targeting the MongoDB Spark Connector with optimized bulk writes (`w:1`, `j:false`, unordered, max batch size, zstd compression)
- **16MB BSON document limit detection** during the design phase, before migration begins
- **AWS EMR and Glue support** for Spark execution with provisioning automation
- **Cost estimation and sizing recommendations** based on source data volume and cluster configuration
- **Post-migration validation** including row counts, sample document checks, and aggregate comparisons
- **Oracle JDBC driver detection and guidance** since the driver cannot be bundled
- **YAML configuration** with secret resolution from environment variables, HashiCorp Vault, and AWS Secrets Manager
- **Three interfaces, one engine** ensuring CLI wizard, CLI subcommands, and web UI all share the same core logic

## Quick Start

### Install

```bash
# Homebrew
brew install reloquent/tap/reloquent

# Go install
go install github.com/reloquent/reloquent@latest

# Docker
docker pull ghcr.io/reloquent/reloquent:latest
```

### Initialize a Project

```bash
reloquent init
```

This creates a `reloquent.yaml` configuration file in the current directory with guided prompts for source and target database connections.

### Run the Wizard

```bash
reloquent
```

Launches the 12-step interactive wizard in your terminal. The wizard walks you through discovery, table selection, schema design, estimation, code generation, provisioning, migration, and validation.

### Launch the Web UI

```bash
reloquent serve
```

Opens the full web-based wizard at `http://localhost:8080` with the visual schema designer for drag-and-drop denormalization.

## Trial Mode

Try Reloquent without any external databases using the included Docker Compose trial environment. It starts a PostgreSQL instance loaded with the Pagila sample dataset and a MongoDB target.

```bash
docker compose -f trial/docker-compose.yml up --build
```

## CLI Reference

| Command | Description |
|---|---|
| `reloquent` | Launch the interactive 12-step wizard |
| `reloquent init` | Initialize a new project configuration file |
| `reloquent discover` | Connect to the source database and discover schema metadata |
| `reloquent select` | Choose tables and columns to include in the migration |
| `reloquent design` | Design the target MongoDB document schema with denormalization |
| `reloquent estimate` | Estimate data volumes, BSON sizes, cluster sizing, and costs |
| `reloquent generate` | Generate PySpark migration scripts |
| `reloquent provision` | Provision AWS EMR or Glue resources |
| `reloquent prepare` | Prepare the target MongoDB environment (databases, collections) |
| `reloquent migrate` | Execute the migration by submitting Spark jobs |
| `reloquent validate` | Run post-migration validation (row counts, samples, aggregates) |
| `reloquent indexes` | Infer and build MongoDB indexes based on source schema and queries |
| `reloquent rollback` | Roll back a migration by dropping target collections |
| `reloquent status` | Show the current state of the migration pipeline |
| `reloquent config` | View or modify the project configuration |
| `reloquent serve` | Start the web UI server |

## Architecture Overview

Reloquent is built around a shared core engine that powers all three interfaces:

```
+------------------+     +------------------+     +------------------+
|   CLI Wizard     |     | CLI Subcommands  |     |     Web UI       |
|  (bubbletea)     |     |    (cobra)       |     |  (React + TS)    |
+--------+---------+     +--------+---------+     +--------+---------+
         |                        |                        |
         +------------------------+------------------------+
                                  |
                      +-----------+-----------+
                      |     Core Engine       |
                      |   (internal/ pkgs)    |
                      +-----------+-----------+
                                  |
              +-------------------+-------------------+
              |                   |                   |
     +--------+-------+  +-------+--------+  +-------+--------+
     | Source Drivers  |  | Spark Codegen  |  | AWS Providers  |
     | (Oracle, PG)   |  | (PySpark)      |  | (EMR, Glue)    |
     +----------------+  +----------------+  +----------------+
```

- **CLI Wizard:** A terminal-based interactive experience using bubbletea and lipgloss. State persists to `~/.reloquent/state.yaml` and is shared across all interfaces.
- **CLI Subcommands:** Scriptable commands for CI/CD pipelines and automation. Each command maps to a phase of the migration pipeline.
- **Web UI:** A React application (TypeScript, Tailwind CSS) embedded in the Go binary via the `embed` package. Uses @xyflow/react for the visual schema designer. Communicates via REST API and WebSocket for live progress updates.

## Configuration Reference

Reloquent uses a YAML configuration file (`reloquent.yaml`) with schema versioning.

```yaml
version: 1

source:
  type: postgresql          # postgresql or oracle
  host: localhost
  port: 5432
  database: pagila
  username: migrator
  password: ${env:SOURCE_DB_PASSWORD}    # secret resolution
  schema: public

target:
  uri: ${vault:secret/data/mongo#uri}    # HashiCorp Vault
  database: pagila_mongo

aws:
  region: us-east-1
  emr:
    cluster_id: ${aws-sm:reloquent/emr-cluster-id}  # AWS Secrets Manager
    instance_type: m5.xlarge
    instance_count: 4
  s3:
    bucket: reloquent-migrations
    prefix: pagila
```

### Secret Resolution Patterns

| Pattern | Source | Example |
|---|---|---|
| `${env:VAR_NAME}` | Environment variable | `${env:DB_PASSWORD}` |
| `${vault:path#key}` | HashiCorp Vault | `${vault:secret/data/mongo#uri}` |
| `${aws-sm:secret-name}` | AWS Secrets Manager | `${aws-sm:reloquent/db-pass}` |

## Development Setup

### Prerequisites

- Go 1.25 or later
- Node.js 20 or later (for web UI development)
- Docker and Docker Compose (for integration tests and trial mode)

### Build

```bash
git clone https://github.com/reloquent/reloquent.git
cd reloquent
make build
```

### Test

```bash
# Unit tests
make test

# Integration tests (requires Docker)
make test-integration
```

### Run Locally

```bash
# CLI
go run ./cmd/reloquent

# Web UI development server
cd web && npm install && npm run dev
```

## License

Reloquent is licensed under the [BSD 3-Clause License](LICENSE).
