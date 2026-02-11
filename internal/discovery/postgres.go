package discovery

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/schema"
)

// Postgres implements Discoverer for PostgreSQL databases.
type Postgres struct {
	cfg    *config.SourceConfig
	pool   *pgxpool.Pool
	schema string // pg schema to discover, defaults to "public"
}

// NewPostgres creates a new PostgreSQL discoverer.
func NewPostgres(cfg *config.SourceConfig) (*Postgres, error) {
	s := cfg.Schema
	if s == "" {
		s = "public"
	}
	return &Postgres{cfg: cfg, schema: s}, nil
}

func (p *Postgres) Connect(ctx context.Context) error {
	connStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s default_query_exec_mode=simple_protocol",
		p.cfg.Host, p.cfg.Port, p.cfg.Database, p.cfg.Username, p.cfg.Password,
	)
	if p.cfg.SSL {
		connStr += " sslmode=require"
	} else {
		connStr += " sslmode=disable"
	}

	poolCfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("parsing connection string: %w", err)
	}
	// Discovery uses a single connection per PLAN.md
	poolCfg.MaxConns = 1

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return fmt.Errorf("connecting to PostgreSQL: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("pinging PostgreSQL: %w", err)
	}

	p.pool = pool
	return nil
}

func (p *Postgres) Discover(ctx context.Context) (*schema.Schema, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("not connected; call Connect first")
	}

	tables, err := p.discoverTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("discovering tables: %w", err)
	}

	tableMap := make(map[string]*schema.Table, len(tables))
	for i := range tables {
		tableMap[tables[i].Name] = &tables[i]
	}

	if err := p.discoverColumns(ctx, tableMap); err != nil {
		return nil, fmt.Errorf("discovering columns: %w", err)
	}

	if err := p.discoverPrimaryKeys(ctx, tableMap); err != nil {
		return nil, fmt.Errorf("discovering primary keys: %w", err)
	}

	if err := p.discoverForeignKeys(ctx, tableMap); err != nil {
		return nil, fmt.Errorf("discovering foreign keys: %w", err)
	}

	if err := p.discoverIndexes(ctx, tableMap); err != nil {
		return nil, fmt.Errorf("discovering indexes: %w", err)
	}

	if err := p.discoverCheckConstraints(ctx, tableMap); err != nil {
		return nil, fmt.Errorf("discovering check constraints: %w", err)
	}

	if err := p.detectSequences(ctx, tableMap); err != nil {
		return nil, fmt.Errorf("detecting sequences: %w", err)
	}

	return &schema.Schema{
		DatabaseType: "postgresql",
		Host:         p.cfg.Host,
		Database:     p.cfg.Database,
		SchemaName:   p.schema,
		Tables:       tables,
	}, nil
}

func (p *Postgres) Close() error {
	if p.pool != nil {
		p.pool.Close()
		p.pool = nil
	}
	return nil
}

