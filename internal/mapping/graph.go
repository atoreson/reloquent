package mapping

import (
	"github.com/reloquent/reloquent/internal/schema"
)

// FKEdge represents a foreign key relationship in the graph.
type FKEdge struct {
	ChildTable    string
	ChildColumns  []string
	ParentTable   string
	ParentColumns []string
	FKName        string
}

// JoinTableInfo describes a many-to-many join table.
type JoinTableInfo struct {
	JoinTable  string
	LeftTable  string
	LeftCols   []string
	RightTable string
	RightCols  []string
}

// FKGraph represents the foreign key relationships between tables.
type FKGraph struct {
	tables map[string]*schema.Table
	edges  []FKEdge
	// adjacency: parent -> children
	children map[string][]FKEdge
	// adjacency: child -> parents
	parents map[string][]FKEdge
}

// NewFKGraph builds a FK relationship graph from a set of tables.
func NewFKGraph(tables []schema.Table) *FKGraph {
	g := &FKGraph{
		tables:   make(map[string]*schema.Table, len(tables)),
		children: make(map[string][]FKEdge),
		parents:  make(map[string][]FKEdge),
	}

	tableSet := make(map[string]bool, len(tables))
	for i := range tables {
		t := &tables[i]
		g.tables[t.Name] = t
		tableSet[t.Name] = true
	}

	for i := range tables {
		t := &tables[i]
		for _, fk := range t.ForeignKeys {
			if !tableSet[fk.ReferencedTable] {
				continue
			}
			edge := FKEdge{
				ChildTable:    t.Name,
				ChildColumns:  fk.Columns,
				ParentTable:   fk.ReferencedTable,
				ParentColumns: fk.ReferencedColumns,
				FKName:        fk.Name,
			}
			g.edges = append(g.edges, edge)
			g.children[fk.ReferencedTable] = append(g.children[fk.ReferencedTable], edge)
			g.parents[t.Name] = append(g.parents[t.Name], edge)
		}
	}

	return g
}

// Edges returns all FK edges in the graph.
func (g *FKGraph) Edges() []FKEdge {
	return g.edges
}

// SelfReferences returns all FK edges where a table references itself.
func (g *FKGraph) SelfReferences() []FKEdge {
	var result []FKEdge
	for _, e := range g.edges {
		if e.ChildTable == e.ParentTable {
			result = append(result, e)
		}
	}
	return result
}

// DetectCycles finds all cycles in the FK graph using DFS.
// Returns each cycle as a list of table names forming the cycle.
func (g *FKGraph) DetectCycles() [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	// Build adjacency for cycle detection: child -> parent (FK direction)
	adj := make(map[string][]string)
	for _, e := range g.edges {
		if e.ChildTable == e.ParentTable {
			continue // skip self-references
		}
		adj[e.ChildTable] = append(adj[e.ChildTable], e.ParentTable)
	}

	var path []string
	var dfs func(node string)
	dfs = func(node string) {
		visited[node] = true
		inStack[node] = true
		path = append(path, node)

		for _, neighbor := range adj[node] {
			if !visited[neighbor] {
				dfs(neighbor)
			} else if inStack[neighbor] {
				// Found a cycle — extract it
				start := -1
				for i, n := range path {
					if n == neighbor {
						start = i
						break
					}
				}
				if start >= 0 {
					cycle := make([]string, len(path)-start)
					copy(cycle, path[start:])
					cycles = append(cycles, cycle)
				}
			}
		}

		path = path[:len(path)-1]
		inStack[node] = false
	}

	for name := range g.tables {
		if !visited[name] {
			dfs(name)
		}
	}

	return cycles
}

// JoinTables detects many-to-many join tables using heuristics:
// - Table has exactly 2 FKs
// - No other tables reference it
// - Has at most 2 non-FK columns (e.g., id, created_at)
func (g *FKGraph) JoinTables() []JoinTableInfo {
	var result []JoinTableInfo

	// Build set of tables that are referenced by other tables
	referenced := make(map[string]bool)
	for _, e := range g.edges {
		referenced[e.ParentTable] = true
	}

	for name, t := range g.tables {
		// Must have exactly 2 FKs
		fks := g.parents[name]
		if len(fks) != 2 {
			continue
		}

		// No other tables should reference this table
		if referenced[name] {
			continue
		}

		// Count non-FK columns
		fkCols := make(map[string]bool)
		for _, fk := range fks {
			for _, c := range fk.ChildColumns {
				fkCols[c] = true
			}
		}
		nonFKCount := 0
		for _, col := range t.Columns {
			if !fkCols[col.Name] {
				nonFKCount++
			}
		}
		if nonFKCount > 2 {
			continue
		}

		result = append(result, JoinTableInfo{
			JoinTable:  name,
			LeftTable:  fks[0].ParentTable,
			LeftCols:   fks[0].ChildColumns,
			RightTable: fks[1].ParentTable,
			RightCols:  fks[1].ChildColumns,
		})
	}

	return result
}

// NestingDepth computes the maximum nesting depth of the embedding tree.
// A depth of 1 means a single-level embed (child→parent), 2 means two levels, etc.
// The embeds map is childTable -> parentTable for tables being embedded.
func (g *FKGraph) NestingDepth(embeds map[string]string) int {
	if len(embeds) == 0 {
		return 0
	}

	// Build reverse map: parent -> children
	children := make(map[string][]string)
	allChildren := make(map[string]bool)
	for child, parent := range embeds {
		children[parent] = append(children[parent], child)
		allChildren[child] = true
	}

	// Find root tables (parents that aren't themselves embedded)
	var roots []string
	for _, parent := range embeds {
		if !allChildren[parent] {
			roots = append(roots, parent)
		}
	}

	// DFS to find max depth from roots
	var maxDepth func(node string) int
	maxDepth = func(node string) int {
		kids := children[node]
		if len(kids) == 0 {
			return 0
		}
		best := 0
		for _, kid := range kids {
			d := 1 + maxDepth(kid)
			if d > best {
				best = d
			}
		}
		return best
	}

	result := 0
	for _, root := range roots {
		d := maxDepth(root)
		if d > result {
			result = d
		}
	}
	return result
}

// TopologicalSort returns tables in bottom-up order for processing embedded tables.
// Leaf tables (those embedded but not containing embeds) come first.
// The embeds map is childTable -> parentTable.
func (g *FKGraph) TopologicalSort(embeds map[string]string) ([]string, error) {
	// Build dependency graph: parent depends on children being processed first
	// in-degree: how many children a table has that are embedded in it
	inDegree := make(map[string]int)
	dependents := make(map[string][]string) // child -> parent(s) that depend on it

	// Collect all tables involved
	allTables := make(map[string]bool)
	for child, parent := range embeds {
		allTables[child] = true
		allTables[parent] = true
		inDegree[parent]++
		dependents[child] = append(dependents[child], parent)
	}

	// Initialize in-degree for leaves
	for t := range allTables {
		if _, ok := inDegree[t]; !ok {
			inDegree[t] = 0
		}
	}

	// Kahn's algorithm
	var queue []string
	for t, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, t)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		sorted = append(sorted, node)

		for _, parent := range dependents[node] {
			inDegree[parent]--
			if inDegree[parent] == 0 {
				queue = append(queue, parent)
			}
		}
	}

	if len(sorted) != len(allTables) {
		return sorted, &CycleError{Tables: sorted}
	}

	return sorted, nil
}

// CycleError indicates a cycle was detected during topological sort.
type CycleError struct {
	Tables []string
}

func (e *CycleError) Error() string {
	return "cycle detected in embedding graph"
}
