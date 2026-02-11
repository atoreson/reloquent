package source

import "context"

// Reader provides read-only access to a source database for validation queries.
type Reader interface {
	Connect(ctx context.Context) error
	RowCount(ctx context.Context, table string) (int64, error)
	SampleRows(ctx context.Context, table string, columns []string, limit int) ([]map[string]interface{}, error)
	AggregateSum(ctx context.Context, table, column string) (float64, error)
	AggregateCountDistinct(ctx context.Context, table, column string) (int64, error)
	QueryRows(ctx context.Context, sql string, args ...interface{}) ([]map[string]interface{}, error)
	Close() error
}
