package wizard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/migration"
)

func TestNewMigrateModel(t *testing.T) {
	m := NewMigrateModel()
	if m.Done() {
		t.Error("should not be done initially")
	}
	if m.Cancelled() {
		t.Error("should not be cancelled initially")
	}
}

func TestMigrateModel_Cancel(t *testing.T) {
	m := NewMigrateModel()
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	rm := result.(MigrateModel)
	if !rm.Cancelled() {
		t.Error("q should cancel")
	}
}

func TestMigrateModel_ProgressDisplay(t *testing.T) {
	m := NewMigrateModel()
	m.SetStatus(&migration.Status{
		Phase: "running",
		Overall: migration.ProgressInfo{
			DocsWritten:     50000,
			DocsTotal:       100000,
			PercentComplete: 50.0,
			ThroughputMBps:  75.5,
		},
		Collections: []migration.CollectionStatus{
			{Name: "users", State: "completed", DocsWritten: 10000, DocsTotal: 10000, PercentComplete: 100},
			{Name: "orders", State: "running", DocsWritten: 40000, DocsTotal: 90000, PercentComplete: 44.4},
		},
	})

	v := m.View()
	if !strings.Contains(v, "running") {
		t.Error("view should show running phase")
	}
	if !strings.Contains(v, "users") {
		t.Error("view should show collection names")
	}
	if !strings.Contains(v, "orders") {
		t.Error("view should show all collections")
	}
}

func TestMigrateModel_FailureDialog(t *testing.T) {
	m := NewMigrateModel()
	m.SetStatus(&migration.Status{
		Phase: "partial_failure",
		Collections: []migration.CollectionStatus{
			{Name: "users", State: "completed"},
			{Name: "orders", State: "failed", Error: "timeout"},
		},
	})

	v := m.View()
	if !strings.Contains(v, "failed") {
		t.Error("view should show failure")
	}

	// Test failure action choices
	// 'r' retries failed
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	rm := result.(MigrateModel)
	if rm.FailureAction() != migration.ActionRetryFailed {
		t.Error("r should set retry failed action")
	}
}

func TestMigrateModel_FailureDialog_RestartAll(t *testing.T) {
	m := NewMigrateModel()
	m.showingFail = true

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	rm := result.(MigrateModel)
	if rm.FailureAction() != migration.ActionRestartAll {
		t.Error("a should set restart all action")
	}
}

func TestMigrateModel_FailureDialog_Abort(t *testing.T) {
	m := NewMigrateModel()
	m.showingFail = true

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	rm := result.(MigrateModel)
	if rm.FailureAction() != migration.ActionAbort {
		t.Error("q should set abort action")
	}
	if !rm.Cancelled() {
		t.Error("q should cancel")
	}
}

func TestMigrateModel_CompletedEnter(t *testing.T) {
	m := NewMigrateModel()
	m.SetStatus(&migration.Status{Phase: "completed"})

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	rm := result.(MigrateModel)
	if !rm.Done() {
		t.Error("enter on completed should finish")
	}
}

func TestMigrateModel_View_Title(t *testing.T) {
	m := NewMigrateModel()
	v := m.View()
	if !strings.Contains(v, "Step 9") {
		t.Error("view should contain step title")
	}
}

func TestRenderProgressBar(t *testing.T) {
	bar := renderProgressBar(50, 20)
	if !strings.Contains(bar, "=") {
		t.Error("progress bar should contain filled characters")
	}
	if !strings.Contains(bar, " ") {
		t.Error("progress bar should contain empty characters")
	}
	if !strings.HasPrefix(bar, "[") || !strings.HasSuffix(bar, "]") {
		t.Error("progress bar should be enclosed in brackets")
	}
}
