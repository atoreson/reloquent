# Reloquent Trial Mode

Try Reloquent with a pre-loaded PostgreSQL database (Pagila DVD rental) and MongoDB target.

## Quick Start

```bash
docker compose -f trial/docker-compose.yml up --build
```

Open **http://localhost:8230** in your browser.

## Credentials

| Service    | Host        | Port  | User       | Password   | Database |
|-----------|-------------|-------|------------|------------|----------|
| PostgreSQL | localhost   | 15432 | reloquent  | reloquent  | pagila   |
| MongoDB    | localhost   | 17017 | (none)     | (none)     | pagila   |
| Web UI     | localhost   | 8230  | —          | —          | —        |

## Walkthrough

1. **Source Connection** — Pre-filled from config. Click "Test Connection" then "Discover Schema".
2. **Target Connection** — Pre-filled. Click "Test Connection".
3. **Table Selection** — Select all ~15 Pagila tables.
4. **Denormalization Design** — Use the visual designer to embed `film_actor` into `film`, `rental` into `customer`, etc.
5. **Type Mapping** — Review PostgreSQL → BSON type mappings.
6. **Sizing** — See estimated cluster sizes and costs.
7. **AWS Setup** — Skip for trial (no real AWS needed).
8. **Pre-Migration** — Creates target collections in MongoDB.
9. **Review** — See the generated PySpark migration script.

## The Spark Gap

Steps 1–9 work fully in the trial. The actual **migration execution** (Step 10) requires a real Apache Spark cluster (AWS EMR or Glue) to run the generated PySpark scripts. The trial demonstrates everything up to and including code generation.

In production, you would:
1. Configure AWS credentials (Step 7)
2. Provision an EMR cluster or Glue job
3. The generated PySpark script reads from PostgreSQL via JDBC and writes to MongoDB

## Connect Directly

```bash
# PostgreSQL
psql -h localhost -p 15432 -U reloquent -d pagila

# MongoDB
mongosh "mongodb://localhost:17017/?replicaSet=rs0"
```

## Cleanup

```bash
docker compose -f trial/docker-compose.yml down -v
```
