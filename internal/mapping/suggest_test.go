package mapping

import (
	"testing"

	"github.com/reloquent/reloquent/internal/schema"
)

func TestSuggest_NoFKs(t *testing.T) {
	s := &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{Name: "users", RowCount: 100},
			{Name: "products", RowCount: 50},
		},
	}
	m := Suggest(s, []string{"users", "products"})
	if len(m.Collections) != 2 {
		t.Fatalf("collections = %d, want 2", len(m.Collections))
	}
	names := collectionNames(m)
	if !names["users"] || !names["products"] {
		t.Errorf("expected users and products collections, got %v", names)
	}
}

func TestSuggest_OneToMany_EmbedArray(t *testing.T) {
	s := &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{Name: "customers", RowCount: 100},
			{Name: "orders", RowCount: 500,
				ForeignKeys: []schema.ForeignKey{
					{Name: "fk_orders_cust", Columns: []string{"customer_id"},
						ReferencedTable: "customers", ReferencedColumns: []string{"id"}},
				},
			},
		},
	}
	m := Suggest(s, []string{"customers", "orders"})

	// customers is root; orders should be embedded as array (500/100 = 5 > 1)
	cust := findCollection(m, "customers")
	if cust == nil {
		t.Fatal("customers collection not found")
	}
	if len(cust.Embedded) != 1 {
		t.Fatalf("embedded count = %d, want 1", len(cust.Embedded))
	}
	if cust.Embedded[0].SourceTable != "orders" {
		t.Errorf("embedded source = %q, want orders", cust.Embedded[0].SourceTable)
	}
	if cust.Embedded[0].Relationship != "array" {
		t.Errorf("relationship = %q, want array", cust.Embedded[0].Relationship)
	}
	if cust.Embedded[0].JoinColumn != "customer_id" {
		t.Errorf("join column = %q", cust.Embedded[0].JoinColumn)
	}
	if cust.Embedded[0].ParentColumn != "id" {
		t.Errorf("parent column = %q", cust.Embedded[0].ParentColumn)
	}
}

func TestSuggest_OneToOne_EmbedSingle(t *testing.T) {
	s := &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{Name: "users", RowCount: 100},
			{Name: "profiles", RowCount: 100,
				ForeignKeys: []schema.ForeignKey{
					{Name: "fk_profiles_user", Columns: []string{"user_id"},
						ReferencedTable: "users", ReferencedColumns: []string{"id"}},
				},
			},
		},
	}
	m := Suggest(s, []string{"users", "profiles"})

	users := findCollection(m, "users")
	if users == nil {
		t.Fatal("users collection not found")
	}
	if len(users.Embedded) != 1 {
		t.Fatalf("embedded count = %d, want 1", len(users.Embedded))
	}
	if users.Embedded[0].Relationship != "single" {
		t.Errorf("relationship = %q, want single (ratio = 1.0)", users.Embedded[0].Relationship)
	}
}

func TestSuggest_OneToOne_ChildFewerRows(t *testing.T) {
	s := &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{Name: "users", RowCount: 200},
			{Name: "profiles", RowCount: 50,
				ForeignKeys: []schema.ForeignKey{
					{Name: "fk_profiles_user", Columns: []string{"user_id"},
						ReferencedTable: "users", ReferencedColumns: []string{"id"}},
				},
			},
		},
	}
	m := Suggest(s, []string{"users", "profiles"})

	users := findCollection(m, "users")
	if users == nil {
		t.Fatal("users collection not found")
	}
	if len(users.Embedded) != 1 {
		t.Fatalf("embedded count = %d, want 1", len(users.Embedded))
	}
	// 50/200 = 0.25 <= 1.0 → single
	if users.Embedded[0].Relationship != "single" {
		t.Errorf("relationship = %q, want single", users.Embedded[0].Relationship)
	}
}

func TestSuggest_SelfReference(t *testing.T) {
	s := &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{Name: "employees", RowCount: 50,
				ForeignKeys: []schema.ForeignKey{
					{Name: "fk_emp_manager", Columns: []string{"manager_id"},
						ReferencedTable: "employees", ReferencedColumns: []string{"id"}},
				},
			},
		},
	}
	m := Suggest(s, []string{"employees"})

	emp := findCollection(m, "employees")
	if emp == nil {
		t.Fatal("employees collection not found")
	}
	// Self-refs should not be embedded, they become references
	if len(emp.Embedded) != 0 {
		t.Errorf("embedded count = %d, want 0 (self-ref should be reference)", len(emp.Embedded))
	}
}

