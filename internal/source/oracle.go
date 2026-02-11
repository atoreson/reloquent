package source

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	// Oracle driver
	_ "github.com/sijms/go-ora/v2"
)

// OracleReader implements Reader for Oracle using go-ora.
type OracleReader struct {
	connStr string
	schema  string
	db      *sql.DB
}

// NewOracleReader creates a new Oracle reader.
func NewOracleReader(connStr, schema string) *OracleReader {
	return &OracleReader{connStr: connStr, schema: strings.ToUpper(schema)}
}

func (r *OracleReader) Connect(ctx context.Context) error {
	db, err := sql.Open("oracle", r.connStr)
	if err != nil {
		return fmt.Errorf("opening Oracle connection: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("pinging Oracle: %w", err)
	}
	r.db = db
	return nil
}

func (r *OracleReader) RowCount(ctx context.Context, table string) (int64, error) {
	var count int64
	q := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s", quoteIdentOra(r.schema), quoteIdentOra(table))
	err := r.db.QueryRowContext(ctx, q).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting rows in %s: %w", table, err)
	}
	return count, nil
}

func (r *OracleReader) SampleRows(ctx context.Context, table string, columns []string, limit int) ([]map[string]interface{}, error) {
	cols := "*"
	if len(columns) > 0 {
		quoted := make([]string, len(columns))
		for i, c := range columns {
			quoted[i] = quoteIdentOra(c)
		}
		cols = strings.Join(quoted, ", ")
	}
	q := fmt.Sprintf("SELECT %s FROM %s.%s WHERE ROWNUM <= %d ORDER BY 1",
		cols, quoteIdentOra(r.schema), quoteIdentOra(table), limit)
	return r.QueryRows(ctx, q)
}

func (r *OracleReader) AggregateSum(ctx context.Context, table, column string) (float64, error) {
	var sum float64
	q := fmt.Sprintf("SELECT COALESCE(SUM(%s), 0) FROM %s.%s",
		quoteIdentOra(column), quoteIdentOra(r.schema), quoteIdentOra(table))
	err := r.db.QueryRowContext(ctx, q).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("summing %s.%s: %w", table, column, err)
	}
	return sum, nil
}

func (r *OracleReader) AggregateCountDistinct(ctx context.Context, table, column string) (int64, error) {
	var count int64
	q := fmt.Sprintf("SELECT COUNT(DISTINCT %s) FROM %s.%s",
		quoteIdentOra(column), quoteIdentOra(r.schema), quoteIdentOra(table))
	err := r.db.QueryRowContext(ctx, q).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting distinct %s.%s: %w", table, column, err)
	}
	return count, nil
}

func (r *OracleReader) QueryRows(ctx context.Context, sqlStr string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := r.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("getting columns: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		row := make(map[string]interface{}, len(cols))
		for i, c := range cols {
			row[c] = vals[i]
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}
	return results, nil
}

func (r *OracleReader) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

func quoteIdentOra(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