// discoverTables lists all user tables with row count estimates and on-disk sizes.
func (p *Postgres) discoverTables(ctx context.Context) ([]schema.Table, error) {
	query := `
		SELECT
			c.relname AS table_name,
			c.reltuples::bigint AS row_estimate,
			pg_total_relation_size(c.oid) AS size_bytes
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1
		  AND c.relkind = 'r'
		ORDER BY c.relname`

	rows, err := p.pool.Query(ctx, query, p.schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []schema.Table
	for rows.Next() {
		var t schema.Table
		if err := rows.Scan(&t.Name, &t.RowCount, &t.SizeBytes); err != nil {
			return nil, err
		}
		// reltuples can be -1 for never-analyzed tables
		if t.RowCount < 0 {
			t.RowCount = 0
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

// discoverColumns fetches all columns for all tables in the schema.
func (p *Postgres) discoverColumns(ctx context.Context, tableMap map[string]*schema.Table) error {
	query := `
		SELECT
			table_name,
			column_name,
			data_type,
			is_nullable,
			column_default,
			character_maximum_length,
			numeric_precision,
			numeric_scale
		FROM information_schema.columns
		WHERE table_schema = $1
		  AND table_name = ANY($2)
		ORDER BY table_name, ordinal_position`

	names := tableNames(tableMap)
	rows, err := p.pool.Query(ctx, query, p.schema, names)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			tableName, colName, dataType, nullable string
			defaultVal                              *string
			maxLen, precision, scale                 *int
		)
		if err := rows.Scan(&tableName, &colName, &dataType, &nullable, &defaultVal, &maxLen, &precision, &scale); err != nil {
			return err
		}

		t, ok := tableMap[tableName]
		if !ok {
			continue
		}

		col := schema.Column{
			Name:         colName,
			DataType:     dataType,
			Nullable:     nullable == "YES",
			DefaultValue: defaultVal,
			MaxLength:    maxLen,
			Precision:    precision,
			Scale:        scale,
		}
		t.Columns = append(t.Columns, col)
	}
	return rows.Err()
}

// discoverPrimaryKeys fetches primary key constraints.
func (p *Postgres) discoverPrimaryKeys(ctx context.Context, tableMap map[string]*schema.Table) error {
	query := `
		SELECT
			tc.table_name,
			tc.constraint_name,
			kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		  AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
		  AND tc.table_schema = $1
		  AND tc.table_name = ANY($2)
		ORDER BY tc.table_name, kcu.ordinal_position`

	names := tableNames(tableMap)
	rows, err := p.pool.Query(ctx, query, p.schema, names)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, constraintName, colName string
		if err := rows.Scan(&tableName, &constraintName, &colName); err != nil {
			return err
		}

		t, ok := tableMap[tableName]
		if !ok {
			continue
		}

		if t.PrimaryKey == nil {
			t.PrimaryKey = &schema.PrimaryKey{Name: constraintName}
		}
		t.PrimaryKey.Columns = append(t.PrimaryKey.Columns, colName)
	}
	return rows.Err()
}

// discoverForeignKeys fetches foreign key relationships including composite keys.
func (p *Postgres) discoverForeignKeys(ctx context.Context, tableMap map[string]*schema.Table) error {
	query := `
		SELECT
			tc.table_name,
			tc.constraint_name,
			kcu.column_name,
			ccu.table_name AS referenced_table,
			ccu.column_name AS referenced_column
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		  AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
		  ON tc.constraint_name = ccu.constraint_name
		  AND tc.table_schema = ccu.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema = $1
		  AND tc.table_name = ANY($2)
		ORDER BY tc.table_name, tc.constraint_name, kcu.ordinal_position`

	names := tableNames(tableMap)
	rows, err := p.pool.Query(ctx, query, p.schema, names)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Group columns by constraint name since composite FKs have multiple rows
	type fkRow struct {
		tableName, constraintName, column, refTable, refColumn string
	}
	var fkRows []fkRow

	for rows.Next() {
		var r fkRow
		if err := rows.Scan(&r.tableName, &r.constraintName, &r.column, &r.refTable, &r.refColumn); err != nil {
			return err
		}
		fkRows = append(fkRows, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Group by table + constraint name
	type fkKey struct{ table, constraint string }
	grouped := make(map[fkKey]*schema.ForeignKey)
	var order []fkKey

	for _, r := range fkRows {
		k := fkKey{r.tableName, r.constraintName}
		fk, exists := grouped[k]
		if !exists {
			fk = &schema.ForeignKey{
				Name:            r.constraintName,
				ReferencedTable: r.refTable,
			}
			grouped[k] = fk
			order = append(order, k)
		}
		fk.Columns = append(fk.Columns, r.column)
		fk.ReferencedColumns = append(fk.ReferencedColumns, r.refColumn)
	}

	for _, k := range order {
		if t, ok := tableMap[k.table]; ok {
			t.ForeignKeys = append(t.ForeignKeys, *grouped[k])
		}
	}

	return nil
}

// discoverIndexes fetches all indexes (excluding primary key indexes which are handled separately).
func (p *Postgres) discoverIndexes(ctx context.Context, tableMap map[string]*schema.Table) error {
	query := `
		SELECT
			t.relname AS table_name,
			i.relname AS index_name,
			ix.indisunique AS is_unique,
			am.amname AS index_type,
			a.attname AS column_name
		FROM pg_index ix
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_am am ON am.oid = i.relam
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname = $1
		  AND t.relname = ANY($2)
		  AND NOT ix.indisprimary
		ORDER BY t.relname, i.relname, array_position(ix.indkey, a.attnum)`

	names := tableNames(tableMap)
	rows, err := p.pool.Query(ctx, query, p.schema, names)
	if err != nil {
		return err
	}
	defer rows.Close()

	type idxKey struct{ table, index string }
	grouped := make(map[idxKey]*schema.Index)
	var order []idxKey

	for rows.Next() {
		var tableName, indexName, indexType, colName string
		var isUnique bool
		if err := rows.Scan(&tableName, &indexName, &isUnique, &indexType, &colName); err != nil {
			return err
		}

		k := idxKey{tableName, indexName}
		idx, exists := grouped[k]
		if !exists {
			idx = &schema.Index{
				Name:   indexName,
				Unique: isUnique,
				Type:   indexType,
			}
			grouped[k] = idx
			order = append(order, k)
		}
		idx.Columns = append(idx.Columns, colName)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, k := range order {
		if t, ok := tableMap[k.table]; ok {
			t.Indexes = append(t.Indexes, *grouped[k])
		}
	}

	return nil
}

// discoverCheckConstraints fetches CHECK constraints (excluding NOT NULL which is on the column).
func (p *Postgres) discoverCheckConstraints(ctx context.Context, tableMap map[string]*schema.Table) error {
	query := `
		SELECT
			tc.table_name,
			tc.constraint_name,
			cc.check_clause
		FROM information_schema.table_constraints tc
		JOIN information_schema.check_constraints cc
		  ON tc.constraint_name = cc.constraint_name
		  AND tc.constraint_schema = cc.constraint_schema
		WHERE tc.constraint_type = 'CHECK'
		  AND tc.table_schema = $1
		  AND tc.table_name = ANY($2)
		  AND tc.constraint_name NOT LIKE '%_not_null'
		ORDER BY tc.table_name, tc.constraint_name`

	names := tableNames(tableMap)
	rows, err := p.pool.Query(ctx, query, p.schema, names)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, constraintName, checkClause string
		if err := rows.Scan(&tableName, &constraintName, &checkClause); err != nil {
			return err
		}

		t, ok := tableMap[tableName]
		if !ok {
			continue
		}

		t.Constraints = append(t.Constraints, schema.Constraint{
			Name:       constraintName,
			Type:       "check",
			Definition: checkClause,
		})
	}
	return rows.Err()
}

// detectSequences marks columns that use sequences (serial/bigserial/identity).
func (p *Postgres) detectSequences(ctx context.Context, tableMap map[string]*schema.Table) error {
	query := `
		SELECT
			table_name,
			column_name
		FROM information_schema.columns
		WHERE table_schema = $1
		  AND table_name = ANY($2)
		  AND (column_default LIKE 'nextval(%' OR is_identity = 'YES')`

	names := tableNames(tableMap)
	rows, err := p.pool.Query(ctx, query, p.schema, names)
	if err != nil {
		// is_identity may not exist on older PG versions; if so, fall back
		return p.detectSequencesFallback(ctx, tableMap)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, colName string
		if err := rows.Scan(&tableName, &colName); err != nil {
			return err
		}

		t, ok := tableMap[tableName]
		if !ok {
			continue
		}
		for i := range t.Columns {
			if t.Columns[i].Name == colName {
				t.Columns[i].IsSequence = true
				break
			}
		}
	}
	return rows.Err()
}

func (p *Postgres) detectSequencesFallback(ctx context.Context, tableMap map[string]*schema.Table) error {
	query := `
		SELECT
			table_name,
			column_name
		FROM information_schema.columns
		WHERE table_schema = $1
		  AND table_name = ANY($2)
		  AND column_default LIKE 'nextval(%'`

	names := tableNames(tableMap)
	rows, err := p.pool.Query(ctx, query, p.schema, names)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, colName string
		if err := rows.Scan(&tableName, &colName); err != nil {
			return err
		}

		t, ok := tableMap[tableName]
		if !ok {
			continue
		}
		for i := range t.Columns {
			if t.Columns[i].Name == colName {
				t.Columns[i].IsSequence = true
				break
			}
		}
	}
	return rows.Err()
}

// ConnString returns a DSN for testing or diagnostics.
func (p *Postgres) ConnString() string {
	ssl := "disable"
	if p.cfg.SSL {
		ssl = "require"
	}
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s sslmode=%s",
		p.cfg.Host, p.cfg.Port, p.cfg.Database, p.cfg.Username, ssl)
}

func tableNames(tableMap map[string]*schema.Table) []string {
	names := make([]string, 0, len(tableMap))
	for name := range tableMap {
		names = append(names, name)
	}
	return names
}

// pgArrayLiteral formats a string slice as a Postgres array literal.
// Not currently used but kept for potential raw SQL needs.
func pgArrayLiteral(vals []string) string {
	quoted := make([]string, len(vals))
	for i, v := range vals {
		quoted[i] = "'" + strings.ReplaceAll(v, "'", "''") + "'"
	}
	return "ARRAY[" + strings.Join(quoted, ",") + "]"
}

// compile-time interface check
var _ Discoverer = (*Postgres)(nil)
