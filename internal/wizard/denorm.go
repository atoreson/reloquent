package wizard

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/schema"
)

// RelChoice represents the user's choice for a FK relationship.
type RelChoice int

const (
	ChoiceReference  RelChoice = iota // keep as separate collection
	ChoiceEmbedArray                  // embed child rows as array in parent
	ChoiceEmbedSingle                 // embed single child doc in parent
)

func (c RelChoice) String() string {
	switch c {
	case ChoiceReference:
		return "reference"
	case ChoiceEmbedArray:
		return "embed array"
	case ChoiceEmbedSingle:
		return "embed single"
	default:
		return "unknown"
	}
}

// fkRelationship is a foreign key with the user's embedding choice.
type fkRelationship struct {
	// The child table that has the FK column
	ChildTable string
	// The FK column(s) on the child table
	ChildColumns []string
	// The parent table being referenced
	ParentTable string
	// The referenced column(s) on the parent table
	ParentColumns []string
	// User's choice
	Choice RelChoice
	// Metadata for display
	IsSelfRef   bool
	IsJoinTable bool
}

// DenormModel is the bubbletea model for the denormalization designer.
type DenormModel struct {
	tables    []schema.Table
	rels      []fkRelationship
	cursor    int
	done      bool
	cancelled bool
	width     int
	height    int
	warnings  []string
	graph     *mapping.FKGraph
}

// NewDenormModel creates a denormalization designer from the selected tables.
func NewDenormModel(tables []schema.Table) DenormModel {
	graph := mapping.NewFKGraph(tables)
	rels := extractRelationships(tables)

	// Mark self-references
	selfRefs := make(map[string]bool)
	for _, sr := range graph.SelfReferences() {
		selfRefs[sr.FKName] = true
	}

	// Mark join tables
	joinTables := make(map[string]bool)
	for _, jt := range graph.JoinTables() {
		joinTables[jt.JoinTable] = true
	}

	for i := range rels {
		if rels[i].ChildTable == rels[i].ParentTable {
			rels[i].IsSelfRef = true
		}
		if joinTables[rels[i].ChildTable] {
			rels[i].IsJoinTable = true
		}
	}

	return DenormModel{
		tables: tables,
		rels:   rels,
		width:  100,
		height: 24,
		graph:  graph,
	}
}

// extractRelationships finds FK relationships between the given tables.
func extractRelationships(tables []schema.Table) []fkRelationship {
	tableSet := make(map[string]bool, len(tables))
	for _, t := range tables {
		tableSet[t.Name] = true
	}

	var rels []fkRelationship
	for _, t := range tables {
		for _, fk := range t.ForeignKeys {
			// Only include FKs where both sides are in the selected set
			if !tableSet[fk.ReferencedTable] {
				continue
			}
			rels = append(rels, fkRelationship{
				ChildTable:    t.Name,
				ChildColumns:  fk.Columns,
				ParentTable:   fk.ReferencedTable,
				ParentColumns: fk.ReferencedColumns,
				Choice:        ChoiceReference,
			})
		}
	}

	// Sort for stable ordering: by parent, then child
	sort.Slice(rels, func(i, j int) bool {
		if rels[i].ParentTable != rels[j].ParentTable {
			return rels[i].ParentTable < rels[j].ParentTable
		}
		return rels[i].ChildTable < rels[j].ChildTable
	})

	return rels
}

func (m DenormModel) Init() tea.Cmd {
	return nil
}

