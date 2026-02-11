package mapping

import (
	"github.com/reloquent/reloquent/internal/schema"
)

// CollectionSizeEstimate holds per-collection BSON document size estimates.
type CollectionSizeEstimate struct {
	Collection     string `json:"collection"`
	SourceTable    string `json:"source_table"`
	AvgDocSizeBytes int64  `json:"avg_doc_size_bytes"`
	MaxDocSizeBytes int64  `json:"max_doc_size_bytes"`
	AvgRowCount     int64  `json:"avg_row_count"`
	ExceedsLimit    bool   `json:"exceeds_limit"`
	Warning         string `json:"warning,omitempty"`
}

const bsonDocumentLimit = 16 * 1024 * 1024 // 16MB

// EstimateSizes estimates per-collection BSON document sizes from source schema and mapping.
// It flags collections that may exceed the 16MB BSON document limit.
func EstimateSizes(s *schema.Schema, m *Mapping) []CollectionSizeEstimate {
	tableMap := make(map[string]*schema.Table, len(s.Tables))
	for i := range s.Tables {
		tableMap[s.Tables[i].Name] = &s.Tables[i]
	}

	var results []CollectionSizeEstimate
	for _, col := range m.Collections {
		est := estimateCollection(col, tableMap)
		results = append(results, est)
	}
	return results
}

func estimateCollection(col Collection, tableMap map[string]*schema.Table) CollectionSizeEstimate {
	srcTable := tableMap[col.SourceTable]
	if srcTable == nil {
		return CollectionSizeEstimate{
			Collection:  col.Name,
			SourceTable: col.SourceTable,
		}
	}

	// Base row size from source table
	baseRowBytes := estimateRowSize(srcTable)
	parentRowCount := srcTable.RowCount
	if parentRowCount == 0 {
		parentRowCount = 1
	}

	// Add embedded document sizes
	var embeddedBytes int64
	var maxEmbeddedBytes int64
	for _, emb := range col.Embedded {
		avgEmb, maxEmb := estimateEmbeddedSize(emb, tableMap, parentRowCount)
		embeddedBytes += avgEmb
		maxEmbeddedBytes += maxEmb
	}

	avgDocSize := baseRowBytes + embeddedBytes
	maxDocSize := baseRowBytes + maxEmbeddedBytes

	// Apply BSON overhead factor (field names, type markers, length prefixes)
	avgDocSize = avgDocSize * 13 / 10 // ~1.3x overhead
	maxDocSize = maxDocSize * 15 / 10 // ~1.5x overhead for worst case

	est := CollectionSizeEstimate{
		Collection:      col.Name,
		SourceTable:     col.SourceTable,
		AvgDocSizeBytes: avgDocSize,
		MaxDocSizeBytes: maxDocSize,
		AvgRowCount:     parentRowCount,
	}

	if maxDocSize > bsonDocumentLimit {
		est.ExceedsLimit = true
		est.Warning = "Estimated maximum document size exceeds 16MB BSON limit. Consider reducing embedding depth or splitting into references."
	}

	return est
}

func estimateEmbeddedSize(emb Embedded, tableMap map[string]*schema.Table, parentRowCount int64) (avgBytes, maxBytes int64) {
	childTable := tableMap[emb.SourceTable]
	if childTable == nil {
		return 0, 0
	}

	childRowSize := estimateRowSize(childTable)

	if emb.Relationship == "single" {
		// 1:1 — one subdocument per parent
		avgBytes = childRowSize
		maxBytes = childRowSize
	} else {
		// 1:N — array of subdocuments
		avgChildrenPerParent := int64(1)
		if parentRowCount > 0 && childTable.RowCount > 0 {
			avgChildrenPerParent = childTable.RowCount / parentRowCount
			if avgChildrenPerParent < 1 {
				avgChildrenPerParent = 1
			}
		}
		avgBytes = childRowSize * avgChildrenPerParent
		// Worst case: 10x average (skewed distribution)
		maxBytes = childRowSize * avgChildrenPerParent * 10
	}

	// Recursively add nested embeds
	for _, nested := range emb.Embedded {
		nestedAvg, nestedMax := estimateEmbeddedSize(nested, tableMap, childTable.RowCount)
		if emb.Relationship == "single" {
			avgBytes += nestedAvg
			maxBytes += nestedMax
		} else {
			avgChildrenPerParent := int64(1)
			if parentRowCount > 0 && childTable.RowCount > 0 {
				avgChildrenPerParent = childTable.RowCount / parentRowCount
				if avgChildrenPerParent < 1 {
					avgChildrenPerParent = 1
				}
			}
			avgBytes += nestedAvg * avgChildrenPerParent
			maxBytes += nestedMax * avgChildrenPerParent * 10
		}
	}

	return avgBytes, maxBytes
}

func estimateRowSize(t *schema.Table) int64 {
	if t.SizeBytes > 0 && t.RowCount > 0 {
		return t.SizeBytes / t.RowCount
	}
	// Estimate from column types
	var size int64
	for _, col := range t.Columns {
		size += estimateColumnSize(col.DataType)
	}
	if size == 0 {
		size = 100 // fallback
	}
	return size
}

func estimateColumnSize(dataType string) int64 {
	switch dataType {
	case "boolean", "bool":
		return 1
	case "smallint", "int2":
		return 2
	case "integer", "int", "int4", "serial":
		return 4
	case "bigint", "int8", "bigserial":
		return 8
	case "real", "float4":
		return 4
	case "double precision", "float8":
		return 8
	case "numeric", "decimal", "NUMBER":
		return 16
	case "date":
		return 4
	case "timestamp", "timestamp without time zone", "timestamp with time zone", "TIMESTAMP":
		return 8
	case "uuid":
		return 16
	case "text", "varchar", "character varying", "VARCHAR2", "CLOB":
		return 100 // average estimate
	case "bytea", "BLOB", "RAW":
		return 256
	case "json", "jsonb":
		return 200
	default:
		return 32
	}
}
