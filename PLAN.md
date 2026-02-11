# Reloquent — Relational to MongoDB Migration Tool

## Project Vision

Reloquent is a migration tool that automates offline migrations from relational databases (Oracle, PostgreSQL) to MongoDB (Atlas, replica sets, sharded clusters). It uses Apache Spark (AWS Glue / EMR) as the compute engine to move terabyte-scale datasets with minimal downtime.

The tool handles the full lifecycle: schema discovery, interactive denormalization design, PySpark code generation, Spark cluster sizing, AWS infrastructure provisioning, migration execution, and post-migration validation.

It offers three interfaces — a full-featured web UI for GUI-first users, an interactive CLI wizard for terminal users, and individual CLI subcommands for power users and CI pipelines — all backed by the same core engine.

**Target audience:** Application developers and DBAs who need a user-friendly, GUI-driven experience. No Spark or MongoDB expertise required.

**Data scale:** 1 TB – 100 TB. Target migration window: under 1 hour.

---

## Architecture

### Tri-Mode Interface

```
┌──────────────────────────────────────────────────────────┐
│                   Go Binary (reloquent)                   │
│                                                           │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────┐  │
│  │ CLI        │  │ CLI        │  │ Web UI Server      │  │
│  │ Subcommands│  │ Wizard     │  │ (localhost:8230)   │  │
│  │ (CI/power) │  │ (terminal) │  │ (full GUI wizard)  │  │
│  └─────┬──────┘  └─────┬──────┘  └─────────┬──────────┘  │
│        │               │                    │             │
│        └───────────────┼────────────────────┘             │
│                        ▼                                  │
│              ┌──────────────────┐                         │
│              │   Core Engine    │                         │
│              └──────────────────┘                         │
└──────────────────────────────────────────────────────────┘
```

- **Web UI** (`reloquent serve`): Full migration wizard in the browser. Covers the entire workflow end-to-end: connect to source and target, explore schema, select tables, drag-and-drop denormalization designer, type mapping, cluster sizing, AWS provisioning, live migration progress dashboard, validation results, index build monitoring, and production readiness signal. This is the recommended experience for most users. All state lives on the Go backend; the browser is a thin client communicating via REST + WebSocket.
- **CLI Wizard** (`reloquent` with no subcommand): Full migration wizard in the terminal using bubbletea for interactive UI. Same end-to-end flow as the web UI, with the option to hand off to the browser for the visual denormalization designer.
- **CLI Subcommands** (`reloquent discover`, `reloquent migrate`, etc.): Individual commands for each phase. For power users, scripting, and CI pipelines.
- **Core Engine**: Shared Go library used by all three interfaces. Handles DB introspection, PySpark generation, AWS provisioning, config management, and state persistence.

All three interfaces share the same config file (`reloquent.yaml`), state file (`state.yaml`), and output artifacts. A user can start in the web UI, close it, and resume from the CLI wizard or vice versa.

### Technology Stack

| Component | Technology |
|---|---|
| CLI + Backend | Go |
| CLI Terminal UI | charmbracelet/bubbletea + lipgloss |
| Backend API | REST (Go net/http or chi router) + WebSocket for live updates |
| Web UI | React (served by Go binary, embedded via `embed` package) |
| Schema Designer | React drag-and-drop (react-flow or similar) |
| PySpark Generation | Go templates producing `.py` files |
| Source DBs | Oracle, PostgreSQL (via JDBC in Spark) |
| Target DB | MongoDB Atlas, replica sets, sharded clusters |
| Compute | AWS Glue, AWS EMR |
| Config Format | YAML |
| Secrets | YAML file, env vars, AWS Secrets Manager, HashiCorp Vault |

#### Version Pinning Strategy

Reloquent targets the **latest stable versions** of all core dependencies:

- **Apache Spark:** Latest stable release. The generated PySpark scripts and EMR/Glue configurations are pinned to a specific Spark major.minor version at build time.
- **MongoDB:** Latest stable server release supported by the Spark Connector.
- **MongoDB Spark Connector:** Latest stable release. The connector version must be compatible with both the Spark version and the MongoDB server version. Reloquent validates this compatibility at generation time and embeds the correct connector JAR version in the EMR bootstrap script / Glue job configuration.
- **PostgreSQL JDBC Driver:** Latest stable release, bundled in the generated artifacts (uploaded to S3 for the Spark cluster).
- **Oracle JDBC Driver:** **Cannot be bundled or distributed** due to Oracle's licensing restrictions. See "Oracle JDBC Driver Handling" below.

When new versions of Spark, MongoDB, or the Spark Connector are released, Reloquent's version pins are updated and a new release is cut. The tool does not support arbitrary user-specified versions in v1 — it always generates code for the pinned versions.

#### Oracle JDBC Driver Handling

Oracle's JDBC driver (`ojdbc11.jar`) cannot be freely redistributed. The tool handles this gracefully:

1. During **Oracle source discovery** (Phase 1), the tool does not need the JDBC driver — it connects to Oracle via Go's `godror` driver.
2. During **PySpark code generation** (Phase 4), the tool checks if an Oracle JDBC JAR is available:
   - Looks in `~/.reloquent/drivers/` for `ojdbc*.jar`.
   - If not found, displays a clear message: "Oracle JDBC driver required for Spark to read from Oracle. Download `ojdbc11.jar` from oracle.com/database/technologies/appdev/jdbc-downloads.html and place it in `~/.reloquent/drivers/`."
   - The web UI includes a direct link and a file upload zone for the JAR.
3. During **AWS provisioning** (Phase 7), the tool uploads the Oracle JDBC JAR to S3 alongside the PySpark script, and configures the EMR bootstrap action or Glue job to include it on the classpath.

### License & Distribution

**License:** BSD 3-Clause. Open source from the first commit.

**Repository structure:**
```
reloquent/
├── LICENSE                 # BSD 3-Clause
├── README.md               # Project overview, quick start, screenshots
├── CONTRIBUTING.md          # Contributor guide (code style, PR process, DCO)
├── CHANGELOG.md            # Version history
├── Makefile                # Build, test, release targets
├── cmd/                    # CLI entry points
├── internal/               # Core engine packages
├── web/                    # React app source
├── docs/                   # Documentation site source (GitHub Pages)
├── scripts/                # Build and release automation
├── test/                   # Integration and E2E test fixtures
└── .github/
    └── workflows/          # CI: test, build, release, scheduled E2E
```

**Distribution channels:**

1. **GitHub Releases:** Prebuilt binaries for macOS (arm64, amd64), Linux (arm64, amd64), and Windows (amd64). Each release includes checksums and a changelog. Built via GitHub Actions on tag push using `goreleaser`.
2. **Homebrew:** Official tap (`brew install reloquent/tap/reloquent`) for macOS and Linux.
3. **Docker image:** `ghcr.io/reloquent/reloquent:latest` — useful for CI pipelines and environments where installing a binary isn't convenient. The image includes the Go binary and the embedded web UI.
4. **Go install:** `go install github.com/reloquent/reloquent@latest` for Go developers.

### Documentation

**Phase 1 (MVP):**
- `README.md`: Project overview, installation instructions (all 4 channels), quick start guide (run `reloquent` and follow the wizard), screenshots/GIFs of the wizard and web UI, license badge, contributing link.

**Phase 2 (Web UI):**
- Documentation site hosted on GitHub Pages (built with a static site generator — Docusaurus or MkDocs):
  - **Getting Started:** Installation, first migration walkthrough (Postgres → MongoDB), configuration reference.
  - **User Guide:** Detailed guide for each wizard step, with screenshots. Covers both web UI and CLI wizard.
  - **Schema Designer Guide:** How to use the drag-and-drop designer, transformation rules, handling complex schemas (self-refs, M2M, circular refs), understanding the document preview and 16MB warnings.
  - **Configuration Reference:** Full `reloquent.yaml` schema documentation, secrets resolution, environment variables.
  - **Type Mapping Reference:** Default type mappings for Oracle and Postgres, how to override.

**Phase 3 (AWS Integration):**
  - **AWS Setup Guide:** IAM permissions required (EMR, Glue, S3, STS), VPC/security group considerations, credential configuration.
  - **Sizing Guide:** How the sizing engine works, how to interpret recommendations, tuning for specific workloads.
  - **Sharding Guide:** When sharding is recommended, how shard keys are chosen, how to override recommendations.
  - **Oracle JDBC Guide:** Step-by-step instructions for downloading and configuring the Oracle JDBC driver.

**Phase 4 (Validation & Production):**
  - **Validation Guide:** How validation works, interpreting results, handling mismatches.
  - **Production Cutover Guide:** Checklist for going live — scaling down clusters, updating connection strings, monitoring.
  - **Rollback Guide:** How to use `reloquent rollback`, manual cleanup steps.
  - **Troubleshooting:** Common errors, connectivity issues, Spark job failures, MongoDB write errors.
  - **CLI Reference:** Auto-generated from cobra command definitions.
  - **FAQ**

---

## Migration Pipeline

### Phase 1: Source Schema Discovery

**Goal:** Understand the source database schema and data volumes without modifying anything.

#### 1a. Direct Connection (Read-Only)

The application connects to the source database in read-only mode and extracts:

- All table names, column names, data types, nullability, defaults
- Primary keys, unique constraints
- Foreign key relationships (including composite FKs)
- Indexes (type, columns, uniqueness)
- Table row counts and on-disk sizes (via `pg_total_relation_size` for Postgres, `DBA_SEGMENTS` for Oracle)
- Sequences and auto-increment configurations
- Check constraints and enums

**Connection is strictly read-only.** The application uses a read-only transaction / session and never issues DDL or DML.

#### 1b. Offline Discovery (Script Export Mode)

For users who cannot or do not want to connect a third-party tool to their production database:

