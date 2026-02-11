package wizard

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/selection"
)

// TableSelectResult is returned when the user confirms their selection.
type TableSelectResult struct {
	Selected []schema.Table
}

// SortField controls the column used for sorting.
type SortField int

const (
	SortByName SortField = iota
	SortByRows
	SortBySize
	SortByFKs
)

// tableEntry represents a table row in the selector.
type tableEntry struct {
	table    schema.Table
	selected bool
	visible  bool // false when filtered out by search
}

// TableSelectModel is the bubbletea model for interactive table selection.
type TableSelectModel struct {
	entries   []tableEntry
	cursor    int
	filter    string
	filtering bool // true when the filter bar is active

	sortField SortField
	sortAsc   bool

	done      bool
	cancelled bool
	width     int
	height    int

	// precomputed visible indexes for fast cursor navigation
	visibleIdxs []int
}

// NewTableSelectModel creates a new table selector from discovered tables.
// preSelected optionally pre-selects tables by name (for resume).
func NewTableSelectModel(tables []schema.Table, preSelected []string) TableSelectModel {
	preMap := make(map[string]bool, len(preSelected))
	for _, n := range preSelected {
		preMap[n] = true
	}

	entries := make([]tableEntry, len(tables))
	for i, t := range tables {
		entries[i] = tableEntry{
			table:    t,
			selected: preMap[t.Name],
			visible:  true,
		}
	}

	m := TableSelectModel{
		entries: entries,
		sortAsc: true,
		width:   100,
		height:  24,
	}
	m.sortEntries()
	m.recomputeVisible()
	return m
}

func (m TableSelectModel) Init() tea.Cmd {
	return nil
}

func (m TableSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.filtering {
			return m.updateFilter(msg)
		}
		return m.updateNormal(msg)
	}
	return m, nil
}

func (m TableSelectModel) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.cancelled = true
		m.done = true
		return m, tea.Quit

	case "up", "k":
		m.moveCursor(-1)

	case "down", "j":
		m.moveCursor(1)

	case "home":
		if len(m.visibleIdxs) > 0 {
			m.cursor = 0
		}

	case "end":
		if len(m.visibleIdxs) > 0 {
			m.cursor = len(m.visibleIdxs) - 1
		}

	case " ":
		m.toggleCurrent()

	case "a":
		m.selectAll()

	case "n":
		m.deselectAll()

	case "/":
		m.filtering = true
		m.filter = ""
		return m, nil

	case "s":
		m.cycleSort()

	case "d":
		m.selectDependencies()

	case "enter":
		if m.selectedCount() == 0 {
			return m, nil // don't allow empty selection
		}
		m.done = true
		return m, tea.Quit
	}

	return m, nil
}

func (m TableSelectModel) updateFilter(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filter = ""
		m.applyFilter()
		return m, nil

	case "enter":
		m.filtering = false
		// Keep the filter applied
		return m, nil

	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.applyFilter()
		}
		return m, nil

	default:
		if len(msg.String()) == 1 {
			m.filter += msg.String()
			m.applyFilter()
		}
		return m, nil
	}
}

func (m TableSelectModel) View() string {
	var b strings.Builder

	title := titleStyle.Render("Step 3: Select Tables")
	b.WriteString(title + "\n\n")

	// Filter bar
	if m.filtering {
		b.WriteString(highlightStyle.Render("  Filter: ") + m.filter + "█\n\n")
	} else if m.filter != "" {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  Filter: %s (/ to change, esc in filter to clear)", m.filter)) + "\n\n")
	}

	// Column headers
	header := fmt.Sprintf("  %-3s %-30s %12s %12s %4s", "", "Table", "Rows", "Size", "FKs")
	b.WriteString(dimStyle.Render(header) + "\n")
	b.WriteString(dimStyle.Render("  " + strings.Repeat("─", min(m.width-4, 70))) + "\n")

	// Calculate visible window
	listHeight := m.height - 12 // Reserve space for header, footer, summary
	if listHeight < 5 {
		listHeight = 5
	}

	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
	}

	end := start + listHeight
	if end > len(m.visibleIdxs) {
		end = len(m.visibleIdxs)
	}

	if len(m.visibleIdxs) == 0 {
		b.WriteString(dimStyle.Render("  No tables match the filter\n"))
	}

	for vi := start; vi < end; vi++ {
		idx := m.visibleIdxs[vi]
		e := m.entries[idx]

		checkbox := "[ ]"
		if e.selected {
			checkbox = selectedStyle.Render("[x]")
		}

		cursor := "  "
		nameStyle := lipgloss.NewStyle()
		if vi == m.cursor {
			cursor = highlightStyle.Render("> ")
			nameStyle = nameStyle.Bold(true)
		}

		name := truncate(e.table.Name, 30)
		rows := formatNumber(e.table.RowCount)
		size := formatBytes(e.table.SizeBytes)
		fks := fmt.Sprintf("%d", len(e.table.ForeignKeys))

		line := fmt.Sprintf("%s%s %-30s %12s %12s %4s",
			cursor, checkbox, nameStyle.Render(name), rows, size, fks)
		b.WriteString(line + "\n")
	}

	// Scroll indicator
	if len(m.visibleIdxs) > listHeight {
		pct := 0
		if len(m.visibleIdxs) > 1 {
			pct = m.cursor * 100 / (len(m.visibleIdxs) - 1)
		}
		b.WriteString(dimStyle.Render(fmt.Sprintf("\n  Showing %d-%d of %d (%d%%)",
			start+1, end, len(m.visibleIdxs), pct)) + "\n")
	}

	b.WriteString("\n")

	// Summary bar
	selTables := m.getSelected()
	totalSize := selection.TotalSize(selTables)
	totalRows := selection.TotalRows(selTables)

	summary := fmt.Sprintf("  Selected: %d tables, %s rows, %s",
		len(selTables), formatNumber(totalRows), formatBytes(totalSize))
	b.WriteString(summaryStyle.Render(summary) + "\n")

	// Orphaned FK warnings
	orphans := selection.FindOrphanedReferences(selTables)
	if len(orphans) > 0 {
		shown := orphans
		if len(shown) > 3 {
			shown = shown[:3]
		}
		for _, o := range shown {
			b.WriteString(warnStyle.Render(fmt.Sprintf(
				"  ⚠ %s references %s (not selected)", o.Table, o.ReferencedTable)) + "\n")
		}
		if len(orphans) > 3 {
			b.WriteString(warnStyle.Render(fmt.Sprintf(
				"  ⚠ ...and %d more orphaned references", len(orphans)-3)) + "\n")
		}
	}

	// Sort indicator
	sortLabels := []string{"name", "rows", "size", "FKs"}
	dir := "↑"
	if !m.sortAsc {
		dir = "↓"
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Sort: %s %s", sortLabels[m.sortField], dir)) + "\n")

	// Keybindings help
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  space toggle • a all • n none • / filter • s sort • d add deps • enter confirm • q quit") + "\n")

	return b.String()
}

