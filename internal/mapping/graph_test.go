package mapping

import (
	"testing"

	"github.com/reloquent/reloquent/internal/schema"
)

func graphTestTables() []schema.Table {
	return []schema.Table{
		{
			Name: "customers",
			Columns: []schema.Column{
				{Name: "id", DataType: "integer"},
				{Name: "name", DataType: "text"},
			},
			PrimaryKey: &schema.PrimaryKey{Name: "pk_customers", Columns: []string{"id"}},
		},
		{
			Name: "orders",
			Columns: []schema.Column{
				{Name: "id", DataType: "integer"},
				{Name: "customer_id", DataType: "integer"},
			},
			ForeignKeys: []schema.ForeignKey{
				{Name: "fk_orders_customer", Columns: []string{"customer_id"}, ReferencedTable: "customers", ReferencedColumns: []string{"id"}},
			},
		},
		{
			Name: "order_items",
			Columns: []schema.Column{
				{Name: "id", DataType: "integer"},
				{Name: "order_id", DataType: "integer"},
				{Name: "product_id", DataType: "integer"},
			},
			ForeignKeys: []schema.ForeignKey{
				{Name: "fk_items_order", Columns: []string{"order_id"}, ReferencedTable: "orders", ReferencedColumns: []string{"id"}},
				{Name: "fk_items_product", Columns: []string{"product_id"}, ReferencedTable: "products", ReferencedColumns: []string{"id"}},
			},
		},
		{
			Name: "products",
			Columns: []schema.Column{
				{Name: "id", DataType: "integer"},
				{Name: "name", DataType: "text"},
			},
			PrimaryKey: &schema.PrimaryKey{Name: "pk_products", Columns: []string{"id"}},
		},
	}
}

func TestNewFKGraph(t *testing.T) {
	g := NewFKGraph(graphTestTables())
	if len(g.edges) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(g.edges))
	}
}

func TestSelfReferences(t *testing.T) {
	tables := []schema.Table{
		{
			Name: "employees",
			Columns: []schema.Column{
				{Name: "id", DataType: "integer"},
				{Name: "manager_id", DataType: "integer"},
			},
			ForeignKeys: []schema.ForeignKey{
				{Name: "fk_emp_manager", Columns: []string{"manager_id"}, ReferencedTable: "employees", ReferencedColumns: []string{"id"}},
			},
		},
	}
	g := NewFKGraph(tables)
	selfRefs := g.SelfReferences()
	if len(selfRefs) != 1 {
		t.Fatalf("expected 1 self-reference, got %d", len(selfRefs))
	}
	if selfRefs[0].ChildTable != "employees" || selfRefs[0].ParentTable != "employees" {
		t.Errorf("expected employees self-ref, got %s->%s", selfRefs[0].ChildTable, selfRefs[0].ParentTable)
	}
}

func TestSelfReferences_None(t *testing.T) {
	g := NewFKGraph(graphTestTables())
	selfRefs := g.SelfReferences()
	if len(selfRefs) != 0 {
		t.Fatalf("expected 0 self-references, got %d", len(selfRefs))
	}
}

func TestDetectCycles_ThreeNodeCycle(t *testing.T) {
	tables := []schema.Table{
		{
			Name: "a",
			ForeignKeys: []schema.ForeignKey{
				{Name: "fk_a_b", Columns: []string{"b_id"}, ReferencedTable: "b", ReferencedColumns: []string{"id"}},
			},
		},
		{
			Name: "b",
			ForeignKeys: []schema.ForeignKey{
				{Name: "fk_b_c", Columns: []string{"c_id"}, ReferencedTable: "c", ReferencedColumns: []string{"id"}},
			},
		},
		{
			Name: "c",
			ForeignKeys: []schema.ForeignKey{
				{Name: "fk_c_a", Columns: []string{"a_id"}, ReferencedTable: "a", ReferencedColumns: []string{"id"}},
			},
		},
	}
	g := NewFKGraph(tables)
	cycles := g.DetectCycles()
	if len(cycles) == 0 {
		t.Fatal("expected at least 1 cycle")
	}
	// The cycle should have 3 nodes
	found3 := false
	for _, c := range cycles {
		if len(c) == 3 {
			found3 = true
		}
	}
	if !found3 {
		t.Error("expected a 3-node cycle")
	}
}

func TestDetectCycles_NoCycle(t *testing.T) {
	g := NewFKGraph(graphTestTables())
	cycles := g.DetectCycles()
	if len(cycles) != 0 {
		t.Fatalf("expected no cycles, got %d", len(cycles))
	}
}

func TestJoinTables_Detected(t *testing.T) {
	// Classic M2M: students <-> courses via enrollments
	tables := []schema.Table{
		{
			Name:    "students",
			Columns: []schema.Column{{Name: "id", DataType: "integer"}},
		},
		{
			Name:    "courses",
			Columns: []schema.Column{{Name: "id", DataType: "integer"}},
		},
		{
			Name: "enrollments",
			Columns: []schema.Column{
				{Name: "student_id", DataType: "integer"},
				{Name: "course_id", DataType: "integer"},
			},
			ForeignKeys: []schema.ForeignKey{
				{Name: "fk_enroll_student", Columns: []string{"student_id"}, ReferencedTable: "students", ReferencedColumns: []string{"id"}},
				{Name: "fk_enroll_course", Columns: []string{"course_id"}, ReferencedTable: "courses", ReferencedColumns: []string{"id"}},
			},
		},
	}
	g := NewFKGraph(tables)
	jts := g.JoinTables()
	if len(jts) != 1 {
		t.Fatalf("expected 1 join table, got %d", len(jts))
	}
	if jts[0].JoinTable != "enrollments" {
		t.Errorf("expected join table 'enrollments', got %q", jts[0].JoinTable)
	}
}