1. The application generates a self-contained SQL script (or shell script wrapping `psql`/`sqlplus`) that:
   - Queries `information_schema` / Oracle data dictionary for schema metadata
   - Queries table sizes and row counts
   - Writes all output to a structured JSON or YAML file
2. The user runs this script in their own environment.
3. The user provides the output file back to Reloquent, which ingests it identically to a direct connection.

**Output format:** `source-schema.yaml` containing the full schema graph.

##### Offline Discovery UX (Web UI)

The offline flow is inherently multi-step and asynchronous (the user leaves, runs a script, and comes back). The web UI handles this gracefully:

- **Step 1 — Generate:** A dedicated "Offline Discovery" page with a dropdown for database type (Postgres / Oracle). The user clicks "Generate Script" and the script is downloaded or displayed with a copy button.
- **Step 2 — Instructions:** Clear, numbered instructions on how to run the script, including example commands for `psql` and `sqlplus`. A "What this script does" expandable section for security-conscious users.
- **Step 3 — Import:** A drag-and-drop file upload zone (or file picker) to import the output YAML. The page validates the file on upload and shows a success/error message immediately.
- **Wizard state:** The wizard saves its state at "waiting for import." If the user closes the browser and returns hours later, the web UI (or CLI wizard) picks up at the import step, not back at the beginning.

### Phase 2: Table Selection

The user selects which tables to migrate. The UI displays:

- Table name, row count, on-disk size
- Foreign key relationships (which tables reference which)
- A visual relationship graph (web UI) or a text-based dependency tree (CLI)

The tool warns if the user selects a table with foreign keys pointing to tables not in the selection (orphaned references).

#### Handling Large Schemas (100+ Tables)

A flat checkbox list breaks down at scale. Both the web UI and CLI support:

- **Text search / filter:** Type to filter tables by name in real-time.
- **Sort:** By name, row count, on-disk size, or number of foreign key relationships.
- **Filter by schema** (Postgres) or **owner** (Oracle) to scope the view.
- **Bulk select by pattern:** e.g., `order_*` selects all tables matching the glob. Web UI has a pattern input field; CLI accepts glob syntax.
- **Dependency-aware selection:** When a table is selected, the UI highlights "you probably also need these" based on foreign key relationships. One-click to add all dependencies.
- **Live size summary:** A running total at the bottom updates as selections change: "Selected: 22 tables, 4.2 TB total."
- **Group by prefix:** Tables with common prefixes (e.g., `order_items`, `order_history`, `order_notes`) are visually grouped in the web UI.

### Phase 3: Denormalization Design

**This is the core UX differentiator.**

The user defines how relational tables map to MongoDB collections and embedded documents.

#### Concepts

- **Root Collection**: A top-level MongoDB collection (e.g., `orders`).
- **Embedded Document**: A table whose rows are embedded as subdocuments or arrays inside a root collection (e.g., `order_items` embedded inside `orders`).
- **Referenced Collection**: A table that remains its own collection, linked by a field reference (not embedded).
- **Nesting Depth**: Supports deep nesting (e.g., `orders → order_items → product_details → supplier`).

#### Web UI: Visual Schema Designer

Core canvas interactions:
- Tables displayed as draggable cards showing columns and types
- Draw edges between tables to define embedding relationships
- Drag a child table onto a parent to embed it (with nesting depth)
- Configure per-relationship: embed as array, embed as single subdocument, or keep as reference
- **Undo/redo** (Ctrl+Z / Ctrl+Y) for all canvas operations — essential for iterative design
- Handle complex cases with explicit UI affordances:
  - **Self-referencing tables** (e.g., `employee.manager_id → employee.id`): option to embed N levels deep or flatten to reference
  - **Many-to-many join tables**: dissolve the join table and embed the relationship on one or both sides
  - **Circular references**: detect and warn; force the user to break the cycle by choosing a reference instead of embedding
  - **Composite foreign keys**: full support, displayed as grouped lines in the UI

##### Handling Large Schemas on the Canvas

With 200+ tables, a flat canvas becomes unusable. The schema designer supports:

- **Search / filter bar** at the top of the canvas to find tables by name.
- **Schema/prefix grouping:** Tables with common prefixes or schemas are clustered together. Clusters can be collapsed/expanded.
- **Minimap** in the corner for navigating a large canvas (standard react-flow feature).
- **Zoom levels:** Zoom out to see the full graph; zoom in to work on a specific cluster.
- **Hide unselected tables:** Toggle to show only the tables included in the migration, hiding everything else.
- **Focus mode:** Click a root collection to show only that collection and its related tables, dimming everything else.

##### Suggested Mappings (Auto-Denormalization)

Before the user starts dragging, the tool analyzes the foreign key graph and **proposes an initial denormalization mapping**:

- Tables with 1:1 relationships → suggest embedding as a single subdocument.
- Tables with 1:N relationships where the child has no other dependents → suggest embedding as an array.
- Many-to-many join tables → suggest dissolving and embedding on the "many" side with fewer rows.
- Tables with no foreign key relationships → suggest keeping as standalone collections.
- Self-referencing tables → suggest keeping as references (safe default).

The suggested mapping is displayed on the canvas with dashed lines (proposed embeddings) that the user can accept, modify, or reject. This dramatically reduces time-to-first-mapping for large schemas — the user *edits* a reasonable starting point instead of building from scratch.

##### Live Document Preview Panel

A **real-time preview panel** on the right side of the canvas shows what the resulting MongoDB document will look like as the user modifies the mapping:

- Pulls a sample row from the source DB (via the read-only connection or from cached discovery data) and renders it as a formatted JSON document.
- Updates instantly when the user adds/removes an embedding, changes nesting depth, renames a field, or applies a transformation.
- Shows the estimated document size in bytes.
- **16MB BSON limit warning:** If the estimated document size for any mapping approaches or exceeds 16MB (based on actual row counts and average row sizes from the source DB), the preview panel displays a prominent warning: "⚠ This embedding could produce documents exceeding MongoDB's 16MB limit. The `users` collection has rows with up to 500K related `audit_events`. Consider keeping `audit_events` as a referenced collection instead."
- The 16MB check uses the source DB's actual cardinality data (max child rows per parent) multiplied by average child row size, not just averages. It flags the worst-case scenario.
- For deeply nested documents, the preview shows the full nesting hierarchy with collapsible sections.

#### CLI: Interactive Prompts

For headless usage, the same decisions are made via an interactive prompt workflow or a pre-built mapping YAML file.

#### Transformation Rules

Beyond structural joins, the user can define per-field transformations:

- **Rename fields**: `order_date` → `orderDate`
- **Computed / derived fields**: e.g., `total = quantity * unit_price`
- **Type coercion**: override the default type mapping (e.g., force a `VARCHAR` to `NumberLong`)
- **Filters**: exclude rows matching a condition (e.g., `WHERE status != 'DELETED'`)
- **Default values**: fill nulls with a specified value
- **Field exclusion**: drop columns that shouldn't migrate

These are stored in the mapping config and translated into PySpark transformations in the generated code.

##### Transformation Rule Builder (Web UI)

Raw expression input (`quantity * unit_price`) is powerful but intimidating for the target audience. The web UI provides a **visual rule builder**:

- Click a field on any table card to open the transformation panel.
- Select an operation from a dropdown: Rename, Compute, Cast Type, Filter Rows, Set Default, Exclude Field.
- Contextual inputs change based on the operation:
  - **Rename:** Single text input for the new field name.
  - **Compute:** Dropdown to select source fields + operator dropdowns (+, -, ×, ÷, concat, etc.) + preview of the result for a sample row.
  - **Cast Type:** Dropdown of available BSON types with a preview of the conversion.
  - **Filter Rows:** Field selector + condition dropdown (equals, not equals, greater than, contains, is null, etc.) + value input.
  - **Set Default:** Value input with type inference.
  - **Exclude:** One-click toggle, no additional input needed.
- **Power user toggle:** A "Switch to expression mode" link reveals a raw text input for PySpark expressions, for users who know what they're doing.
- Transformations are shown as badges on the affected fields in the canvas cards, so the user can see at a glance which fields have rules applied.

#### Data Type Mapping

A configurable mapping layer translates source types to BSON types:

| Oracle | PostgreSQL | BSON (default) |
|---|---|---|
| `NUMBER(p,0)` | `INTEGER` / `BIGINT` | `NumberLong` |
| `NUMBER(p,s)` | `NUMERIC` / `DECIMAL` | `Decimal128` |
| `VARCHAR2` | `VARCHAR` / `TEXT` | `String` |
| `DATE` | `DATE` | `ISODate` |
| `TIMESTAMP` | `TIMESTAMP` | `ISODate` |
| `CLOB` | `TEXT` | `String` |
| `BLOB` | `BYTEA` | `BinData` |
| `RAW` | `UUID` | `String` or `UUID` (configurable) |
| — | `JSONB` | Parsed into BSON subdocument |
| — | `ARRAY` | BSON array |
| `SDO_GEOMETRY` | `GEOMETRY` (PostGIS) | GeoJSON subdocument |

Users can override any mapping in the config file or interactively. The tool generates a `type-mapping.yaml` during Phase 1 that users can review and edit before proceeding.

### Phase 4: PySpark Code Generation

The tool generates a self-contained PySpark script (`.py`) that:

1. **Reads source tables via JDBC with parallel partitioning.**
   - Auto-detects good partition columns: numeric primary keys first, then date/timestamp columns.
   - Computes `lowerBound` and `upperBound` from table statistics or a lightweight `MIN/MAX` query.
   - Sets `numPartitions` based on table size and target cluster capacity.
   - Generates explicit `.read.jdbc(...)` calls with `partitionColumn`, `lowerBound`, `upperBound`, `numPartitions`.
   - This is critical: without explicit JDBC partitioning, Spark reads the entire table through a single connection, which is the primary bottleneck.

