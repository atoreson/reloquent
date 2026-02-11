package codegen

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/drivers"
	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/transform"
	"github.com/reloquent/reloquent/internal/typemap"
)

// Generator produces PySpark migration scripts.
type Generator struct {
	Config  *config.Config
	Schema  *schema.Schema
	Mapping *mapping.Mapping
	TypeMap *typemap.TypeMap
}

// GenerateResult contains the generated PySpark code.
type GenerateResult struct {
	MigrationScript string
	OracleGuidance  string // non-empty if Oracle JDBC is missing
}

// Generate produces the PySpark migration script.
func (g *Generator) Generate() (*GenerateResult, error) {
	var buf bytes.Buffer

	tmpl, err := template.New("migration").Parse(migrationTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	data := g.buildTemplateData()
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	result := &GenerateResult{
		MigrationScript: buf.String(),
	}

	// Check Oracle JDBC
	if g.Config.Source.Type == "oracle" {
		if _, err := drivers.FindOracleJDBC(); err != nil {
			result.OracleGuidance = drivers.OracleJDBCGuidance()
		}
	}

	return result, nil
}

type templateData struct {
	SourceType     string
	JDBCUrl        string
	MongoURI       string
	MongoDatabase  string
	Collections    []collectionData
	MaxConnections int
	HasTransforms  bool
	OracleGuidance string
}

type collectionData struct {
	Name          string
	SourceTable   string
	PartitionCol  string
	NumPartitions int
	Operations    []string // ordered PySpark operation lines
}

func (g *Generator) buildTemplateData() templateData {
	jdbcURL := buildJDBCURL(g.Config.Source)

	var hasTransforms bool
	var collections []collectionData
	for _, c := range g.Mapping.Collections {
		partCol := findPartitionColumn(g.Schema, c.SourceTable)
		ops := g.buildPySparkOperations(c.Name, &c, g.Config.Source.MaxConnections, jdbcURL)

		// Check if any transforms are present
		if len(c.Transformations) > 0 {
			hasTransforms = true
		}
		for _, e := range c.Embedded {
			if hasTransformsInEmbedded(e) {
				hasTransforms = true
			}
		}

		collections = append(collections, collectionData{
			Name:          c.Name,
			SourceTable:   c.SourceTable,
			PartitionCol:  partCol,
			NumPartitions: g.Config.Source.MaxConnections,
			Operations:    ops,
		})
	}

	var guidance string
	if g.Config.Source.Type == "oracle" {
		if _, err := drivers.FindOracleJDBC(); err != nil {
			guidance = drivers.OracleJDBCGuidance()
		}
	}

	return templateData{
		SourceType:     g.Config.Source.Type,
		JDBCUrl:        jdbcURL,
		MongoURI:       g.Config.Target.ConnectionString,
		MongoDatabase:  g.Config.Target.Database,
		Collections:    collections,
		MaxConnections: g.Config.Source.MaxConnections,
		HasTransforms:  hasTransforms,
		OracleGuidance: guidance,
	}
}

func hasTransformsInEmbedded(e mapping.Embedded) bool {
	if len(e.Transformations) > 0 {
		return true
	}
	for _, child := range e.Embedded {
		if hasTransformsInEmbedded(child) {
			return true
		}
	}
	return false
}

// buildPySparkOperations generates the ordered code blocks for a collection.
// Bottom-up: read leaves first, groupBy+collect_list, join into parent, repeat upward.
func (g *Generator) buildPySparkOperations(rootDF string, c *mapping.Collection, numPartitions int, jdbcURL string) []string {
	var ops []string

	// Read root table
	partCol := findPartitionColumn(g.Schema, c.SourceTable)
	ops = append(ops, fmt.Sprintf(`%s_df = spark.read.jdbc(
    url=jdbc_url,
    table="%s",
    column="%s",
    lowerBound=0,
    upperBound=1000000,
    numPartitions=%d,
    properties=jdbc_properties,
)`, rootDF, c.SourceTable, partCol, numPartitions))

	// Apply collection-level transforms
	if len(c.Transformations) > 0 {
		transformLines := transform.ToPySparkAll(c.Transformations, rootDF+"_df")
		ops = append(ops, transformLines...)
	}

	// Process embedded tables bottom-up recursively
	for _, emb := range c.Embedded {
		embOps := g.buildEmbeddedOperations(rootDF+"_df", &emb, numPartitions)
		ops = append(ops, embOps...)
	}

	return ops
}

// buildEmbeddedOperations generates PySpark code for an embedded table and its children.
// Processes bottom-up: children first, then this level.
func (g *Generator) buildEmbeddedOperations(parentDFName string, emb *mapping.Embedded, numPartitions int) []string {
	var ops []string
	childDF := emb.SourceTable + "_df"

	// Read child table
	partCol := findPartitionColumn(g.Schema, emb.SourceTable)
	ops = append(ops, fmt.Sprintf(`%s = spark.read.jdbc(
    url=jdbc_url,
    table="%s",
    column="%s",
    lowerBound=0,
    upperBound=1000000,
    numPartitions=%d,
    properties=jdbc_properties,
)`, childDF, emb.SourceTable, partCol, numPartitions))

	// Apply embedded-level transforms
	if len(emb.Transformations) > 0 {
		transformLines := transform.ToPySparkAll(emb.Transformations, childDF)
		ops = append(ops, transformLines...)
	}

	// Process nested children first (bottom-up)
	for _, nested := range emb.Embedded {
		nestedOps := g.buildEmbeddedOperations(childDF, &nested, numPartitions)
		ops = append(ops, nestedOps...)
	}

	// GroupBy + collect_list + join into parent
	nestedDF := emb.SourceTable + "_nested"
	ops = append(ops, fmt.Sprintf(`%s = %s.groupBy("%s").agg(
    collect_list(struct("*")).alias("%s")
)`, nestedDF, childDF, emb.JoinColumn, emb.FieldName))

	ops = append(ops, fmt.Sprintf(`%s = %s.join(
    %s,
    %s["%s"] == %s["%s"],
    "left",
).drop(%s["%s"])`, parentDFName, parentDFName, nestedDF,
		parentDFName, emb.ParentColumn, nestedDF, emb.JoinColumn,
		nestedDF, emb.JoinColumn))

	return ops
}

func buildJDBCURL(src config.SourceConfig) string {
	switch src.Type {
	case "postgresql":
		ssl := "false"
		if src.SSL {
			ssl = "true"
		}
		return fmt.Sprintf("jdbc:postgresql://%s:%d/%s?ssl=%s", src.Host, src.Port, src.Database, ssl)
	case "oracle":
		return fmt.Sprintf("jdbc:oracle:thin:@%s:%d/%s", src.Host, src.Port, src.Database)
	default:
		return ""
	}
}

// findPartitionColumn selects the best column for JDBC partitioning.
func findPartitionColumn(s *schema.Schema, tableName string) string {
	for _, t := range s.Tables {
		if t.Name != tableName {
			continue
		}
		if t.PrimaryKey != nil {
			for _, pkCol := range t.PrimaryKey.Columns {
				for _, col := range t.Columns {
					if col.Name == pkCol && isNumericType(col.DataType) {
						return col.Name
					}
				}
			}
		}
		for _, col := range t.Columns {
			if isNumericType(col.DataType) {
				return col.Name
			}
		}
	}
	return "id"
}

func isNumericType(dataType string) bool {
	switch dataType {
	case "integer", "bigint", "smallint", "serial", "bigserial",
		"int", "int4", "int8", "NUMBER", "INTEGER", "BIGINT", "SMALLINT":
		return true
	}
	return false
}

var migrationTemplate = `"""
Reloquent Migration Script
Generated by Reloquent -- https://github.com/reloquent/reloquent

Source: {{ .SourceType }} ({{ .JDBCUrl }})
Target: MongoDB ({{ .MongoDatabase }})
"""
{{ if .OracleGuidance }}{{ .OracleGuidance }}{{ end }}
from pyspark.sql import SparkSession
from pyspark.sql.functions import collect_list, struct{{ if .HasTransforms }}, coalesce, lit, expr, col{{ end }}

spark = SparkSession.builder \
    .appName("reloquent-migration") \
    .config("spark.mongodb.write.connection.uri", "{{ .MongoURI }}") \
    .config("spark.mongodb.write.database", "{{ .MongoDatabase }}") \
    .getOrCreate()

jdbc_url = "{{ .JDBCUrl }}"
jdbc_properties = {
    "driver": "{{ if eq .SourceType "postgresql" }}org.postgresql.Driver{{ else }}oracle.jdbc.OracleDriver{{ end }}",
}
{{ range .Collections }}
# === Collection: {{ .Name }} (from: {{ .SourceTable }}) ===
{{ range .Operations }}
{{ . }}
{{ end }}
{{ .Name }}_df.write \
    .format("mongodb") \
    .mode("overwrite") \
    .option("collection", "{{ .Name }}") \
    .option("ordered", "false") \
    .option("writeConcern.w", "1") \
    .option("writeConcern.journal", "false") \
    .option("maxBatchSize", "100000") \
    .option("compressors", "zstd") \
    .save()

print(f"Done: {{ .Name }}: { {{ .Name }}_df.count()} documents written")
{{ end }}
print("Migration complete.")
spark.stop()
`

// dfName returns a safe DataFrame variable name from a table name.
func dfName(table string) string {
	return strings.ReplaceAll(table, ".", "_") + "_df"
}
