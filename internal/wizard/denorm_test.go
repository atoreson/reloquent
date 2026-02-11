package wizard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/schema"
)

func testTablesWithFKs() []schema.Table {
	return []schema.Table{
		{Name: "customers", RowCount: 1000, PrimaryKey: &schema.PrimaryKey{Name: "pk_customers", Columns: []string{"id"}}},
		{Name: "orders", RowCount: 5000, ForeignKeys: []schema.ForeignKey{
			{Name: "fk_orders_customer", Columns: []string{"customer_id"}, ReferencedTable: "customers", ReferencedColumns: []string{"id"}},
		}},
		{Name: "order_items", RowCount: 20000, ForeignKeys: []schema.ForeignKey{
			{Name: "fk_items_order", Columns: []string{"order_id"}, ReferencedTable: "orders", ReferencedColumns: []string{"id"}},
		}},
		{Name: "products", RowCount: 500},
	}
}

func TestExtractRelationships(t *testing.T) {
	rels := extractRelationships(testTablesWithFKs())
	if len(rels) != 2 {
		t.Fatalf("expected 2 relationships, got %d", len(rels))
	}

	// Should be sorted by parent then child
	if rels[0].ParentTable != "customers" || rels[0].ChildTable != "orders" {
		t.Errorf("first rel should be orders→customers, got %s→%s", rels[0].ChildTable, rels[0].ParentTable)
	}
	if rels[1].ParentTable != "orders" || rels[1].ChildTable != "order_items" {
		t.Errorf("second rel should be order_items→orders, got %s→%s", rels[1].ChildTable, rels[1].ParentTable)
	}
}

func TestExtractRelationships_FiltersCrossSelection(t *testing.T) {
	// Only select customers and orders, not order_items
	tables := []schema.Table{
		{Name: "customers"},
		{Name: "orders", ForeignKeys: []schema.ForeignKey{
			{Name: "fk_orders_customer", Columns: []string{"customer_id"}, ReferencedTable: "customers", ReferencedColumns: []string{"id"}},
		}},
		// order_items excluded — its FK to orders should not appear
	}
	rels := extractRelationships(tables)
	if len(rels) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(rels))
	}
}

func TestExtractRelationships_FKToUnselectedTable(t *testing.T) {
	// orders references customers, but customers is not selected
	tables := []schema.Table{
		{Name: "orders", ForeignKeys: []schema.ForeignKey{
			{Name: "fk_orders_customer", Columns: []string{"customer_id"}, ReferencedTable: "customers", ReferencedColumns: []string{"id"}},
		}},
	}
	rels := extractRelationships(tables)
	if len(rels) != 0 {
		t.Fatalf("expected 0 relationships (FK target not selected), got %d", len(rels))
	}
}

func TestExtractRelationships_NoFKs(t *testing.T) {
	tables := []schema.Table{
		{Name: "customers"},
		{Name: "products"},
	}
	rels := extractRelationships(tables)
	if len(rels) != 0 {
		t.Fatalf("expected 0 relationships, got %d", len(rels))
	}
}

func TestNewDenormModel(t *testing.T) {
	m := NewDenormModel(testTablesWithFKs())
	if len(m.rels) != 2 {
		t.Fatalf("expected 2 rels, got %d", len(m.rels))
	}
	if m.cursor != 0 {
		t.Errorf("initial cursor should be 0, got %d", m.cursor)
	}
	if m.done {
		t.Error("should not be done initially")
	}
}

func TestDenormCursorNavigation(t *testing.T) {
	m := NewDenormModel(testTablesWithFKs())

	// Move down
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = result.(DenormModel)
	if m.cursor != 1 {
		t.Errorf("after j: cursor should be 1, got %d", m.cursor)
	}

	// Move down again — should clamp
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = result.(DenormModel)
	if m.cursor != 1 {
		t.Errorf("cursor should clamp at 1, got %d", m.cursor)
	}

	// Move up
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = result.(DenormModel)
	if m.cursor != 0 {
		t.Errorf("after k: cursor should be 0, got %d", m.cursor)
	}

	// Move up again — should clamp
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = result.(DenormModel)
	if m.cursor != 0 {
		t.Errorf("cursor should clamp at 0, got %d", m.cursor)
	}
}