2. **Applies transformations.**
   - Column renames, type casts, computed columns, row filters, null defaults.
   - Expressed as PySpark DataFrame operations (`.withColumnRenamed`, `.withColumn`, `.filter`, etc.).

3. **Performs joins and nesting.**
   - Joins child DataFrames to parent DataFrames.
   - Uses `collect_list` / `struct` to nest child rows as arrays of subdocuments.
   - For deep nesting, joins are applied bottom-up: deepest children first, then intermediate levels, then root.
   - Join strategy selection:
     - Small dimension tables: broadcast join (`broadcast()` hint).
     - Large fact tables: sort-merge join (Spark default).
     - The generated code includes comments explaining the join strategy.

4. **Writes to MongoDB (maximum throughput configuration).**
   - Uses the MongoDB Spark Connector (`mongodb-spark-connector`).
   - **Write concern during migration:** `w: 1, j: false` — no journal acknowledgment during bulk load. Durability is not critical since we can restart the migration. Switched to `w: majority` for post-migration validation queries.
   - **Unordered bulk inserts:** `ordered=false` — allows the driver to continue inserting on error and maximizes parallelism. Errors are collected and reported at the end.
   - **Maximum batch size:** Set `maxBatchSize` to the connector's maximum (typically 100,000 documents or 48MB per batch, whichever is hit first). Larger batches reduce round-trip overhead.
   - **Connection pooling:** Set `maxPoolSize` on the MongoDB connection string to match the number of Spark executor cores writing concurrently.
   - **Compression:** Enable `compressors=zstd` on the connection string to reduce network bandwidth between Spark and MongoDB.
   - **Sharded collections — pre-split chunks:** For sharded target collections, pre-split the chunk ranges before inserting so writes distribute evenly across shards from the start. This avoids the "hot shard" problem where all writes funnel through one shard until the balancer catches up.
   - **Disable the balancer** during migration (`sh.stopBalancer()`) if chunks are pre-split correctly. Re-enable post-migration.
   - **Avoid monotonic `_id` values:** If `_id` values would be sequential (e.g., auto-increment from the source), either use a hashed shard key or generate well-distributed `_id` values (e.g., UUIDs) to prevent write hotspotting on sharded clusters.

5. **Outputs a migration summary.**
   - Row counts per source table read.
   - Document counts per target collection written.
   - Elapsed time per stage.

#### Generated File Structure

```
output/
├── migration.py              # Main PySpark entry point
├── config/
│   ├── source-schema.yaml    # Discovered schema
│   ├── mapping.yaml          # Denormalization + transformation rules
│   ├── type-mapping.yaml     # Data type mapping overrides
│   └── credentials.yaml      # DB credentials (gitignored)
├── plans/
│   ├── sizing-plan.yaml      # Spark + MongoDB cluster sizing (migration + production)
│   └── sharding-plan.yaml    # Shard key, shard count, pre-split commands (if >3TB)
├── validation/
│   └── validate.py           # Post-migration validation script
├── reports/
│   ├── migration-report.json # Full migration report (generated post-migration)
│   └── migration-report.txt  # Human-readable summary
└── infrastructure/
    ├── emr-cluster.yaml      # EMR cluster config (or CloudFormation)
    └── glue-job.yaml         # Glue job config
```

### Phase 5: Cluster Sizing & Recommendations

#### Spark Cluster Sizing

**Default platform: AWS EMR.** EMR provides full control over cluster configuration, supports the full data range (1 TB – 100 TB), and has no practical job duration limits. AWS Glue is offered as a fallback for users who lack EMR permissions in their AWS environment.

**Platform selection logic:**
1. Verify EMR access by calling `sts:GetCallerIdentity` and simulating `emr:RunJobFlow` permissions via the IAM policy simulator API.
2. If EMR is available → use EMR (recommended).
3. If EMR is unavailable and data size ≤ 500 GB → offer Glue as a fallback.
4. If EMR is unavailable and data size > 500 GB → warn the user that Glue is unlikely to complete within the 1-hour window and recommend they pursue EMR access.

**Glue practical limits:** Glue caps at 299 DPUs, with a 48-hour max job timeout. Shuffle space and network throughput make it impractical beyond ~500 GB – 1 TB for a sub-1-hour migration window.

##### EMR Sizing (Default)

| Data Size | Worker Nodes | Instance Type | Estimated Cost |
|---|---|---|---|
| 1 TB – 5 TB | 10–20 nodes | r5.4xlarge | ~$50–200 |
| 5 TB – 20 TB | 20–50 nodes | r5.8xlarge | ~$100–500 |
| 20 TB – 50 TB | 50–100 nodes | r5.8xlarge | ~$200–1000 |
| 50 TB – 100 TB | 100–200 nodes | r5.12xlarge | ~$500–3000 |

##### Glue Sizing (Fallback)

| Data Size | DPU Count | Estimated Cost |
|---|---|---|
| < 50 GB | 10 DPU | ~$5–10 |
| 50 GB – 250 GB | 50–100 DPU | ~$25–100 |
| 250 GB – 500 GB | 150–299 DPU | ~$100–250 |
| > 500 GB | ⚠️ Not recommended | — |

The tool calculates sizing using:
- Total data size across selected tables
- Denormalization expansion factor (joins increase document size)
- Target migration time (< 1 hour)
- JDBC read parallelism (bounded by source DB max connections — default 20, max 50)
- MongoDB write throughput (bounded by target cluster tier)

**Key constraint:** The bottleneck is almost always source DB read speed, not Spark compute or MongoDB write speed.

#### Source DB Read Throughput Benchmark

Before committing to a cluster size and migration window estimate, the tool offers an optional benchmark:

1. Select a medium-sized table (or let the user choose).
2. Read a sample partition (e.g., 1% of rows) using the configured `numPartitions` and `max_connections`.
3. Measure MB/s throughput and extrapolate to the full dataset.
4. Report: estimated total read time, recommended parallelism adjustments, and whether the 1-hour window is achievable.
5. If the benchmark suggests the 1-hour window is not achievable, recommend increasing `max_connections` (if the source DB can handle it) or accepting a longer migration window.

This benchmark runs against the live source DB with a single lightweight query per partition. It does not write anything or lock tables.

#### MongoDB Target Cluster Sizing

The tool outputs a **full MongoDB sizing plan** covering both the migration phase and post-migration production use.

##### Migration-Phase Sizing (Oversize for Speed)

The target cluster should be oversized during migration to maximize write throughput, then scaled down after migration completes.

- **Storage:** Estimated document size × document count × 1.5 (overhead for indexes, oplog, and working set headroom during bulk writes).
- **Write throughput:** Total data size / target migration window = required MB/s. Map this to Atlas tier or self-hosted replica set specs.
- **RAM:** During migration, WiredTiger cache should be large enough to hold the working set of active index builds. Recommend instance types with at least 2× the total index size in RAM.
- **IOPS:** Bulk inserts are I/O intensive. Recommend provisioned IOPS or NVMe-backed instances during migration.
- **Atlas tier mapping:** Output specific Atlas tier recommendations (e.g., M40, M50, M60, M80) based on the above calculations.

##### Post-Migration Production Sizing (Scale Down)

After migration, validation, and index builds complete, the tool recommends a right-sized production cluster:

- **Storage:** Actual data size + indexes + 30% headroom for growth.
- **RAM:** Working set estimate based on index sizes and expected query patterns (inferred from source DB indexes).
- **Atlas tier mapping:** Recommend a smaller production-appropriate tier.
- **Output:** Clear instructions for the user to scale down their Atlas cluster via the Atlas UI.

##### Sharding Plan (Data > 3 TB)

For datasets exceeding 3 TB, the tool generates a sharding plan:

1. **Shard key recommendation:**
   - Analyze the root collection's document structure and the source DB's query patterns (inferred from indexes and foreign keys).
   - Prefer high-cardinality fields that distribute writes evenly.
   - Warn against monotonically increasing shard keys (timestamps, auto-incremented IDs) — recommend hashed shard keys in these cases.
   - If the user's documents have a natural partition field (e.g., `region`, `tenant_id`), recommend ranged sharding on that field.
   - For collections with no obvious shard key, recommend a hashed `_id` shard key.

2. **Shard count recommendation:**
   - Based on total data size, target ~1–2 TB per shard (balancing cost vs. performance).
   - Account for index overhead and working set.

3. **Pre-split chunk ranges:**
   - Calculate initial chunk boundaries based on the shard key distribution.
   - Generate `sh.splitAt()` commands to pre-split before migration begins.

4. **Output:** A `sharding-plan.yaml` file containing the shard key, shard count, and pre-split commands, plus a human-readable summary explaining the rationale.

##### Plain-Language Explanations for All Recommendations

Every sizing and sharding recommendation is accompanied by a brief, non-technical explanation of *why*. The target audience may not have MongoDB or Spark expertise. Examples:

- "We recommend **3 shards** because your data is 8 TB and each shard handles up to 3 TB comfortably. Fewer shards would mean each one stores too much data, slowing down queries."
- "We chose a **hashed `_id` shard key** for the `orders` collection because your primary key is auto-incremented. Without hashing, all new writes would go to a single shard (the one holding the highest key range), creating a bottleneck."
- "We recommend an **Atlas M60** during migration because bulk inserts require high IOPS and write throughput. After migration, you can scale down to an **M40**, which is sufficient for your estimated production workload."
- "The Spark cluster is sized at **30 × r5.4xlarge** to read 4.2 TB from your source database in parallel. Each node reads a portion of the data simultaneously, like 30 librarians copying books at once."

These explanations appear in the sizing plan output files, in the web UI alongside each recommendation, and in the CLI wizard's sizing step.

### Phase 5.5: Pre-Migration MongoDB Setup

