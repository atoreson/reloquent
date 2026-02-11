# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-02-11

### Added

- 12-step interactive wizard available as both a CLI terminal UI (bubbletea) and a full web UI (React/TypeScript).
- Source database discovery for PostgreSQL and Oracle, including tables, columns, primary keys, foreign keys, indexes, and row counts.
- Visual schema designer with drag-and-drop denormalization of foreign key relationships using @xyflow/react.
- PySpark migration script generation targeting the MongoDB Spark Connector with optimized bulk write settings.
- AWS EMR and Glue provisioning support with cluster configuration and job submission.
- 16MB BSON document limit estimation during the schema design phase, surfacing warnings before migration begins.
- Post-migration validation framework with row count comparison, sample document checks, and aggregate value verification.
- MongoDB index inference and building based on source database indexes and foreign key relationships.
- Production readiness checks covering connection health, resource sizing, and configuration completeness.
- Cost and sizing estimation based on source data volumes, target document structure, and AWS instance pricing.
- YAML configuration with schema versioning (`version: 1`) and secret resolution from environment variables, HashiCorp Vault, and AWS Secrets Manager.
- Docker trial mode with the Pagila sample dataset (PostgreSQL to MongoDB) for hands-on evaluation.
