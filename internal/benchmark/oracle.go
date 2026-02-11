package benchmark

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/sijms/go-ora/v2"
)

// OracleReader implements SourceReader for Oracle databases.
type OracleReader struct {
	ConnString string
}

// ReadSample reads a sample from an Oracle table using SAMPLE().
func (r *OracleReader) ReadSample(ctx context.Context, tableName, partitionCol string, samplePct float64) (int64, time.Duration, error) {
	db, err := sql.Open("oracle", r.ConnString)
	if err != nil {
		return 0, 0, fmt.Errorf("connecting to Oracle: %w", err)
	}
	defer db.Close()

	query := fmt.Sprintf("SELECT * FROM %s SAMPLE(%.2f)", tableName, samplePct)

	start := time.Now()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return 0, 0, fmt.Errorf("executing sample query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return 0, 0, fmt.Errorf("getting columns: %w", err)
	}

	var bytesRead int64
	scanDest := make([]any, len(cols))
	scanPtrs := make([]any, len(cols))
	for i := range scanDest {
		scanPtrs[i] = &scanDest[i]
	}

	for rows.Next() {
		if err := rows.Scan(scanPtrs...); err != nil {
			return 0, 0, fmt.Errorf("scanning row: %w", err)
		}
		for _, v := range scanDest {
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
