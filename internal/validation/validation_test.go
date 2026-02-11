package validation

import (
	"context"
	"testing"

	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/source"
	"github.com/reloquent/reloquent/internal/target"
)

func makeTestValidator(src *source.MockReader, tgt *target.MockOperator, s *schema.Schema, m *mapping.Mapping) *Validator {
	return &Validator{
		Source:     src,
		Target:     tgt,
		Schema:     s,
		Mapping:    m,
		SampleSize: 10,
	}
}

func TestValidateRowCounts_Match(t *testing.T) {
	src := &source.MockReader{
		RowCounts: map[string]int64{"users": 1000},
	}
	tgt := &target.MockOperator{
		DocCounts: map[string]int64{"users": 1000},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "users", SourceTable: "users"},
		},
	}

	v := makeTestValidator(src, tgt, nil, m)
	result, err := v.ValidateRowCounts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "PASS" {
		t.Errorf("expected PASS, got %s", result.Status)
	}
	if !result.Collections[0].RowCountCheck.Match {
		t.Error("row counts should match")
	}
}

func TestValidateRowCounts_Mismatch(t *testing.T) {
	src := &source.MockReader{
		RowCounts: map[string]int64{"users": 1000},
	}
	tgt := &target.MockOperator{
		DocCounts: map[string]int64{"users": 990},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "users", SourceTable: "users"},
		},
	}

	v := makeTestValidator(src, tgt, nil, m)
	result, err := v.ValidateRowCounts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "FAIL" {
		t.Errorf("expected FAIL, got %s", result.Status)
	}
	if result.Collections[0].RowCountCheck.Match {
		t.Error("row counts should not match")
	}
}

func TestValidateRowCounts_Partial(t *testing.T) {
	src := &source.MockReader{
		RowCounts: map[string]int64{"users": 100, "orders": 500},
	}
	tgt := &target.MockOperator{
		DocCounts: map[string]int64{"users": 100, "orders": 499},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "users", SourceTable: "users"},
			{Name: "orders", SourceTable: "orders"},
		},
	}

	v := makeTestValidator(src, tgt, nil, m)
	result, err := v.ValidateRowCounts(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "PARTIAL" {
		t.Errorf("expected PARTIAL, got %s", result.Status)
	}
}

func TestValidateSamples_AllFieldsPresent(t *testing.T) {
	src := &source.MockReader{}
	tgt := &target.MockOperator{
		SampleDocs: map[string][]map[string]interface{}{
			"users": {
				{"_id": "1", "name": "Alice", "email": "alice@example.com"},
			},
		},
	}
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "name", DataType: "varchar"},
					{Name: "email", DataType: "varchar"},
				},
			},
		},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "users", SourceTable: "users"},
		},
	}

	v := makeTestValidator(src, tgt, s, m)
	result, err := v.ValidateSamples(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "PASS" {
		t.Errorf("expected PASS, got %s", result.Status)
	}
	if result.Collections[0].SampleCheck.MismatchCount != 0 {
		t.Error("no mismatches expected")
	}
}

func TestValidateSamples_MissingField(t *testing.T) {
	src := &source.MockReader{}
	tgt := &target.MockOperator{
		SampleDocs: map[string][]map[string]interface{}{
			"users": {
				{"_id": "1", "name": "Alice"},
			},
		},
	}
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "name", DataType: "varchar"},
					{Name: "email", DataType: "varchar"},
				},
			},
		},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "users", SourceTable: "users"},
		},
	}

	v := makeTestValidator(src, tgt, s, m)
	result, err := v.ValidateSamples(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "FAIL" {
		t.Errorf("expected FAIL, got %s", result.Status)
	}
	if result.Collections[0].SampleCheck.MismatchCount == 0 {
		t.Error("expected mismatches for missing field")
	}
}

func TestValidateAggregates_Match(t *testing.T) {
	src := &source.MockReader{
		CountDistincts: map[string]int64{"users.user_id": 1000},
		Sums:           map[string]float64{"users.balance": 50000.0},
	}
	tgt := &target.MockOperator{
		CountDistincts: map[string]int64{"users.user_id": 1000},
		Sums:           map[string]float64{"users.balance": 50000.0},
	}
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name:       "users",
				PrimaryKey: &schema.PrimaryKey{Name: "pk", Columns: []string{"user_id"}},
				Columns: []schema.Column{
					{Name: "user_id", DataType: "integer"},
					{Name: "balance", DataType: "numeric"},
				},
			},
		},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "users", SourceTable: "users"},
		},
	}

	v := makeTestValidator(src, tgt, s, m)
	result, err := v.ValidateAggregates(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "PASS" {
		t.Errorf("expected PASS, got %s", result.Status)
	}
}

