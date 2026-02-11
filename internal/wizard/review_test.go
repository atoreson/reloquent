package wizard

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/sizing"
)

func TestNewReviewModel(t *testing.T) {
	plan := &sizing.SizingPlan{
		SparkPlan:     sizing.SparkPlan{Platform: "emr", InstanceType: "r5.4xlarge", WorkerCount: 10, CostEstimate: "$100"},
		MongoPlan:     sizing.MongoPlan{MigrationTier: "M60", ProductionTier: "M40", StorageGB: 100},
		EstimatedTime: time.Hour,
	}
	m := NewReviewModel(plan, "# pyspark script\nprint('hello')")

	if m.Done() {
		t.Error("should not be done initially")
	}
	if m.Confirmed() {
		t.Error("should not be confirmed initially")
	}
}

func TestReviewModel_Confirm(t *testing.T) {
	m := NewReviewModel(nil, "")
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	rm := result.(ReviewModel)
	if !rm.Done() {
		t.Error("enter should finish")
	}
	if !rm.Confirmed() {
		t.Error("enter should confirm")
	}
	if rm.Cancelled() {
		t.Error("enter should not cancel")
	}
}

func TestReviewModel_Cancel(t *testing.T) {
	m := NewReviewModel(nil, "")
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	rm := result.(ReviewModel)
	if !rm.Cancelled() {
		t.Error("q should cancel")
	}
	if rm.Confirmed() {
		t.Error("q should not confirm")
	}
}

func TestReviewModel_ScriptToggle(t *testing.T) {
	m := NewReviewModel(nil, "print('hello')")

	if m.showScript {
		t.Error("script should be hidden initially")
	}

	// Toggle on
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	rm := result.(ReviewModel)
	if !rm.showScript {
		t.Error("v should show script")
	}

	// Toggle off
	result, _ = rm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	rm = result.(ReviewModel)
	if rm.showScript {
		t.Error("second v should hide script")
	}
}

func TestReviewModel_View_Summary(t *testing.T) {
	plan := &sizing.SizingPlan{
		SparkPlan:     sizing.SparkPlan{Platform: "emr", InstanceType: "r5.4xlarge", WorkerCount: 10, CostEstimate: "$100-200"},
		MongoPlan:     sizing.MongoPlan{MigrationTier: "M60", ProductionTier: "M40", StorageGB: 100},
		EstimatedTime: time.Hour,
	}
	m := NewReviewModel(plan, "print('test')")
	m.width = 100
	m.height = 30

	v := m.View()
	if !strings.Contains(v, "Step 8b") {
		t.Error("view should contain step title")
	}
	if !strings.Contains(v, "EMR") {
		t.Error("view should show platform")
	}
	if !strings.Contains(v, "$100-200") {
		t.Error("view should show cost")
	}
	if !strings.Contains(v, "WARNING") {
		t.Error("view should show warning")
	}
}

func TestReviewModel_View_ScriptVisible(t *testing.T) {
	m := NewReviewModel(nil, "print('hello world')")
	m.showScript = true
	m.width = 100
	m.height = 30

	v := m.View()
	if !strings.Contains(v, "hello world") {
		t.Error("view should show script when toggled")
	}
}