Before the migration job runs, the tool prepares and validates the target MongoDB cluster.

#### Target Cluster Validation

The tool connects to the target MongoDB cluster (using the connection string from config) and auto-detects the topology via `db.hello()`. It then validates that the cluster matches the sizing recommendations:

- **Topology detection:** Automatically determine if the target is Atlas, a replica set, or a sharded cluster. No manual `topology` config needed — the tool infers it from the connection string (`mongodb+srv://` = Atlas) and server response.
- **Storage check:** Query available storage and compare against the estimated data size (including index overhead). Warn if the cluster doesn't have sufficient space for the migration. "⚠ Your cluster has 2 TB of available storage, but the estimated migration size is 5.8 TB. Please scale up your Atlas cluster before proceeding."
- **Tier/RAM check:** If Atlas, compare the detected tier against the recommended tier. "⚠ Your cluster is an M40, but we recommended M60 for this migration. The migration may take longer than estimated due to lower write throughput."
- **Shard check:** If sharding was recommended and the cluster is not sharded, warn. If sharded, verify the shard count matches the recommendation.
- **Connection validation:** Verify read and write permissions on the target database.

The validation results are displayed in the UI and the user can proceed (with warnings acknowledged) or go back to resize their cluster.

#### Pre-Migration Setup Steps

1. **Create target collections** with the correct names (derived from the denormalization mapping).
2. **If sharding is required (data > 3 TB):**
   - Enable sharding on the database: `sh.enableSharding("dbName")`.
   - Create the shard key index on each collection that will be sharded.
   - Shard each collection: `sh.shardCollection("dbName.collectionName", shardKey)`.
   - Pre-split chunks using the calculated chunk boundaries.
   - Disable the balancer: `sh.stopBalancer()`.
3. **If not sharding:** Create empty collections (MongoDB creates them implicitly on first insert, but explicit creation allows us to set options like collation if needed).
4. **Output the sizing plan** to the user (both migration-phase and production sizing), displayed in the UI and saved as `sizing-plan.yaml`.
5. **Prompt the user to confirm** before proceeding to migration.

### Phase 6: Index Planning

The tool generates a list of indexes to be created *after* data insertion completes (actual index creation happens in Phase 9):

1. **Inferred indexes** from source DB:
   - Source primary keys → unique index on corresponding document field
   - Source foreign keys that became references (not embedded) → index on the reference field
   - Source indexes on frequently queried columns → equivalent MongoDB indexes
   - Composite indexes preserved with correct field order

2. **Recommended indexes** based on document structure:
   - `_id` (always exists)
   - Fields used in the denormalization join conditions (likely query patterns)
   - Array fields that will be queried → multikey index recommendations

3. **Output:** A list of `createIndex` commands. The tool can execute these programmatically against the target cluster after data insertion is complete.

**Why post-insert:** Building indexes on an empty collection means every insert must update every index incrementally. For bulk loads, it is significantly faster to insert all data first, then build indexes as a single operation. MongoDB's index builder is optimized for this pattern. Exception: the `_id` index always exists, and shard key indexes must exist before writes to a sharded collection.

### Phase 7: AWS Infrastructure Provisioning

If the user opts in, the application can:

1. **Authenticate** using the user's existing AWS credentials (environment variables, `~/.aws/credentials`, IAM role, or SSO session).
2. **Verify platform access:**
   - Call `sts:GetCallerIdentity` to confirm valid AWS credentials.
   - Simulate `emr:RunJobFlow` permissions via the IAM policy simulator API to check EMR access.
   - If EMR is unavailable, check Glue permissions (`glue:CreateJob`, `glue:StartJobRun`) and offer Glue as a fallback (with data size warnings if > 500 GB).
   - Report findings clearly: "EMR access confirmed" or "EMR unavailable — Glue available as fallback" or "Neither EMR nor Glue permissions found."
3. **Provision the Spark cluster:**
   - **EMR (default):** Launch an EMR cluster with the recommended instance types and count, bootstrap actions for MongoDB Spark Connector, and the PySpark script uploaded to S3.
   - **Glue (fallback):** Create a Glue job with the generated PySpark script, configured DPU count, and appropriate IAM role.
4. **Upload artifacts** to S3: PySpark script, config files (credentials encrypted or pulled from Secrets Manager at runtime).
5. **Pre-flight connectivity check (mandatory):** After the Spark cluster is provisioned and before the migration job starts, the tool runs a lightweight connectivity verification from the Spark cluster:
   - Test JDBC connection from a Spark executor to the source database. Execute a simple `SELECT 1` or equivalent.
   - Test MongoDB connection from a Spark executor to the target cluster. Execute a `ping` command.
   - Verify that the MongoDB Spark Connector JAR is available on the cluster.
   - If either check fails, report the specific error (connection refused, timeout, authentication failure, security group/VPC issue) and **do not start the migration**. This saves the user from a 5-minute cluster spin-up followed by an immediate connection failure.
6. **Start the migration job** and poll for status.
7. **Report progress:** Job stage, records processed, elapsed time, estimated time remaining.
8. **Tear down** the Spark cluster after completion (Glue does this automatically; EMR cluster is terminated).

The provisioning logic uses AWS SDK for Go. All infrastructure is created with tags for cost tracking and cleanup.

### Phase 8: Migration Execution & Live Monitoring

#### Live Status Dashboard (Web UI + CLI)

During the migration, the UI provides real-time status updates:

**Web UI dashboard displays:**
- Overall migration progress (percentage complete, documents inserted / total estimated)
- Per-collection progress bars (source reads, Spark processing, MongoDB writes)
- Current throughput (MB/s read from source, documents/s written to MongoDB)
- Elapsed time and estimated time remaining
- Spark job stage visualization (which stage is active, shuffle stats)
- Error count and last error details (if any)
- Source DB connection utilization (active connections / max)

**CLI outputs:**
- Periodic progress lines (every 10 seconds) with the same metrics
- Final summary on completion

The Go backend polls the Spark job status via the EMR/Glue API and the MongoDB Spark Connector's write metrics. The web UI uses WebSocket for real-time updates.

#### Browser Disconnection Handling

Migration takes 30–60 minutes. Users will close tabs, laptops will sleep, WiFi will drop. The system handles this gracefully:

- **The Go backend is the source of truth.** Migration state is tracked server-side, not in the browser. Closing the browser does not stop the migration.
- **WebSocket reconnection:** When the browser reconnects (tab reopened, WiFi restored), the React app automatically reconnects the WebSocket and re-syncs the full migration state. No data loss, no stale UI.
- **Browser notifications:** With user permission, the web UI sends a browser notification when the migration completes (or fails), even if the tab is in the background.
- **CLI resume:** If the user started in the web UI but closes the browser, running `reloquent` or `reloquent status` in the terminal shows the current migration state.
- **`reloquent serve` keeps running:** The Go process manages the migration lifecycle independently of the browser. The user can close and reopen `localhost:8230` at any time.

#### Partial Failure Handling

If some collections succeed and others fail during migration:

1. The tool detects per-collection success/failure status from the Spark job output.
2. The UI presents the results clearly: "✓ 5 of 6 collections migrated successfully. ✗ `order_items` failed: MongoDB write timeout after 3 retries."
3. **The user is asked what to do:**
   - **Retry failed only:** Re-run the Spark job for only the failed collection(s). The tool drops the partially-written data for the failed collection first, then re-runs.
   - **Restart everything:** Drop all target collections and re-run the entire migration from scratch.
   - **Abort:** Stop here. Leave successful collections in place. The user can investigate and manually retry later via `reloquent migrate --collection order_items`.
4. This decision is presented in the web UI as a dialog and in the CLI wizard as an interactive prompt.

### Phase 9: Post-Migration Validation & Index Builds

After all data has been inserted:

#### Step 1: Validation

1. **Row count validation:** Compare source table row counts against target collection document counts (accounting for denormalization — e.g., 1000 orders with 5000 order_items should produce 1000 documents, not 5000).

2. **Statistical sample validation:**
   - Select N random documents from the target collection (configurable, default 1000).
   - For each sampled document, reconstruct what the document *should* look like by running the equivalent SQL joins on the source database.
   - Diff the reconstructed document against the actual MongoDB document.
   - Report: match percentage, list of mismatches with details.

3. **Aggregate validation:**
   - Run aggregate queries on both source and target (e.g., `SUM(amount)`, `COUNT(DISTINCT customer_id)`) and compare results.

4. **UI displays validation results** as each check completes:
   - Green checkmark for passed checks
   - Red alert with details for failed checks
   - Overall validation status: PASS / FAIL / PARTIAL

#### Step 2: Index Builds

After validation passes (or the user chooses to proceed despite warnings):

1. Execute all `createIndex` commands against the target collections.
2. **UI displays index build progress:**
   - Per-index status: queued → building → complete
   - Build progress percentage (MongoDB 4.2+ reports this via `currentOp`)
   - Estimated time remaining per index
3. If sharding was used: re-enable the balancer (`sh.startBalancer()`) after index builds complete.
4. Restore write concern to production settings (`w: majority, j: true`).

#### Step 3: Production Readiness Signal

When all of the following are true, the UI displays a clear **"READY FOR PRODUCTION"** signal:

- All validation checks passed
- All indexes built successfully
- Balancer re-enabled (if sharded)
- Write concern restored to production settings

The signal includes:
- Summary of migration: total documents, total data size, elapsed time
- Recommended next steps: scale down the Atlas cluster to production tier, update application connection strings, resume application writes
- A `migration-report.json` file saved to the output directory with full details

If any step failed, the UI displays a **"REQUIRES ATTENTION"** status with specific remediation steps.

#### Validation Report

4. **Output:** A validation report (JSON + human-readable summary) with pass/fail status and details on any discrepancies. Saved as `migration-report.json` and `migration-report.txt`.