func TestValidateAggregates_Mismatch(t *testing.T) {
	src := &source.MockReader{
		CountDistincts: map[string]int64{"users.user_id": 1000},
	}
	tgt := &target.MockOperator{
		CountDistincts: map[string]int64{"users.user_id": 999},
	}
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name:       "users",
				PrimaryKey: &schema.PrimaryKey{Name: "pk", Columns: []string{"user_id"}},
				Columns: []schema.Column{
					{Name: "user_id", DataType: "integer"},
				},
			},
		},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "users", SourceTable: "users"},
		},
	}

	v := makeTestValidator(src, tgt, s, m)
	result, err := v.ValidateAggregates(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "FAIL" {
		t.Errorf("expected FAIL, got %s", result.Status)
	}
}

func TestValidate_FullPipeline(t *testing.T) {
	src := &source.MockReader{
		RowCounts:      map[string]int64{"users": 100},
		CountDistincts: map[string]int64{"users.user_id": 100},
	}
	tgt := &target.MockOperator{
		DocCounts:      map[string]int64{"users": 100},
		CountDistincts: map[string]int64{"users.user_id": 100},
		SampleDocs: map[string][]map[string]interface{}{
			"users": {{"_id": "1", "user_id": 1, "name": "Alice"}},
		},
	}
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name:       "users",
				PrimaryKey: &schema.PrimaryKey{Name: "pk", Columns: []string{"user_id"}},
				Columns: []schema.Column{
					{Name: "user_id", DataType: "integer"},
					{Name: "name", DataType: "varchar"},
				},
			},
		},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "users", SourceTable: "users"},
		},
	}

	callbackCalls := 0
	v := &Validator{
		Source:     src,
		Target:     tgt,
		Schema:     s,
		Mapping:    m,
		SampleSize: 10,
		Callback: func(collection, checkType string, passed bool) {
			callbackCalls++
		},
	}

	result, err := v.Validate(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "PASS" {
		t.Errorf("expected PASS, got %s", result.Status)
	}
	if callbackCalls != 3 {
		t.Errorf("expected 3 callback calls (row_count, sample, aggregate), got %d", callbackCalls)
	}
	if result.StartedAt.IsZero() || result.CompletedAt.IsZero() {
		t.Error("timestamps should be set")
	}
}

func TestValidate_EmptyCollections(t *testing.T) {
	src := &source.MockReader{
		RowCounts: map[string]int64{"empty": 0},
	}
	tgt := &target.MockOperator{
		DocCounts: map[string]int64{"empty": 0},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "empty", SourceTable: "empty"},
		},
	}

	v := makeTestValidator(src, tgt, &schema.Schema{Tables: []schema.Table{{Name: "empty"}}}, m)
	result, err := v.Validate(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "PASS" {
		t.Errorf("empty collections should PASS, got %s", result.Status)
	}
}

func TestComputeOverallStatus(t *testing.T) {
	tests := []struct {
		name     string
		colls    []CollectionResult
		expected string
	}{
		{"empty", nil, "PASS"},
		{"all_pass", []CollectionResult{{Status: "PASS"}, {Status: "PASS"}}, "PASS"},
		{"all_fail", []CollectionResult{{Status: "FAIL"}, {Status: "FAIL"}}, "FAIL"},
		{"mixed", []CollectionResult{{Status: "PASS"}, {Status: "FAIL"}}, "PARTIAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeOverallStatus(tt.colls)
			if got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestReconstructSQL(t *testing.T) {
	col := mapping.Collection{
		Name:        "orders",
		SourceTable: "orders",
		Embedded: []mapping.Embedded{
			{
				SourceTable:  "order_items",
				FieldName:    "items",
				Relationship: "array",
				JoinColumn:   "order_id",
				ParentColumn: "id",
			},
		},
	}

	sql := ReconstructSQL(col, "public")
	if sql == "" {
		t.Error("expected non-empty SQL")
	}
	if !contains(sql, "LEFT JOIN") {
		t.Error("expected LEFT JOIN for embedded table")
	}
	if !contains(sql, "public.orders") {
		t.Error("expected qualified table name")
	}
}

func TestReconstructSQL_NoEmbedded(t *testing.T) {
	col := mapping.Collection{
		Name:        "users",
		SourceTable: "users",
	}
	sql := ReconstructSQL(col, "")
	if contains(sql, "JOIN") {
		t.Error("should not have JOIN without embedded tables")
	}
}

func TestFloatClose(t *testing.T) {
	if !floatClose(100.0, 100.0) {
		t.Error("identical values should match")
	}
	if !floatClose(100.0, 100.005) {
		t.Error("values within tolerance should match")
	}
	if floatClose(100.0, 200.0) {
		t.Error("values far apart should not match")
	}
	if !floatClose(0.0, 0.0) {
		t.Error("zeros should match")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
