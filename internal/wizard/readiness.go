package wizard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/report"
)

// ReadinessModel is the bubbletea model for the production readiness display.
type ReadinessModel struct {
	report    *report.MigrationReport
	done      bool
	cancelled bool
	width     int
	height    int
}

// NewReadinessModel creates a readiness display model.
func NewReadinessModel() ReadinessModel {
	return ReadinessModel{
		width:  100,
		height: 24,
	}
}

func (m ReadinessModel) Init() tea.Cmd {
	return nil
}

func (m ReadinessModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "enter":
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m ReadinessModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Migration Complete"))
	b.WriteString("\n\n")

	if m.report == nil {
		b.WriteString("  Generating readiness report...\n")
		return b.String()
	}

	// Banner
	if m.report.ProductionReady {
		b.WriteString(successStyle.Render("  READY FOR PRODUCTION"))
	} else {
		b.WriteString(errStyle.Render("  REQUIRES ATTENTION"))
	}
	b.WriteString("\n\n")

	// Migration summary
	b.WriteString(highlightStyle.Render("  Summary:"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("    Source: %s (%s/%s, %d tables)\n",
		m.report.Source.Type, m.report.Source.Host, m.report.Source.Database, m.report.Source.Tables))
	b.WriteString(fmt.Sprintf("    Target: %s (%s, %d collections)\n",
		m.report.Target.Database, m.report.Target.Topology, m.report.Target.Collections))
	b.WriteString(fmt.Sprintf("    Indexes: %d (%s)\n", m.report.Indexes.TotalIndexes, m.report.Indexes.Status))
	b.WriteString("\n")

	// Readiness checklist
	b.WriteString(highlightStyle.Render("  Readiness Checks:"))
	b.WriteString("\n")
	for _, rc := range m.report.ReadinessChecks {
		icon := successStyle.Render("PASS")
		if !rc.Passed {
			icon = errStyle.Render("FAIL")
		}
		b.WriteString(fmt.Sprintf("    [%s] %s\n", icon, rc.Name))
	}
	b.WriteString("\n")

	// Next steps
	if len(m.report.NextSteps) > 0 {
		b.WriteString(highlightStyle.Render("  Next Steps:"))
		b.WriteString("\n")
		for i, s := range m.report.NextSteps {
			b.WriteString(fmt.Sprintf("    %d. %s\n", i+1, s))
		}
		b.WriteString("\n")
	}

	// Report path
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Report: %s", "~/.reloquent/migration-report.json")))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("  Press enter to exit"))
	b.WriteString("\n")

	return b.String()
}

// Done returns true when the model is finished.
func (m ReadinessModel) Done() bool {
	return m.done
}

// Cancelled returns true if the user cancelled.
func (m ReadinessModel) Cancelled() bool {
	return m.cancelled
}

// SetReport sets the migration report for display.
func (m *ReadinessModel) SetReport(r *report.MigrationReport) {
	m.report = r
}
