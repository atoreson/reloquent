package wizard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/typemap"
)

func testSchemaForTypeMap() *schema.Schema {
	return &schema.Schema{
		DatabaseType: "postgresql",
		Tables: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", DataType: "integer"},
					{Name: "name", DataType: "character varying"},
					{Name: "email", DataType: "text"},
					{Name: "active", DataType: "boolean"},
					{Name: "metadata", DataType: "jsonb"},
				},
			},
			{
				Name: "orders",
				Columns: []schema.Column{
					{Name: "id", DataType: "integer"},
					{Name: "amount", DataType: "numeric"},
					{Name: "created_at", DataType: "timestamp with time zone"},
				},
			},
		},
	}
}

func TestTypeMapModel_Construction(t *testing.T) {
	s := testSchemaForTypeMap()
	m := NewTypeMapModel(s, "postgresql", nil)

	// Should contain only types actually in use (7 unique: boolean, character varying,
	// integer, jsonb, numeric, text, timestamp with time zone)
	if len(m.types) != 7 {
		t.Errorf("expected 7 unique types, got %d: %v", len(m.types), m.types)
	}

	// Should be sorted
	for i := 1; i < len(m.types); i++ {
		if m.types[i] < m.types[i-1] {
			t.Errorf("types not sorted: %s before %s", m.types[i-1], m.types[i])
		}
	}
}

func TestTypeMapModel_Navigation(t *testing.T) {
	s := testSchemaForTypeMap()
	m := NewTypeMapModel(s, "postgresql", nil)

	// Move down
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = result.(TypeMapModel)
	if m.cursor != 1 {
		t.Errorf("after j: cursor should be 1, got %d", m.cursor)
	}

	// Move up
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = result.(TypeMapModel)
	if m.cursor != 0 {
		t.Errorf("after k: cursor should be 0, got %d", m.cursor)
	}

	// Move up at top — should clamp
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = result.(TypeMapModel)
	if m.cursor != 0 {
		t.Errorf("cursor should clamp at 0, got %d", m.cursor)
	}
}

func TestTypeMapModel_EditCycling(t *testing.T) {
	s := testSchemaForTypeMap()
	m := NewTypeMapModel(s, "postgresql", nil)

	// Find integer's initial type
	initialType := m.typeMap.Resolve(m.types[m.cursor])

	// Press 'e' to cycle
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = result.(TypeMapModel)

	newType := m.typeMap.Resolve(m.types[m.cursor])
	if newType == initialType {
		t.Error("pressing 'e' should change the BSON type")
	}
}

func TestTypeMapModel_OverrideTracking(t *testing.T) {
	s := testSchemaForTypeMap()
	m := NewTypeMapModel(s, "postgresql", nil)

	sourceType := m.types[m.cursor]

	// Initially not overridden
	if m.typeMap.IsOverridden(sourceType) {
		t.Error("should not be overridden initially")
	}

	// Edit
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = result.(TypeMapModel)

	if !m.typeMap.IsOverridden(sourceType) {
		t.Error("should be overridden after edit")
	}
}

func TestTypeMapModel_RestoreDefault(t *testing.T) {
	s := testSchemaForTypeMap()
	m := NewTypeMapModel(s, "postgresql", nil)

	sourceType := m.types[m.cursor]
	originalType := m.typeMap.Resolve(sourceType)

	// Edit then restore
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = result.(TypeMapModel)
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = result.(TypeMapModel)

	restored := m.typeMap.Resolve(sourceType)
	if restored != originalType {
		t.Errorf("after restore: expected %s, got %s", originalType, restored)
	}
	if m.typeMap.IsOverridden(sourceType) {
		t.Error("should not be overridden after restore")
	}
}

func TestTypeMapModel_Confirm(t *testing.T) {
	s := testSchemaForTypeMap()
	m := NewTypeMapModel(s, "postgresql", nil)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	rm := result.(TypeMapModel)
	if !rm.Done() {
		t.Error("enter should finish")
	}
	if rm.Cancelled() {
		t.Error("enter should not cancel")
	}
	if rm.Result() == nil {
		t.Error("result should not be nil")
	}
}

func TestTypeMapModel_Cancel(t *testing.T) {
	s := testSchemaForTypeMap()
	m := NewTypeMapModel(s, "postgresql", nil)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	rm := result.(TypeMapModel)
	if !rm.Done() {
		t.Error("q should finish")
	}
	if !rm.Cancelled() {
		t.Error("q should cancel")
	}
	if rm.Result() != nil {
		t.Error("result should be nil on cancel")
	}
}

func TestTypeMapModel_View(t *testing.T) {
	s := testSchemaForTypeMap()
	m := NewTypeMapModel(s, "postgresql", nil)
	m.width = 100
	m.height = 30

	v := m.View()

	if !strings.Contains(v, "Step 5: Type Mapping Review") {
		t.Error("view should contain title")
	}
	if !strings.Contains(v, "Source Type") {
		t.Error("view should contain header")
	}
	if !strings.Contains(v, "BSON Type") {
		t.Error("view should contain BSON Type header")
	}
	if !strings.Contains(v, "default") {
		t.Error("view should show default status")
	}
}

func TestTypeMapModel_ExistingOverrides(t *testing.T) {
	s := testSchemaForTypeMap()
	existing := typemap.ForDatabase("postgresql")
	existing.Override("integer", typemap.BSONDecimal128)

	m := NewTypeMapModel(s, "postgresql", existing)

	// integer should still be overridden
	if !m.typeMap.IsOverridden("integer") {
		t.Error("existing override should be preserved")
	}
	if m.typeMap.Resolve("integer") != typemap.BSONDecimal128 {
		t.Errorf("expected Decimal128, got %s", m.typeMap.Resolve("integer"))
	}
}

func TestNextBSONType(t *testing.T) {
	// Should cycle through all types
	current := typemap.AllBSONTypes[0]
	seen := make(map[typemap.BSONType]bool)
	for i := 0; i < len(typemap.AllBSONTypes); i++ {
		seen[current] = true
		current = nextBSONType(current)
	}
	// Should have seen all types
	if len(seen) != len(typemap.AllBSONTypes) {
		t.Errorf("expected to cycle through all %d types, saw %d", len(typemap.AllBSONTypes), len(seen))
	}
	// Should wrap around
	if current != typemap.AllBSONTypes[0] {
		t.Error("should wrap around to first type")
	}
}
