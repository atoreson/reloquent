package discovery

import "fmt"

// ScriptGenerator generates offline SQL scripts that produce YAML-formatted schema output.
type ScriptGenerator struct {
	DBType string
	Schema string // schema/owner name
}

// GenerateScript returns a SQL script that queries the database catalog
// and outputs YAML matching the schema.Schema format.
func (sg *ScriptGenerator) GenerateScript() string {
	switch sg.DBType {
	case "postgresql":
		return sg.postgresScript()
	case "oracle":
		return sg.oracleScript()
	default:
		return fmt.Sprintf("-- Unsupported database type: %s\n", sg.DBType)
	}
}

// GenerateShellWrapper returns a bash wrapper script that runs the SQL
// and captures the output.
func (sg *ScriptGenerator) GenerateShellWrapper() string {
	switch sg.DBType {
	case "postgresql":
		return sg.postgresWrapper()
	case "oracle":
		return sg.oracleWrapper()
	default:
		return fmt.Sprintf("#!/bin/bash\necho 'Unsupported database type: %s'\nexit 1\n", sg.DBType)
	}
}

func (sg *ScriptGenerator) postgresScript() string {
	schemaName := sg.Schema
	if schemaName == "" {
		schemaName = "public"
	}

	return fmt.Sprintf(`-- Reloquent Offline Discovery Script (PostgreSQL)
-- Run: psql -h HOST -U USER -d DB -f this_script.sql -o output.yaml -t -A

-- Tables with row estimates and sizes
SELECT 'database_type: postgresql';
SELECT 'tables:';

SELECT '- name: ' || c.relname ||
       E'\n  row_count: ' || GREATEST(c.reltuples::bigint, 0) ||
       E'\n  size_bytes: ' || pg_total_relation_size(c.oid)
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = '%s'
  AND c.relkind = 'r'
ORDER BY c.relname;

-- Columns
SELECT '  columns:' FROM (SELECT 1) x WHERE EXISTS (
  SELECT 1 FROM information_schema.columns WHERE table_schema = '%s' LIMIT 1
);
SELECT '  - name: ' || column_name ||
       E'\n    data_type: ' || data_type ||
       E'\n    nullable: ' || CASE WHEN is_nullable = 'YES' THEN 'true' ELSE 'false' END
FROM information_schema.columns
WHERE table_schema = '%s'
ORDER BY table_name, ordinal_position;

-- Primary keys
SELECT '  primary_key:';
SELECT '    name: ' || tc.constraint_name ||
       E'\n    columns: [' || string_agg(kcu.column_name, ', ' ORDER BY kcu.ordinal_position) || ']'
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
  ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
WHERE tc.constraint_type = 'PRIMARY KEY'
  AND tc.table_schema = '%s'
GROUP BY tc.table_name, tc.constraint_name
ORDER BY tc.table_name;

-- Foreign keys
SELECT '  foreign_keys:';
SELECT '  - name: ' || tc.constraint_name ||
       E'\n    referenced_table: ' || ccu.table_name ||
       E'\n    columns: [' || string_agg(DISTINCT kcu.column_name, ', ') || ']' ||
       E'\n    referenced_columns: [' || string_agg(DISTINCT ccu.column_name, ', ') || ']'
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
  ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
JOIN information_schema.constraint_column_usage ccu
  ON tc.constraint_name = ccu.constraint_name AND tc.table_schema = ccu.table_schema
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND tc.table_schema = '%s'
GROUP BY tc.table_name, tc.constraint_name, ccu.table_name
ORDER BY tc.table_name;
`, schemaName, schemaName, schemaName, schemaName, schemaName)
}