func TestJoinTables_FalsePositive_ManyColumns(t *testing.T) {
	// Has 2 FKs but too many non-FK columns — not a join table
	tables := []schema.Table{
		{Name: "a", Columns: []schema.Column{{Name: "id", DataType: "integer"}}},
		{Name: "b", Columns: []schema.Column{{Name: "id", DataType: "integer"}}},
		{
			Name: "c",
			Columns: []schema.Column{
				{Name: "a_id", DataType: "integer"},
				{Name: "b_id", DataType: "integer"},
				{Name: "col1", DataType: "text"},
				{Name: "col2", DataType: "text"},
				{Name: "col3", DataType: "text"},
			},
			ForeignKeys: []schema.ForeignKey{
				{Name: "fk_c_a", Columns: []string{"a_id"}, ReferencedTable: "a", ReferencedColumns: []string{"id"}},
				{Name: "fk_c_b", Columns: []string{"b_id"}, ReferencedTable: "b", ReferencedColumns: []string{"id"}},
			},
		},
	}
	g := NewFKGraph(tables)
	jts := g.JoinTables()
	if len(jts) != 0 {
		t.Fatalf("expected 0 join tables (false positive), got %d", len(jts))
	}
}

func TestJoinTables_FalsePositive_Referenced(t *testing.T) {
	// Has 2 FKs but is referenced by another table — not a join table
	tables := []schema.Table{
		{Name: "a", Columns: []schema.Column{{Name: "id", DataType: "integer"}}},
		{Name: "b", Columns: []schema.Column{{Name: "id", DataType: "integer"}}},
		{
			Name: "c",
			Columns: []schema.Column{
				{Name: "id", DataType: "integer"},
				{Name: "a_id", DataType: "integer"},
				{Name: "b_id", DataType: "integer"},
			},
			ForeignKeys: []schema.ForeignKey{
				{Name: "fk_c_a", Columns: []string{"a_id"}, ReferencedTable: "a", ReferencedColumns: []string{"id"}},
				{Name: "fk_c_b", Columns: []string{"b_id"}, ReferencedTable: "b", ReferencedColumns: []string{"id"}},
			},
		},
		{
			Name: "d",
			Columns: []schema.Column{
				{Name: "c_id", DataType: "integer"},
			},
			ForeignKeys: []schema.ForeignKey{
				{Name: "fk_d_c", Columns: []string{"c_id"}, ReferencedTable: "c", ReferencedColumns: []string{"id"}},
			},
		},
	}
	g := NewFKGraph(tables)
	jts := g.JoinTables()
	if len(jts) != 0 {
		t.Fatalf("expected 0 join tables (referenced by d), got %d", len(jts))
	}
}

func TestTopologicalSort(t *testing.T) {
	// order_items embedded in orders, orders embedded in customers
	embeds := map[string]string{
		"order_items": "orders",
		"orders":      "customers",
	}
	g := NewFKGraph(graphTestTables())
	sorted, err := g.TopologicalSort(embeds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// order_items should come before orders, orders before customers
	indexOf := func(name string) int {
		for i, n := range sorted {
			if n == name {
				return i
			}
		}
		return -1
	}

	if indexOf("order_items") > indexOf("orders") {
		t.Error("order_items should come before orders")
	}
	if indexOf("orders") > indexOf("customers") {
		t.Error("orders should come before customers")
	}
}

func TestTopologicalSort_DiamondDependency(t *testing.T) {
	// Both B and C embedded in A, D embedded in both B and C
	embeds := map[string]string{
		"b": "a",
		"c": "a",
	}
	tables := []schema.Table{
		{Name: "a"},
		{Name: "b", ForeignKeys: []schema.ForeignKey{{Name: "fk_b_a", Columns: []string{"a_id"}, ReferencedTable: "a", ReferencedColumns: []string{"id"}}}},
		{Name: "c", ForeignKeys: []schema.ForeignKey{{Name: "fk_c_a", Columns: []string{"a_id"}, ReferencedTable: "a", ReferencedColumns: []string{"id"}}}},
	}
	g := NewFKGraph(tables)
	sorted, err := g.TopologicalSort(embeds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// b and c should both come before a
	indexOf := func(name string) int {
		for i, n := range sorted {
			if n == name {
				return i
			}
		}
		return -1
	}
	if indexOf("b") > indexOf("a") {
		t.Error("b should come before a")
	}
	if indexOf("c") > indexOf("a") {
		t.Error("c should come before a")
	}
}

func TestNestingDepth(t *testing.T) {
	tests := []struct {
		name   string
		embeds map[string]string
		want   int
	}{
		{
			name:   "empty",
			embeds: map[string]string{},
			want:   0,
		},
		{
			name:   "single level",
			embeds: map[string]string{"orders": "customers"},
			want:   1,
		},
		{
			name:   "two levels",
			embeds: map[string]string{"order_items": "orders", "orders": "customers"},
			want:   2,
		},
		{
			name:   "three levels",
			embeds: map[string]string{"d": "c", "c": "b", "b": "a"},
			want:   3,
		},
	}

	g := NewFKGraph(graphTestTables())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.NestingDepth(tt.embeds)
			if got != tt.want {
				t.Errorf("NestingDepth() = %d, want %d", got, tt.want)
			}
		})
	}
}
