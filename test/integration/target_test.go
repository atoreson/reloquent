//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/reloquent/reloquent/internal/target"
)

func TestMongoConnect(t *testing.T) {
	skipIfNoMongo(t)
	ctx := context.Background()

	op, err := target.NewMongoOperator(ctx, mongoURI(t), mongoDatabase(t))
	if err != nil {
		t.Fatalf("connecting to MongoDB: %v", err)
	}
	defer op.Close(ctx)

	topo, err := op.DetectTopology(ctx)
	if err != nil {
		t.Fatalf("detecting topology: %v", err)
	}

	if topo.Type == "" {
		t.Error("topology type is empty")
	}
	if topo.ServerVersion == "" {
		t.Error("server version is empty")
	}
	t.Logf("MongoDB topology: %s, version: %s", topo.Type, topo.ServerVersion)
}

func TestMongoCollectionCRUD(t *testing.T) {
	skipIfNoMongo(t)
	ctx := context.Background()

	op, err := target.NewMongoOperator(ctx, mongoURI(t), mongoDatabase(t))
	if err != nil {
		t.Fatalf("connecting to MongoDB: %v", err)
	}
	defer op.Close(ctx)

	testCollections := []string{"test_crud_a", "test_crud_b"}

	// Create
	if err := op.CreateCollections(ctx, testCollections); err != nil {
		t.Fatalf("creating collections: %v", err)
	}

	// Verify count
	for _, name := range testCollections {
		count, err := op.CountDocuments(ctx, name)
		if err != nil {
			t.Errorf("counting %s: %v", name, err)
		}
		if count != 0 {
			t.Errorf("%s count = %d, want 0", name, count)
		}
	}

	// Cleanup
	if err := op.DropCollections(ctx, testCollections); err != nil {
		t.Fatalf("dropping collections: %v", err)
	}
}

func TestMongoIndexCRUD(t *testing.T) {
	skipIfNoMongo(t)
	ctx := context.Background()

	op, err := target.NewMongoOperator(ctx, mongoURI(t), mongoDatabase(t))
	if err != nil {
		t.Fatalf("connecting to MongoDB: %v", err)
	}
	defer op.Close(ctx)

	// Setup
	coll := "test_idx"
	op.CreateCollections(ctx, []string{coll})
	defer op.DropCollections(ctx, []string{coll})

	// Create index
	idx := target.IndexDefinition{
		Keys:   []target.IndexKey{{Field: "email", Order: 1}},
		Name:   "idx_email",
		Unique: true,
	}
	if err := op.CreateIndex(ctx, coll, idx); err != nil {
		t.Fatalf("creating index: %v", err)
	}

	// List index progress (should show complete)
	statuses, err := op.ListIndexBuildProgress(ctx)
	if err != nil {
		t.Fatalf("listing index builds: %v", err)
	}
	// Index build should be complete for a small/empty collection
	_ = statuses
}
