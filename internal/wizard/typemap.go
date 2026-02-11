package wizard

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/typemap"
)

// TypeMapModel is the bubbletea model for the type mapping review step (Step 5).
type TypeMapModel struct {
	typeMap    *typemap.TypeMap
	types     []string // source types actually in use, sorted
	cursor    int
	done      bool
	cancelled bool
	width     int
	height    int
}

// NewTypeMapModel creates a type mapping review model.
// It scans the schema for types actually in use by the given tables and
// initializes the type map with defaults for the database type.
func NewTypeMapModel(s *schema.Schema, dbType string, existing *typemap.TypeMap) TypeMapModel {
	var tm *typemap.TypeMap
	if existing != nil {
		tm = existing
	} else {
		tm = typemap.ForDatabase(dbType)
	}

	// Collect types actually in use
	typeSet := make(map[string]bool)
	for _, t := range s.Tables {
		for _, col := range t.Columns {
			typeSet[col.DataType] = true
		}
	}

	// Ensure all in-use types are in the map
	for typ := range typeSet {
		if _, ok := tm.AllMappings()[typ]; !ok {
			tm.AllMappings()[typ] = typemap.BSONString
		}
	}

	types := make([]string, 0, len(typeSet))
	for typ := range typeSet {
		types = append(types, typ)
	}
	sort.Strings(types)

	return TypeMapModel{
		typeMap: tm,
		types:  types,
		width:  100,
		height: 24,
	}
}

func (m TypeMapModel) Init() tea.Cmd {
	return nil
}

func (m TypeMapModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if len(m.types) == 0 {
			switch msg.String() {
			case "enter", "f":
				m.done = true
				return m, tea.Quit
			case "q", "esc", "ctrl+c":
				m.done = true
				m.cancelled = true
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.done = true
			m.cancelled = true
			return m, tea.Quit

		case "j", "down":
			if m.cursor < len(m.types)-1 {
				m.cursor++
			}

		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}

		case "e": // edit: cycle through BSON types
			if m.cursor < len(m.types) {
				sourceType := m.types[m.cursor]
				current := m.typeMap.Resolve(sourceType)
				next := nextBSONType(current)
				m.typeMap.Override(sourceType, next)
			}

		case "d": // restore default
			if m.cursor < len(m.types) {
				sourceType := m.types[m.cursor]
				m.typeMap.RestoreDefault(sourceType)
			}

		case "enter", "f":
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m TypeMapModel) View() string {
	var b strings.Builder

	title := titleStyle.Render("Step 5: Type Mapping Review")
	b.WriteString(title + "\n\n")

	if len(m.types) == 0 {
		b.WriteString("  No types found in selected tables.\n\n")
		b.WriteString(dimStyle.Render("  Press enter to confirm • q to cancel\n"))
		return b.String()
	}

	// Header
	b.WriteString(fmt.Sprintf("  %-30s %-16s %s\n", "Source Type", "BSON Type", "Status"))
	b.WriteString("  " + strings.Repeat("─", 60) + "\n")

	// Visible window
	maxVisible := m.height - 10
	if maxVisible < 5 {
		maxVisible = 5
	}
	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(m.types) {
		end = len(m.types)
	}

	for i := start; i < end; i++ {
		sourceType := m.types[i]
		bsonType := m.typeMap.Resolve(sourceType)

		cursor := "  "
		if i == m.cursor {
			cursor = highlightStyle.Render("> ")
		}

		status := dimStyle.Render("default")
		if m.typeMap.IsOverridden(sourceType) {
			status = successStyle.Render("override ★")
		}

		b.WriteString(fmt.Sprintf("%s%-30s %-16s %s\n",
			cursor, sourceType, string(bsonType), status))
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  e edit • d restore default • enter confirm • q cancel\n"))

	return b.String()
}

// Result returns the type mapping.
func (m TypeMapModel) Result() *typemap.TypeMap {
	if m.cancelled {
		return nil
	}
	return m.typeMap
}

// Done returns true if the model has finished.
func (m TypeMapModel) Done() bool {
	return m.done
}

// Cancelled returns true if the user cancelled.
func (m TypeMapModel) Cancelled() bool {
	return m.done && m.cancelled
}

// nextBSONType returns the next BSON type in the cycle.
func nextBSONType(current typemap.BSONType) typemap.BSONType {
	types := typemap.AllBSONTypes
	for i, t := range types {
		if t == current {
			return types[(i+1)%len(types)]
		}
	}
	return types[0]
}