// Result returns the selection result, or nil if cancelled.
func (m TableSelectModel) Result() *TableSelectResult {
	if m.cancelled {
		return nil
	}
	selected := m.getSelected()
	if len(selected) == 0 {
		return nil
	}
	return &TableSelectResult{Selected: selected}
}

// Done returns true if the model finished.
func (m TableSelectModel) Done() bool {
	return m.done
}

// Cancelled returns true if the user cancelled.
func (m TableSelectModel) Cancelled() bool {
	return m.cancelled
}

// --- internal helpers ---

func (m *TableSelectModel) moveCursor(delta int) {
	if len(m.visibleIdxs) == 0 {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.visibleIdxs) {
		m.cursor = len(m.visibleIdxs) - 1
	}
}

func (m *TableSelectModel) toggleCurrent() {
	if m.cursor < 0 || m.cursor >= len(m.visibleIdxs) {
		return
	}
	idx := m.visibleIdxs[m.cursor]
	m.entries[idx].selected = !m.entries[idx].selected
}

func (m *TableSelectModel) selectAll() {
	for _, vi := range m.visibleIdxs {
		m.entries[vi].selected = true
	}
}

func (m *TableSelectModel) deselectAll() {
	for _, vi := range m.visibleIdxs {
		m.entries[vi].selected = false
	}
}

func (m *TableSelectModel) selectDependencies() {
	// Build a set of currently selected table names
	selectedNames := make(map[string]bool)
	for _, e := range m.entries {
		if e.selected {
			selectedNames[e.table.Name] = true
		}
	}

	// Find all tables referenced by selected tables via FKs
	needed := make(map[string]bool)
	for _, e := range m.entries {
		if !e.selected {
			continue
		}
		for _, fk := range e.table.ForeignKeys {
			if !selectedNames[fk.ReferencedTable] {
				needed[fk.ReferencedTable] = true
			}
		}
	}

	// Select the needed tables
	for i := range m.entries {
		if needed[m.entries[i].table.Name] {
			m.entries[i].selected = true
		}
	}
}

func (m *TableSelectModel) applyFilter() {
	lower := strings.ToLower(m.filter)
	for i := range m.entries {
		if m.filter == "" {
			m.entries[i].visible = true
		} else {
			m.entries[i].visible = strings.Contains(
				strings.ToLower(m.entries[i].table.Name), lower)
		}
	}
	m.recomputeVisible()
	if m.cursor >= len(m.visibleIdxs) {
		m.cursor = max(0, len(m.visibleIdxs)-1)
	}
}

func (m *TableSelectModel) recomputeVisible() {
	m.visibleIdxs = m.visibleIdxs[:0]
	for i, e := range m.entries {
		if e.visible {
			m.visibleIdxs = append(m.visibleIdxs, i)
		}
	}
}

func (m *TableSelectModel) cycleSort() {
	if m.sortAsc {
		m.sortAsc = false
	} else {
		m.sortField = (m.sortField + 1) % 4
		m.sortAsc = true
	}
	m.sortEntries()
	m.recomputeVisible()
	m.cursor = 0
}

func (m *TableSelectModel) sortEntries() {
	sort.SliceStable(m.entries, func(i, j int) bool {
		a, b := m.entries[i].table, m.entries[j].table
		var less bool
		switch m.sortField {
		case SortByName:
			less = a.Name < b.Name
		case SortByRows:
			less = a.RowCount < b.RowCount
		case SortBySize:
			less = a.SizeBytes < b.SizeBytes
		case SortByFKs:
			less = len(a.ForeignKeys) < len(b.ForeignKeys)
		}
		if !m.sortAsc {
			return !less
		}
		return less
	})
}

func (m *TableSelectModel) selectedCount() int {
	n := 0
	for _, e := range m.entries {
		if e.selected {
			n++
		}
	}
	return n
}

func (m *TableSelectModel) getSelected() []schema.Table {
	var tables []schema.Table
	for _, e := range m.entries {
		if e.selected {
			tables = append(tables, e.table)
		}
	}
	return tables
}

// SelectedNames returns the names of selected tables.
func (m *TableSelectModel) SelectedNames() []string {
	var names []string
	for _, e := range m.entries {
		if e.selected {
			names = append(names, e.table.Name)
		}
	}
	return names
}

// --- formatting helpers ---

func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)
	switch {
	case b >= tb:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(tb))
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatNumber(n int64) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

// styles for table select
var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	summaryStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)
