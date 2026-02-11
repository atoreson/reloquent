package wizard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/reloquent/reloquent/internal/config"
)

// TargetResult is returned when the target step completes.
type TargetResult struct {
	Config *config.TargetConfig
}

// target field indexes
const (
	targetFieldConnStr = iota
	targetFieldDatabase
	targetFieldCount
)

// targetConnectDoneMsg is sent when the MongoDB ping completes.
type targetConnectDoneMsg struct {
	err error
}

// TargetModel is the bubbletea model for the target MongoDB connection form.
type TargetModel struct {
	inputs     []textinput.Model
	focused    int
	err        error
	connecting bool
	spinner    spinner.Model
	result     *TargetResult
	done       bool
	statusMsg  string
	width      int
}

// NewTargetModel creates a new target connection model.
func NewTargetModel() TargetModel {
	inputs := make([]textinput.Model, targetFieldCount)

	inputs[targetFieldConnStr] = textinput.New()
	inputs[targetFieldConnStr].Placeholder = "mongodb://localhost:27017"
	inputs[targetFieldConnStr].CharLimit = 512
	inputs[targetFieldConnStr].Focus()

	inputs[targetFieldDatabase] = textinput.New()
	inputs[targetFieldDatabase].Placeholder = "mydb"
	inputs[targetFieldDatabase].CharLimit = 128

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return TargetModel{
		inputs:  inputs,
		focused: targetFieldConnStr,
		spinner: s,
		width:   80,
	}
}

func (m TargetModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m TargetModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		if m.connecting {
			return m, nil // ignore input during connection test
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			m.done = true
			m.err = fmt.Errorf("cancelled")
			return m, tea.Quit

		case "tab", "down":
			m.focused = (m.focused + 1) % targetFieldCount
			return m, m.updateFocus()

		case "shift+tab", "up":
			m.focused--
			if m.focused < 0 {
				m.focused = targetFieldCount - 1
			}
			return m, m.updateFocus()

		case "enter":
			if m.focused == targetFieldDatabase {
				return m, m.startConnect()
			}
			m.focused = (m.focused + 1) % targetFieldCount
			return m, m.updateFocus()
		}

	case targetConnectDoneMsg:
		m.connecting = false
		if msg.err != nil {
			m.err = msg.err
			m.statusMsg = fmt.Sprintf("Connection failed: %v", msg.err)
			return m, nil
		}
		m.result = &TargetResult{Config: m.buildConfig()}
		m.done = true
		return m, tea.Quit

	case spinner.TickMsg:
		if m.connecting {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Update the focused text input
	if !m.connecting && m.focused >= 0 && m.focused < targetFieldCount {
		var cmd tea.Cmd
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m TargetModel) View() string {
	var b strings.Builder

	title := titleStyle.Render("Step 2: Target MongoDB")
	b.WriteString(title + "\n\n")

	labels := []string{"Connection String", "Database"}
	for i := 0; i < targetFieldCount; i++ {
		label := fmt.Sprintf("  %-20s ", labels[i])
		cursor := "  "
		if i == m.focused {
			cursor = highlightStyle.Render("> ")
		}
		b.WriteString(cursor + dimStyle.Render(label) + m.inputs[i].View() + "\n")
	}

	b.WriteString("\n")

	if m.connecting {
		b.WriteString(fmt.Sprintf("  %s Testing connection...\n", m.spinner.View()))
	} else if m.err != nil {
		b.WriteString(errStyle.Render("  "+m.statusMsg) + "\n")
		b.WriteString(dimStyle.Render("  Fix the issue and press Enter to retry\n"))
	} else {
		b.WriteString(dimStyle.Render("  Press Enter on Database to connect • tab/shift-tab to navigate • esc to cancel\n"))
	}

	return b.String()
}

// Result returns the target config result, or nil if not completed.
func (m TargetModel) Result() *TargetResult {
	return m.result
}

// Done returns true if the model has finished (success or cancelled).
func (m TargetModel) Done() bool {
	return m.done
}

// Cancelled returns true if the user cancelled.
func (m TargetModel) Cancelled() bool {
	return m.done && m.result == nil
}

func (m *TargetModel) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, targetFieldCount)
	for i := 0; i < targetFieldCount; i++ {
		if i == m.focused {
			cmds[i] = m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *TargetModel) startConnect() tea.Cmd {
	m.connecting = true
	m.err = nil
	m.statusMsg = ""

	cfg := m.buildConfig()

	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			client, err := mongo.Connect(options.Client().ApplyURI(cfg.ConnectionString))
			if err != nil {
				return targetConnectDoneMsg{err: err}
			}
			defer client.Disconnect(ctx)

			if err := client.Ping(ctx, nil); err != nil {
				return targetConnectDoneMsg{err: err}
			}

			return targetConnectDoneMsg{}
		},
	)
}

func (m *TargetModel) buildConfig() *config.TargetConfig {
	connStr := m.inputs[targetFieldConnStr].Value()
	if connStr == "" {
		connStr = "mongodb://localhost:27017"
	}

	database := m.inputs[targetFieldDatabase].Value()
	if database == "" {
		database = "mydb"
	}

	return &config.TargetConfig{
		Type:             "mongodb",
		ConnectionString: connStr,
		Database:         database,
	}
}
