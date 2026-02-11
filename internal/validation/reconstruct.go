package validation

import (
	"fmt"
	"strings"

	"github.com/reloquent/reloquent/internal/mapping"
)

// ReconstructSQL builds a SQL SELECT that reconstructs the data for a collection
// by joining the root table with embedded tables according to the mapping.
// This is primarily used for documentation/debugging purposes.
func ReconstructSQL(col mapping.Collection, schemaName string) string {
	rootAlias := "t0"
	var joins []string
	var aliasIdx int

	rootTable := qualifiedTable(schemaName, col.SourceTable)
	selectCols := []string{rootAlias + ".*"}

	for _, emb := range col.Embedded {
		aliasIdx++
		alias := fmt.Sprintf("t%d", aliasIdx)
		joinTable := qualifiedTable(schemaName, emb.SourceTable)
		join := fmt.Sprintf("LEFT JOIN %s %s ON %s.%s = %s.%s",
			joinTable, alias, alias, emb.JoinColumn, rootAlias, emb.ParentColumn)
		joins = append(joins, join)
		selectCols = append(selectCols, alias+".*")

		// Recurse into nested embeds
		aliasIdx = buildNestedJoins(&joins, &selectCols, emb.Embedded, alias, schemaName, aliasIdx)
	}

	sql := fmt.Sprintf("SELECT %s\nFROM %s %s",
		strings.Join(selectCols, ", "),
		rootTable, rootAlias)

	if len(joins) > 0 {
		sql += "\n" + strings.Join(joins, "\n")
	}

	return sql
}

func buildNestedJoins(joins *[]string, selectCols *[]string, embedded []mapping.Embedded, parentAlias, schemaName string, aliasIdx int) int {
	for _, emb := range embedded {
		aliasIdx++
		alias := fmt.Sprintf("t%d", aliasIdx)
		joinTable := qualifiedTable(schemaName, emb.SourceTable)
		join := fmt.Sprintf("LEFT JOIN %s %s ON %s.%s = %s.%s",
			joinTable, alias, alias, emb.JoinColumn, parentAlias, emb.ParentColumn)
		*joins = append(*joins, join)
		*selectCols = append(*selectCols, alias+".*")

		aliasIdx = buildNestedJoins(joins, selectCols, emb.Embedded, alias, schemaName, aliasIdx)
	}
	return aliasIdx
}

func qualifiedTable(schemaName, table string) string {
	if schemaName == "" {
		return table
	}
	return schemaName + "." + table
}