---

## Configuration & Secrets

### Config File: `reloquent.yaml`

Default location: `~/.reloquent/reloquent.yaml` (overridable via `--config` flag or `RELOQUENT_CONFIG` env var).

```yaml
version: 1  # Config schema version (for future migration)

source:
  type: postgresql  # or oracle
  host: source-db.example.com
  port: 5432
  database: myapp
  schema: public
  username: readonly_user
  password: "${VAULT:secret/data/reloquent/source#password}"  # Vault reference
  # OR
  # password: "${ENV:SOURCE_DB_PASSWORD}"  # Env var reference
  # OR
  # password: "plaintext-password"  # Direct (not recommended)
  ssl: true
  read_only: true
  max_connections: 20  # JDBC read parallelism during migration (default: 20, max: 50)
  # Discovery phase uses a single connection regardless of this setting

target:
  type: mongodb
  connection_string: "${AWS_SM:reloquent/mongo-uri}"  # AWS Secrets Manager
  # OR
  # connection_string: "mongodb+srv://user:pass@cluster.mongodb.net/mydb"
  database: myapp
  # Topology (atlas, replica_set, sharded) is auto-detected from connection string + db.hello()
  # No manual topology configuration needed

aws:
  region: us-east-1
  profile: default  # AWS CLI profile
  platform: emr  # emr (default) | glue (fallback if EMR unavailable)
  s3_bucket: reloquent-migrations
  tags:
    project: reloquent
    environment: migration

logging:
  level: info  # debug | info | warn | error
  directory: ~/.reloquent/logs/  # Plain text log files
  # Log files are rotated daily: reloquent-2026-02-11.log
  # Retained for 30 days by default
  retention_days: 30
```

### Secrets Resolution Order

1. HashiCorp Vault (`${VAULT:path#key}`)
2. AWS Secrets Manager (`${AWS_SM:secret-name}`)
3. Environment variable (`${ENV:VAR_NAME}`)
4. Direct value in YAML (with warning on startup: "Credentials stored in plaintext")

The tool generates a `.gitignore` entry for `reloquent.yaml` and warns if it detects the file in a git repository.

---

## CLI Design: Interactive Wizard + Subcommands

### Primary Experience: Interactive Wizard

Running `reloquent` with no subcommand launches a guided, interactive wizard that walks the user through the entire migration process step by step. The wizard uses terminal UI elements (powered by a library like `bubbletea` or `charmbracelet/huh`) for a polished interactive experience: selectable lists, confirmation prompts, progress spinners, and formatted tables.

#### Wizard Flow

```
$ reloquent

  ╔══════════════════════════════════════╗
  ║   Welcome to Reloquent              ║
  ║   Relational → Spark → MongoDB      ║
  ╚══════════════════════════════════════╝

  No config file found. Let's set one up.

  ┌─ Step 1: Source Database ─────────────────────────────┐
  │                                                        │
  │  Database type:  ● PostgreSQL  ○ Oracle                │
  │  Host: ________________________________________        │
  │  Port: [5432]                                          │
  │  Database: ____________________________________        │
  │  Username: ____________________________________        │
  │  Password: ************************************        │
  │  Max connections for migration (1-50): [20]            │
  │                                                        │
  │  [ Test Connection ]    [ Use Offline Script Instead ] │
  └────────────────────────────────────────────────────────┘

  ✓ Connected to PostgreSQL 15.4 on source-db.example.com
  ✓ Found 47 tables, 312 columns, 89 foreign keys
  ✓ Total data size: 4.2 TB

  ┌─ Step 2: Target MongoDB ──────────────────────────────┐
  │                                                        │
  │  Connection string: mongodb+srv://...                  │
  │  Database: ____________________________________        │
  │  Topology:  ● Atlas  ○ Replica Set  ○ Sharded          │
  │                                                        │
  │  [ Test Connection ]                                   │
  └────────────────────────────────────────────────────────┘

  ✓ Connected to MongoDB Atlas M50 cluster (3-node replica set)

  ┌─ Step 3: Select Tables ───────────────────────────────┐
  │                                                        │
  │  [x] orders          1.2M rows    180 GB               │
  │  [x] order_items     8.4M rows    1.2 TB               │
  │  [x] customers       340K rows    12 GB                 │
  │  [x] products        45K rows     2.1 GB               │
  │  [ ] audit_log       94M rows     2.8 TB  ← excluded   │
  │  [ ] session_data    12M rows     89 GB   ← excluded   │
  │  ...                                                    │
  │                                                        │
  │  Selected: 22 tables, 4.2 TB total                     │
  │  ⚠ Warning: order_items references products            │
  │    (selected — OK)                                      │
  │                                                        │
  │  [ Continue ]  [ Select All ]  [ Deselect All ]        │
  └────────────────────────────────────────────────────────┘

  ┌─ Step 4: Denormalization Design ──────────────────────┐
  │                                                        │
  │  Would you like to:                                    │
  │    ● Open the visual designer (launches web browser)   │
  │    ○ Use CLI-guided prompts                            │
  │    ○ Import a mapping file                             │
  │                                                        │
  └────────────────────────────────────────────────────────┘

  → Launching web designer at http://localhost:8230/design
    (Waiting for you to save your mapping in the browser...)

  ✓ Mapping saved: 22 tables → 6 collections
    • orders (with embedded order_items, shipping_details)
    • customers (with embedded addresses)
    • products (with embedded categories)
    • ...

  ┌─ Step 5: Type Mapping Review ─────────────────────────┐
  │                                                        │
  │  Default type mappings applied. Review?                │
  │    ● Accept defaults                                   │
  │    ○ Review and edit                                   │
  │    ○ Open editor in browser                            │
  │                                                        │
  └────────────────────────────────────────────────────────┘

  ┌─ Step 6: Cluster Sizing ──────────────────────────────┐
  │                                                        │
  │  Data size: 4.2 TB (estimated 5.8 TB after joins)     │
  │                                                        │
  │  Run source DB read benchmark? (optional, ~2 min)      │
  │    ● Yes, benchmark now                                │
  │    ○ Skip                                              │
  │                                                        │
  │  Benchmarking... reading sample from orders table      │
  │  ████████████████████████████░░  82%  ETA: 24s         │
  │                                                        │
  │  Benchmark results:                                    │
  │    Read throughput: 420 MB/s at 20 connections          │
  │    Estimated full read time: 38 minutes                │
  │    ✓ 1-hour migration window is achievable              │
  │                                                        │
  │  ── Spark Cluster (EMR) ──                             │
  │  Recommended: 30x r5.4xlarge ($180–240 estimated)      │
  │                                                        │
  │  ── MongoDB (Migration Phase) ──                       │
  │  Recommended: Atlas M60 (scale down to M40 after)      │
  │                                                        │
  │  ── Sharding ──                                        │
  │  Data exceeds 3 TB. Sharding recommended.              │
  │  Shard key: hashed _id (orders), region (customers)    │
  │  Shard count: 3 shards                                 │
  │                                                        │
  │  Full sizing plan saved to output/plans/               │
  │                                                        │
  │  [ Continue ]  [ Adjust Settings ]                     │
  └────────────────────────────────────────────────────────┘

  ┌─ Step 7: AWS Setup ───────────────────────────────────┐
  │                                                        │
  │  Using AWS profile: default (us-east-1)                │
  │  ✓ EMR access confirmed                                │
  │                                                        │
  │  How would you like to proceed?                        │
  │    ● Auto-provision EMR cluster and run migration      │
  │    ○ Generate scripts only (I'll provision manually)   │
  │                                                        │
  └────────────────────────────────────────────────────────┘

  ┌─ Step 8: Pre-Migration MongoDB Setup ─────────────────┐
  │                                                        │
  │  Validating target cluster...                          │
  │  ✓ Connected to MongoDB Atlas M60 (3-shard cluster)    │
  │  ✓ Available storage: 8.2 TB (need 5.8 TB) — OK       │
  │  ✓ Topology matches sharding recommendation            │
  │                                                        │
  │  Creating collections and sharding config...           │
  │  ✓ Created collection: orders                          │
  │  ✓ Created collection: customers                       │
  │  ✓ Sharded orders on { _id: "hashed" }                │
  │  ✓ Pre-split 12 chunks across 3 shards                │
  │  ✓ Balancer disabled for migration                     │
  │                                                        │
  └────────────────────────────────────────────────────────┘

  ┌─ Step 8b: Review Migration Plan ──────────────────────┐
  │                                                        │
  │  Summary of what will happen:                          │
  │  • Read 22 tables (4.2 TB) from PostgreSQL             │
  │  • Join and embed into 6 MongoDB collections           │
  │  • Write to Atlas M60 cluster (3 shards)               │
  │  • Estimated time: 38 minutes                          │
  │                                                        │
  │  ▸ View generated PySpark code (optional)  ────────┐  │
  │  │  # migration.py — annotated                     │  │
  │  │  # Read 'orders' table with 20 parallel          │  │
  │  │  # partitions on column 'order_id'               │  │
  │  │  orders_df = spark.read.jdbc(                    │  │
  │  │      url=jdbc_url,                               │  │
  │  │      table="orders",                             │  │
  │  │      ...                                         │  │
  │  └─────────────────────────────────────────────────┘  │
  │                                                        │
  │  ════════════════════════════════════════════════════   │
  │  ⚠ POINT OF NO RETURN                                 │
  │  After this step, data will be written to MongoDB.     │
  │  You can go back to change settings now.               │
  │  If you need to undo after starting, use               │
  │  'reloquent rollback' to drop target collections.      │
  │  ════════════════════════════════════════════════════   │
  │                                                        │
  │  [ Start Migration ]  [ Go Back ]                      │
  └────────────────────────────────────────────────────────┘

  ┌─ Step 9: Migration ───────────────────────────────────┐
  │                                                        │
  │  Pre-flight check...                                   │
  │  ✓ Source DB reachable from Spark cluster               │
  │  ✓ MongoDB reachable from Spark cluster                 │
  │  ✓ Spark Connector JAR available                        │
  │                                                        │
  │  Provisioning EMR cluster... (3-5 minutes)             │
  │  ████████████████████████████████  100%  ✓ Ready       │
  │                                                        │
  │  Migration in progress:                                │
  │                                                        │
  │  orders        ██████████████░░░░░░  68%   820K docs   │
  │  customers     ████████████████████  100%  340K docs ✓ │
  │  products      ████████████████████  100%  45K docs  ✓ │
  │  order_items   ████████████░░░░░░░░  58%   4.9M docs  │
  │                                                        │
  │  Throughput: 285 MB/s read │ 142K docs/s write         │
  │  Elapsed: 22m 14s │ Estimated remaining: 16m           │
  │                                                        │
  │  (If a collection fails:)                              │
  │  ┌──────────────────────────────────────────────────┐  │
  │  │  ✗ order_items failed: write timeout             │  │
  │  │  5 of 6 collections succeeded.                   │  │
  │  │                                                  │  │
  │  │  What would you like to do?                      │  │
  │  │    ○ Retry failed collection(s) only             │  │
  │  │    ○ Restart entire migration                    │  │
  │  │    ○ Abort (keep successful collections)         │  │
  │  └──────────────────────────────────────────────────┘  │
  └────────────────────────────────────────────────────────┘

  ✓ Migration complete. 6 collections, 12.4M documents.

  ┌─ Step 10: Validation ─────────────────────────────────┐
  │                                                        │
  │  ✓ Row counts match (6/6 collections)                  │
  │  ✓ Sample validation: 1000/1000 documents match        │
  │  ✓ Aggregate checks: 12/12 passed                      │
  │                                                        │
  │  Validation: PASSED                                    │
  └────────────────────────────────────────────────────────┘

  ┌─ Step 11: Index Builds ───────────────────────────────┐
  │                                                        │
  │  orders.customer_id_1     ████████████████████  ✓      │
  │  orders.order_date_1      ████████████████░░░░  78%    │
  │  customers.email_1        ████████████████████  ✓      │
  │  products.sku_1           ████████████████████  ✓      │
  │                                                        │
  │  Building 8 indexes... 6/8 complete                    │
  └────────────────────────────────────────────────────────┘

  ╔══════════════════════════════════════════════════════════╗
  ║  ✓ READY FOR PRODUCTION                                 ║
  ║                                                         ║
  ║  Migration:  12.4M documents, 5.8 TB, 38 minutes        ║
  ║  Validation: PASSED (row counts, samples, aggregates)   ║
  ║  Indexes:    8/8 built                                  ║
  ║  Balancer:   Re-enabled                                 ║
  ║  Write concern: Restored to w:majority                  ║
  ║                                                         ║
  ║  Next steps:                                            ║
  ║  1. Scale Atlas cluster down from M60 → M40             ║
  ║  2. Update application connection strings               ║
  ║  3. Resume application writes                           ║
  ║                                                         ║
  ║  Full report: output/reports/migration-report.json      ║
  ╚══════════════════════════════════════════════════════════╝
```

