package wizard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/report"
)

func TestNewReadinessModel(t *testing.T) {
	m := NewReadinessModel()
	if m.Done() {
		t.Error("should not be done initially")
	}
}

func TestReadinessModel_Ready(t *testing.T) {
	m := NewReadinessModel()
	m.SetReport(&report.MigrationReport{
		ProductionReady: true,
		Source:          report.SourceSummary{Type: "postgresql", Host: "localhost", Database: "mydb", Tables: 5},
		Target:          report.TargetSummary{Database: "target", Topology: "replica_set", Collections: 3},
		Indexes:         report.IndexSummary{TotalIndexes: 4, Status: "complete"},
		ReadinessChecks: []report.ReadinessCheck{
			{Name: "validation", Passed: true},
			{Name: "indexes", Passed: true},
		},
		NextSteps: []string{"Switch connection strings"},
	})

	v := m.View()
	if !strings.Contains(v, "READY FOR PRODUCTION") {
		t.Error("should show ready banner")
	}
	if !strings.Contains(v, "postgresql") {
		t.Error("should show source info")
	}
	if !strings.Contains(v, "Switch connection strings") {
		t.Error("should show next steps")
	}
}

func TestReadinessModel_NotReady(t *testing.T) {
	m := NewReadinessModel()
	m.SetReport(&report.MigrationReport{
		ProductionReady: false,
		Source:          report.SourceSummary{Type: "oracle"},
		Target:          report.TargetSummary{Database: "target"},
		ReadinessChecks: []report.ReadinessCheck{
			{Name: "validation", Passed: false},
		},
	})

	v := m.View()
	if !strings.Contains(v, "REQUIRES ATTENTION") {
		t.Error("should show requires attention banner")
	}
}

func TestReadinessModel_Enter(t *testing.T) {
	m := NewReadinessModel()
	m.SetReport(&report.MigrationReport{ProductionReady: true})

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	rm := result.(ReadinessModel)
	if !rm.Done() {
		t.Error("enter should exit")
	}
}

func TestReadinessModel_View_Title(t *testing.T) {
	m := NewReadinessModel()
	v := m.View()
	if !strings.Contains(v, "Migration Complete") {
		t.Error("view should contain title")
	}
}
