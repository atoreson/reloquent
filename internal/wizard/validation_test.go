package wizard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/validation"
)

func TestNewValidationModel(t *testing.T) {
	m := NewValidationModel()
	if m.Done() {
		t.Error("should not be done initially")
	}
	if m.Cancelled() {
		t.Error("should not be cancelled initially")
	}
	if m.Result() != nil {
		t.Error("result should be nil initially")
	}
}

func TestValidationModel_AddCheck(t *testing.T) {
	m := NewValidationModel()
	m.AddCheck("users", "row_count", true)
	m.AddCheck("users", "sample", false)

	v := m.View()
	if !strings.Contains(v, "users") {
		t.Error("view should contain collection name")
	}
	if !strings.Contains(v, "row_count") {
		t.Error("view should contain check type")
	}
}

func TestValidationModel_SetResult_Pass(t *testing.T) {
	m := NewValidationModel()
	m.SetResult(&validation.Result{Status: "PASS"})

	v := m.View()
	if !strings.Contains(v, "PASS") {
		t.Error("view should show PASS")
	}
	if !strings.Contains(v, "enter") {
		t.Error("should show enter prompt for passing result")
	}
}

func TestValidationModel_SetResult_Fail(t *testing.T) {
	m := NewValidationModel()
	m.SetResult(&validation.Result{Status: "FAIL"})

	v := m.View()
	if !strings.Contains(v, "FAIL") {
		t.Error("view should show FAIL")
	}
	if !strings.Contains(v, "proceed") {
		t.Error("should show proceed option on failure")
	}
}

func TestValidationModel_ProceedOnFailure(t *testing.T) {
	m := NewValidationModel()
	m.SetResult(&validation.Result{Status: "FAIL"})

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	rm := result.(ValidationModel)
	if !rm.Done() {
		t.Error("p should finish on failure")
	}
	if rm.Cancelled() {
		t.Error("p should not cancel")
	}
}

func TestValidationModel_CancelOnFailure(t *testing.T) {
	m := NewValidationModel()
	m.SetResult(&validation.Result{Status: "FAIL"})

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	rm := result.(ValidationModel)
	if !rm.Cancelled() {
		t.Error("q should cancel")
	}
}

func TestValidationModel_EnterOnPass(t *testing.T) {
	m := NewValidationModel()
	m.SetResult(&validation.Result{Status: "PASS"})

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	rm := result.(ValidationModel)
	if !rm.Done() {
		t.Error("enter should finish on pass")
	}
}

func TestValidationModel_View_Title(t *testing.T) {
	m := NewValidationModel()
	v := m.View()
	if !strings.Contains(v, "Step 10") {
		t.Error("view should contain step title")
	}
}