Key wizard behaviors (both CLI and Web UI):

- **Resume capability:** The wizard saves progress to `~/.reloquent/state.yaml` at each step. Both the CLI wizard and web UI read from the same state file. If the user exits and re-runs, either interface offers to resume from where they left off.
- **Cross-interface switching:** A user can start in the web UI, close the browser, and resume from the CLI wizard (or vice versa). State is shared.
- **Back navigation:** The user can go back to any previous step and change decisions — up until the point of no return.
- **Point of no return:** Step 8b explicitly warns the user that proceeding will write data to MongoDB. Before this point, back navigation is unlimited. After migration starts (Step 9+), the "back" button is disabled for Steps 1–8. If the user needs to change configuration after a partial or full migration, they must use `reloquent rollback` to clean up and start over.
- **PySpark code review (optional):** Step 8b includes a collapsible "View generated PySpark code" section with syntax highlighting and inline annotations explaining what each block does. Collapsed by default — power users can expand it, GUI-first users skip it entirely.
- **Web UI handoff (CLI only):** At Step 4 (Denormalization Design), the CLI wizard can launch the browser for the visual designer and wait for the user to save, then resume in the terminal. (The web UI handles this step natively.)

CLI wizard-specific:
- **Keyboard navigation:** Arrow keys, Enter, and Escape for all interactions. Type-ahead filtering for table selection.
- **Terminal UI library:** `charmbracelet/bubbletea` + `charmbracelet/lipgloss` for styled terminal rendering.

Web UI wizard-specific:
- **Full browser experience:** All steps rendered as React pages with step navigation sidebar. No terminal required.
- **Real-time updates:** Migration progress, validation results, and index builds use WebSocket for live updates without polling.
- **Visual schema designer:** Step 4 uses react-flow for drag-and-drop denormalization — the primary reason many users will choose the web UI.
- **Responsive:** Works on desktop browsers. Tablet support is a nice-to-have.

### Secondary: Individual Subcommands

For power users, CI pipelines, and re-running specific phases, all wizard steps are also available as standalone commands:

```
reloquent
├── (no subcommand)         # Launch interactive wizard (primary experience)
├── init                    # Create default config file interactively
├── discover                # Phase 1: Schema discovery
│   ├── --direct            # Connect to source DB directly
│   └── --script            # Generate offline discovery script
├── ingest                  # Ingest offline discovery output file
│   └── --file <path>
├── select                  # Phase 2: Interactive table selection
├── design                  # Phase 3: Denormalization
│   ├── --import <mapping.yaml>  # Use pre-built mapping file
│   ├── --export <mapping.yaml>  # Export current mapping
│   └── --web               # Launch browser-based designer
├── generate                # Phase 4: Generate PySpark script
│   └── --output <dir>
├── estimate                # Phase 5: Cluster sizing recommendation
│   └── --benchmark         # Run source DB read throughput benchmark
├── prepare                 # Phase 5.5: Pre-migration MongoDB setup
│   ├── --dry-run           # Show what would be created
│   └── --skip-shard        # Skip sharding setup even if recommended
├── provision               # Phase 7: Provision AWS infrastructure
│   ├── --dry-run           # Show what would be created
│   ├── --teardown          # Destroy provisioned resources
│   └── --check             # Verify AWS credentials and platform permissions
├── migrate                 # Run the full migration
│   ├── --skip-provision    # Use existing cluster
│   └── --collection <name> # Retry a specific failed collection only
├── validate                # Phase 9: Post-migration validation
│   ├── --samples <N>       # Number of documents to sample
│   └── --full              # Full row count + aggregate validation
├── indexes                 # Build indexes on target collections
│   ├── --dry-run           # Show indexes that would be created
│   └── --monitor           # Watch index build progress
├── status                  # Check migration readiness
├── rollback                # Drop target collections and clean up migration artifacts
│   ├── --collections <names>  # Drop specific collections only
│   └── --confirm           # Skip confirmation prompt (for scripting)
├── serve                   # Start the full web UI on localhost:8230
│   └── --port <port>
└── config
    ├── show                # Display current config (secrets masked)
    ├── validate            # Validate config file
    └── type-mapping        # Interactive type mapping editor
```

**Lock file:** When any interface (web UI, CLI wizard, or subcommand) starts a migration, it creates `~/.reloquent/reloquent.lock`. If another instance tries to start, it sees the lock and warns: "Another Reloquent instance is running (PID 12345). Only one migration can run at a time." The lock is released on completion, failure, or via `reloquent rollback`.

These subcommands read from the same config and state files the wizard uses. A user can run the wizard partway through, exit, then use subcommands to re-run or adjust specific steps.

---

## Development Phases

### Phase 1: Foundation (MVP)

- [ ] Go project scaffolding (CLI framework: cobra, terminal UI: bubbletea + lipgloss)
- [ ] REST API scaffolding (Go backend serving React frontend via `embed`)
- [ ] WebSocket endpoint for live updates
- [ ] React app scaffolding with wizard step navigation
- [ ] Interactive CLI wizard flow (Steps 1–3: source connection, target connection, table selection)
- [ ] Web UI wizard flow (Steps 1–3: same steps, browser-based forms)
- [ ] Wizard state persistence (`~/.reloquent/state.yaml`) with resume capability (shared across CLI and web UI)
- [ ] Config file loading with `version: 1` schema versioning, secrets resolution (YAML, env vars)
- [ ] Plain text logging to `~/.reloquent/logs/` with daily rotation and configurable log levels
- [ ] Lock file (`~/.reloquent/reloquent.lock`) for single-operator safety
- [ ] PostgreSQL schema discovery (direct connection)
- [ ] Schema output as YAML
- [ ] Table selection (CLI: interactive terminal UI with checkboxes and search; Web: filterable/sortable table with checkboxes, dependency highlighting, live size summary, bulk select by pattern)
- [ ] Basic denormalization mapping (CLI prompts, single-level embedding)
- [ ] PySpark code generation (JDBC reads with partitioning, basic joins, MongoDB writes with max throughput config)
- [ ] Row count validation
- [ ] **Distribution:** goreleaser configuration, GitHub Actions release workflow, Homebrew tap, Docker image, `go install` support
- [ ] **Docs:** README.md (overview, installation for all 4 channels, quick start, screenshots), CONTRIBUTING.md, CHANGELOG.md, LICENSE (BSD 3-Clause)
- [ ] **Testing:** Unit tests for config parsing (including version field), schema discovery, type mapping, PySpark code generation output correctness. REST API endpoint tests. React component tests for wizard steps 1–3.

### Phase 2: Full Denormalization & Web UI

