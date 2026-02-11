package discovery

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/sijms/go-ora/v2"
	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/schema"
)

// Oracle implements Discoverer for Oracle databases using go-ora (pure Go, no Instant Client).
type Oracle struct {
	cfg   *config.SourceConfig
	db    *sql.DB
	owner string // Oracle schema owner, defaults to username uppercased
}

// NewOracle creates a new Oracle discoverer.
func NewOracle(cfg *config.SourceConfig) (*Oracle, error) {
	owner := cfg.Schema
	if owner == "" {
		owner = strings.ToUpper(cfg.Username)
	}
	return &Oracle{cfg: cfg, owner: owner}, nil
}

// ConnString returns the go-ora connection string.
func (o *Oracle) ConnString() string {
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		o.cfg.Username, o.cfg.Password, o.cfg.Host, o.cfg.Port, o.cfg.Database)
}

func (o *Oracle) Connect(ctx context.Context) error {
	connStr := o.ConnString()

	db, err := sql.Open("oracle", connStr)
	if err != nil {
		return fmt.Errorf("opening Oracle connection: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("pinging Oracle: %w", err)
	}

	o.db = db
	return nil
}

func (o *Oracle) Discover(ctx context.Context) (*schema.Schema, error) {
	if o.db == nil {
		return nil, fmt.Errorf("not connected; call Connect first")
	}

	tables, err := o.discoverTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("discovering tables: %w", err)
	}

	tableMap := make(map[string]*schema.Table, len(tables))
	for i := range tables {
		tableMap[tables[i].Name] = &tables[i]
	}

	if err := o.discoverColumns(ctx, tableMap); err != nil {
		return nil, fmt.Errorf("discovering columns: %w", err)
	}

	if err := o.discoverPrimaryKeys(ctx, tableMap); err != nil {
		return nil, fmt.Errorf("discovering primary keys: %w", err)
	}

	if err := o.discoverForeignKeys(ctx, tableMap); err != nil {
		return nil, fmt.Errorf("discovering foreign keys: %w", err)
	}

	if err := o.discoverIndexes(ctx, tableMap); err != nil {
		return nil, fmt.Errorf("discovering indexes: %w", err)
	}

	if err := o.discoverCheckConstraints(ctx, tableMap); err != nil {
		return nil, fmt.Errorf("discovering check constraints: %w", err)
	}

	if err := o.detectSequences(ctx, tableMap); err != nil {
		return nil, fmt.Errorf("detecting sequences: %w", err)
	}

	return &schema.Schema{
		DatabaseType: "oracle",
		Host:         o.cfg.Host,
		Database:     o.cfg.Database,
		SchemaName:   o.owner,
		Tables:       tables,
	}, nil
}

func (o *Oracle) Close() error {
	if o.db != nil {
		err := o.db.Close()
		o.db = nil
		return err
	}
	return nil
}