func TestDenormChoiceCycling(t *testing.T) {
	m := NewDenormModel(testTablesWithFKs())

	// Default is reference
	if m.rels[0].Choice != ChoiceReference {
		t.Fatalf("default choice should be reference, got %v", m.rels[0].Choice)
	}

	// Space cycles: reference → embed array
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = result.(DenormModel)
	if m.rels[0].Choice != ChoiceEmbedArray {
		t.Errorf("after first space: expected embed array, got %v", m.rels[0].Choice)
	}

	// Space again: embed array → embed single
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = result.(DenormModel)
	if m.rels[0].Choice != ChoiceEmbedSingle {
		t.Errorf("after second space: expected embed single, got %v", m.rels[0].Choice)
	}

	// Space again: embed single → reference
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = result.(DenormModel)
	if m.rels[0].Choice != ChoiceReference {
		t.Errorf("after third space: expected reference, got %v", m.rels[0].Choice)
	}
}

func TestDenormDirectSetKeys(t *testing.T) {
	m := NewDenormModel(testTablesWithFKs())

	// 'a' sets embed array
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = result.(DenormModel)
	if m.rels[0].Choice != ChoiceEmbedArray {
		t.Errorf("'a' should set embed array, got %v", m.rels[0].Choice)
	}

	// 's' sets embed single
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = result.(DenormModel)
	if m.rels[0].Choice != ChoiceEmbedSingle {
		t.Errorf("'s' should set embed single, got %v", m.rels[0].Choice)
	}

	// 'r' sets reference
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = result.(DenormModel)
	if m.rels[0].Choice != ChoiceReference {
		t.Errorf("'r' should set reference, got %v", m.rels[0].Choice)
	}
}

func TestDenormConfirm(t *testing.T) {
	m := NewDenormModel(testTablesWithFKs())
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	rm := result.(DenormModel)
	if !rm.Done() {
		t.Error("f should finish")
	}
	if rm.Cancelled() {
		t.Error("f should not cancel")
	}
}

func TestDenormCancel(t *testing.T) {
	m := NewDenormModel(testTablesWithFKs())
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	rm := result.(DenormModel)
	if !rm.Done() {
		t.Error("q should finish")
	}
	if !rm.Cancelled() {
		t.Error("q should cancel")
	}
}

func TestDenormCancelEsc(t *testing.T) {
	m := NewDenormModel(testTablesWithFKs())
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	rm := result.(DenormModel)
	if !rm.Cancelled() {
		t.Error("esc should cancel")
	}
}

func TestBuildMapping_AllReferences(t *testing.T) {
	m := NewDenormModel(testTablesWithFKs())
	// All defaults are reference
	mp := m.BuildMapping()

	if len(mp.Collections) != 4 {
		t.Fatalf("expected 4 collections (all reference), got %d", len(mp.Collections))
	}

	// All tables should be root collections
	names := make(map[string]bool)
	for _, c := range mp.Collections {
		names[c.Name] = true
	}
	for _, name := range []string{"customers", "orders", "order_items", "products"} {
		if !names[name] {
			t.Errorf("expected collection %q", name)
		}
	}

	// customers should have a reference to orders
	var customers *mapping.Collection
	for i := range mp.Collections {
		if mp.Collections[i].Name == "customers" {
			c := mp.Collections[i]
			customers = &c
		}
	}
	if customers == nil {
		t.Fatal("customers collection not found")
	}
	if len(customers.References) != 1 {
		t.Fatalf("expected 1 reference on customers, got %d", len(customers.References))
	}
	if customers.References[0].SourceTable != "orders" {
		t.Errorf("expected reference to orders, got %q", customers.References[0].SourceTable)
	}
}

