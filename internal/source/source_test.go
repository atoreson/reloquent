package source

import (
	"context"
	"errors"
	"testing"
)

func TestMockReader_Connect(t *testing.T) {
	m := &MockReader{}
	if err := m.Connect(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Connected {
		t.Error("should be connected")
	}
}

func TestMockReader_ConnectError(t *testing.T) {
	m := &MockReader{ConnectErr: errors.New("refused")}
	if err := m.Connect(context.Background()); err == nil {
		t.Error("expected error")
	}
}

func TestMockReader_RowCount(t *testing.T) {
	m := &MockReader{
		RowCounts: map[string]int64{
			"users":  1000,
			"orders": 5000,
		},
	}

	tests := []struct {
		table string
		want  int64
	}{
		{"users", 1000},
		{"orders", 5000},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			got, err := m.RowCount(context.Background(), tt.table)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("RowCount(%s) = %d, want %d", tt.table, got, tt.want)
			}
		})
	}
}

func TestMockReader_RowCount_Missing(t *testing.T) {
	m := &MockReader{RowCounts: map[string]int64{}}
	_, err := m.RowCount(context.Background(), "missing")
	if err == nil {
		t.Error("expected error for missing table")
	}
}

func TestMockReader_SampleRows(t *testing.T) {
	m := &MockReader{
		Samples: map[string][]map[string]interface{}{
			"users": {
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			},
		},
	}

	rows, err := m.SampleRows(context.Background(), "users", nil, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
}

func TestMockReader_AggregateSum(t *testing.T) {
	m := &MockReader{
		Sums: map[string]float64{
			"orders.total": 99999.50,
		},
	}

	sum, err := m.AggregateSum(context.Background(), "orders", "total")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum != 99999.50 {
		t.Errorf("expected 99999.50, got %f", sum)
	}
}

func TestMockReader_AggregateCountDistinct(t *testing.T) {
	m := &MockReader{
		CountDistincts: map[string]int64{
			"users.id": 1000,
		},
	}

	count, err := m.AggregateCountDistinct(context.Background(), "users", "id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1000 {
		t.Errorf("expected 1000, got %d", count)
	}
}

func TestMockReader_Close(t *testing.T) {
	m := &MockReader{}
	if err := m.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Closed {
		t.Error("should be closed")
	}
}

func TestMockReader_Errors(t *testing.T) {
	testErr := errors.New("test error")

	t.Run("RowCountErr", func(t *testing.T) {
		m := &MockReader{RowCountErr: testErr}
		_, err := m.RowCount(context.Background(), "x")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("SampleErr", func(t *testing.T) {
		m := &MockReader{SampleErr: testErr}
		_, err := m.SampleRows(context.Background(), "x", nil, 1)
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("SumErr", func(t *testing.T) {
		m := &MockReader{SumErr: testErr}
		_, err := m.AggregateSum(context.Background(), "x", "y")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("CountDistinctErr", func(t *testing.T) {
		m := &MockReader{CountDistinctErr: testErr}
		_, err := m.AggregateCountDistinct(context.Background(), "x", "y")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("QueryErr", func(t *testing.T) {
		m := &MockReader{QueryErr: testErr}
		_, err := m.QueryRows(context.Background(), "SELECT 1")
		if err == nil {
			t.Error("expected error")
		}
	})
}