func (o *Oracle) discoverTables(ctx context.Context) ([]schema.Table, error) {
	query := `
		SELECT t.TABLE_NAME, NVL(t.NUM_ROWS, 0),
			NVL((SELECT SUM(s.BYTES) FROM DBA_SEGMENTS s WHERE s.SEGMENT_NAME = t.TABLE_NAME AND s.OWNER = t.OWNER), 0)
		FROM ALL_TABLES t
		WHERE t.OWNER = :1
		ORDER BY t.TABLE_NAME`

	rows, err := o.db.QueryContext(ctx, query, o.owner)
	if err != nil {
		// Fall back to version without DBA_SEGMENTS (requires privileges)
		return o.discoverTablesFallback(ctx)
	}
	defer rows.Close()

	var tables []schema.Table
	for rows.Next() {
		var t schema.Table
		if err := rows.Scan(&t.Name, &t.RowCount, &t.SizeBytes); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

func (o *Oracle) discoverTablesFallback(ctx context.Context) ([]schema.Table, error) {
	query := `
		SELECT TABLE_NAME, NVL(NUM_ROWS, 0), 0
		FROM ALL_TABLES
		WHERE OWNER = :1
		ORDER BY TABLE_NAME`

	rows, err := o.db.QueryContext(ctx, query, o.owner)
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
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

func (o *Oracle) discoverColumns(ctx context.Context, tableMap map[string]*schema.Table) error {
	query := `
		SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE,
			CASE WHEN NULLABLE = 'Y' THEN 'YES' ELSE 'NO' END,
			DATA_DEFAULT, CHAR_LENGTH, DATA_PRECISION, DATA_SCALE
		FROM ALL_TAB_COLUMNS
		WHERE OWNER = :1
		ORDER BY TABLE_NAME, COLUMN_ID`

	rows, err := o.db.QueryContext(ctx, query, o.owner)
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

func (o *Oracle) discoverPrimaryKeys(ctx context.Context, tableMap map[string]*schema.Table) error {
	query := `
		SELECT c.TABLE_NAME, c.CONSTRAINT_NAME, cc.COLUMN_NAME
		FROM ALL_CONSTRAINTS c
		JOIN ALL_CONS_COLUMNS cc ON c.CONSTRAINT_NAME = cc.CONSTRAINT_NAME AND c.OWNER = cc.OWNER
		WHERE c.OWNER = :1
		  AND c.CONSTRAINT_TYPE = 'P'
		ORDER BY c.TABLE_NAME, cc.POSITION`

	rows, err := o.db.QueryContext(ctx, query, o.owner)
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

func (o *Oracle) discoverForeignKeys(ctx context.Context, tableMap map[string]*schema.Table) error {
	query := `
		SELECT c.TABLE_NAME, c.CONSTRAINT_NAME,
			cc.COLUMN_NAME,
			rc.TABLE_NAME AS REF_TABLE,
			rcc.COLUMN_NAME AS REF_COLUMN
		FROM ALL_CONSTRAINTS c
		JOIN ALL_CONS_COLUMNS cc ON c.CONSTRAINT_NAME = cc.CONSTRAINT_NAME AND c.OWNER = cc.OWNER
		JOIN ALL_CONSTRAINTS rc ON c.R_CONSTRAINT_NAME = rc.CONSTRAINT_NAME AND c.R_OWNER = rc.OWNER
		JOIN ALL_CONS_COLUMNS rcc ON rc.CONSTRAINT_NAME = rcc.CONSTRAINT_NAME AND rc.OWNER = rcc.OWNER
			AND cc.POSITION = rcc.POSITION
		WHERE c.OWNER = :1
		  AND c.CONSTRAINT_TYPE = 'R'
		ORDER BY c.TABLE_NAME, c.CONSTRAINT_NAME, cc.POSITION`

	rows, err := o.db.QueryContext(ctx, query, o.owner)
	if err != nil {
		return err
	}
	defer rows.Close()

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

func (o *Oracle) discoverIndexes(ctx context.Context, tableMap map[string]*schema.Table) error {
	query := `
		SELECT i.TABLE_NAME, i.INDEX_NAME, i.UNIQUENESS, i.INDEX_TYPE, ic.COLUMN_NAME
		FROM ALL_INDEXES i
		JOIN ALL_IND_COLUMNS ic ON i.INDEX_NAME = ic.INDEX_NAME AND i.TABLE_OWNER = ic.TABLE_OWNER
		WHERE i.TABLE_OWNER = :1
		  AND i.INDEX_NAME NOT IN (
			SELECT CONSTRAINT_NAME FROM ALL_CONSTRAINTS
			WHERE OWNER = :2 AND CONSTRAINT_TYPE = 'P'
		  )
		ORDER BY i.TABLE_NAME, i.INDEX_NAME, ic.COLUMN_POSITION`

	rows, err := o.db.QueryContext(ctx, query, o.owner, o.owner)
	if err != nil {
		return err
	}
	defer rows.Close()

	type idxKey struct{ table, index string }
	grouped := make(map[idxKey]*schema.Index)
	var order []idxKey

	for rows.Next() {
		var tableName, indexName, uniqueness, indexType, colName string
		if err := rows.Scan(&tableName, &indexName, &uniqueness, &indexType, &colName); err != nil {
			return err
		}

		k := idxKey{tableName, indexName}
		idx, exists := grouped[k]
		if !exists {
			idx = &schema.Index{
				Name:   indexName,
				Unique: uniqueness == "UNIQUE",
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

func (o *Oracle) discoverCheckConstraints(ctx context.Context, tableMap map[string]*schema.Table) error {
	query := `
		SELECT TABLE_NAME, CONSTRAINT_NAME, SEARCH_CONDITION
		FROM ALL_CONSTRAINTS
		WHERE OWNER = :1
		  AND CONSTRAINT_TYPE = 'C'
		  AND GENERATED != 'GENERATED NAME'
		ORDER BY TABLE_NAME, CONSTRAINT_NAME`

	rows, err := o.db.QueryContext(ctx, query, o.owner)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, constraintName string
		var searchCondition *string
		if err := rows.Scan(&tableName, &constraintName, &searchCondition); err != nil {
			return err
		}

		t, ok := tableMap[tableName]
		if !ok {
			continue
		}

		def := ""
		if searchCondition != nil {
			def = *searchCondition
		}

		// Filter out NOT NULL constraints (they show up as "COL" IS NOT NULL)
		if strings.Contains(def, "IS NOT NULL") {
			continue
		}

		t.Constraints = append(t.Constraints, schema.Constraint{
			Name:       constraintName,
			Type:       "check",
			Definition: def,
		})
	}
	return rows.Err()
}

func (o *Oracle) detectSequences(ctx context.Context, tableMap map[string]*schema.Table) error {
	query := `
		SELECT TABLE_NAME, COLUMN_NAME
		FROM ALL_TAB_COLUMNS
		WHERE OWNER = :1
		  AND IDENTITY_COLUMN = 'YES'`

	rows, err := o.db.QueryContext(ctx, query, o.owner)
	if err != nil {
		// IDENTITY_COLUMN may not exist on older Oracle versions
		return nil
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

// compile-time interface check
var _ Discoverer = (*Oracle)(nil)