func TestSuggest_JoinTable_Skipped(t *testing.T) {
	s := &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{Name: "students", RowCount: 100,
				Columns: []schema.Column{{Name: "id", DataType: "integer"}},
			},
			{Name: "courses", RowCount: 50,
				Columns: []schema.Column{{Name: "id", DataType: "integer"}},
			},
			{Name: "enrollments", RowCount: 300,
				Columns: []schema.Column{
					{Name: "student_id", DataType: "integer"},
					{Name: "course_id", DataType: "integer"},
				},
				ForeignKeys: []schema.ForeignKey{
					{Name: "fk_enroll_student", Columns: []string{"student_id"},
						ReferencedTable: "students", ReferencedColumns: []string{"id"}},
					{Name: "fk_enroll_course", Columns: []string{"course_id"},
						ReferencedTable: "courses", ReferencedColumns: []string{"id"}},
				},
			},
		},
	}
	m := Suggest(s, []string{"students", "courses", "enrollments"})

	// enrollments is a join table — should not get its own collection
	names := collectionNames(m)
	if names["enrollments"] {
		t.Error("join table 'enrollments' should not have its own collection")
	}
	// students and courses should each get their own collection
	if !names["students"] {
		t.Error("students collection missing")
	}
	if !names["courses"] {
		t.Error("courses collection missing")
	}
}

func TestSuggest_SubsetSelection(t *testing.T) {
	s := &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{Name: "users", RowCount: 100},
			{Name: "orders", RowCount: 500,
				ForeignKeys: []schema.ForeignKey{
					{Name: "fk_orders_user", Columns: []string{"user_id"},
						ReferencedTable: "users", ReferencedColumns: []string{"id"}},
				},
			},
			{Name: "products", RowCount: 200},
		},
	}

	// Only select users and products — orders not selected
	m := Suggest(s, []string{"users", "products"})

	names := collectionNames(m)
	if !names["users"] || !names["products"] {
		t.Errorf("expected users and products, got %v", names)
	}
	if names["orders"] {
		t.Error("orders should not be in mapping (not selected)")
	}

	// users should have no embeds since orders isn't selected
	users := findCollection(m, "users")
	if users != nil && len(users.Embedded) != 0 {
		t.Errorf("users should have 0 embeds, got %d", len(users.Embedded))
	}
}

func TestSuggest_FKToUnselectedTable(t *testing.T) {
	s := &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{Name: "users", RowCount: 100},
			{Name: "orders", RowCount: 500,
				ForeignKeys: []schema.ForeignKey{
					{Name: "fk_orders_user", Columns: []string{"user_id"},
						ReferencedTable: "users", ReferencedColumns: []string{"id"}},
				},
			},
		},
	}

	// Only select orders — users is not selected, FK target is outside selection
	m := Suggest(s, []string{"orders"})

	names := collectionNames(m)
	if !names["orders"] {
		t.Error("orders collection missing")
	}
	// orders references users, but users isn't selected, so orders becomes a root
	orders := findCollection(m, "orders")
	if orders == nil {
		t.Fatal("orders collection not found")
	}
	if len(orders.Embedded) != 0 {
		t.Errorf("orders should have 0 embeds, got %d", len(orders.Embedded))
	}
}

func TestSuggest_EmptySchema(t *testing.T) {
	s := &schema.Schema{
		DatabaseType: "postgresql",
		Tables:       []schema.Table{},
	}
	m := Suggest(s, []string{})
	if len(m.Collections) != 0 {
		t.Errorf("collections = %d, want 0", len(m.Collections))
	}
}

func TestSuggest_MultiLevelHierarchy(t *testing.T) {
	s := &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{Name: "customers", RowCount: 100},
			{Name: "orders", RowCount: 500,
				ForeignKeys: []schema.ForeignKey{
					{Name: "fk_orders_cust", Columns: []string{"customer_id"},
						ReferencedTable: "customers", ReferencedColumns: []string{"id"}},
				},
			},
			{Name: "order_items", RowCount: 2000,
				ForeignKeys: []schema.ForeignKey{
					{Name: "fk_items_order", Columns: []string{"order_id"},
						ReferencedTable: "orders", ReferencedColumns: []string{"id"}},
				},
			},
		},
	}

	m := Suggest(s, []string{"customers", "orders", "order_items"})

	// customers is the root (no outgoing FKs)
	cust := findCollection(m, "customers")
	if cust == nil {
		t.Fatal("customers collection not found")
	}
	// orders should be embedded in customers
	if len(cust.Embedded) != 1 {
		t.Fatalf("customers.embedded = %d, want 1", len(cust.Embedded))
	}
	if cust.Embedded[0].SourceTable != "orders" {
		t.Errorf("embedded = %q, want orders", cust.Embedded[0].SourceTable)
	}
}

