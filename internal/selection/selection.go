package selection

import (
	"strings"

	"github.com/reloquent/reloquent/internal/schema"
)

// FilterByPattern returns tables matching a glob-like pattern (e.g., "order_*").
func FilterByPattern(tables []schema.Table, pattern string) []schema.Table {
	var matched []schema.Table
	for _, t := range tables {
		if matchGlob(t.Name, pattern) {
			matched = append(matched, t)
		}
	}
	return matched
}

// TotalSize returns the sum of SizeBytes for the given tables.
func TotalSize(tables []schema.Table) int64 {
	var total int64
	for _, t := range tables {
		total += t.SizeBytes
	}
	return total
}

// TotalRows returns the sum of RowCount for the given tables.
func TotalRows(tables []schema.Table) int64 {
	var total int64
	for _, t := range tables {
		total += t.RowCount
	}
	return total
}

// OrphanedRef represents a foreign key pointing to a table not in the selection.
type OrphanedRef struct {
	Table           string
	ForeignKey      string
	ReferencedTable string
}

// FindOrphanedReferences returns foreign keys that reference tables not in the selection.
func FindOrphanedReferences(selected []schema.Table) []OrphanedRef {
	selectedNames := make(map[string]bool)
	for _, t := range selected {
		selectedNames[t.Name] = true
	}

	var orphans []OrphanedRef
	for _, t := range selected {
		for _, fk := range t.ForeignKeys {
			if !selectedNames[fk.ReferencedTable] {
				orphans = append(orphans, OrphanedRef{
					Table:           t.Name,
					ForeignKey:      fk.Name,
					ReferencedTable: fk.ReferencedTable,
				})
			}
		}
	}
	return orphans
}

func matchGlob(name, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(name, pattern[:len(pattern)-1])
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(name, pattern[1:])
	}
	return name == pattern
}
