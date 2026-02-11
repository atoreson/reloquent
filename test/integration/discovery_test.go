//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/discovery"
)

func TestDiscoverPagila(t *testing.T) {
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

	// Verify table count
	if len(s.Tables) < 10 {
		t.Errorf("expected at least 10 tables, got %d", len(s.Tables))
	}

	// Check specific tables exist
	tableNames := make(map[string]bool)
	for _, table := range s.Tables {
		tableNames[table.Name] = true
	}

	expectedTables := []string{
		"actor", "film", "customer", "rental", "payment",
		"category", "language", "country", "city", "address",
		"store", "staff", "inventory", "film_actor", "film_category",
	}
	for _, name := range expectedTables {
		if !tableNames[name] {
			t.Errorf("expected table %q not found", name)
		}
	}

	// Verify FK relationships
	var filmActorFKs int
	for _, table := range s.Tables {
		if table.Name == "film_actor" {
			filmActorFKs = len(table.ForeignKeys)
		}
	}
	if filmActorFKs < 2 {
		t.Errorf("film_actor should have at least 2 FKs, got %d", filmActorFKs)
	}

	// Verify data types
	for _, table := range s.Tables {
		if table.Name == "film" {
			for _, col := range table.Columns {
				if col.Name == "rental_rate" {
					if col.DataType != "numeric" {
						t.Errorf("film.rental_rate type = %q, want %q", col.DataType, "numeric")
					}
				}
			}
		}
	}

	// Verify row counts are populated (from ANALYZE)
	for _, table := range s.Tables {
		if table.Name == "actor" && table.RowCount == 0 {
			t.Errorf("actor table has 0 row count; did ANALYZE run?")
		}
	}

	// Check serial/sequence detection on PKs
	for _, table := range s.Tables {
		if table.Name == "actor" && table.PrimaryKey != nil {
			if len(table.PrimaryKey.Columns) != 1 || table.PrimaryKey.Columns[0] != "actor_id" {
				t.Errorf("actor PK = %v, want [actor_id]", table.PrimaryKey.Columns)
			}
		}
	}
}
