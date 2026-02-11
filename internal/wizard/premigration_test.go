package wizard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/target"
)

func TestNewPreMigrationModel(t *testing.T) {
	m := NewPreMigrationModel([]string{"users", "orders"})
	if m.Done() {
		t.Error("should not be done initially")
	}
	if len(m.collections) != 2 {
		t.Errorf("expected 2 collections, got %d", len(m.collections))
	}
}

func TestPreMigrationModel_Confirm(t *testing.T) {
	m := NewPreMigrationModel([]string{"users"})
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	rm := result.(PreMigrationModel)
	if !rm.Done() {
		t.Error("enter should finish")
	}
	if rm.Cancelled() {
		t.Error("enter should not cancel")
	}
}

func TestPreMigrationModel_Cancel(t *testing.T) {
	m := NewPreMigrationModel([]string{"users"})
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	rm := result.(PreMigrationModel)
	if !rm.Cancelled() {
		t.Error("q should cancel")
	}
}

func TestPreMigrationModel_TopologyDisplay(t *testing.T) {
	m := NewPreMigrationModel([]string{"users"})
	m.SetTopology(&target.TopologyInfo{
		Type:          "atlas",
		IsAtlas:       true,
		ServerVersion: "7.0.0",
		ShardCount:    3,
	})

	v := m.View()
	if !strings.Contains(v, "atlas") {
		t.Error("view should show topology type")
	}
	if !strings.Contains(v, "Atlas") {
		t.Error("view should show Atlas label")
	}
	if !strings.Contains(v, "7.0.0") {
		t.Error("view should show server version")
	}
}

func TestPreMigrationModel_ValidationResults(t *testing.T) {
	m := NewPreMigrationModel([]string{"users"})
	m.SetValidation(&target.ValidationResult{
		Passed: true,
		Warnings: []target.ValidationIssue{
			{Category: "tier", Message: "Standalone instance"},
		},
	})

	v := m.View()
	if !strings.Contains(v, "PASSED") {
		t.Error("view should show PASSED")
	}
	if !strings.Contains(v, "Standalone") {
		t.Error("view should show warning")
	}
}

func TestPreMigrationModel_SetupProgress(t *testing.T) {
	m := NewPreMigrationModel([]string{"users"})
	m.SetSetupDone()

	v := m.View()
	if !strings.Contains(v, "Created") {
		t.Error("view should show Created after setup")
	}
}

func TestPreMigrationModel_View(t *testing.T) {
	m := NewPreMigrationModel([]string{"users", "orders"})
	v := m.View()
	if !strings.Contains(v, "Step 8") {
		t.Error("view should contain step title")
	}
	if !strings.Contains(v, "users") {
		t.Error("view should list collections")
	}
}
