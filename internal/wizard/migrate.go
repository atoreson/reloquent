package wizard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/migration"
	"github.com/reloquent/reloquent/internal/sizing"
)

// MigrateModel is the bubbletea model for Step 9: Migration Execution.
type MigrateModel struct {
	status      *migration.Status
	failAction  migration.FailureAction
	showingFail bool
	done        bool
	cancelled   bool
	width       int
	height      int
}

// NewMigrateModel creates a migration execution model.
func NewMigrateModel() MigrateModel {
	return MigrateModel{
		status: &migration.Status{Phase: "pending"},
		width:  100,
		height: 24,
	}
}

func (m MigrateModel) Init() tea.Cmd {
	return nil
}

func (m MigrateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.showingFail {
			switch msg.String() {
			case "r":
				m.failAction = migration.ActionRetryFailed
				m.done = true
				return m, tea.Quit
			case "a":
				m.failAction = migration.ActionRestartAll
				m.done = true
				return m, tea.Quit
			case "q":
				m.failAction = migration.ActionAbort
				m.done = true
				m.cancelled = true
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "esc":
			m.done = true
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			if m.status.Phase == "completed" || m.status.Phase == "failed" || m.status.Phase == "partial_failure" {
				m.done = true
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m MigrateModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Step 9: Migration"))
	b.WriteString("\n\n")

	if m.status == nil {
		b.WriteString("  Waiting for migration to start...\n")
		return b.String()
	}

	// Phase
	phaseStyle := dimStyle
	switch m.status.Phase {
	case "running":
		phaseStyle = highlightStyle
	case "completed":
		phaseStyle = successStyle
	case "failed", "partial_failure":
		phaseStyle = errStyle
	}
	b.WriteString(fmt.Sprintf("  Phase: %s\n", phaseStyle.Render(m.status.Phase)))

	// Overall progress
	if m.status.Overall.DocsTotal > 0 {
		pct := m.status.Overall.PercentComplete
		bar := renderProgressBar(pct, m.width-20)
		b.WriteString(fmt.Sprintf("  %s %.1f%%\n", bar, pct))
		b.WriteString(fmt.Sprintf("  %d / %d docs", m.status.Overall.DocsWritten, m.status.Overall.DocsTotal))
		if m.status.Overall.ThroughputMBps > 0 {
			b.WriteString(fmt.Sprintf("  (%.1f MB/s)", m.status.Overall.ThroughputMBps))
		}
		b.WriteString("\n")
	}

	// Time
	if m.status.ElapsedTime > 0 {
		b.WriteString(fmt.Sprintf("  Elapsed: %s", sizing.FormatDuration(m.status.ElapsedTime)))
		if m.status.EstimatedRemain > 0 {
			b.WriteString(fmt.Sprintf("  Remaining: ~%s", sizing.FormatDuration(m.status.EstimatedRemain)))
		}
		b.WriteString("\n")
	}

	// Per-collection progress
	if len(m.status.Collections) > 0 {
		b.WriteString("\n")
		b.WriteString(highlightStyle.Render("  Collections:"))
		b.WriteString("\n")
		for _, col := range m.status.Collections {
			stateIcon := " "
			switch col.State {
			case "completed":
				stateIcon = successStyle.Render("OK")
			case "running":
				stateIcon = highlightStyle.Render(">>")
			case "failed":
				stateIcon = errStyle.Render("XX")
			case "pending":
				stateIcon = dimStyle.Render("..")
			}
			line := fmt.Sprintf("  %s %-30s", stateIcon, col.Name)
			if col.State == "running" && col.DocsTotal > 0 {
				line += fmt.Sprintf(" %5.1f%%", col.PercentComplete)
			} else if col.State == "failed" {
				line += fmt.Sprintf(" %s", errStyle.Render(col.Error))
			}
			b.WriteString(line + "\n")
		}
	}

	// Errors
	if len(m.status.Errors) > 0 {
		b.WriteString("\n")
		b.WriteString(errStyle.Render("  Errors:"))
		b.WriteString("\n")
		for _, e := range m.status.Errors {
			b.WriteString(fmt.Sprintf("  - %s\n", e))
		}
	}

	// Failure dialog
	if m.showingFail {
		b.WriteString("\n")
		b.WriteString(errStyle.Render("  Some collections failed. What would you like to do?"))
		b.WriteString("\n")
		b.WriteString("  r: Retry failed only  a: Restart all  q: Abort\n")
	} else if m.status.Phase == "completed" {
		b.WriteString("\n")
		b.WriteString(successStyle.Render("  Migration completed successfully!"))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  Press enter to continue"))
	} else if m.status.Phase != "failed" && m.status.Phase != "partial_failure" {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  q: cancel migration"))
	}

	return b.String()
}

// Done returns true when the model is finished.
func (m MigrateModel) Done() bool {
	return m.done
}

// Cancelled returns true if the user cancelled.
func (m MigrateModel) Cancelled() bool {
	return m.cancelled
}

// FailureAction returns the chosen failure action.
func (m MigrateModel) FailureAction() migration.FailureAction {
	return m.failAction
}

// SetStatus updates the migration status for display.
func (m *MigrateModel) SetStatus(status *migration.Status) {
	m.status = status
	if status.Phase == "partial_failure" {
		m.showingFail = true
	}
}

func renderProgressBar(pct float64, width int) string {
	if width < 10 {
		width = 10
	}
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled
	return "[" + strings.Repeat("=", filled) + strings.Repeat(" ", empty) + "]"
}