func TestBuildMapping_EmbedArray(t *testing.T) {
	m := NewDenormModel(testTablesWithFKs())
	// Embed orders into customers
	m.rels[0].Choice = ChoiceEmbedArray // orders→customers

	mp := m.BuildMapping()

	// orders should be embedded, so only 3 root collections
	if len(mp.Collections) != 3 {
		t.Fatalf("expected 3 collections, got %d", len(mp.Collections))
	}

	names := make(map[string]bool)
	for _, c := range mp.Collections {
		names[c.Name] = true
	}
	if names["orders"] {
		t.Error("orders should be embedded, not a root collection")
	}

	// Find customers
	for _, c := range mp.Collections {
		if c.Name == "customers" {
			if len(c.Embedded) != 1 {
				t.Fatalf("expected 1 embedded in customers, got %d", len(c.Embedded))
			}
			if c.Embedded[0].SourceTable != "orders" {
				t.Errorf("expected embedded orders, got %q", c.Embedded[0].SourceTable)
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
		}
	}
}

func TestBuildMapping_EmbedSingle(t *testing.T) {
	m := NewDenormModel(testTablesWithFKs())
	m.rels[0].Choice = ChoiceEmbedSingle // orders→customers as single

	mp := m.BuildMapping()

	for _, c := range mp.Collections {
		if c.Name == "customers" {
			if len(c.Embedded) != 1 {
				t.Fatalf("expected 1 embedded, got %d", len(c.Embedded))
			}
			if c.Embedded[0].Relationship != "single" {
				t.Errorf("expected 'single', got %q", c.Embedded[0].Relationship)
			}
			return
		}
	}
	t.Error("customers collection not found")
}

func TestBuildMapping_NoFKs(t *testing.T) {
	tables := []schema.Table{
		{Name: "customers"},
		{Name: "products"},
	}
	m := NewDenormModel(tables)
	mp := m.BuildMapping()

	if len(mp.Collections) != 2 {
		t.Fatalf("expected 2 collections, got %d", len(mp.Collections))
	}
	for _, c := range mp.Collections {
		if len(c.Embedded) != 0 {
			t.Errorf("collection %q should have no embedded entries", c.Name)
		}
		if len(c.References) != 0 {
			t.Errorf("collection %q should have no references", c.Name)
		}
	}
}

func TestDenormView_Renders(t *testing.T) {
	m := NewDenormModel(testTablesWithFKs())
	m.width = 100
	m.height = 30
	v := m.View()

	if !strings.Contains(v, "Step 4: Denormalization Design") {
		t.Error("view should contain title")
	}
	if !strings.Contains(v, "customer_id") {
		t.Error("view should show FK column names")
	}
	if !strings.Contains(v, "reference") {
		t.Error("view should show default choice 'reference'")
	}
	if !strings.Contains(v, "Preview") {
		t.Error("view should contain preview section")
	}
	if !strings.Contains(v, "(collection)") {
		t.Error("view should show collection labels in preview")
	}
}

func TestDenormView_NoFKs(t *testing.T) {
	tables := []schema.Table{
		{Name: "products"},
	}
	m := NewDenormModel(tables)
	v := m.View()

	if !strings.Contains(v, "No foreign key relationships") {
		t.Error("view should indicate no FKs")
	}
	if !strings.Contains(v, "standalone collections") {
		t.Error("view should mention standalone collections")
	}
}

func TestDenormNoFKs_ConfirmWithF(t *testing.T) {
	tables := []schema.Table{
		{Name: "products"},
	}
	m := NewDenormModel(tables)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	rm := result.(DenormModel)
	if !rm.Done() {
		t.Error("f should finish even with no FKs")
	}
	if rm.Cancelled() {
		t.Error("f should not cancel")
	}
}

func TestDenormPreview_ShowsEmbedded(t *testing.T) {
	m := NewDenormModel(testTablesWithFKs())
	m.rels[0].Choice = ChoiceEmbedArray // orders embedded in customers

	preview := m.buildPreview()
	joined := strings.Join(preview, "\n")

	if !strings.Contains(joined, "customers (collection)") {
		t.Error("preview should show customers as collection")
	}
	if !strings.Contains(joined, "orders[]") {
		t.Error("preview should show orders[] as embedded array")
	}
}

func TestDenormEnterConfirms(t *testing.T) {
	m := NewDenormModel(testTablesWithFKs())
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	rm := result.(DenormModel)
	if !rm.Done() {
		t.Error("enter should confirm")
	}
	if rm.Cancelled() {
		t.Error("enter should not cancel")
	}
}

func TestBuildMapping_DeepNesting(t *testing.T) {
	// 3-level chain: order_items → orders → customers, all embedded
	tables := testTablesWithFKs()
	m := NewDenormModel(tables)
	m.rels[0].Choice = ChoiceEmbedArray // orders → customers
	m.rels[1].Choice = ChoiceEmbedArray // order_items → orders

	mp := m.BuildMapping()

	// Should have 2 root collections: customers, products
	if len(mp.Collections) != 2 {
		t.Fatalf("expected 2 collections, got %d", len(mp.Collections))
	}

	var customers *mapping.Collection
	for i := range mp.Collections {
		if mp.Collections[i].Name == "customers" {
			c := mp.Collections[i]
			customers = &c
		}
	}
	if customers == nil {
		t.Fatal("customers collection not found")
	}

	// customers should have orders embedded
	if len(customers.Embedded) != 1 {
		t.Fatalf("expected 1 embedded in customers, got %d", len(customers.Embedded))
	}
	if customers.Embedded[0].SourceTable != "orders" {
		t.Errorf("expected embedded orders, got %q", customers.Embedded[0].SourceTable)
	}

	// orders should have order_items nested inside it
	if len(customers.Embedded[0].Embedded) != 1 {
		t.Fatalf("expected 1 nested embedded in orders, got %d", len(customers.Embedded[0].Embedded))
	}
	if customers.Embedded[0].Embedded[0].SourceTable != "order_items" {
		t.Errorf("expected nested order_items, got %q", customers.Embedded[0].Embedded[0].SourceTable)
	}
}

func TestSelfReferenceDisplay(t *testing.T) {
	tables := []schema.Table{
		{Name: "employees", ForeignKeys: []schema.ForeignKey{
			{Name: "fk_emp_manager", Columns: []string{"manager_id"}, ReferencedTable: "employees", ReferencedColumns: []string{"id"}},
		}},
	}
	m := NewDenormModel(tables)

	if len(m.rels) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(m.rels))
	}
	if !m.rels[0].IsSelfRef {
		t.Error("should be marked as self-reference")
	}

	v := m.View()
	if !strings.Contains(v, "self-ref") {
		t.Error("view should show self-ref label")
	}
}

