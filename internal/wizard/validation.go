package wizard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/validation"
)

// ValidationModel is the bubbletea model for Step 10: Validation.
type ValidationModel struct {
	result     *validation.Result
	checks     []validationCheck
	done       bool
	cancelled  bool
	failed     bool
	width      int
	height     int
}

type validationCheck struct {
	Collection string
	CheckType  string
	Passed     bool
}

// NewValidationModel creates a validation step model.
func NewValidationModel() ValidationModel {
	return ValidationModel{
		width:  100,
		height: 24,
	}
}

func (m ValidationModel) Init() tea.Cmd {
	return nil
}

func (m ValidationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "p":
			if m.failed {
				m.done = true
				return m, tea.Quit
			}
		case "enter":
			if m.result != nil {
				m.done = true
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m ValidationModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Step 10: Validation"))
	b.WriteString("\n\n")

	if len(m.checks) == 0 && m.result == nil {
		b.WriteString("  Waiting for validation to start...\n")
		return b.String()
	}

	// Show per-collection, per-check results
	collChecks := make(map[string][]validationCheck)
	for _, c := range m.checks {
		collChecks[c.Collection] = append(collChecks[c.Collection], c)
	}

	for coll, checks := range collChecks {
		b.WriteString(fmt.Sprintf("  %s:\n", highlightStyle.Render(coll)))
		for _, c := range checks {
			icon := successStyle.Render("PASS")
			if !c.Passed {
				icon = errStyle.Render("FAIL")
			}
			b.WriteString(fmt.Sprintf("    [%s] %s\n", icon, c.CheckType))
		}
	}

	// Overall status
	if m.result != nil {
		b.WriteString("\n")
		switch m.result.Status {
		case "PASS":
			b.WriteString(successStyle.Render("  Validation: PASS"))
		case "FAIL":
			b.WriteString(errStyle.Render("  Validation: FAIL"))
		case "PARTIAL":
			b.WriteString(errStyle.Render("  Validation: PARTIAL (some checks failed)"))
		}
		b.WriteString("\n\n")

		if m.failed {
			b.WriteString(dimStyle.Render("  p: proceed anyway  q: cancel"))
			b.WriteString("\n")
		} else {
			b.WriteString(dimStyle.Render("  Press enter to continue"))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// Done returns true when the model is finished.
func (m ValidationModel) Done() bool {
	return m.done
}

// Cancelled returns true if the user cancelled.
func (m ValidationModel) Cancelled() bool {
	return m.cancelled
}

// Result returns the validation result.
func (m ValidationModel) Result() *validation.Result {
	return m.result
}

// AddCheck records a validation check result.
func (m *ValidationModel) AddCheck(collection, checkType string, passed bool) {
	m.checks = append(m.checks, validationCheck{
		Collection: collection,
		CheckType:  checkType,
		Passed:     passed,
	})
}

// SetResult sets the final validation result.
func (m *ValidationModel) SetResult(result *validation.Result) {
	m.result = result
	m.failed = result.Status != "PASS"
}
