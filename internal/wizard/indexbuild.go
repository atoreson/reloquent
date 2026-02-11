package wizard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/target"
)

// IndexBuildModel is the bubbletea model for Step 11: Index Builds.
type IndexBuildModel struct {
	totalIndexes int
	completed    int
	statuses     []target.IndexBuildStatus
	done         bool
	cancelled    bool
	finished     bool
	width        int
	height       int
}

// NewIndexBuildModel creates an index build step model.
func NewIndexBuildModel(totalIndexes int) IndexBuildModel {
	return IndexBuildModel{
		totalIndexes: totalIndexes,
		width:        100,
		height:       24,
	}
}

func (m IndexBuildModel) Init() tea.Cmd {
	return nil
}

func (m IndexBuildModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			m.done = true
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			if m.finished {
				m.done = true
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m IndexBuildModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Step 11: Index Builds"))
	b.WriteString("\n\n")

	if m.totalIndexes == 0 {
		b.WriteString("  No indexes to build.\n")
		b.WriteString(dimStyle.Render("  Press enter to continue"))
		b.WriteString("\n")
		return b.String()
	}

	// Summary
	if m.finished {
		b.WriteString(successStyle.Render(fmt.Sprintf("  All %d indexes built successfully.", m.totalIndexes)))
		b.WriteString("\n\n")
	} else {
		b.WriteString(fmt.Sprintf("  Building %d indexes... %d/%d complete.\n\n",
			m.totalIndexes, m.completed, m.totalIndexes))
	}

	// Per-index status
	for _, s := range m.statuses {
		var icon string
		switch s.Phase {
		case "complete":
			icon = successStyle.Render("OK")
		case "building":
			icon = highlightStyle.Render(">>")
		default:
			icon = dimStyle.Render("..")
		}

		line := fmt.Sprintf("  %s %-30s %-20s", icon, s.IndexName, s.Collection)
		if s.Phase == "building" && s.Progress > 0 {
			barWidth := m.width - 60
			if barWidth < 10 {
				barWidth = 10
			}
			bar := renderProgressBar(s.Progress, barWidth)
			line += fmt.Sprintf(" %s %.0f%%", bar, s.Progress)
		}
		b.WriteString(line + "\n")
	}

	if m.finished {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  Press enter to continue"))
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  q: cancel"))
		b.WriteString("\n")
	}

	return b.String()
}

// Done returns true when the model is finished.
func (m IndexBuildModel) Done() bool {
	return m.done
}

// Cancelled returns true if the user cancelled.
func (m IndexBuildModel) Cancelled() bool {
	return m.cancelled
}

// UpdateProgress sets the current index build statuses.
func (m *IndexBuildModel) UpdateProgress(statuses []target.IndexBuildStatus) {
	m.statuses = statuses
	m.completed = 0
	for _, s := range statuses {
		if s.Phase == "complete" {
			m.completed++
		}
	}
}

// SetFinished marks index builds as complete.
func (m *IndexBuildModel) SetFinished() {
	m.finished = true
	m.completed = m.totalIndexes
}
