package wizard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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
		}},
		{Name: "products", RowCount: 500, SizeBytes: 32768},
	}
}

func TestNewTableSelectModel(t *testing.T) {
	m := NewTableSelectModel(testTables(), nil)
	if len(m.entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(m.entries))
	}
	if m.selectedCount() != 0 {
		t.Errorf("expected 0 selected initially, got %d", m.selectedCount())
	}
	if len(m.visibleIdxs) != 4 {
		t.Errorf("expected 4 visible, got %d", len(m.visibleIdxs))
	}
}

func TestNewTableSelectModel_PreSelected(t *testing.T) {
	m := NewTableSelectModel(testTables(), []string{"customers", "orders"})
	if m.selectedCount() != 2 {
		t.Errorf("expected 2 pre-selected, got %d", m.selectedCount())
	}
}

func TestToggleCurrent(t *testing.T) {
	m := NewTableSelectModel(testTables(), nil)
	m.toggleCurrent()
	if m.selectedCount() != 1 {
		t.Errorf("expected 1 selected after toggle, got %d", m.selectedCount())
	}
	m.toggleCurrent()
	if m.selectedCount() != 0 {
		t.Errorf("expected 0 selected after second toggle, got %d", m.selectedCount())
	}
}

func TestSelectAll_DeselectAll(t *testing.T) {
	m := NewTableSelectModel(testTables(), nil)
	m.selectAll()
	if m.selectedCount() != 4 {
		t.Errorf("selectAll: expected 4, got %d", m.selectedCount())
	}
	m.deselectAll()
	if m.selectedCount() != 0 {
		t.Errorf("deselectAll: expected 0, got %d", m.selectedCount())
	}
}

func TestMoveCursor(t *testing.T) {
	m := NewTableSelectModel(testTables(), nil)
	if m.cursor != 0 {
		t.Fatalf("initial cursor should be 0, got %d", m.cursor)
	}
	m.moveCursor(1)
	if m.cursor != 1 {
		t.Errorf("cursor should be 1 after down, got %d", m.cursor)
	}
	m.moveCursor(-1)
	if m.cursor != 0 {
		t.Errorf("cursor should be 0 after up, got %d", m.cursor)
	}
	// Should clamp at boundaries
	m.moveCursor(-5)
	if m.cursor != 0 {
		t.Errorf("cursor should clamp at 0, got %d", m.cursor)
	}
	m.moveCursor(100)
	if m.cursor != 3 {
		t.Errorf("cursor should clamp at 3, got %d", m.cursor)
	}
}

func TestApplyFilter(t *testing.T) {
	m := NewTableSelectModel(testTables(), nil)
	m.filter = "order"
	m.applyFilter()
	if len(m.visibleIdxs) != 2 {
		t.Errorf("expected 2 visible with 'order' filter, got %d", len(m.visibleIdxs))
	}

	// Clear filter
	m.filter = ""
	m.applyFilter()
	if len(m.visibleIdxs) != 4 {
		t.Errorf("expected 4 visible with empty filter, got %d", len(m.visibleIdxs))
	}
}

func TestApplyFilter_CaseInsensitive(t *testing.T) {
	m := NewTableSelectModel(testTables(), nil)
	m.filter = "ORDER"
	m.applyFilter()
	if len(m.visibleIdxs) != 2 {
		t.Errorf("expected 2 visible with 'ORDER' filter, got %d", len(m.visibleIdxs))
	}
}

func TestSelectDependencies(t *testing.T) {
	m := NewTableSelectModel(testTables(), nil)

	// Select only order_items (which references orders)
	m.cursor = 2 // order_items (sorted by name: customers=0, order_items=1, orders=2, products=3)
	// Actually, sort is by name ascending, so let's find order_items
	for i, idx := range m.visibleIdxs {
		if m.entries[idx].table.Name == "order_items" {
			m.cursor = i
			break
		}
	}
	m.toggleCurrent()

	m.selectDependencies()

	// order_items references orders, so orders should now be selected
	selected := m.getSelected()
	selectedNames := make(map[string]bool)
	for _, t := range selected {
		selectedNames[t.Name] = true
	}
	if !selectedNames["order_items"] {
		t.Error("order_items should be selected")
	}
	if !selectedNames["orders"] {
		t.Error("orders should be auto-selected as dependency")
	}
}

func TestCycleSort(t *testing.T) {
	m := NewTableSelectModel(testTables(), nil)
	if m.sortField != SortByName || !m.sortAsc {
		t.Fatalf("initial sort should be name ascending")
	}
	m.cycleSort() // name desc
	if m.sortField != SortByName || m.sortAsc {
		t.Errorf("after first cycle: expected name desc")
	}
	m.cycleSort() // rows asc
	if m.sortField != SortByRows || !m.sortAsc {
		t.Errorf("after second cycle: expected rows asc")
	}
}

func TestViewRenders(t *testing.T) {
	m := NewTableSelectModel(testTables(), nil)
	m.width = 80
	m.height = 24
	v := m.View()
	if !strings.Contains(v, "Step 3: Select Tables") {
		t.Error("view should contain title")
	}
	if !strings.Contains(v, "customers") {
		t.Error("view should contain table name 'customers'")
	}
	if !strings.Contains(v, "Selected: 0 tables") {
		t.Error("view should show 0 selected")
	}
}

func TestUpdateEnterWithNoSelection(t *testing.T) {
	m := NewTableSelectModel(testTables(), nil)
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, cmd := m.updateNormal(msg)
	rm := result.(TableSelectModel)
	if rm.Done() {
		t.Error("enter with no selection should not finish")
	}
	if cmd != nil {
		t.Error("enter with no selection should return nil cmd")
	}
}

func TestUpdateEnterWithSelection(t *testing.T) {
	m := NewTableSelectModel(testTables(), []string{"customers"})
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	result, _ := m.updateNormal(msg)
	rm := result.(TableSelectModel)
	if !rm.Done() {
		t.Error("enter with selection should finish")
	}
	if rm.Cancelled() {
		t.Error("should not be cancelled")
	}
	r := rm.Result()
	if r == nil {
		t.Fatal("result should not be nil")
	}
	if len(r.Selected) != 1 {
		t.Errorf("expected 1 selected, got %d", len(r.Selected))
	}
}

func TestResultNilWhenCancelled(t *testing.T) {
	m := NewTableSelectModel(testTables(), nil)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	result, _ := m.updateNormal(msg)
	rm := result.(TableSelectModel)
	if !rm.Cancelled() {
		t.Error("q should cancel")
	}
	if rm.Result() != nil {
		t.Error("result should be nil when cancelled")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, tt := range tests {
		got := formatBytes(tt.input)
		if got != tt.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0K"},
		{1500000, "1.5M"},
		{2000000000, "2.0B"},
	}
	for _, tt := range tests {
		got := formatNumber(tt.input)
		if got != tt.want {
			t.Errorf("formatNumber(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate short string: got %q", got)
	}
	got := truncate("a_very_long_table_name", 10)
	// truncate keeps maxLen-1 bytes + "…", so visual width is maxLen
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncated string should end with ellipsis, got %q", got)
	}
	if !strings.HasPrefix(got, "a_very_lo") {
		t.Errorf("truncated string should start with prefix, got %q", got)
	}
}
