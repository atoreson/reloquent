package wizard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/target"
)

// PreMigrationModel is the bubbletea model for Step 8: Pre-Migration Setup.
type PreMigrationModel struct {
	topology   *target.TopologyInfo
	validation *target.ValidationResult
	collections []string
	setupDone   bool
	done        bool
	cancelled   bool
	width       int
	height      int
}

// NewPreMigrationModel creates a pre-migration setup model.
func NewPreMigrationModel(collections []string) PreMigrationModel {
	return PreMigrationModel{
		collections: collections,
		width:       100,
		height:      24,
	}
}

func (m PreMigrationModel) Init() tea.Cmd {
	return nil
}

func (m PreMigrationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "f":
			m.done = true
			return m, tea.Quit
		case "q", "esc":
			m.done = true
			m.cancelled = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m PreMigrationModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Step 8: Pre-Migration Setup"))
	b.WriteString("\n\n")

	// Topology info
	if m.topology != nil {
		b.WriteString(highlightStyle.Render("  Topology: "))
		b.WriteString(fmt.Sprintf("%s", m.topology.Type))
		if m.topology.IsAtlas {
			b.WriteString(" (Atlas)")
		}
		b.WriteString(fmt.Sprintf("  Version: %s\n", m.topology.ServerVersion))
		if m.topology.ShardCount > 0 {
			b.WriteString(fmt.Sprintf("  Shards: %d\n", m.topology.ShardCount))
		}
		b.WriteString("\n")
	}

	// Validation results
	if m.validation != nil {
		if m.validation.Passed {
			b.WriteString(successStyle.Render("  Validation: PASSED"))
			b.WriteString("\n")
		} else {
			b.WriteString(errStyle.Render("  Validation: FAILED"))
			b.WriteString("\n")
		}

		for _, w := range m.validation.Warnings {
			b.WriteString(fmt.Sprintf("  %s %s: %s\n", "!", w.Category, w.Message))
			if w.Suggestion != "" {
				b.WriteString(fmt.Sprintf("    %s\n", dimStyle.Render(w.Suggestion)))
			}
		}
		for _, e := range m.validation.Errors {
			b.WriteString(fmt.Sprintf("  %s %s: %s\n", errStyle.Render("X"), e.Category, e.Message))
			if e.Suggestion != "" {
				b.WriteString(fmt.Sprintf("    %s\n", dimStyle.Render(e.Suggestion)))
			}
		}
		b.WriteString("\n")
	}

	// Collections setup
	b.WriteString(highlightStyle.Render("  Collections:"))
	if m.setupDone {
		b.WriteString(successStyle.Render(" Created"))
	} else {
		b.WriteString(" Pending")
	}
	b.WriteString("\n")
	for _, name := range m.collections {
		b.WriteString(fmt.Sprintf("    %s\n", name))
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  enter: continue  q: cancel"))

	return b.String()
}

// Done returns true when the model is finished.
func (m PreMigrationModel) Done() bool {
	return m.done
}

// Cancelled returns true if the user cancelled.
func (m PreMigrationModel) Cancelled() bool {
	return m.cancelled
}

// SetTopology sets the detected topology for display.
func (m *PreMigrationModel) SetTopology(topo *target.TopologyInfo) {
	m.topology = topo
}

// SetValidation sets the validation result for display.
func (m *PreMigrationModel) SetValidation(result *target.ValidationResult) {
	m.validation = result
}

// SetSetupDone marks collection creation as complete.
func (m *PreMigrationModel) SetSetupDone() {
	m.setupDone = true
}