- [ ] Oracle schema discovery support
- [ ] Offline discovery script generation (Postgres + Oracle)
- [ ] Offline discovery import UX (web UI: drag-and-drop file upload with instructions and "waiting for import" state)
- [ ] Deep nesting support in denormalization
- [ ] Complex schema handling (self-refs, M2M, circular ref detection, composite FKs)
- [ ] Transformation rules (rename, compute, filter, type coercion, field exclusion)
- [ ] CLI wizard Steps 4–5: denormalization design (CLI prompts) and type mapping review
- [ ] Web UI wizard Steps 4–5: drag-and-drop denormalization designer (react-flow), type mapping editor
- [ ] Schema designer at-scale UX: search/filter bar, schema/prefix grouping, minimap, zoom levels, hide unselected tables, focus mode
- [ ] Suggested mappings: auto-analyze foreign key graph and propose initial denormalization (1:1 → embed single, 1:N → embed array, M2M → dissolve, self-ref → reference)
- [ ] Live document preview panel: sample document from source DB, updates in real-time as mapping changes, shows estimated document size
- [ ] 16MB BSON limit detection: estimate worst-case document sizes using actual cardinality (max child rows per parent × avg row size), warn during design
- [ ] Undo/redo (Ctrl+Z / Ctrl+Y) for all canvas operations
- [ ] Visual transformation rule builder: operation dropdown, contextual inputs, sample row preview, power user expression mode toggle
- [ ] Transformation badges on affected fields in canvas cards
- [ ] CLI wizard web handoff: option to launch browser for visual designer from terminal, wait for save, resume wizard
- [ ] Oracle JDBC driver handling: detection, user guidance (download link, file upload in web UI), JAR placement in `~/.reloquent/drivers/`
- [ ] Data type mapping configuration (CLI + Web)
- [ ] **Docs:** Documentation site scaffolding (Docusaurus or MkDocs on GitHub Pages). Getting Started guide, User Guide (wizard walkthrough with screenshots for each step), Schema Designer Guide, Configuration Reference, Type Mapping Reference.
- [ ] **Testing:** Unit tests for deep nesting join generation, transformation rule application, circular ref detection, 16MB estimation logic, suggested mapping algorithm. Integration tests for Oracle + Postgres discovery against containerized DBs (Docker). Web UI component tests for schema designer interactions (drag, drop, edge creation, nesting configuration, undo/redo, document preview updates).

### Phase 3: AWS Integration & Sizing

- [ ] CLI wizard Steps 6–9: sizing, AWS setup, MongoDB preparation, code review (optional), point-of-no-return confirmation, migration execution with live terminal progress bars
- [ ] Web UI wizard Steps 6–9: sizing dashboard with plain-language explanations, AWS config panel, MongoDB preparation status with target cluster validation results, collapsible PySpark code review with annotations, point-of-no-return warning, live migration progress dashboard (WebSocket-powered real-time updates with per-collection progress, throughput graphs, error feed)
- [ ] Cluster sizing engine (Spark + MongoDB target, migration-phase and production sizing) with plain-language explanations for every recommendation
- [ ] Source DB read throughput benchmark
- [ ] MongoDB sharding plan generation (shard key recommendation, pre-split commands for data > 3 TB)
- [ ] Target cluster validation: auto-detect topology via `db.hello()`, verify storage capacity, check tier against recommendation, verify shard count
- [ ] Pre-migration MongoDB setup (create collections, shard if needed, disable balancer)
- [ ] AWS credential verification and platform access checks (EMR/Glue)
- [ ] AWS EMR cluster provisioning and execution (default)
- [ ] AWS Glue job provisioning and execution (fallback)
- [ ] S3 artifact upload (including Oracle JDBC JAR if Oracle source, with classpath configuration on EMR/Glue)
- [ ] MongoDB Spark Connector version pinning and compatibility validation at generation time
- [ ] Mandatory pre-flight connectivity check (source DB + MongoDB reachable from Spark cluster, Spark Connector JAR available)
- [ ] Partial failure handling: detect per-collection success/failure, prompt user (retry failed, restart all, or abort)
- [ ] Browser disconnection handling: server-side migration state, WebSocket reconnection with full state re-sync, browser notifications on completion
- [ ] `reloquent rollback` command: drop target collections, clean up migration artifacts, release lock file
- [ ] Infrastructure teardown
- [ ] **Docs:** AWS Setup Guide (IAM permissions, VPC/security groups, credentials), Sizing Guide, Sharding Guide, Oracle JDBC Guide.
- [ ] **Testing:** Unit tests for sizing calculations, shard key selection logic, target cluster validation logic. Integration tests for AWS provisioning (mock AWS SDK or use LocalStack), pre-flight connectivity check. End-to-end test with small dataset against real EMR + MongoDB Atlas. WebSocket integration tests for live progress updates and reconnection. Rollback command tests.

### Phase 4: Validation, Index Builds & Production Readiness

- [ ] CLI wizard Steps 10–11: validation results display, index build progress, production readiness signal (terminal UI)
- [ ] Web UI wizard Steps 10–11: validation results dashboard (real-time pass/fail indicators), index build progress bars (per-index via `currentOp`), production readiness banner with next-steps checklist
- [ ] Statistical sample validation (SQL reconstruction + MongoDB comparison)
- [ ] Aggregate validation
- [ ] Validation results UI (real-time display as checks complete, pass/fail indicators)
- [ ] Index inference and generation
- [ ] Post-insert index creation execution
- [ ] Index build progress monitoring (per-index status via `currentOp`)
- [ ] Balancer re-enablement after index builds (sharded clusters)
- [ ] Write concern restoration to production settings
- [ ] Production readiness signal ("READY FOR PRODUCTION" / "REQUIRES ATTENTION")
- [ ] Migration report generation (JSON + human-readable)
- [ ] HashiCorp Vault integration
- [ ] AWS Secrets Manager integration
- [ ] Error handling, retry logic, and user-friendly error messages
- [ ] **Docs:** Validation Guide, Production Cutover Guide (checklist), Rollback Guide, Troubleshooting (common errors, connectivity, Spark failures, MongoDB write errors), CLI Reference (auto-generated from cobra), FAQ.
- [ ] **Testing:** Integration tests for validation logic (pre-seeded source + target with known data, verify correct diff detection). Test index build monitoring against a real MongoDB instance. Test production readiness state machine transitions.

### Phase 5: Future Enhancements (Backlog)

- [ ] CDC-based live migration (Postgres logical replication → Spark Structured Streaming)
- [ ] Additional source databases (MySQL, SQL Server)
- [ ] Additional cloud platforms (GCP Dataproc, Azure HDInsight)
- [ ] MongoDB schema validation rule generation (JSON Schema)
- [ ] Migration dry-run mode (generate documents without writing to MongoDB)
- [ ] Cost estimation before provisioning
- [ ] Migration history and audit log
- [ ] Multi-database migration in a single run

---

## Testing Strategy

### Unit Tests

Every Go package has corresponding unit tests. Key areas:

- **Config parsing:** Valid YAML, invalid YAML, secrets resolution (env vars, Vault syntax, AWS SM syntax), missing required fields, version field migration.
- **Schema discovery:** Mock database responses → verify correct schema YAML output. Test all supported data types for both Postgres and Oracle.
- **Type mapping:** Default mappings, user overrides, edge cases (unknown types, spatial types, array types).
- **Denormalization logic:** Single-level embedding, deep nesting, self-references, M2M dissolution, circular reference detection and error reporting, composite FK handling.
- **Suggested mappings:** Given a foreign key graph, verify the auto-generated initial mapping is sensible (1:1 → embed single, 1:N → embed array, etc.).
- **16MB document size estimation:** Given cardinality data (max child rows per parent, avg row size), verify the estimator correctly flags mappings that would produce oversized documents.
- **PySpark code generation:** Verify generated Python code is syntactically valid. Verify JDBC partitioning parameters are correct. Verify join order (bottom-up). Verify transformation expressions. Verify MongoDB write configuration (unordered, batch size, write concern, compression).
- **Sizing engine:** Given known data sizes, verify correct EMR/Glue recommendations. Verify Glue fallback warnings for >500 GB. Verify MongoDB tier mapping. Verify shard count calculations. Verify plain-language explanation generation.
- **Shard key selection:** Given document structures and source indexes, verify shard key recommendations are sensible (high cardinality, no monotonic keys unless hashed).
- **Target cluster validation:** Given a `db.hello()` response and sizing recommendations, verify correct pass/warn/fail outcomes.
- **Lock file management:** Verify lock creation, detection, and release. Verify stale lock detection (PID no longer running).

### Integration Tests

Run against real (containerized) infrastructure:

- **Source DB discovery:** Docker containers running Postgres 15 and Oracle XE. Pre-loaded with test schemas covering all edge cases (self-refs, composite FKs, all data types, large table simulation).
- **PySpark execution:** Run generated PySpark scripts against a local Spark instance reading from containerized source DBs and writing to a containerized MongoDB replica set. Verify document structure and data correctness.
- **MongoDB operations:** Test collection creation, sharding commands, index builds, and balancer control against a containerized MongoDB sharded cluster.
- **AWS provisioning:** Use LocalStack for EMR/Glue/S3/IAM API mocking. Verify correct API calls, IAM permission checks, and teardown behavior.
- **Validation pipeline:** Pre-seed source and target with known data (including intentional mismatches). Verify the validation pipeline correctly identifies matches and mismatches.

### End-to-End Tests

Full migration runs with real infrastructure (run in CI on a schedule, not on every commit):

- **Small dataset (1 GB):** Postgres → EMR → MongoDB Atlas free tier. Verify complete pipeline from discovery through production readiness signal.
- **Medium dataset (50 GB):** Postgres → EMR → MongoDB Atlas M30. Verify performance characteristics and sizing accuracy.
- **Sharded dataset (5 TB simulated):** Verify sharding plan generation, pre-split, and balanced write distribution (can use synthetic data).

