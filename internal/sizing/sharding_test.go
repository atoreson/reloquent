package sizing

import (
	"path/filepath"
	"testing"
)

func TestCalculateSharding_BelowThreshold(t *testing.T) {
	plan := CalculateSharding(tbToBytes(2), nil)

	if plan.Recommended {
		t.Error("sharding should not be recommended below 3 TB")
	}
	if len(plan.Explanations) == 0 {
		t.Error("expected explanation for why sharding is not recommended")
	}
	if plan.Explanations[0].Summary == "" {
		t.Error("explanation summary should not be empty")
	}
}

func TestCalculateSharding_AboveThreshold(t *testing.T) {
	collections := []ShardKeyInput{
		{
			CollectionName: "orders",
			PKFields:       []string{"id"},
			PKIsSequential: true,
			IndexedFields:  []string{"customer_id", "created_at"},
			EstimatedCount: 100_000_000,
		},
	}

	plan := CalculateSharding(tbToBytes(5), collections)

	if !plan.Recommended {
		t.Error("sharding should be recommended above 3 TB")
	}
	if plan.ShardCount < 2 {
		t.Errorf("expected at least 2 shards, got %d", plan.ShardCount)
	}
	if len(plan.Collections) != 1 {
		t.Fatalf("expected 1 collection shard, got %d", len(plan.Collections))
	}
}

func TestCalculateSharding_SequentialPK_Hashed(t *testing.T) {
	collections := []ShardKeyInput{
		{
			CollectionName: "users",
			PKFields:       []string{"id"},
			PKIsSequential: true,
		},
	}

	plan := CalculateSharding(tbToBytes(5), collections)

	cs := plan.Collections[0]
	if !cs.IsHashed {
		t.Error("sequential PK should result in hashed shard key")
	}
	if cs.ShardKey["id"] != "hashed" {
		t.Errorf("expected shard key {id: hashed}, got %v", cs.ShardKey)
	}
}

func TestCalculateSharding_NonSequentialPK_WithIndex_Ranged(t *testing.T) {
	collections := []ShardKeyInput{
		{
			CollectionName: "events",
			PKFields:       []string{"event_id"},
			PKIsSequential: false,
			IndexedFields:  []string{"tenant_id", "created_at"},
		},
	}

	plan := CalculateSharding(tbToBytes(5), collections)

	cs := plan.Collections[0]
	if cs.IsHashed {
		t.Error("non-sequential PK with indexed field should use ranged shard key")
	}
	// Should use first non-PK indexed field
	if cs.ShardKey["tenant_id"] != "1" {
		t.Errorf("expected ranged shard key on tenant_id, got %v", cs.ShardKey)
	}
}

func TestCalculateSharding_NoObviousKey_DefaultHashedID(t *testing.T) {
	collections := []ShardKeyInput{
		{
			CollectionName: "logs",
			PKFields:       []string{},
			PKIsSequential: false,
			IndexedFields:  []string{},
		},
	}

	plan := CalculateSharding(tbToBytes(5), collections)

	cs := plan.Collections[0]
	if !cs.IsHashed {
		t.Error("no obvious key should default to hashed")
	}
	if cs.ShardKey["_id"] != "hashed" {
		t.Errorf("expected shard key {_id: hashed}, got %v", cs.ShardKey)
	}
}

func TestCalculateSharding_PreSplitCount(t *testing.T) {
	collections := []ShardKeyInput{
		{
			CollectionName: "orders",
			PKFields:       []string{"id"},
			PKIsSequential: true,
		},
	}

	plan := CalculateSharding(tbToBytes(6), collections) // ~4 shards

	cs := plan.Collections[0]
	expectedSplits := plan.ShardCount * 4
	if cs.PreSplitCount != expectedSplits {
		t.Errorf("expected %d pre-splits, got %d", expectedSplits, cs.PreSplitCount)
	}
	// Commands should be splits - 1 (splitting N regions requires N-1 split points)
	if len(cs.PreSplitCmds) != cs.PreSplitCount-1 {
		t.Errorf("expected %d pre-split commands, got %d", cs.PreSplitCount-1, len(cs.PreSplitCmds))
	}
}

func TestCalculateSharding_ExplanationText(t *testing.T) {
	collections := []ShardKeyInput{
		{
			CollectionName: "orders",
			PKFields:       []string{"id"},
			PKIsSequential: true,
		},
	}

	plan := CalculateSharding(tbToBytes(5), collections)

	if len(plan.Explanations) == 0 {
		t.Fatal("expected explanations")
	}
	for _, exp := range plan.Explanations {
		if exp.Summary == "" {
			t.Error("explanation summary should not be empty")
		}
		if exp.Detail == "" {
			t.Error("explanation detail should not be empty")
		}
	}

	cs := plan.Collections[0]
	if cs.Explanation == "" {
		t.Error("collection shard explanation should not be empty")
	}
}

func TestCalculateSharding_ShardCount(t *testing.T) {
	tests := []struct {
		name      string
		dataBytes int64
		minShards int
	}{
		{"3 TB", tbToBytes(3), 2},
		{"6 TB", tbToBytes(6), 4},
		{"15 TB", tbToBytes(15), 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := CalculateSharding(tt.dataBytes, nil)
			if plan.ShardCount < tt.minShards {
				t.Errorf("expected at least %d shards for %s, got %d", tt.minShards, tt.name, plan.ShardCount)
			}
		})
	}
}

func TestShardingPlan_WriteYAML(t *testing.T) {
	plan := CalculateSharding(tbToBytes(5), []ShardKeyInput{
		{CollectionName: "orders", PKFields: []string{"id"}, PKIsSequential: true},
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "sharding.yaml")

	if err := plan.WriteYAML(path); err != nil {
		t.Fatalf("WriteYAML: %v", err)
	}
}

func TestShardKeyString(t *testing.T) {
	cs := CollectionShard{
		ShardKey: map[string]string{"_id": "hashed"},
	}
	got := cs.ShardKeyString()
	if got != "{_id: hashed}" {
		t.Errorf("ShardKeyString() = %q, want %q", got, "{_id: hashed}")
	}
}

func TestCalculate_IntegratesSharding(t *testing.T) {
	// Verify that Calculate() integrates sharding when called via the sizing engine
	input := Input{
		TotalDataBytes:        tbToBytes(5),
		TotalRowCount:         1_000_000_000,
		DenormExpansionFactor: 1.0, // No expansion to keep at 5TB
		CollectionCount:       3,
	}

	plan := Calculate(input)

	// With 5TB data and expansion factor 1.0, sharding should be recommended
	if plan.ShardPlan == nil {
		t.Error("expected sharding plan for 5 TB dataset")
	}
	if plan.ShardPlan != nil && !plan.ShardPlan.Recommended {
		t.Error("sharding should be recommended for 5 TB")
	}
}
