package transform

import (
	"strings"
	"testing"

	"github.com/reloquent/reloquent/internal/mapping"
)

func TestToPySpark_Rename(t *testing.T) {
	tr := mapping.Transformation{
		Operation:   OpRename,
		SourceField: "first_name",
		TargetField: "firstName",
	}
	got := ToPySpark(tr, "df")
	want := `df = df.withColumnRenamed("first_name", "firstName")`
	if got != want {
		t.Errorf("got:\n  %s\nwant:\n  %s", got, want)
	}
}

func TestToPySpark_Compute(t *testing.T) {
	tr := mapping.Transformation{
		Operation:   OpCompute,
		TargetField: "full_name",
		Expression:  "concat(first_name, ' ', last_name)",
	}
	got := ToPySpark(tr, "df")
	want := `df = df.withColumn("full_name", expr("concat(first_name, ' ', last_name)"))`
	if got != want {
		t.Errorf("got:\n  %s\nwant:\n  %s", got, want)
	}
}

func TestToPySpark_Cast(t *testing.T) {
	tr := mapping.Transformation{
		Operation:   OpCast,
		SourceField: "price",
		TargetType:  "double",
	}
	got := ToPySpark(tr, "df")
	want := `df = df.withColumn("price", col("price").cast("double"))`
	if got != want {
		t.Errorf("got:\n  %s\nwant:\n  %s", got, want)
	}
}

func TestToPySpark_Filter(t *testing.T) {
	tr := mapping.Transformation{
		Operation:  OpFilter,
		Expression: "status = 'active'",
	}
	got := ToPySpark(tr, "df")
	want := `df = df.filter("status = 'active'")`
	if got != want {
		t.Errorf("got:\n  %s\nwant:\n  %s", got, want)
	}
}

func TestToPySpark_Default(t *testing.T) {
	tr := mapping.Transformation{
		Operation:   OpDefault,
		SourceField: "status",
		Value:       "unknown",
	}
	got := ToPySpark(tr, "df")
	want := `df = df.withColumn("status", coalesce(col("status"), lit("unknown")))`
	if got != want {
		t.Errorf("got:\n  %s\nwant:\n  %s", got, want)
	}
}

func TestToPySpark_Default_Numeric(t *testing.T) {
	tr := mapping.Transformation{
		Operation:   OpDefault,
		SourceField: "count",
		Value:       "0",
	}
	got := ToPySpark(tr, "df")
	want := `df = df.withColumn("count", coalesce(col("count"), lit(0)))`
	if got != want {
		t.Errorf("got:\n  %s\nwant:\n  %s", got, want)
	}
}

func TestToPySpark_Exclude(t *testing.T) {
	tr := mapping.Transformation{
		Operation:   OpExclude,
		SourceField: "internal_notes",
	}
	got := ToPySpark(tr, "df")
	want := `df = df.drop("internal_notes")`
	if got != want {
		t.Errorf("got:\n  %s\nwant:\n  %s", got, want)
	}
}

