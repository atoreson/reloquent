//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/reloquent/reloquent/internal/codegen"
	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/discovery"
	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/typemap"
)

func TestFullPipelineCodegen(t *testing.T) {
	skipIfNoPostgres(t)
	ctx := context.Background()

	cfg := &config.Config{
		Version: 1,
		Source: config.SourceConfig{
			Type:           "postgresql",
			Host:           pgHost(t),
			Port:           pgPort(t),
			Database:       pgDatabase(t),
			Schema:         "public",
			Username:       pgUser(t),
			Password:       pgPassword(t),
			MaxConnections: 20,
		},
		Target: config.TargetConfig{
			Type:             "mongodb",
			ConnectionString: mongoURI(t),
			Database:         mongoDatabase(t),
		},
	}

	// Discover
	d, err := discovery.New(&cfg.Source)
	if err != nil {
		t.Fatalf("creating discoverer: %v", err)
	}
	defer d.Close()

	if err := d.Connect(ctx); err != nil {
		t.Fatalf("connecting: %v", err)
	}

	s, err := d.Discover(ctx)
	if err != nil {
		t.Fatalf("discovering: %v", err)
	}

	// Select all tables
	var tableNames []string
	for _, table := range s.Tables {
		tableNames = append(tableNames, table.Name)
	}

	// Generate mapping
	m := mapping.Suggest(s, tableNames)

	// Generate code
	gen := &codegen.Generator{
		Config:  cfg,
		Schema:  s,
		Mapping: m,
		TypeMap: typemap.ForDatabase("postgresql"),
	}

	result, err := gen.Generate()
	if err != nil {
		t.Fatalf("generating code: %v", err)
	}

	script := result.MigrationScript
	if script == "" {
		t.Fatal("empty migration script")
	}

	// Verify JDBC partitioning is always specified
	if !strings.Contains(script, "numPartitions") {
		t.Error("generated script missing numPartitions (JDBC partitioning)")
	}
	if !strings.Contains(script, "lowerBound") {
		t.Error("generated script missing lowerBound")
	}
	if !strings.Contains(script, "upperBound") {
		t.Error("generated script missing upperBound")
	}

	// Verify MongoDB write settings match CLAUDE.md requirements
	if !strings.Contains(script, `"writeConcern.w", "1"`) {
		t.Error("generated script missing w:1 write concern")
	}
	if !strings.Contains(script, `"writeConcern.journal", "false"`) {
		t.Error("generated script missing j:false")
	}
	if !strings.Contains(script, `"ordered", "false"`) {
		t.Error("generated script missing unordered writes")
	}
	if !strings.Contains(script, `"compressors", "zstd"`) {
		t.Error("generated script missing zstd compression")
	}

	// Verify it references the Pagila database
	if !strings.Contains(script, "postgresql") {
		t.Error("generated script doesn't mention postgresql")
	}

	t.Logf("Generated PySpark script: %d bytes, %d lines",
		len(script), strings.Count(script, "\n"))
}