### Web UI Tests

- **Component tests:** React Testing Library for all components (connection forms, table selector with search/filter/sort/bulk-select, schema designer with minimap/grouping/focus mode, mapping editor, type mapping editor, transformation rule builder, document preview panel, sizing dashboard, progress dashboard, validation results, index build monitor, production readiness banner, partial failure dialog, offline discovery upload zone).
- **Wizard flow tests:** Verify complete step navigation (forward, back, resume from state file). Verify state persistence at each step. Verify point-of-no-return disables back navigation for completed irreversible steps. Verify cross-interface resume (start in web UI, resume in CLI, and vice versa).
- **Schema designer interaction tests:** Drag-and-drop produces correct mapping YAML. Edge creation sets correct embedding relationships. Nesting depth configuration works. Circular reference detection shows warnings. Undo/redo correctly reverts/reapplies all operations. 16MB document size warning appears when cardinality thresholds are exceeded.
- **Suggested mapping tests:** Verify auto-generated initial mapping renders correctly on canvas with dashed lines. Verify user can accept, modify, or reject individual suggestions.
- **Document preview tests:** Verify preview updates in real-time when mapping changes. Verify estimated size calculation is correct. Verify 16MB warning triggers at correct threshold. Verify deep nesting renders with collapsible sections.
- **Transformation rule builder tests:** Each operation type (rename, compute, cast, filter, default, exclude) produces correct config. Power user expression mode toggle works. Sample row preview renders correctly. Transformation badges appear on affected fields.
- **Offline discovery flow tests:** Script generation produces downloadable file. File upload zone accepts valid YAML. Invalid files show error messages immediately. Wizard resumes correctly from "waiting for import" state.
- **WebSocket tests:** Mock migration progress events and verify dashboard updates in real-time. Verify reconnection behavior on WebSocket disconnect (full state re-sync). Verify browser notification fires on completion/failure.
- **REST API tests:** All backend endpoints return correct responses for valid and invalid inputs. Error handling for connection failures, invalid configs, and edge cases.
- **Partial failure dialog tests:** Verify the retry/restart/abort dialog renders correctly and each option triggers the correct backend action.

### Test Infrastructure

- CI: GitHub Actions
- Containerized DBs: Docker Compose (Postgres, Oracle XE, MongoDB replica set, MongoDB sharded cluster)
- AWS mocking: LocalStack
- Spark local mode for integration tests
- Test fixtures: YAML schema files, mapping configs, and expected PySpark outputs for snapshot testing

---

## Key Technical Decisions

1. **Indexes are built post-insert** for bulk load performance. Shard key indexes are the exception and must exist before writes.
2. **JDBC read parallelism is explicit.** Generated PySpark code always specifies `partitionColumn`, `lowerBound`, `upperBound`, `numPartitions`. Default parallelism is 20 connections, configurable up to 50 max. Discovery phase always uses a single connection.
3. **Partial failures ask the user.** On per-collection failure, the user chooses: retry failed only, restart everything, or abort. No silent data loss.
4. **The web UI is served by the Go binary** using Go's `embed` package. No separate frontend deployment. The Go backend exposes a REST API + WebSocket endpoint consumed by the React frontend.
5. **All operations available via all three interfaces.** The web UI, CLI wizard, and CLI subcommands are equal peers backed by the same core engine. Users can switch between them mid-workflow.
6. **Interactive wizard is the primary CLI experience.** Running `reloquent` with no subcommand launches a guided wizard. Individual subcommands exist for power users and CI. Wizard state is persisted to disk for resume capability.
7. **PySpark is generated, not executed by the Go tool directly.** The Go tool produces `.py` files that run on Spark. This keeps the tool simple and the generated code inspectable/auditable. An optional annotated code review step is available for power users.
8. **MongoDB writes are tuned for maximum bulk throughput:** `w:1, j:false` during migration, unordered bulk inserts, max batch size, zstd compression, connection pooling matched to Spark executor cores. Production write concern restored post-migration.
9. **Sharding for data > 3 TB.** The tool generates a full sharding plan with shard key recommendations, pre-split chunk boundaries, and balancer management.
10. **Oversized migration cluster, right-sized production cluster.** The sizing plan explicitly covers both phases with specific Atlas tier recommendations for each.
11. **MongoDB topology is auto-detected** from the connection string and `db.hello()`. No manual topology configuration. The tool validates the target cluster matches sizing recommendations before migration.
12. **Mandatory pre-flight connectivity check** from the Spark cluster to both source DB and MongoDB before starting the migration job.
13. **Single-operator lock file** prevents concurrent migrations against the same target. `~/.reloquent/reloquent.lock` is created on migration start, released on completion or rollback.
14. **Config files are versioned** (`version: 1`) for future schema migration.
15. **Plain text logging** to `~/.reloquent/logs/` with daily rotation and configurable log levels.
16. **16MB BSON document limit is surfaced during design.** The schema designer estimates worst-case document sizes using actual cardinality data and warns the user before migration, not after.
17. **Testing at every layer.** Unit tests per package, integration tests against containerized DBs and LocalStack, end-to-end tests against real infrastructure on a schedule.
18. **Latest versions only.** Reloquent targets the latest stable Spark, MongoDB, and MongoDB Spark Connector releases. Version pins are updated with each Reloquent release. No user-configurable version overrides in v1.
19. **Oracle JDBC driver is not bundled.** Due to Oracle licensing, users must provide their own `ojdbc11.jar`. The tool detects, guides, and uploads it to S3 for the Spark cluster.
20. **Open source under BSD 3-Clause.** Public repository from the first commit.
21. **Cross-platform distribution** via GitHub Releases (goreleaser), Homebrew tap, Docker image, and `go install`.
22. **Documentation ships with each phase.** README in Phase 1, full docs site (GitHub Pages) starting Phase 2, expanded through Phases 3–4.

---

## Resolved Decisions

- **Project name:** Reloquent (relational + eloquent).
- **CLI design:** Three interfaces — (1) Web UI via `reloquent serve` is the full GUI wizard in the browser (recommended for most users), (2) CLI wizard via `reloquent` is the full terminal wizard, (3) CLI subcommands for power users and CI. All share the same core engine, config, and state. Users can switch between interfaces mid-workflow.
- **Source DB connection limits:** Default 20, max 50. Discovery uses a single connection. Migration phase uses configurable parallelism.
- **Source DB read throughput:** Optional benchmark test against a sample partition to estimate read speed and validate the 1-hour window.
- **Glue vs EMR:** EMR is the default. Glue is a fallback for users without EMR permissions, with a practical ceiling of ~500 GB for sub-1-hour migrations. The tool auto-detects platform availability via IAM permission checks.
- **Web UI framework:** React with react-flow for the schema designer.
- **MongoDB writes:** Maximum throughput configuration — `w:1, j:false` during migration, unordered bulk inserts, max batch size, zstd compression. Production settings restored post-migration.
- **MongoDB sizing:** Output a full sizing plan covering both migration (oversized) and production (right-sized) tiers. No Atlas Admin API integration — user provisions manually via Atlas UI.
- **MongoDB topology:** Auto-detected from connection string + `db.hello()`. No manual config. Target cluster validated against sizing recommendations (storage, tier, shard count) before migration.
- **Sharding:** For data > 3 TB, generate a sharding plan with shard key recommendations, pre-split commands, and balancer management. The tool creates and shards collections before migration.
- **Migration monitoring:** Real-time UI dashboard (WebSocket) with per-collection progress, throughput metrics, and error reporting. CLI gets periodic progress lines. Browser disconnection handled gracefully via server-side state + WebSocket reconnection.
- **Partial failure:** Ask the user — retry failed collection(s) only, restart everything, or abort.
- **Post-migration flow:** Validation → index builds → balancer re-enable → write concern restore → production readiness signal. All displayed in UI with clear pass/fail status.
- **Rollback:** `reloquent rollback` command drops target collections and cleans up migration artifacts.
- **Pre-flight check:** Mandatory connectivity verification from Spark cluster to both source DB and MongoDB before starting migration.
- **PySpark code review:** Optional collapsible step with syntax highlighting and annotations. Visible to power users, hidden by default.
- **Multi-user:** Single-operator lock file (`~/.reloquent/reloquent.lock`).
- **Config versioning:** `version: 1` field in `reloquent.yaml` for future schema migration.
- **Logging:** Plain text logs to `~/.reloquent/logs/` with daily rotation, configurable log levels, 30-day retention.
- **Testing:** Unit tests per package, integration tests against containerized DBs (Docker) and LocalStack, end-to-end tests against real infrastructure on a schedule.
- **License:** BSD 3-Clause, open source from the first commit.
- **Distribution:** GitHub Releases with goreleaser (macOS, Linux, Windows binaries), Homebrew tap, Docker image (`ghcr.io`), `go install`.
- **Version pinning:** Latest stable Spark + MongoDB + Spark Connector. Pins updated per Reloquent release. No user-configurable overrides.
- **Oracle JDBC:** Not bundled due to licensing. Tool detects missing JAR, guides user to download, uploads to S3 during provisioning.
- **Documentation:** README + CONTRIBUTING + CHANGELOG in Phase 1. Full docs site (GitHub Pages via Docusaurus/MkDocs) starting Phase 2, covering: Getting Started, User Guide, Schema Designer Guide, Configuration Reference, Type Mapping Reference, AWS Setup Guide, Sizing Guide, Sharding Guide, Oracle JDBC Guide, Validation Guide, Production Cutover Guide, Rollback Guide, Troubleshooting, CLI Reference (auto-generated), FAQ.

## Open Questions

(None at this time. All major architecture, UX, and operational decisions have been resolved.)