func (m DenormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// If no relationships, only f/q/esc are valid
		if len(m.rels) == 0 {
			switch msg.String() {
			case "f", "enter":
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
			if m.cursor < len(m.rels)-1 {
				m.cursor++
			}

		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}

		case " ": // cycle: reference → embed array → embed single → reference
			m.rels[m.cursor].Choice = (m.rels[m.cursor].Choice + 1) % 3

		case "a": // direct set: embed array
			m.rels[m.cursor].Choice = ChoiceEmbedArray

		case "s": // direct set: embed single
			m.rels[m.cursor].Choice = ChoiceEmbedSingle

		case "r": // direct set: reference
			m.rels[m.cursor].Choice = ChoiceReference

		case "f", "enter":
			m.enforceCycleConstraints()
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// enforceCycleConstraints detects cycles where all edges are "embed" and forces one to "reference".
func (m *DenormModel) enforceCycleConstraints() {
	m.warnings = nil

	// Build embed adjacency: child→parent for embed choices only
	embedEdges := make(map[string]string) // child→parent
	for _, rel := range m.rels {
		if rel.Choice == ChoiceEmbedArray || rel.Choice == ChoiceEmbedSingle {
			if rel.ChildTable != rel.ParentTable { // skip self-refs
				embedEdges[rel.ChildTable] = rel.ParentTable
			}
		}
	}

	// Check for cycles in the embed graph
	for child := range embedEdges {
		visited := map[string]bool{child: true}
		current := child
		for {
			parent, ok := embedEdges[current]
			if !ok {
				break
			}
			if visited[parent] {
				// Cycle detected — force this edge to reference
				for i := range m.rels {
					if m.rels[i].ChildTable == current &&
						m.rels[i].ParentTable == parent &&
						(m.rels[i].Choice == ChoiceEmbedArray || m.rels[i].Choice == ChoiceEmbedSingle) {
						m.rels[i].Choice = ChoiceReference
						m.warnings = append(m.warnings,
							fmt.Sprintf("Cycle detected: %s→%s forced to reference", current, parent))
						break
					}
				}
				break
			}
			visited[parent] = true
			current = parent
		}
	}
}

func (m DenormModel) View() string {
	var b strings.Builder

	title := titleStyle.Render("Step 4: Denormalization Design")
	b.WriteString(title + "\n\n")

	if len(m.rels) == 0 {
		b.WriteString("  No foreign key relationships between selected tables.\n")
		b.WriteString("  All tables will become standalone collections.\n\n")
		b.WriteString(dimStyle.Render("  Press f to confirm • q to cancel\n"))
		return b.String()
	}

	// Relationship list
	b.WriteString(dimStyle.Render("  Relationships:") + "\n\n")

	for i, rel := range m.rels {
		cursor := "  "
		if i == m.cursor {
			cursor = highlightStyle.Render("> ")
		}

		arrow := fmt.Sprintf("%s.%s → %s.%s",
			rel.ChildTable, strings.Join(rel.ChildColumns, ","),
			rel.ParentTable, strings.Join(rel.ParentColumns, ","))

		// Add metadata labels
		labels := ""
		if rel.IsSelfRef {
			labels = " (self-ref)"
		}
		if rel.IsJoinTable {
			labels = " (M2M join)"
		}

		choiceStr := m.choiceLabel(rel.Choice)

		b.WriteString(fmt.Sprintf("%s%-50s  [%s]%s\n", cursor, arrow, choiceStr, labels))
	}

	// Warnings
	if len(m.warnings) > 0 {
		b.WriteString("\n")
		for _, w := range m.warnings {
			b.WriteString(errStyle.Render("  ⚠ "+w) + "\n")
		}
	}

	// Preview panel
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Preview:") + "\n\n")
	preview := m.buildPreview()
	for _, line := range preview {
		b.WriteString("  " + line + "\n")
	}

	// Help
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  j/k navigate • space cycle • a embed array • s embed single • r reference • f confirm • q cancel\n"))

	return b.String()
}

func (m DenormModel) choiceLabel(c RelChoice) string {
	switch c {
	case ChoiceReference:
		return dimStyle.Render("reference")
	case ChoiceEmbedArray:
		return successStyle.Render("embed array")
	case ChoiceEmbedSingle:
		return successStyle.Render("embed single")
	default:
		return "unknown"
	}
}

// buildPreview generates a recursive tree-like preview of the resulting collection structure.
func (m DenormModel) buildPreview() []string {
	// Build parent→children map for embed relationships
	type embedInfo struct {
		childTable string
		relType    string // "array" or "single"
	}

	// parentTable → list of children embedded into it
	childrenOf := make(map[string][]embedInfo)
	embeddedSet := make(map[string]bool)

	for _, rel := range m.rels {
		if rel.Choice == ChoiceEmbedArray || rel.Choice == ChoiceEmbedSingle {
			if rel.ChildTable == rel.ParentTable {
				continue // skip self-refs for preview tree
			}
			relType := "array"
			if rel.Choice == ChoiceEmbedSingle {
				relType = "single"
			}
			childrenOf[rel.ParentTable] = append(childrenOf[rel.ParentTable],
				embedInfo{childTable: rel.ChildTable, relType: relType})
			embeddedSet[rel.ChildTable] = true
		}
	}

	// Sort children for stable output
	for k := range childrenOf {
		sort.Slice(childrenOf[k], func(i, j int) bool {
			return childrenOf[k][i].childTable < childrenOf[k][j].childTable
		})
	}

	// Root collections: tables not embedded
	var rootNames []string
	for _, t := range m.tables {
		if !embeddedSet[t.Name] {
			rootNames = append(rootNames, t.Name)
		}
	}
	sort.Strings(rootNames)

	// Recursive tree builder
	var lines []string
	var buildTree func(name string, indent string)
	buildTree = func(name string, indent string) {
		children := childrenOf[name]
		for _, child := range children {
			suffix := "[]"
			label := "embedded array"
			if child.relType == "single" {
				suffix = ""
				label = "embedded single"
			}
			lines = append(lines, fmt.Sprintf("%s└─ %s%s (%s)", indent, child.childTable, suffix, label))
			buildTree(child.childTable, indent+"   ")
		}
	}

	for _, name := range rootNames {
		lines = append(lines, fmt.Sprintf("%s (collection)", name))
		buildTree(name, "")
	}

	return lines
}

// BuildMapping converts the current choices into a mapping.Mapping.
// Supports deep nesting: if a parent is also embedded, the child becomes nested inside it.
func (m DenormModel) BuildMapping() *mapping.Mapping {
	// Track which tables are embedded (child→parent)
	type embedEntry struct {
		parentTable  string
		childTable   string
		joinColumn   string
		parentColumn string
		relationship string
	}

	var embeds []embedEntry
	embeddedSet := make(map[string]bool) // tables that are embedded into another

	for _, rel := range m.rels {
		if rel.Choice == ChoiceReference {
			continue
		}
		if rel.ChildTable == rel.ParentTable {
			continue // self-refs default to reference
		}
		relType := "array"
		if rel.Choice == ChoiceEmbedSingle {
			relType = "single"
		}
		embeds = append(embeds, embedEntry{
			parentTable:  rel.ParentTable,
			childTable:   rel.ChildTable,
			joinColumn:   strings.Join(rel.ChildColumns, ","),
			parentColumn: strings.Join(rel.ParentColumns, ","),
			relationship: relType,
		})
		embeddedSet[rel.ChildTable] = true
	}

	// Build a map of parentTable → embedded entries
	parentToEmbeds := make(map[string][]embedEntry)
	for _, e := range embeds {
		parentToEmbeds[e.parentTable] = append(parentToEmbeds[e.parentTable], e)
	}

	// Recursive function to build nested Embedded structs
	var buildEmbedded func(tableName string) []mapping.Embedded
	buildEmbedded = func(tableName string) []mapping.Embedded {
		entries := parentToEmbeds[tableName]
		if len(entries) == 0 {
			return nil
		}
		result := make([]mapping.Embedded, 0, len(entries))
		for _, e := range entries {
			emb := mapping.Embedded{
				SourceTable:  e.childTable,
				FieldName:    e.childTable,
				Relationship: e.relationship,
				JoinColumn:   e.joinColumn,
				ParentColumn: e.parentColumn,
				Embedded:     buildEmbedded(e.childTable), // recurse
			}
			result = append(result, emb)
		}
		return result
	}

	// Build reference list
	type refInfo struct {
		parentTable  string
		childTable   string
		joinColumn   string
		parentColumn string
	}
	var refs []refInfo
	for _, rel := range m.rels {
		if rel.Choice != ChoiceReference {
			// Self-refs also become references
			if rel.ChildTable != rel.ParentTable {
				continue
			}
		}
		refs = append(refs, refInfo{
			parentTable:  rel.ParentTable,
			childTable:   rel.ChildTable,
			joinColumn:   strings.Join(rel.ChildColumns, ","),
			parentColumn: strings.Join(rel.ParentColumns, ","),
		})
	}

	// Create collections: one per non-embedded table
	collMap := make(map[string]*mapping.Collection)
	var collOrder []string
	for _, t := range m.tables {
		if embeddedSet[t.Name] {
			continue
		}
		c := &mapping.Collection{
			Name:        t.Name,
			SourceTable: t.Name,
			Embedded:    buildEmbedded(t.Name),
		}
		collMap[t.Name] = c
		collOrder = append(collOrder, t.Name)
	}
	sort.Strings(collOrder)

	// Attach references to parent collections
	for _, r := range refs {
		parent, ok := collMap[r.parentTable]
		if !ok {
			continue
		}
		parent.References = append(parent.References, mapping.Reference{
			SourceTable:  r.childTable,
			FieldName:    r.childTable,
			JoinColumn:   r.joinColumn,
			ParentColumn: r.parentColumn,
		})
	}

	// Deduplicate collection order
	seen := make(map[string]bool)
	var collections []mapping.Collection
	for _, name := range collOrder {
		if seen[name] {
			continue
		}
		seen[name] = true
		collections = append(collections, *collMap[name])
	}

	return &mapping.Mapping{Collections: collections}
}

// Done returns true if the model has finished.
func (m DenormModel) Done() bool {
	return m.done
}

// Cancelled returns true if the user cancelled.
func (m DenormModel) Cancelled() bool {
	return m.done && m.cancelled
}
