package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndLoadYAML(t *testing.T) {
	s := &Schema{
		DatabaseType: "postgresql",
		Host:         "localhost",
		Database:     "testdb",
		SchemaName:   "public",
		Tables: []Table{
			{
				Name:      "users",
				RowCount:  1000,
				SizeBytes: 65536,
				Columns: []Column{
					{Name: "id", DataType: "integer", Nullable: false, IsSequence: true},
					{Name: "name", DataType: "character varying", Nullable: false},
					{Name: "email", DataType: "character varying", Nullable: true},
				},
				PrimaryKey: &PrimaryKey{Name: "users_pkey", Columns: []string{"id"}},
			},
			{
				Name:      "posts",
				RowCount:  5000,
				SizeBytes: 262144,
				Columns: []Column{
					{Name: "id", DataType: "integer", Nullable: false},
					{Name: "user_id", DataType: "integer", Nullable: false},
					{Name: "title", DataType: "text", Nullable: false},
				},
				ForeignKeys: []ForeignKey{
					{
						Name:              "fk_posts_user",
						Columns:           []string{"user_id"},
						ReferencedTable:   "users",
						ReferencedColumns: []string{"id"},
					},
				},
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "schema.yaml")

	if err := s.WriteYAML(path); err != nil {
		t.Fatalf("WriteYAML: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("schema file not created: %v", err)
	}

	loaded, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}

	if loaded.DatabaseType != "postgresql" {
		t.Errorf("DatabaseType = %q, want %q", loaded.DatabaseType, "postgresql")
	}
	if len(loaded.Tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(loaded.Tables))
	}
	if loaded.Tables[0].Name != "users" {
		t.Errorf("first table = %q, want %q", loaded.Tables[0].Name, "users")
	}
	if loaded.Tables[0].RowCount != 1000 {
		t.Errorf("users RowCount = %d, want 1000", loaded.Tables[0].RowCount)
	}
	if len(loaded.Tables[0].Columns) != 3 {
		t.Errorf("users columns = %d, want 3", len(loaded.Tables[0].Columns))
	}
	if loaded.Tables[0].PrimaryKey == nil {
		t.Fatal("users primary key should not be nil")
	}
	if len(loaded.Tables[1].ForeignKeys) != 1 {
		t.Errorf("posts FKs = %d, want 1", len(loaded.Tables[1].ForeignKeys))
	}
	if loaded.Tables[1].ForeignKeys[0].ReferencedTable != "users" {
		t.Errorf("FK ref table = %q, want %q", loaded.Tables[1].ForeignKeys[0].ReferencedTable, "users")
	}
}

func TestLoadYAML_NotFound(t *testing.T) {
	_, err := LoadYAML("/nonexistent/path/schema.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestSummary(t *testing.T) {
	s := &Schema{
		Tables: []Table{
			{Name: "a", RowCount: 100, SizeBytes: 1024, Columns: []Column{{Name: "id"}}, ForeignKeys: []ForeignKey{{Name: "fk1"}}},
			{Name: "b", RowCount: 200, SizeBytes: 2048, Columns: []Column{{Name: "id"}, {Name: "val"}}},
		},
	}
	summary := s.Summary()
	if summary == "" {
		t.Error("summary should not be empty")
	}
}
