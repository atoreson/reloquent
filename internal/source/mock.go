package source

import (
	"context"
	"fmt"
)

// MockReader is a test double for the Reader interface.
type MockReader struct {
	ConnectErr error

	RowCounts          map[string]int64
	RowCountErr        error
	Samples            map[string][]map[string]interface{}
	SampleErr          error
	Sums               map[string]float64 // key: "table.column"
	SumErr             error
	CountDistincts     map[string]int64 // key: "table.column"
	CountDistinctErr   error
	QueryResult        []map[string]interface{}
	QueryErr           error

	Connected bool
	Closed    bool
}

func (m *MockReader) Connect(_ context.Context) error {
	if m.ConnectErr != nil {
		return m.ConnectErr
	}
	m.Connected = true
	return nil
}

func (m *MockReader) RowCount(_ context.Context, table string) (int64, error) {
	if m.RowCountErr != nil {
		return 0, m.RowCountErr
	}
	if m.RowCounts != nil {
		if c, ok := m.RowCounts[table]; ok {
			return c, nil
		}
	}
	return 0, fmt.Errorf("no row count configured for table %s", table)
}

func (m *MockReader) SampleRows(_ context.Context, table string, _ []string, _ int) ([]map[string]interface{}, error) {
	if m.SampleErr != nil {
		return nil, m.SampleErr
	}
	if m.Samples != nil {
		if s, ok := m.Samples[table]; ok {
			return s, nil
		}
	}
	return nil, nil
}

func (m *MockReader) AggregateSum(_ context.Context, table, column string) (float64, error) {
	if m.SumErr != nil {
		return 0, m.SumErr
	}
	key := table + "." + column
	if m.Sums != nil {
		if s, ok := m.Sums[key]; ok {
			return s, nil
		}
	}
	return 0, nil
}

func (m *MockReader) AggregateCountDistinct(_ context.Context, table, column string) (int64, error) {
	if m.CountDistinctErr != nil {
		return 0, m.CountDistinctErr
	}
	key := table + "." + column
	if m.CountDistincts != nil {
		if c, ok := m.CountDistincts[key]; ok {
			return c, nil
		}
	}
	return 0, nil
}

func (m *MockReader) QueryRows(_ context.Context, _ string, _ ...interface{}) ([]map[string]interface{}, error) {
	if m.QueryErr != nil {
		return nil, m.QueryErr
	}
	return m.QueryResult, nil
}

func (m *MockReader) Close() error {
	m.Closed = true
	return nil
}