func TestValidate_ValidOperations(t *testing.T) {
	tests := []struct {
		name string
		t    mapping.Transformation
	}{
		{"rename", mapping.Transformation{Operation: OpRename, SourceField: "a", TargetField: "b"}},
		{"compute", mapping.Transformation{Operation: OpCompute, TargetField: "x", Expression: "a + b"}},
		{"cast", mapping.Transformation{Operation: OpCast, SourceField: "a", TargetType: "int"}},
		{"filter", mapping.Transformation{Operation: OpFilter, Expression: "x > 0"}},
		{"default", mapping.Transformation{Operation: OpDefault, SourceField: "a", Value: "none"}},
		{"exclude", mapping.Transformation{Operation: OpExclude, SourceField: "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Validate(tt.t); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidate_InvalidOperation(t *testing.T) {
	tr := mapping.Transformation{Operation: "unknown"}
	err := Validate(tr)
	if err == nil {
		t.Error("expected error for unknown operation")
	}
}

func TestValidate_MissingFields(t *testing.T) {
	tests := []struct {
		name    string
		t       mapping.Transformation
		wantErr string
	}{
		{"rename no source", mapping.Transformation{Operation: OpRename, TargetField: "b"}, "source_field"},
		{"rename no target", mapping.Transformation{Operation: OpRename, SourceField: "a"}, "target_field"},
		{"compute no target", mapping.Transformation{Operation: OpCompute, Expression: "a+b"}, "target_field"},
		{"compute no expr", mapping.Transformation{Operation: OpCompute, TargetField: "x"}, "expression"},
		{"cast no source", mapping.Transformation{Operation: OpCast, TargetType: "int"}, "source_field"},
		{"cast no type", mapping.Transformation{Operation: OpCast, SourceField: "a"}, "target_type"},
		{"filter no expr", mapping.Transformation{Operation: OpFilter}, "expression"},
		{"default no source", mapping.Transformation{Operation: OpDefault, Value: "x"}, "source_field"},
		{"default no value", mapping.Transformation{Operation: OpDefault, SourceField: "a"}, "value"},
		{"exclude no source", mapping.Transformation{Operation: OpExclude}, "source_field"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.t)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidateAll_ConflictRenameExclude(t *testing.T) {
	transforms := []mapping.Transformation{
		{Operation: OpRename, SourceField: "a", TargetField: "b"},
		{Operation: OpExclude, SourceField: "a"},
	}
	err := ValidateAll(transforms)
	if err == nil {
		t.Error("expected conflict error")
	}
}

func TestValidateAll_ConflictExcludeRename(t *testing.T) {
	transforms := []mapping.Transformation{
		{Operation: OpExclude, SourceField: "a"},
		{Operation: OpRename, SourceField: "a", TargetField: "b"},
	}
	err := ValidateAll(transforms)
	if err == nil {
		t.Error("expected conflict error")
	}
}

func TestToPySparkAll_Ordering(t *testing.T) {
	// Mix up the order; should come out: filter, compute, rename, cast, default, exclude
	transforms := []mapping.Transformation{
		{Operation: OpExclude, SourceField: "temp"},
		{Operation: OpRename, SourceField: "a", TargetField: "b"},
		{Operation: OpFilter, Expression: "x > 0"},
		{Operation: OpCompute, TargetField: "y", Expression: "x * 2"},
		{Operation: OpDefault, SourceField: "z", Value: "0"},
		{Operation: OpCast, SourceField: "p", TargetType: "double"},
	}
	lines := ToPySparkAll(transforms, "df")
	if len(lines) != 6 {
		t.Fatalf("expected 6 lines, got %d", len(lines))
	}

	// Verify ordering by checking the operation type in each line
	if !strings.Contains(lines[0], "filter") {
		t.Errorf("line 0 should be filter, got: %s", lines[0])
	}
	if !strings.Contains(lines[1], "expr") {
		t.Errorf("line 1 should be compute (expr), got: %s", lines[1])
	}
	if !strings.Contains(lines[2], "withColumnRenamed") {
		t.Errorf("line 2 should be rename, got: %s", lines[2])
	}
	if !strings.Contains(lines[3], "cast") {
		t.Errorf("line 3 should be cast, got: %s", lines[3])
	}
	if !strings.Contains(lines[4], "coalesce") {
		t.Errorf("line 4 should be default (coalesce), got: %s", lines[4])
	}
	if !strings.Contains(lines[5], "drop") {
		t.Errorf("line 5 should be exclude (drop), got: %s", lines[5])
	}
}

func TestIsNumber(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"0", true},
		{"123", true},
		{"-1", true},
		{"3.14", true},
		{"-0.5", true},
		{"", false},
		{"abc", false},
		{"12a", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isNumber(tt.input); got != tt.want {
				t.Errorf("isNumber(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
