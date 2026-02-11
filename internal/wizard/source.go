package wizard

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/discovery"
	"github.com/reloquent/reloquent/internal/schema"
)

// SourceResult is returned when the source step completes.
type SourceResult struct {
	Config *config.SourceConfig
	Schema *schema.Schema
}

// field indexes
const (
	fieldDBType = iota
	fieldHost
	fieldPort
	fieldDatabase
	fieldUsername
	fieldPassword
	fieldCount
)

// SourceModel is the bubbletea model for the source connection form.
type SourceModel struct {
	inputs       []textinput.Model
	focused      int
	dbTypeChoice int // 0=PostgreSQL, 1=Oracle
	err          error
	discovering  bool
	spinner      spinner.Model
	result       *SourceResult
	done         bool
	statusMsg    string
	width        int
}

type discoveryDoneMsg struct {
	cfg    *config.SourceConfig
	schema *schema.Schema
	err    error
}

func NewSourceModel() SourceModel {
	inputs := make([]textinput.Model, fieldCount)

	inputs[fieldDBType] = textinput.New()
	inputs[fieldDBType].Placeholder = "postgresql"
	inputs[fieldDBType].CharLimit = 20

	inputs[fieldHost] = textinput.New()
	inputs[fieldHost].Placeholder = "localhost"
	inputs[fieldHost].CharLimit = 256
	inputs[fieldHost].Focus()

	inputs[fieldPort] = textinput.New()
	inputs[fieldPort].Placeholder = "5432"
	inputs[fieldPort].CharLimit = 5

	inputs[fieldDatabase] = textinput.New()
	inputs[fieldDatabase].Placeholder = "mydb"
	inputs[fieldDatabase].CharLimit = 128

	inputs[fieldUsername] = textinput.New()
	inputs[fieldUsername].Placeholder = "postgres"
	inputs[fieldUsername].CharLimit = 128

	inputs[fieldPassword] = textinput.New()
	inputs[fieldPassword].Placeholder = ""
	inputs[fieldPassword].EchoMode = textinput.EchoPassword
	inputs[fieldPassword].EchoCharacter = '*'
	inputs[fieldPassword].CharLimit = 256

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return SourceModel{
		inputs:  inputs,
		focused: fieldHost, // skip DB type field, use tab to toggle
		spinner: s,
		width:   80,
	}
}

func (m SourceModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SourceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		if m.discovering {
			return m, nil // ignore input during discovery
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			m.done = true
			m.err = fmt.Errorf("cancelled")
			return m, tea.Quit

		case "tab", "down":
			m.focused = (m.focused + 1) % fieldCount
			// Skip fieldDBType, we use ctrl+t to toggle
			if m.focused == fieldDBType {
				m.focused = fieldHost
			}
			return m, m.updateFocus()

		case "shift+tab", "up":
			m.focused--
			if m.focused < fieldHost {
				m.focused = fieldPassword
			}
			return m, m.updateFocus()

		case "ctrl+t":
			m.dbTypeChoice = (m.dbTypeChoice + 1) % 2
			return m, nil

		case "enter":
			if m.focused == fieldPassword {
				return m, m.startDiscovery()
			}
			m.focused = (m.focused + 1) % fieldCount
			if m.focused == fieldDBType {
				m.focused = fieldHost
			}
			return m, m.updateFocus()
		}

	case discoveryDoneMsg:
		m.discovering = false
		if msg.err != nil {
			m.err = msg.err
			m.statusMsg = fmt.Sprintf("Connection failed: %v", msg.err)
			return m, nil
		}
		m.result = &SourceResult{Config: msg.cfg, Schema: msg.schema}
		m.done = true
		return m, tea.Quit

	case spinner.TickMsg:
		if m.discovering {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Update the focused text input
	if !m.discovering && m.focused >= fieldHost && m.focused < fieldCount {
		var cmd tea.Cmd
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m SourceModel) View() string {
	var b strings.Builder

	title := titleStyle.Render("Step 1: Source Database")
	b.WriteString(title + "\n\n")

	// DB type selector
	pg := "● PostgreSQL"
	oracle := "○ Oracle"
	if m.dbTypeChoice == 1 {
		pg = "○ PostgreSQL"
		oracle = "● Oracle"
	}
	b.WriteString(fmt.Sprintf("  Database type: %s  %s  (ctrl+t to toggle)\n\n",
		pg, oracle))

	labels := []string{"Host", "Port", "Database", "Username", "Password"}
	for i := fieldHost; i < fieldCount; i++ {
		label := fmt.Sprintf("  %-10s ", labels[i-fieldHost])
		cursor := "  "
		if i == m.focused {
			cursor = highlightStyle.Render("> ")
		}
		b.WriteString(cursor + dimStyle.Render(label) + m.inputs[i].View() + "\n")
	}

	b.WriteString("\n")

	if m.discovering {
		b.WriteString(fmt.Sprintf("  %s Connecting and discovering schema...\n", m.spinner.View()))
	} else if m.err != nil {
		b.WriteString(errStyle.Render("  "+m.statusMsg) + "\n")
		b.WriteString(dimStyle.Render("  Fix the issue and press Enter to retry\n"))
	} else {
		b.WriteString(dimStyle.Render("  Press Enter on Password to connect • tab/shift-tab to navigate • esc to cancel\n"))
	}

	return b.String()
}

// Result returns the discovery result, or nil if not completed.
func (m SourceModel) Result() *SourceResult {
	return m.result
}

// Done returns true if the model has finished (success or cancelled).
func (m SourceModel) Done() bool {
	return m.done
}

// Cancelled returns true if the user cancelled.
func (m SourceModel) Cancelled() bool {
	return m.done && m.result == nil
}

func (m *SourceModel) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, fieldCount)
	for i := fieldHost; i < fieldCount; i++ {
		if i == m.focused {
			cmds[i] = m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *SourceModel) startDiscovery() tea.Cmd {
	m.discovering = true
	m.err = nil
	m.statusMsg = ""

	cfg := m.buildConfig()

	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			d, err := discovery.New(cfg)
			if err != nil {
				return discoveryDoneMsg{err: err}
			}
			defer d.Close()

			if err := d.Connect(ctx); err != nil {
				return discoveryDoneMsg{err: err}
			}

			s, err := d.Discover(ctx)
			if err != nil {
				return discoveryDoneMsg{err: err}
			}

			return discoveryDoneMsg{cfg: cfg, schema: s}
		},
	)
}

func (m *SourceModel) buildConfig() *config.SourceConfig {
	dbType := "postgresql"
	if m.dbTypeChoice == 1 {
		dbType = "oracle"
	}

	host := m.inputs[fieldHost].Value()
	if host == "" {
		host = "localhost"
	}

	portStr := m.inputs[fieldPort].Value()
	port := 5432
	if dbType == "oracle" {
		port = 1521
	}
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	database := m.inputs[fieldDatabase].Value()
	username := m.inputs[fieldUsername].Value()
	password := m.inputs[fieldPassword].Value()

	return &config.SourceConfig{
		Type:           dbType,
		Host:           host,
		Port:           port,
		Database:       database,
		Username:       username,
		Password:       password,
		ReadOnly:       true,
		MaxConnections: 20,
	}
}

// styles
var (
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).BorderStyle(lipgloss.DoubleBorder()).BorderBottom(true).Padding(0, 1)
	highlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	dimStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	successStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
)
