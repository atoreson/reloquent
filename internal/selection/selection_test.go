package selection

import (
	"testing"

	"github.com/reloquent/reloquent/internal/schema"
)

func testTables() []schema.Table {
	return []schema.Table{
		{Name: "customers", RowCount: 1000, SizeBytes: 65536},
		{Name: "orders", RowCount: 5000, SizeBytes: 262144, ForeignKeys: []schema.ForeignKey{
			{Name: "fk_orders_customer", Columns: []string{"customer_id"}, ReferencedTable: "customers", ReferencedColumns: []string{"id"}},
		}},
		{Name: "order_items", RowCount: 20000, SizeBytes: 524288, ForeignKeys: []schema.ForeignKey{
			{Name: "fk_items_order", Columns: []string{"order_id"}, ReferencedTable: "orders", ReferencedColumns: []string{"id"}},
			{Name: "fk_items_product", Columns: []string{"product_id"}, ReferencedTable: "products", ReferencedColumns: []string{"id"}},
		}},
		{Name: "products", RowCount: 500, SizeBytes: 32768},
		{Name: "audit_log", RowCount: 100000, SizeBytes: 1048576},
	}
}

func TestFilterByPattern(t *testing.T) {
	tables := testTables()

	tests := []struct {
		name    string
		pattern string
		want    int
	}{
		{"wildcard all", "*", 5},
		{"prefix match", "order*", 2},
		{"suffix match", "*log", 1},
		{"exact match", "customers", 1},
		{"no match", "nonexistent", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterByPattern(tables, tt.pattern)
			if len(got) != tt.want {
				t.Errorf("FilterByPattern(%q) returned %d tables, want %d", tt.pattern, len(got), tt.want)
			}
		})
	}
}

func TestTotalSize(t *testing.T) {
	tables := testTables()[:2] // customers + orders
	got := TotalSize(tables)
	want := int64(65536 + 262144)
	if got != want {
		t.Errorf("TotalSize = %d, want %d", got, want)
	}
}

func TestTotalRows(t *testing.T) {
	tables := testTables()[:2]
	got := TotalRows(tables)
	want := int64(1000 + 5000)
	if got != want {
		t.Errorf("TotalRows = %d, want %d", got, want)
	}
}

func TestFindOrphanedReferences_NoOrphans(t *testing.T) {
	tables := testTables() // all tables present
	orphans := FindOrphanedReferences(tables)
	if len(orphans) != 0 {
		t.Errorf("expected no orphans with all tables selected, got %d", len(orphans))
	}
}

func TestFindOrphanedReferences_WithOrphans(t *testing.T) {
	// Select only orders and order_items — customers and products are missing
	tables := []schema.Table{
		testTables()[1], // orders (refs customers)
		testTables()[2], // order_items (refs orders, products)
	}
	orphans := FindOrphanedReferences(tables)
	if len(orphans) != 2 {
		t.Fatalf("expected 2 orphans, got %d: %v", len(orphans), orphans)
	}

	// Should find: orders->customers and order_items->products
	refTables := map[string]bool{}
	for _, o := range orphans {
		refTables[o.ReferencedTable] = true
	}
	if !refTables["customers"] {
		t.Error("expected orphan reference to 'customers'")
	}
	if !refTables["products"] {
		t.Error("expected orphan reference to 'products'")
	}
}

func TestTotalSizeEmpty(t *testing.T) {
	got := TotalSize(nil)
	if got != 0 {
		t.Errorf("TotalSize(nil) = %d, want 0", got)
	}
}
