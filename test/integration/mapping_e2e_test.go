//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/discovery"
	"github.com/reloquent/reloquent/internal/mapping"
)

func TestDiscoverAndSuggestMapping(t *testing.T) {
	skipIfNoPostgres(t)
	ctx := context.Background()

	cfg := &config.SourceConfig{
		Type:     "postgresql",
		Host:     pgHost(t),
		Port:     pgPort(t),
		Database: pgDatabase(t),
		Schema:   "public",
		Username: pgUser(t),
		Password: pgPassword(t),
	}

	d, err := discovery.New(cfg)
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

	m := mapping.Suggest(s, tableNames)
	if m == nil {
		t.Fatal("Suggest returned nil")
	}

	if len(m.Collections) == 0 {
		t.Fatal("no collections in suggested mapping")
	}

	// Verify that the mapping assigns all selected tables
	usedTables := make(map[string]bool)
	for _, col := range m.Collections {
		usedTables[col.SourceTable] = true
		for _, emb := range col.Embedded {
			usedTables[emb.SourceTable] = true
		}
		for _, ref := range col.References {
			usedTables[ref.SourceTable] = true
		}
	}

	for _, name := range tableNames {
		if !usedTables[name] {
			t.Errorf("table %q not assigned in mapping", name)
		}
	}

	// Film should have some embedded tables (film_actor, film_category are children)
	var filmCollections int
	for _, col := range m.Collections {
		if col.SourceTable == "film" {
			filmCollections++
			t.Logf("film collection has %d embedded, %d references",
				len(col.Embedded), len(col.References))
		}
	}

	t.Logf("Suggested mapping: %d collections from %d tables",
		len(m.Collections), len(tableNames))
}

func TestMappingSizeEstimate(t *testing.T) {
	skipIfNoPostgres(t)
	ctx := context.Background()

	cfg := &config.SourceConfig{
		Type:     "postgresql",
		Host:     pgHost(t),
		Port:     pgPort(t),
		Database: pgDatabase(t),
		Schema:   "public",
		Username: pgUser(t),
		Password: pgPassword(t),
	}

	d, err := discovery.New(cfg)
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

	var tableNames []string
	for _, table := range s.Tables {
		tableNames = append(tableNames, table.Name)
	}

	m := mapping.Suggest(s, tableNames)
	estimates := mapping.EstimateSizes(s, m)

	if len(estimates) == 0 {
		t.Fatal("no size estimates returned")
	}

	// No Pagila collection should exceed 16MB
	for _, est := range estimates {
		if est.ExceedsLimit {
			t.Errorf("collection %q flagged as exceeding 16MB limit (max: %d bytes)",
				est.Collection, est.MaxDocSizeBytes)
		}
	}
}