func TestSuggest_ZeroRowCounts(t *testing.T) {
	s := &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{Name: "parent", RowCount: 0},
			{Name: "child", RowCount: 0,
				ForeignKeys: []schema.ForeignKey{
					{Name: "fk_child_parent", Columns: []string{"parent_id"},
						ReferencedTable: "parent", ReferencedColumns: []string{"id"}},
				},
			},
		},
	}
	m := Suggest(s, []string{"parent", "child"})

	parent := findCollection(m, "parent")
	if parent == nil {
		t.Fatal("parent collection not found")
	}
	if len(parent.Embedded) != 1 {
		t.Fatalf("embedded count = %d, want 1", len(parent.Embedded))
	}
	// With zero rows, ratio calculation is skipped; default is "array"
	if parent.Embedded[0].Relationship != "array" {
		t.Errorf("relationship = %q, want array (zero row counts)", parent.Embedded[0].Relationship)
	}
}

func TestSuggest_MultipleChildrenSameParent(t *testing.T) {
	s := &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{Name: "users", RowCount: 100},
			{Name: "orders", RowCount: 500,
				ForeignKeys: []schema.ForeignKey{
					{Name: "fk_orders_user", Columns: []string{"user_id"},
						ReferencedTable: "users", ReferencedColumns: []string{"id"}},
				},
			},
			{Name: "addresses", RowCount: 80,
				ForeignKeys: []schema.ForeignKey{
					{Name: "fk_addr_user", Columns: []string{"user_id"},
						ReferencedTable: "users", ReferencedColumns: []string{"id"}},
				},
			},
		},
	}
	m := Suggest(s, []string{"users", "orders", "addresses"})

	users := findCollection(m, "users")
	if users == nil {
		t.Fatal("users collection not found")
	}
	if len(users.Embedded) != 2 {
		t.Fatalf("embedded count = %d, want 2", len(users.Embedded))
	}

	embeddedNames := make(map[string]string)
	for _, e := range users.Embedded {
		embeddedNames[e.SourceTable] = e.Relationship
	}
	// orders: 500/100 = 5 > 1 → array
	if embeddedNames["orders"] != "array" {
		t.Errorf("orders relationship = %q, want array", embeddedNames["orders"])
	}
	// addresses: 80/100 = 0.8 <= 1 → single
	if embeddedNames["addresses"] != "single" {
		t.Errorf("addresses relationship = %q, want single", embeddedNames["addresses"])
	}
}

func TestSuggest_CollectionFieldNames(t *testing.T) {
	s := &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{Name: "users", RowCount: 100},
			{Name: "orders", RowCount: 500,
				ForeignKeys: []schema.ForeignKey{
					{Name: "fk_orders_user", Columns: []string{"user_id"},
						ReferencedTable: "users", ReferencedColumns: []string{"id"}},
				},
			},
		},
	}
	m := Suggest(s, []string{"users", "orders"})

	users := findCollection(m, "users")
	if users == nil {
		t.Fatal("users collection not found")
	}
	// Collection name should match source table
	if users.Name != "users" {
		t.Errorf("collection name = %q", users.Name)
	}
	if users.SourceTable != "users" {
		t.Errorf("source table = %q", users.SourceTable)
	}
	// Embedded field name should be the child table name
	if len(users.Embedded) > 0 && users.Embedded[0].FieldName != "orders" {
		t.Errorf("field name = %q, want orders", users.Embedded[0].FieldName)
	}
}

func TestSuggest_AllCyclicFKs_FallbackToRoots(t *testing.T) {
	// All tables have outgoing FKs (circular) — the fallback should use all as roots
	s := &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{Name: "a", RowCount: 10,
				ForeignKeys: []schema.ForeignKey{
					{Name: "fk_a_b", Columns: []string{"b_id"},
						ReferencedTable: "b", ReferencedColumns: []string{"id"}},
				},
			},
			{Name: "b", RowCount: 10,
				ForeignKeys: []schema.ForeignKey{
					{Name: "fk_b_a", Columns: []string{"a_id"},
						ReferencedTable: "a", ReferencedColumns: []string{"id"}},
				},
			},
		},
	}
	m := Suggest(s, []string{"a", "b"})

	// Both have outgoing FKs, neither is a pure root. Should still produce collections.
	if len(m.Collections) == 0 {
		t.Fatal("expected at least 1 collection")
	}
	names := collectionNames(m)
	if !names["a"] && !names["b"] {
		t.Errorf("expected at least one of a, b in collections: %v", names)
	}
}

// helpers

func collectionNames(m *Mapping) map[string]bool {
	names := make(map[string]bool)
	for _, c := range m.Collections {
		names[c.Name] = true
	}
	return names
}

func findCollection(m *Mapping, name string) *Collection {
	for i := range m.Collections {
		if m.Collections[i].Name == name {
			return &m.Collections[i]
		}
	}
	return nil
}
