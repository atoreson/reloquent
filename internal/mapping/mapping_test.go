package mapping

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndLoadYAML(t *testing.T) {
	m := &Mapping{
		Collections: []Collection{
			{
				Name:        "customers",
				SourceTable: "customers",
				Embedded: []Embedded{
					{
						SourceTable:  "orders",
						FieldName:    "orders",
						Relationship: "array",
						JoinColumn:   "customer_id",
						ParentColumn: "id",
					},
				},
			},
			{
				Name:        "products",
				SourceTable: "products",
				References: []Reference{
					{
						SourceTable:  "categories",
						FieldName:    "category",
						JoinColumn:   "category_id",
						ParentColumn: "id",
					},
				},
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "mapping.yaml")

	if err := m.WriteYAML(path); err != nil {
		t.Fatalf("WriteYAML: %v", err)
	}

	// File should exist
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}

	loaded, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}

	if len(loaded.Collections) != 2 {
		t.Fatalf("expected 2 collections, got %d", len(loaded.Collections))
	}

	// Check first collection
	c := loaded.Collections[0]
	if c.Name != "customers" {
		t.Errorf("expected name 'customers', got %q", c.Name)
	}
	if c.SourceTable != "customers" {
		t.Errorf("expected source_table 'customers', got %q", c.SourceTable)
	}
	if len(c.Embedded) != 1 {
		t.Fatalf("expected 1 embedded, got %d", len(c.Embedded))
	}
	if c.Embedded[0].SourceTable != "orders" {
		t.Errorf("expected embedded source_table 'orders', got %q", c.Embedded[0].SourceTable)
	}
	if c.Embedded[0].Relationship != "array" {
		t.Errorf("expected relationship 'array', got %q", c.Embedded[0].Relationship)
	}
	if c.Embedded[0].JoinColumn != "customer_id" {
		t.Errorf("expected join_column 'customer_id', got %q", c.Embedded[0].JoinColumn)
	}
	if c.Embedded[0].ParentColumn != "id" {
		t.Errorf("expected parent_column 'id', got %q", c.Embedded[0].ParentColumn)
	}

	// Check second collection references
	p := loaded.Collections[1]
	if p.Name != "products" {
		t.Errorf("expected name 'products', got %q", p.Name)
	}
	if len(p.References) != 1 {
		t.Fatalf("expected 1 reference, got %d", len(p.References))
	}
	if p.References[0].SourceTable != "categories" {
		t.Errorf("expected reference source_table 'categories', got %q", p.References[0].SourceTable)
	}
}

func TestLoadYAML_NotFound(t *testing.T) {
	_, err := LoadYAML("/nonexistent/path/mapping.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestWriteYAML_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "mapping.yaml")

	m := &Mapping{
		Collections: []Collection{
			{Name: "test", SourceTable: "test"},
		},
	}

	if err := m.WriteYAML(path); err != nil {
		t.Fatalf("WriteYAML should create subdirectories: %v", err)
	}

	loaded, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}
	if len(loaded.Collections) != 1 {
		t.Errorf("expected 1 collection, got %d", len(loaded.Collections))
	}
}

func TestWriteAndLoadYAML_NestedEmbedded(t *testing.T) {
	m := &Mapping{
		Collections: []Collection{
			{
				Name:        "customers",
				SourceTable: "customers",
				Embedded: []Embedded{
					{
						SourceTable:  "orders",
						FieldName:    "orders",
						Relationship: "array",
						JoinColumn:   "customer_id",
						ParentColumn: "id",
						Embedded: []Embedded{
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
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "nested.yaml")

	if err := m.WriteYAML(path); err != nil {
		t.Fatalf("WriteYAML: %v", err)
	}

	loaded, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}

	if len(loaded.Collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(loaded.Collections))
	}

	c := loaded.Collections[0]
	if len(c.Embedded) != 1 {
		t.Fatalf("expected 1 embedded, got %d", len(c.Embedded))
	}
	if c.Embedded[0].SourceTable != "orders" {
		t.Errorf("expected orders, got %q", c.Embedded[0].SourceTable)
	}
	if len(c.Embedded[0].Embedded) != 1 {
		t.Fatalf("expected 1 nested embedded, got %d", len(c.Embedded[0].Embedded))
	}
	nested := c.Embedded[0].Embedded[0]
	if nested.SourceTable != "order_items" {
		t.Errorf("expected order_items, got %q", nested.SourceTable)
	}
	if nested.FieldName != "items" {
		t.Errorf("expected field name 'items', got %q", nested.FieldName)
	}
}

func TestWriteAndLoadYAML_WithTransformations(t *testing.T) {
	m := &Mapping{
		Collections: []Collection{
			{
				Name:        "users",
				SourceTable: "users",
				Transformations: []Transformation{
					{
						SourceField: "first_name",
						Operation:   "rename",
						TargetField: "firstName",
					},
				},
				Embedded: []Embedded{
					{
						SourceTable:  "addresses",
						FieldName:    "addresses",
						Relationship: "array",
						JoinColumn:   "user_id",
						ParentColumn: "id",
						Transformations: []Transformation{
							{
								SourceField: "internal_code",
								Operation:   "exclude",
							},
						},
					},
				},
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "transforms.yaml")

	if err := m.WriteYAML(path); err != nil {
		t.Fatalf("WriteYAML: %v", err)
	}

	loaded, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}

	c := loaded.Collections[0]
	if len(c.Transformations) != 1 {
		t.Fatalf("expected 1 collection transform, got %d", len(c.Transformations))
	}
	if c.Transformations[0].Operation != "rename" {
		t.Errorf("expected rename, got %q", c.Transformations[0].Operation)
	}

	if len(c.Embedded) != 1 {
		t.Fatalf("expected 1 embedded, got %d", len(c.Embedded))
	}
	if len(c.Embedded[0].Transformations) != 1 {
		t.Fatalf("expected 1 embedded transform, got %d", len(c.Embedded[0].Transformations))
	}
	if c.Embedded[0].Transformations[0].Operation != "exclude" {
		t.Errorf("expected exclude, got %q", c.Embedded[0].Transformations[0].Operation)
	}
}

func TestWriteAndLoadYAML_EmbedSingle(t *testing.T) {
	m := &Mapping{
		Collections: []Collection{
			{
				Name:        "orders",
				SourceTable: "orders",
				Embedded: []Embedded{
					{
						SourceTable:  "shipping_address",
						FieldName:    "shipping_address",
						Relationship: "single",
						JoinColumn:   "order_id",
						ParentColumn: "id",
					},
				},
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "mapping.yaml")

	if err := m.WriteYAML(path); err != nil {
		t.Fatalf("WriteYAML: %v", err)
	}

	loaded, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}

	if loaded.Collections[0].Embedded[0].Relationship != "single" {
		t.Errorf("expected relationship 'single', got %q",
			loaded.Collections[0].Embedded[0].Relationship)
	}
}
