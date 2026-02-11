package target

import (
	"context"
	"errors"
	"testing"

	"github.com/reloquent/reloquent/internal/sizing"
)

func TestMockOperator_DetectTopology(t *testing.T) {
	tests := []struct {
		name     string
		topo     *TopologyInfo
		wantType string
	}{
		{"atlas", &TopologyInfo{Type: "atlas", IsAtlas: true, ServerVersion: "7.0.0"}, "atlas"},
		{"replica_set", &TopologyInfo{Type: "replica_set", ServerVersion: "7.0.0"}, "replica_set"},
		{"sharded", &TopologyInfo{Type: "sharded", ShardCount: 3, ServerVersion: "7.0.0"}, "sharded"},
		{"standalone", &TopologyInfo{Type: "standalone", ServerVersion: "7.0.0"}, "standalone"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockOperator{TopologyResult: tt.topo}
			got, err := mock.DetectTopology(context.Background())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
		})
	}
}

func TestMockOperator_DetectTopology_Error(t *testing.T) {
	mock := &MockOperator{TopologyErr: errors.New("connection refused")}
	_, err := mock.DetectTopology(context.Background())
	if err == nil {
		t.Error("expected error")
	}
}

func TestMockOperator_Validate_StorageSufficient(t *testing.T) {
	mock := &MockOperator{
		ValidationResult: &ValidationResult{Passed: true},
	}

	plan := &sizing.SizingPlan{}
	result, err := mock.Validate(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("validation should pass")
	}
}

func TestMockOperator_Validate_StorageInsufficient(t *testing.T) {
	mock := &MockOperator{
		ValidationResult: &ValidationResult{
			Passed: false,
			Errors: []ValidationIssue{
				{Category: "storage", Message: "Insufficient storage", Suggestion: "Upgrade to larger tier"},
			},
		},
	}

	plan := &sizing.SizingPlan{}
	result, err := mock.Validate(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("validation should fail with insufficient storage")
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
}

func TestMockOperator_Validate_ShardMismatch(t *testing.T) {
	mock := &MockOperator{
		ValidationResult: &ValidationResult{
			Passed: false,
			Errors: []ValidationIssue{
				{Category: "shard", Message: "Sharding required but not available", Suggestion: "Deploy sharded cluster"},
			},
		},
	}

	plan := &sizing.SizingPlan{
		ShardPlan: &sizing.ShardingPlan{Recommended: true},
	}
	result, err := mock.Validate(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("validation should fail with shard mismatch")
	}
}

func TestMockOperator_CreateCollections(t *testing.T) {
	mock := &MockOperator{}
	err := mock.CreateCollections(context.Background(), []string{"users", "orders", "products"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.CreatedCollections) != 3 {
		t.Errorf("expected 3 created collections, got %d", len(mock.CreatedCollections))
	}
}

func TestMockOperator_ShardingSetup(t *testing.T) {
	mock := &MockOperator{}
	plan := &sizing.ShardingPlan{Recommended: true}
	err := mock.SetupSharding(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.ShardingSetup {
		t.Error("sharding should be set up")
	}
}

func TestMockOperator_DropCollections(t *testing.T) {
	mock := &MockOperator{}
	err := mock.DropCollections(context.Background(), []string{"users", "orders"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.DroppedCollections) != 2 {
		t.Errorf("expected 2 dropped collections, got %d", len(mock.DroppedCollections))
	}
}

func TestMockOperator_DropCollections_Error(t *testing.T) {
	mock := &MockOperator{DropErr: errors.New("permission denied")}
	err := mock.DropCollections(context.Background(), []string{"users"})
	if err == nil {
		t.Error("expected error")
	}
}

func TestMockOperator_BalancerOperations(t *testing.T) {
	mock := &MockOperator{}

	if err := mock.DisableBalancer(context.Background()); err != nil {
		t.Fatalf("DisableBalancer: %v", err)
	}
	if !mock.BalancerDisabled {
		t.Error("balancer should be disabled")
	}

	if err := mock.EnableBalancer(context.Background()); err != nil {
		t.Fatalf("EnableBalancer: %v", err)
	}
	if !mock.BalancerEnabled {
		t.Error("balancer should be enabled")
	}
}

func TestMockOperator_CountDocuments(t *testing.T) {
	mock := &MockOperator{
		DocCounts: map[string]int64{"users": 1000, "orders": 5000},
	}
	count, err := mock.CountDocuments(context.Background(), "users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1000 {
		t.Errorf("expected 1000, got %d", count)
	}
}

func TestMockOperator_SampleDocuments(t *testing.T) {
	mock := &MockOperator{
		SampleDocs: map[string][]map[string]interface{}{
			"users": {{"_id": "1", "name": "Alice"}},
		},
	}
	docs, err := mock.SampleDocuments(context.Background(), "users", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 {
		t.Errorf("expected 1 doc, got %d", len(docs))
	}
}

func TestMockOperator_AggregateSum(t *testing.T) {
	mock := &MockOperator{
		Sums: map[string]float64{"orders.total": 50000.0},
	}
	sum, err := mock.AggregateSum(context.Background(), "orders", "total")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum != 50000.0 {
		t.Errorf("expected 50000, got %f", sum)
	}
}

func TestMockOperator_AggregateCountDistinct(t *testing.T) {
	mock := &MockOperator{
		CountDistincts: map[string]int64{"users.id": 999},
	}
	count, err := mock.AggregateCountDistinct(context.Background(), "users", "id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 999 {
		t.Errorf("expected 999, got %d", count)
	}
}

func TestMockOperator_CreateIndex(t *testing.T) {
	mock := &MockOperator{}
	idx := IndexDefinition{
		Keys:   []IndexKey{{Field: "email", Order: 1}},
		Name:   "idx_email",
		Unique: true,
	}
	err := mock.CreateIndex(context.Background(), "users", idx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.CreatedIndexes) != 1 {
		t.Errorf("expected 1 index, got %d", len(mock.CreatedIndexes))
	}
	if mock.CreatedIndexes[0].Collection != "users" {
		t.Errorf("expected collection 'users', got %s", mock.CreatedIndexes[0].Collection)
	}
}

func TestMockOperator_CreateIndexes(t *testing.T) {
	mock := &MockOperator{}
	indexes := []CollectionIndex{
		{Collection: "users", Index: IndexDefinition{Keys: []IndexKey{{Field: "email", Order: 1}}}},
		{Collection: "orders", Index: IndexDefinition{Keys: []IndexKey{{Field: "user_id", Order: 1}}}},
	}
	err := mock.CreateIndexes(context.Background(), indexes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.CreatedIndexes) != 2 {
		t.Errorf("expected 2 indexes, got %d", len(mock.CreatedIndexes))
	}
}

func TestMockOperator_SetWriteConcern(t *testing.T) {
	mock := &MockOperator{}
	err := mock.SetWriteConcern(context.Background(), "majority", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.WriteConcernSet {
		t.Error("write concern should be set")
	}
	if mock.WriteConcernW != "majority" {
		t.Errorf("expected w=majority, got %s", mock.WriteConcernW)
	}
	if !mock.WriteConcernJ {
		t.Error("expected journal=true")
	}
}

func TestMockOperator_ListIndexBuildProgress(t *testing.T) {
	mock := &MockOperator{
		IndexBuildStatuses: []IndexBuildStatus{
			{Collection: "users", IndexName: "idx_email", Phase: "building", Progress: 50.0},
		},
	}
	statuses, err := mock.ListIndexBuildProgress(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 1 {
		t.Errorf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].Progress != 50.0 {
		t.Errorf("expected progress 50, got %f", statuses[0].Progress)
	}
}