func (sg *ScriptGenerator) oracleScript() string {
	owner := sg.Schema
	if owner == "" {
		owner = "CURRENT_USER"
	}

	return fmt.Sprintf(`-- Reloquent Offline Discovery Script (Oracle)
-- Run: sqlplus USER/PASS@HOST:PORT/SID @this_script.sql > output.yaml

SET LINESIZE 500
SET PAGESIZE 0
SET FEEDBACK OFF
SET HEADING OFF

SELECT 'database_type: oracle' FROM DUAL;
SELECT 'tables:' FROM DUAL;

-- Tables with row counts
SELECT '- name: ' || TABLE_NAME ||
       CHR(10) || '  row_count: ' || NVL(NUM_ROWS, 0)
FROM ALL_TABLES
WHERE OWNER = '%s'
ORDER BY TABLE_NAME;

-- Columns
SELECT '  columns:' FROM DUAL;
SELECT '  - name: ' || COLUMN_NAME ||
       CHR(10) || '    data_type: ' || DATA_TYPE ||
       CHR(10) || '    nullable: ' || CASE WHEN NULLABLE = 'Y' THEN 'true' ELSE 'false' END
FROM ALL_TAB_COLUMNS
WHERE OWNER = '%s'
ORDER BY TABLE_NAME, COLUMN_ID;

-- Primary keys
SELECT '  primary_key:' FROM DUAL;
SELECT '    name: ' || c.CONSTRAINT_NAME ||
       CHR(10) || '    columns: [' || LISTAGG(cc.COLUMN_NAME, ', ') WITHIN GROUP (ORDER BY cc.POSITION) || ']'
FROM ALL_CONSTRAINTS c
JOIN ALL_CONS_COLUMNS cc ON c.CONSTRAINT_NAME = cc.CONSTRAINT_NAME AND c.OWNER = cc.OWNER
WHERE c.OWNER = '%s'
  AND c.CONSTRAINT_TYPE = 'P'
GROUP BY c.TABLE_NAME, c.CONSTRAINT_NAME
ORDER BY c.TABLE_NAME;

-- Foreign keys
SELECT '  foreign_keys:' FROM DUAL;
SELECT '  - name: ' || c.CONSTRAINT_NAME ||
       CHR(10) || '    referenced_table: ' || rc.TABLE_NAME ||
       CHR(10) || '    columns: [' || LISTAGG(cc.COLUMN_NAME, ', ') WITHIN GROUP (ORDER BY cc.POSITION) || ']'
FROM ALL_CONSTRAINTS c
JOIN ALL_CONS_COLUMNS cc ON c.CONSTRAINT_NAME = cc.CONSTRAINT_NAME AND c.OWNER = cc.OWNER
JOIN ALL_CONSTRAINTS rc ON c.R_CONSTRAINT_NAME = rc.CONSTRAINT_NAME AND c.R_OWNER = rc.OWNER
WHERE c.OWNER = '%s'
  AND c.CONSTRAINT_TYPE = 'R'
GROUP BY c.TABLE_NAME, c.CONSTRAINT_NAME, rc.TABLE_NAME
ORDER BY c.TABLE_NAME;

EXIT;
`, owner, owner, owner, owner)
}

func (sg *ScriptGenerator) postgresWrapper() string {
	return `#!/bin/bash
# Reloquent Offline Discovery Wrapper (PostgreSQL)
# Usage: ./discover.sh -h HOST -p PORT -U USER -d DATABASE

set -euo pipefail

OUTPUT="${RELOQUENT_OUTPUT:-discovery-output.yaml}"

echo "Running PostgreSQL schema discovery..."
echo "Output: ${OUTPUT}"

psql "$@" -f "$(dirname "$0")/discover.sql" -t -A -o "${OUTPUT}"

echo "Done. Import with: reloquent ingest --file ${OUTPUT}"
`
}

func (sg *ScriptGenerator) oracleWrapper() string {
	return `#!/bin/bash
# Reloquent Offline Discovery Wrapper (Oracle)
# Usage: ./discover.sh USER/PASS@HOST:PORT/SID

set -euo pipefail

OUTPUT="${RELOQUENT_OUTPUT:-discovery-output.yaml}"

echo "Running Oracle schema discovery..."
echo "Output: ${OUTPUT}"

sqlplus -S "$1" @"$(dirname "$0")/discover.sql" > "${OUTPUT}"

echo "Done. Import with: reloquent ingest --file ${OUTPUT}"
`
}
