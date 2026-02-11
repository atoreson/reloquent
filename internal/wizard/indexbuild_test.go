package wizard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/target"
)

func TestNewIndexBuildModel(t *testing.T) {
	m := NewIndexBuildModel(5)
	if m.Done() {
		t.Error("should not be done initially")
	}
	if m.Cancelled() {
		t.Error("should not be cancelled initially")
	}
}

func TestIndexBuildModel_NoIndexes(t *testing.T) {
	m := NewIndexBuildModel(0)
	v := m.View()
	if !strings.Contains(v, "No indexes") {
		t.Error("should show no indexes message")
	}
}

func TestIndexBuildModel_UpdateProgress(t *testing.T) {
	m := NewIndexBuildModel(3)
	m.UpdateProgress([]target.IndexBuildStatus{
		{Collection: "users", IndexName: "idx_email", Phase: "complete", Progress: 100},
		{Collection: "orders", IndexName: "idx_user", Phase: "building", Progress: 50},
		{Collection: "products", IndexName: "idx_sku", Phase: "not_started"},
	})

	v := m.View()
	if !strings.Contains(v, "idx_email") {
		t.Error("should show index names")
	}
	if !strings.Contains(v, "1/3") {
		t.Error("should show completed count")
	}
}

func TestIndexBuildModel_Finished(t *testing.T) {
	m := NewIndexBuildModel(2)
	m.SetFinished()

	v := m.View()
	if !strings.Contains(v, "successfully") {
		t.Error("should show success message when finished")
	}
	if !strings.Contains(v, "enter") {
		t.Error("should show enter prompt when finished")
	}
}

func TestIndexBuildModel_EnterOnFinished(t *testing.T) {
	m := NewIndexBuildModel(1)
	m.SetFinished()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	rm := result.(IndexBuildModel)
	if !rm.Done() {
		t.Error("enter on finished should complete")
	}
}

func TestIndexBuildModel_Cancel(t *testing.T) {
	m := NewIndexBuildModel(1)
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	rm := result.(IndexBuildModel)
	if !rm.Cancelled() {
		t.Error("q should cancel")
	}
}

func TestIndexBuildModel_View_Title(t *testing.T) {
	m := NewIndexBuildModel(1)
	v := m.View()
	if !strings.Contains(v, "Step 11") {
		t.Error("view should contain step title")
	}
}
