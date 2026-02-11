package source

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresReader implements Reader for PostgreSQL using pgx.
type PostgresReader struct {
	connStr string
	schema  string
	pool    *pgxpool.Pool
}

// NewPostgresReader creates a new PostgreSQL reader.
func NewPostgresReader(connStr, schema string) *PostgresReader {
	if schema == "" {
		schema = "public"
	}
	return &PostgresReader{connStr: connStr, schema: schema}
}

func (r *PostgresReader) Connect(ctx context.Context) error {
	cfg, err := pgxpool.ParseConfig(r.connStr)
	if err != nil {
		return fmt.Errorf("parsing connection string: %w", err)
	}
	cfg.MaxConns = 1 // single connection for validation
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connecting to PostgreSQL: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("pinging PostgreSQL: %w", err)
	}
	r.pool = pool
	return nil
}

func (r *PostgresReader) RowCount(ctx context.Context, table string) (int64, error) {
	var count int64
	sql := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s", quoteIdentPg(r.schema), quoteIdentPg(table))
	err := r.pool.QueryRow(ctx, sql).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting rows in %s: %w", table, err)
	}
	return count, nil
}

func (r *PostgresReader) SampleRows(ctx context.Context, table string, columns []string, limit int) ([]map[string]interface{}, error) {
	cols := "*"
	if len(columns) > 0 {
		quoted := make([]string, len(columns))
		for i, c := range columns {
			quoted[i] = quoteIdentPg(c)
		}
		cols = strings.Join(quoted, ", ")
	}
	sql := fmt.Sprintf("SELECT %s FROM %s.%s ORDER BY 1 LIMIT %d", cols, quoteIdentPg(r.schema), quoteIdentPg(table), limit)
	return r.QueryRows(ctx, sql)
}

func (r *PostgresReader) AggregateSum(ctx context.Context, table, column string) (float64, error) {
	var sum float64
	sql := fmt.Sprintf("SELECT COALESCE(SUM(%s)::float8, 0) FROM %s.%s",
		quoteIdentPg(column), quoteIdentPg(r.schema), quoteIdentPg(table))
	err := r.pool.QueryRow(ctx, sql).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("summing %s.%s: %w", table, column, err)
	}
	return sum, nil
}

func (r *PostgresReader) AggregateCountDistinct(ctx context.Context, table, column string) (int64, error) {
	var count int64
	sql := fmt.Sprintf("SELECT COUNT(DISTINCT %s) FROM %s.%s",
		quoteIdentPg(column), quoteIdentPg(r.schema), quoteIdentPg(table))
	err := r.pool.QueryRow(ctx, sql).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting distinct %s.%s: %w", table, column, err)
	}
	return count, nil
}

func (r *PostgresReader) QueryRows(ctx context.Context, sql string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}
	defer rows.Close()

	descs := rows.FieldDescriptions()
	var results []map[string]interface{}
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		row := make(map[string]interface{}, len(descs))
		for i, d := range descs {
			row[d.Name] = vals[i]
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}
	return results, nil
}

func (r *PostgresReader) Close() error {
	if r.pool != nil {
		r.pool.Close()
	}
	return nil
}

func quoteIdentPg(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
