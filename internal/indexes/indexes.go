package indexes

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/target"
)

// IndexPlan describes the set of indexes to create on the target.
type IndexPlan struct {
	Indexes      []target.CollectionIndex `yaml:"indexes"`
	Explanations []string                 `yaml:"explanations"`
}

// Infer generates an IndexPlan from the source schema and mapping.
func Infer(s *schema.Schema, m *mapping.Mapping) *IndexPlan {
	plan := &IndexPlan{}
	tableMap := buildTableMap(s)

	for _, col := range m.Collections {
		srcTable := tableMap[col.SourceTable]
		if srcTable == nil {
			continue
		}

		// 1. Primary key → unique index (skip if single-column PK that maps to _id)
		if srcTable.PrimaryKey != nil {
			pkCols := srcTable.PrimaryKey.Columns
			if !isSingleID(pkCols) {
				keys := make([]target.IndexKey, len(pkCols))
				for i, c := range pkCols {
					keys[i] = target.IndexKey{Field: c, Order: 1}
				}
				idx := target.IndexDefinition{
					Keys:   keys,
					Name:   fmt.Sprintf("pk_%s", col.Name),
					Unique: true,
				}
				plan.addIfNew(col.Name, idx)
				plan.Explanations = append(plan.Explanations,
					fmt.Sprintf("Unique index on %s(%s) from primary key", col.Name, strings.Join(pkCols, ", ")))
			}
		}

		// 2. References → index on reference field
		for _, ref := range col.References {
			idx := target.IndexDefinition{
				Keys: []target.IndexKey{{Field: ref.FieldName, Order: 1}},
				Name: fmt.Sprintf("ref_%s_%s", col.Name, ref.FieldName),
			}
			plan.addIfNew(col.Name, idx)
			plan.Explanations = append(plan.Explanations,
				fmt.Sprintf("Index on %s.%s from reference to %s", col.Name, ref.FieldName, ref.SourceTable))
		}

		// 3. Source indexes → equivalent MongoDB index
		for _, srcIdx := range srcTable.Indexes {
			// Skip if this is the PK index (already handled above)
			if srcTable.PrimaryKey != nil && sameColumns(srcIdx.Columns, srcTable.PrimaryKey.Columns) {
				continue
			}
			keys := make([]target.IndexKey, 0, len(srcIdx.Columns))
			for _, c := range srcIdx.Columns {
				keys = append(keys, target.IndexKey{Field: c, Order: 1})
			}
			if len(keys) == 0 {
				continue
			}
			idx := target.IndexDefinition{
				Keys:   keys,
				Name:   fmt.Sprintf("idx_%s_%s", col.Name, strings.Join(srcIdx.Columns, "_")),
				Unique: srcIdx.Unique,
			}
			plan.addIfNew(col.Name, idx)
			plan.Explanations = append(plan.Explanations,
				fmt.Sprintf("Index on %s(%s) from source index %s", col.Name, strings.Join(srcIdx.Columns, ", "), srcIdx.Name))
		}

		// 4. Embedded fields → dot notation indexes for their source indexes
		inferEmbeddedIndexes(plan, col.Name, col.Embedded, tableMap, "")
	}

	return plan
}

func inferEmbeddedIndexes(plan *IndexPlan, collName string, embedded []mapping.Embedded, tableMap map[string]*schema.Table, prefix string) {
	for _, emb := range embedded {
		fieldPrefix := emb.FieldName
		if prefix != "" {
			fieldPrefix = prefix + "." + emb.FieldName
		}

		srcTable := tableMap[emb.SourceTable]
		if srcTable == nil {
			continue
		}

		// FK that became an embedded join → index on the join field using dot notation
		idx := target.IndexDefinition{
			Keys: []target.IndexKey{{Field: fieldPrefix + "." + emb.JoinColumn, Order: 1}},
			Name: fmt.Sprintf("idx_%s_%s", collName, strings.ReplaceAll(fieldPrefix+"_"+emb.JoinColumn, ".", "_")),
		}
		plan.addIfNew(collName, idx)
		plan.Explanations = append(plan.Explanations,
			fmt.Sprintf("Index on %s.%s.%s from embedded join", collName, fieldPrefix, emb.JoinColumn))

		// Source indexes on embedded table → dot notation
		for _, srcIdx := range srcTable.Indexes {
			keys := make([]target.IndexKey, 0, len(srcIdx.Columns))
			for _, c := range srcIdx.Columns {
				keys = append(keys, target.IndexKey{Field: fieldPrefix + "." + c, Order: 1})
			}
			if len(keys) == 0 {
				continue
			}
			colNames := make([]string, len(srcIdx.Columns))
			for i, c := range srcIdx.Columns {
				colNames[i] = fieldPrefix + "." + c
			}
			idx := target.IndexDefinition{
				Keys:   keys,
				Name:   fmt.Sprintf("idx_%s_%s", collName, strings.ReplaceAll(strings.Join(colNames, "_"), ".", "_")),
				Unique: srcIdx.Unique,
			}
			plan.addIfNew(collName, idx)
			plan.Explanations = append(plan.Explanations,
				fmt.Sprintf("Index on %s(%s) from embedded table %s index %s",
					collName, strings.Join(colNames, ", "), emb.SourceTable, srcIdx.Name))
		}

		// Recurse into nested embeds
		inferEmbeddedIndexes(plan, collName, emb.Embedded, tableMap, fieldPrefix)
	}
}

func (p *IndexPlan) addIfNew(collection string, idx target.IndexDefinition) {
	// Never generate _id index
	if len(idx.Keys) == 1 && idx.Keys[0].Field == "_id" {
		return
	}

	// Deduplicate by collection + key fields
	keyStr := indexKeyString(idx.Keys)
	for _, existing := range p.Indexes {
		if existing.Collection == collection && indexKeyString(existing.Index.Keys) == keyStr {
			return
		}
	}
	p.Indexes = append(p.Indexes, target.CollectionIndex{Collection: collection, Index: idx})
}

func indexKeyString(keys []target.IndexKey) string {
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s:%d", k.Field, k.Order)
	}
	return strings.Join(parts, ",")
}

func isSingleID(cols []string) bool {
	return len(cols) == 1 && (cols[0] == "_id" || cols[0] == "id")
}

func sameColumns(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func buildTableMap(s *schema.Schema) map[string]*schema.Table {
	m := make(map[string]*schema.Table, len(s.Tables))
	for i := range s.Tables {
		m[s.Tables[i].Name] = &s.Tables[i]
	}
	return m
}

// WriteYAML writes the index plan to a YAML file.
func (p *IndexPlan) WriteYAML(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	data, err := yaml.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshaling index plan: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadYAML reads an index plan from a YAML file.
func LoadYAML(path string) (*IndexPlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading index plan: %w", err)
	}
	p := &IndexPlan{}
	if err := yaml.Unmarshal(data, p); err != nil {
		return nil, fmt.Errorf("parsing index plan: %w", err)
	}
	return p, nil
}