func TestSelfReference_DefaultsToReference(t *testing.T) {
	tables := []schema.Table{
		{Name: "employees", ForeignKeys: []schema.ForeignKey{
			{Name: "fk_emp_manager", Columns: []string{"manager_id"}, ReferencedTable: "employees", ReferencedColumns: []string{"id"}},
		}},
	}
	m := NewDenormModel(tables)
	mp := m.BuildMapping()

	// Self-ref should result in a single collection with itself as reference
	if len(mp.Collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(mp.Collections))
	}
	if mp.Collections[0].Name != "employees" {
		t.Errorf("expected employees collection, got %q", mp.Collections[0].Name)
	}
}

func TestM2MJoinTableDisplay(t *testing.T) {
	tables := []schema.Table{
		{Name: "students", Columns: []schema.Column{{Name: "id", DataType: "integer"}}},
		{Name: "courses", Columns: []schema.Column{{Name: "id", DataType: "integer"}}},
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
	m := NewDenormModel(tables)

	// enrollments' rels should be marked as join table
	joinCount := 0
	for _, rel := range m.rels {
		if rel.IsJoinTable {
			joinCount++
		}
	}
	if joinCount != 2 {
		t.Errorf("expected 2 rels marked as join table, got %d", joinCount)
	}

	v := m.View()
	if !strings.Contains(v, "M2M join") {
		t.Error("view should show M2M join label")
	}
}

func TestCycleForcedReference(t *testing.T) {
	// A→B and B→A, both set to embed — should force one to reference
	tables := []schema.Table{
		{Name: "a", ForeignKeys: []schema.ForeignKey{
			{Name: "fk_a_b", Columns: []string{"b_id"}, ReferencedTable: "b", ReferencedColumns: []string{"id"}},
		}},
		{Name: "b", ForeignKeys: []schema.ForeignKey{
			{Name: "fk_b_a", Columns: []string{"a_id"}, ReferencedTable: "a", ReferencedColumns: []string{"id"}},
		}},
	}
	m := NewDenormModel(tables)

	// Set both to embed
	for i := range m.rels {
		m.rels[i].Choice = ChoiceEmbedArray
	}

	m.enforceCycleConstraints()

	// At least one should have been forced to reference
	embedCount := 0
	for _, rel := range m.rels {
		if rel.Choice == ChoiceEmbedArray || rel.Choice == ChoiceEmbedSingle {
			embedCount++
		}
	}
	if embedCount >= 2 {
		t.Error("cycle should have forced at least one edge to reference")
	}
	if len(m.warnings) == 0 {
		t.Error("should have generated a warning")
	}
}

func TestBuildPreview_DeepNesting(t *testing.T) {
	tables := testTablesWithFKs()
	m := NewDenormModel(tables)
	m.rels[0].Choice = ChoiceEmbedArray // orders → customers
	m.rels[1].Choice = ChoiceEmbedArray // order_items → orders

	preview := m.buildPreview()
	joined := strings.Join(preview, "\n")

	if !strings.Contains(joined, "customers (collection)") {
		t.Error("preview should show customers as collection")
	}
	if !strings.Contains(joined, "orders[]") {
		t.Error("preview should show orders embedded")
	}
	if !strings.Contains(joined, "order_items[]") {
		t.Error("preview should show order_items nested inside orders")
	}
}

func TestRelChoiceString(t *testing.T) {
	tests := []struct {
		choice RelChoice
		want   string
	}{
		{ChoiceReference, "reference"},
		{ChoiceEmbedArray, "embed array"},
		{ChoiceEmbedSingle, "embed single"},
	}
	for _, tt := range tests {
		if got := tt.choice.String(); got != tt.want {
			t.Errorf("RelChoice(%d).String() = %q, want %q", tt.choice, got, tt.want)
		}
	}
}

