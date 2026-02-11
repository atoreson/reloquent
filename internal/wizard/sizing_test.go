package wizard

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/sizing"
)

func testSizingPlan() *sizing.SizingPlan {
	return &sizing.SizingPlan{
		SparkPlan: sizing.SparkPlan{
			Platform:     "emr",
			InstanceType: "r5.4xlarge",
			WorkerCount:  15,
			CostEstimate: "$100-200",
		},
		MongoPlan: sizing.MongoPlan{
			MigrationTier:  "M60 (64 GB RAM)",
			ProductionTier: "M40 (16 GB RAM)",
			StorageGB:      500,
		},
		EstimatedTime: 2 * time.Hour,
		Explanations: []sizing.Explanation{
			{Category: "overview", Summary: "Migrating 1 TB", Detail: "Details about the migration."},
			{Category: "spark", Summary: "EMR cluster", Detail: "Details about EMR."},
		},
	}
}

func TestNewSizingModel(t *testing.T) {
	plan := testSizingPlan()
	m := NewSizingModel(plan)
	if m.Done() {
		t.Error("should not be done initially")
	}
	if m.Cancelled() {
		t.Error("should not be cancelled initially")
	}
}

func TestSizingModel_Navigation(t *testing.T) {
	m := NewSizingModel(testSizingPlan())

	// Enter confirms
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	rm := result.(SizingModel)
	if !rm.Done() {
		t.Error("enter should finish")
	}
	if rm.Cancelled() {
		t.Error("enter should not cancel")
	}
}

func TestSizingModel_Cancel(t *testing.T) {
	m := NewSizingModel(testSizingPlan())

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	rm := result.(SizingModel)
	if !rm.Done() {
		t.Error("q should finish")
	}
	if !rm.Cancelled() {
		t.Error("q should cancel")
	}
}

func TestSizingModel_BenchmarkToggle(t *testing.T) {
	m := NewSizingModel(testSizingPlan())

	// 'b' should not crash or change done state
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	rm := result.(SizingModel)
	if rm.Done() {
		t.Error("b should not finish")
	}
}

func TestSizingModel_View(t *testing.T) {
	m := NewSizingModel(testSizingPlan())
	m.width = 100
	m.height = 30

	v := m.View()
	if !strings.Contains(v, "Step 6") {
		t.Error("view should contain step title")
	}
	if !strings.Contains(v, "EMR") || !strings.Contains(v, "r5.4xlarge") {
		t.Error("view should contain Spark plan details")
	}
	if !strings.Contains(v, "M60") {
		t.Error("view should contain MongoDB tier")
	}
}

func TestSizingModel_NilPlan(t *testing.T) {
	m := NewSizingModel(nil)
	v := m.View()
	if !strings.Contains(v, "No sizing plan") {
		t.Error("view should indicate no plan")
	}
}

func TestSizingModel_ConfirmWithF(t *testing.T) {
	m := NewSizingModel(testSizingPlan())
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	rm := result.(SizingModel)
	if !rm.Done() {
		t.Error("f should finish")
	}
}
