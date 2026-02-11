package indexes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/schema"
)

func TestInfer_PKToUniqueIndex(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name:       "users",
				PrimaryKey: &schema.PrimaryKey{Name: "pk_users", Columns: []string{"user_id"}},
			},
		},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "users", SourceTable: "users"},
		},
	}

	plan := Infer(s, m)
	found := false
	for _, ci := range plan.Indexes {
		if ci.Collection == "users" && ci.Index.Unique && len(ci.Index.Keys) == 1 && ci.Index.Keys[0].Field == "user_id" {
			found = true
		}
	}
	if !found {
		t.Error("expected unique index on users.user_id from PK")
	}
}

func TestInfer_PKSkipID(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name:       "users",
				PrimaryKey: &schema.PrimaryKey{Name: "pk_users", Columns: []string{"id"}},
			},
		},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "users", SourceTable: "users"},
		},
	}

	plan := Infer(s, m)
	for _, ci := range plan.Indexes {
		if ci.Collection == "users" && len(ci.Index.Keys) == 1 && ci.Index.Keys[0].Field == "id" {
			t.Error("should skip index on 'id' (maps to _id)")
		}
	}
}

func TestInfer_FKRefToIndex(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{Name: "orders"},
			{Name: "products"},
		},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{
				Name:        "orders",
				SourceTable: "orders",
				References: []mapping.Reference{
					{SourceTable: "products", FieldName: "product_ref", JoinColumn: "product_id", ParentColumn: "id"},
				},
			},
		},
	}

	plan := Infer(s, m)
	found := false
	for _, ci := range plan.Indexes {
		if ci.Collection == "orders" && len(ci.Index.Keys) == 1 && ci.Index.Keys[0].Field == "product_ref" {
			found = true
		}
	}
	if !found {
		t.Error("expected index on orders.product_ref from FK reference")
	}
}

func TestInfer_CompositePreservesOrder(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "order_items",
				Indexes: []schema.Index{
					{Name: "idx_compound", Columns: []string{"order_id", "product_id"}, Unique: true},
				},
			},
		},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "order_items", SourceTable: "order_items"},
		},
	}

	plan := Infer(s, m)
	found := false
	for _, ci := range plan.Indexes {
		if ci.Collection == "order_items" && len(ci.Index.Keys) == 2 {
			if ci.Index.Keys[0].Field == "order_id" && ci.Index.Keys[1].Field == "product_id" {
				found = true
				if !ci.Index.Unique {
					t.Error("composite index should be unique")
				}
			}
		}
	}
	if !found {
		t.Error("expected composite index preserving field order")
	}
}

func TestInfer_EmbeddedDotNotation(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{Name: "orders"},
			{
				Name: "order_items",
				Indexes: []schema.Index{
					{Name: "idx_product", Columns: []string{"product_id"}},
				},
			},
		},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{
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
			},
		},
	}

	plan := Infer(s, m)
	found := false
	for _, ci := range plan.Indexes {
		if ci.Collection == "orders" {
			for _, k := range ci.Index.Keys {
				if k.Field == "items.product_id" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected dot notation index items.product_id for embedded table")
	}
}

func TestInfer_Deduplication(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name:       "users",
				PrimaryKey: &schema.PrimaryKey{Name: "pk_users", Columns: []string{"user_id"}},
				Indexes: []schema.Index{
					{Name: "idx_user_id", Columns: []string{"user_id"}, Unique: true},
				},
			},
		},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "users", SourceTable: "users"},
		},
	}

	plan := Infer(s, m)
	count := 0
	for _, ci := range plan.Indexes {
		if ci.Collection == "users" && len(ci.Index.Keys) == 1 && ci.Index.Keys[0].Field == "user_id" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 index on user_id (deduplicated), got %d", count)
	}
}

func TestInfer_NoIDIndex(t *testing.T) {
	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "users",
				Indexes: []schema.Index{
					{Name: "idx_id", Columns: []string{"_id"}},
				},
			},
		},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "users", SourceTable: "users"},
		},
	}

	plan := Infer(s, m)
	for _, ci := range plan.Indexes {
		for _, k := range ci.Index.Keys {
			if k.Field == "_id" {
				t.Error("should never generate _id index")
			}
		}
	}
}

func TestIndexPlan_YAML_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "indexes.yaml")

	plan := &IndexPlan{
		Explanations: []string{"test explanation"},
	}

	if err := plan.WriteYAML(path); err != nil {
		t.Fatalf("WriteYAML: %v", err)
	}

	loaded, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}
	if len(loaded.Explanations) != 1 || loaded.Explanations[0] != "test explanation" {
		t.Error("round-trip failed")
	}
}

func TestLoadYAML_NotFound(t *testing.T) {
	_, err := LoadYAML("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestWriteYAML_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "indexes.yaml")

	plan := &IndexPlan{}
	if err := plan.WriteYAML(path); err != nil {
		t.Fatalf("WriteYAML should create subdirectory: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("file should exist after write")
	}
}
