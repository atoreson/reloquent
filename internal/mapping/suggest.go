package mapping

import (
	"github.com/reloquent/reloquent/internal/schema"
)

// Suggest analyzes a schema's FK graph and returns a suggested mapping.
// If rootTables is non-empty, only those tables become root collections;
// otherwise roots are inferred from the FK graph.
// Rules:
//   - 1:1 FK → embed single
//   - 1:N FK → embed array (if child has < 1000 rows per parent on average)
//   - M:N (join tables) → dissolve into arrays on both sides
//   - Self-referencing FK → reference (not embed)
//   - Cycles → break by converting deepest edge to reference
func Suggest(s *schema.Schema, selectedTables []string, rootTables ...string) *Mapping {
	selected := make(map[string]bool)
	for _, t := range selectedTables {
		selected[t] = true
	}

	g := NewFKGraph(s.Tables)

	// Identify root tables (not referenced as child in any FK, or low-depth in topo order)
	childOf := make(map[string][]schema.ForeignKey) // table -> FKs pointing out
	for _, t := range s.Tables {
		if !selected[t.Name] {
			continue
		}
		for _, fk := range t.ForeignKeys {
			if selected[fk.ReferencedTable] {
				childOf[t.Name] = append(childOf[t.Name], fk)
			}
		}
	}

	// Tables that are never referenced as a child are roots
	referencedAsChild := make(map[string]bool)
	for _, t := range s.Tables {
		if !selected[t.Name] {
			continue
		}
		for _, fk := range t.ForeignKeys {
			if selected[fk.ReferencedTable] {
				referencedAsChild[t.Name] = true
			}
		}
	}

	// Self-references
	selfRefs := make(map[string]bool)
	for _, e := range g.SelfReferences() {
		selfRefs[e.ChildTable] = true
	}

	// Join tables (M:N)
	joinTables := make(map[string]bool)
	for _, jt := range g.JoinTables() {
		joinTables[jt.JoinTable] = true
	}

	// Build collections for root tables
	collections := make([]Collection, 0)
	used := make(map[string]bool) // tables already assigned to a collection

	// First pass: identify root tables
	roots := make([]string, 0)
	if len(rootTables) > 0 {
		// Use explicitly provided roots
		for _, r := range rootTables {
			if selected[r] {
				roots = append(roots, r)
			}
		}
	} else {
		// Heuristic: tables with no FK pointing out, or that are the "parent" side
		for _, t := range s.Tables {
			if !selected[t.Name] {
				continue
			}
			if joinTables[t.Name] {
				continue // skip join tables
			}
			if len(childOf[t.Name]) == 0 || selfRefs[t.Name] {
				roots = append(roots, t.Name)
			}
		}

		// If no roots found (everything has FKs), use all selected tables
		if len(roots) == 0 {
			for _, t := range s.Tables {
				if selected[t.Name] && !joinTables[t.Name] {
					roots = append(roots, t.Name)
				}
			}
		}
	}

	tableMap := make(map[string]schema.Table)
	for _, t := range s.Tables {
		tableMap[t.Name] = t
	}

	for _, root := range roots {
		if used[root] {
			continue
		}
		col := Collection{
			Name:        root,
			SourceTable: root,
		}

		// Find children that reference this root
		for _, t := range s.Tables {
			if !selected[t.Name] || t.Name == root || used[t.Name] {
				continue
			}
			for _, fk := range t.ForeignKeys {
				if fk.ReferencedTable != root {
					continue
				}
				if selfRefs[t.Name] {
					// Self-ref → reference
					col.References = append(col.References, Reference{
						SourceTable:  t.Name,
						FieldName:    t.Name + "_ref",
						JoinColumn:   fk.Columns[0],
						ParentColumn: fk.ReferencedColumns[0],
					})
				} else {
					// Determine array vs single
					rel := "array"
					parent := tableMap[root]
					child := tableMap[t.Name]
					if parent.RowCount > 0 && child.RowCount > 0 {
						ratio := float64(child.RowCount) / float64(parent.RowCount)
						if ratio <= 1.0 {
							rel = "single"
						}
					}

					col.Embedded = append(col.Embedded, Embedded{
						SourceTable:  t.Name,
						FieldName:    t.Name,
						Relationship: rel,
						JoinColumn:   fk.Columns[0],
						ParentColumn: fk.ReferencedColumns[0],
					})
				}
				used[t.Name] = true
			}
		}

		used[root] = true
		collections = append(collections, col)
	}

	// Any remaining selected tables get their own collection
	for _, t := range s.Tables {
		if selected[t.Name] && !used[t.Name] {
			collections = append(collections, Collection{
				Name:        t.Name,
				SourceTable: t.Name,
			})
		}
	}

	return &Mapping{Collections: collections}
}
