package typemap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPostgresMapping(t *testing.T) {
	tm := DefaultPostgres()

	tests := []struct {
		sourceType string
		want       BSONType
	}{
		{"integer", BSONNumberLong},
		{"bigint", BSONNumberLong},
		{"text", BSONString},
		{"boolean", BSONBoolean},
		{"timestamp with time zone", BSONISODate},
		{"bytea", BSONBinData},
		{"jsonb", BSONDocument},
		{"numeric", BSONDecimal128},
		{"double precision", BSONDouble},
	}

	for _, tt := range tests {
		t.Run(tt.sourceType, func(t *testing.T) {
			got := tm.Resolve(tt.sourceType)
			if got != tt.want {
				t.Errorf("Resolve(%q) = %s, want %s", tt.sourceType, got, tt.want)
			}
		})
	}
}

func TestUnknownTypeFallsBackToString(t *testing.T) {
	tm := DefaultPostgres()
	got := tm.Resolve("some_unknown_type")
	if got != BSONString {
		t.Errorf("expected fallback to String, got %s", got)
	}
}

func TestDefaultOracleMapping(t *testing.T) {
	tm := DefaultOracle()

	if tm.Resolve("NUMBER") != BSONNumberLong {
		t.Error("expected NUMBER -> NumberLong")
	}
	if tm.Resolve("VARCHAR2") != BSONString {
		t.Error("expected VARCHAR2 -> String")
	}
	if tm.Resolve("BLOB") != BSONBinData {
		t.Error("expected BLOB -> BinData")
	}
}

func TestForDatabase(t *testing.T) {
	pg := ForDatabase("postgresql")
	if pg.Resolve("integer") != BSONNumberLong {
		t.Error("ForDatabase(postgresql) should return PostgreSQL defaults")
	}

	ora := ForDatabase("oracle")
	if ora.Resolve("NUMBER") != BSONNumberLong {
		t.Error("ForDatabase(oracle) should return Oracle defaults")
	}
}

func TestOverride(t *testing.T) {
	tm := ForDatabase("postgresql")

	// Override integer mapping
	tm.Override("integer", BSONDecimal128)
	if tm.Resolve("integer") != BSONDecimal128 {
		t.Errorf("expected Decimal128 after override, got %s", tm.Resolve("integer"))
	}
	if !tm.IsOverridden("integer") {
		t.Error("integer should be marked as overridden")
	}

	// Restore default
	tm.RestoreDefault("integer")
	if tm.Resolve("integer") != BSONNumberLong {
		t.Errorf("expected NumberLong after restore, got %s", tm.Resolve("integer"))
	}
	if tm.IsOverridden("integer") {
		t.Error("integer should not be overridden after restore")
	}
}

func TestOverride_SameAsDefault(t *testing.T) {
	tm := ForDatabase("postgresql")

	// Overriding to the same value as default should not mark as override
	tm.Override("integer", BSONNumberLong)
	if tm.IsOverridden("integer") {
		t.Error("overriding to default value should not be tracked as override")
	}
}

func TestWriteAndLoadYAML(t *testing.T) {
	tm := ForDatabase("postgresql")
	tm.Override("integer", BSONDecimal128)

	dir := t.TempDir()
	path := filepath.Join(dir, "typemap.yaml")

	if err := tm.WriteYAML(path); err != nil {
		t.Fatalf("WriteYAML: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}

	loaded, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}

	if loaded.Resolve("integer") != BSONDecimal128 {
		t.Errorf("loaded mapping: expected Decimal128, got %s", loaded.Resolve("integer"))
	}

	if loaded.Resolve("text") != BSONString {
		t.Errorf("loaded mapping: expected String for text, got %s", loaded.Resolve("text"))
	}
}

func TestLoadYAML_NotFound(t *testing.T) {
	_, err := LoadYAML("/nonexistent/typemap.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestSortedTypes(t *testing.T) {
	tm := DefaultPostgres()
	types := tm.SortedTypes()

	if len(types) == 0 {
		t.Fatal("expected non-empty sorted types")
	}

	// Check sorted order
	for i := 1; i < len(types); i++ {
		if types[i] < types[i-1] {
			t.Errorf("types not sorted: %s before %s", types[i-1], types[i])
		}
	}
}
