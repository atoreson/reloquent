package benchmark

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// PostgresReader implements SourceReader for PostgreSQL databases.
type PostgresReader struct {
	ConnString string
}

// ReadSample reads a sample from a PostgreSQL table using TABLESAMPLE.
func (r *PostgresReader) ReadSample(ctx context.Context, tableName, partitionCol string, samplePct float64) (int64, time.Duration, error) {
	conn, err := pgx.Connect(ctx, r.ConnString)
	if err != nil {
		return 0, 0, fmt.Errorf("connecting to PostgreSQL: %w", err)
	}
	defer conn.Close(ctx)

	query := fmt.Sprintf("SELECT * FROM %s TABLESAMPLE SYSTEM(%.2f)", pgx.Identifier{tableName}.Sanitize(), samplePct)

	start := time.Now()

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return 0, 0, fmt.Errorf("executing sample query: %w", err)
	}
	defer rows.Close()

	var bytesRead int64
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return 0, 0, fmt.Errorf("reading row: %w", err)
		}
		for _, v := range values {
			if v != nil {
				bytesRead += estimateValueSize(v)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("iterating rows: %w", err)
	}

	elapsed := time.Since(start)
	return bytesRead, elapsed, nil
}

func estimateValueSize(v any) int64 {
	switch val := v.(type) {
	case string:
		return int64(len(val))
	case []byte:
		return int64(len(val))
	case int, int32, int64, float32, float64:
		return 8
	case bool:
		return 1
	default:
		return 16 // rough estimate for other types
	}
}
