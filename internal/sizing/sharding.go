package sizing

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ShardingPlan describes whether and how to shard the MongoDB deployment.
type ShardingPlan struct {
	Recommended  bool              `yaml:"recommended"`
	Collections  []CollectionShard `yaml:"collections"`
	ShardCount   int               `yaml:"shard_count"`
	Explanations []Explanation     `yaml:"explanations"`
}

// CollectionShard describes the sharding configuration for a single collection.
type CollectionShard struct {
	CollectionName string            `yaml:"collection_name"`
	ShardKey       map[string]string `yaml:"shard_key"`
	IsHashed       bool              `yaml:"is_hashed"`
	PreSplitCount  int               `yaml:"pre_split_count"`
	PreSplitCmds   []string          `yaml:"pre_split_commands"`
	Explanation    string            `yaml:"explanation"`
}

// ShardKeyInput provides information needed to recommend a shard key for a collection.
type ShardKeyInput struct {
	CollectionName   string
	DocumentFields   []string
	PKFields         []string
	PKIsSequential   bool
	IndexedFields    []string
	EstimatedDocSize int64
	EstimatedCount   int64
}

const shardingThreshold = 3 * 1024 * 1024 * 1024 * 1024 // 3 TB

// CalculateSharding determines whether sharding is recommended and produces a plan.
func CalculateSharding(totalDataBytes int64, collections []ShardKeyInput) *ShardingPlan {
	plan := &ShardingPlan{}

	if totalDataBytes < shardingThreshold {
		plan.Recommended = false
		plan.Explanations = append(plan.Explanations, Explanation{
			Category: "sharding",
			Summary:  fmt.Sprintf("Sharding not recommended for %s", FormatBytes(totalDataBytes)),
			Detail: fmt.Sprintf(
				"Your estimated data size of %s is below the 3 TB threshold where sharding becomes beneficial. "+
					"A replica set can handle this volume efficiently. Sharding adds operational complexity — "+
					"like splitting a library across multiple buildings. It only makes sense when a single building can't hold all the books.",
				FormatBytes(totalDataBytes)),
		})
		return plan
	}

	plan.Recommended = true

	// Shard count: ~1-2 TB per shard
	tbData := bytesToTB(totalDataBytes)
	plan.ShardCount = ceilInt(tbData / 1.5)
	if plan.ShardCount < 2 {
		plan.ShardCount = 2
	}

	plan.Explanations = append(plan.Explanations, Explanation{
		Category: "sharding",
		Summary:  fmt.Sprintf("Sharding recommended: %d shards for %s", plan.ShardCount, FormatBytes(totalDataBytes)),
		Detail: fmt.Sprintf(
			"With %s of data, sharding across %d shards is recommended (targeting ~1.5 TB per shard). "+
				"This distributes both storage and write throughput across multiple servers, like splitting a highway into "+
				"multiple lanes to handle more traffic.",
			FormatBytes(totalDataBytes), plan.ShardCount),
	})

	for _, col := range collections {
		cs := calculateCollectionShard(col, plan.ShardCount)
		plan.Collections = append(plan.Collections, cs)
	}

	return plan
}

func calculateCollectionShard(input ShardKeyInput, shardCount int) CollectionShard {
	cs := CollectionShard{
		CollectionName: input.CollectionName,
		ShardKey:       make(map[string]string),
	}

	// Decision logic:
	// 1. Sequential PK → hashed shard key (avoids hotspot on last shard)
	// 2. High-cardinality indexed field → ranged shard key
	// 3. No obvious key → hashed _id

	if input.PKIsSequential && len(input.PKFields) > 0 {
		// Sequential PK → hash it to distribute evenly
		keyField := input.PKFields[0]
		cs.ShardKey[keyField] = "hashed"
		cs.IsHashed = true
		cs.Explanation = fmt.Sprintf(
			"Using hashed shard key on '%s' because the primary key is sequential. "+
				"Hashing distributes writes evenly across shards instead of sending all new documents to the last shard.",
			keyField)
	} else if len(input.IndexedFields) > 0 {
		// Use the first indexed field as a ranged shard key
		keyField := bestIndexedField(input.IndexedFields, input.PKFields)
		cs.ShardKey[keyField] = "1"
		cs.IsHashed = false
		cs.Explanation = fmt.Sprintf(
			"Using ranged shard key on '%s' because it's an indexed field with high cardinality. "+
				"Range-based sharding allows efficient queries on this field.",
			keyField)
	} else {
		// Default: hashed _id
		cs.ShardKey["_id"] = "hashed"
		cs.IsHashed = true
		cs.Explanation = "Using hashed shard key on '_id' as a safe default. " +
			"This distributes documents evenly across shards."
	}

	// Pre-split: shardCount × 4 chunks
	cs.PreSplitCount = shardCount * 4
	cs.PreSplitCmds = generatePreSplitCmds(input.CollectionName, cs.ShardKey, cs.PreSplitCount)

	return cs
}

func bestIndexedField(indexedFields, pkFields []string) string {
	// Prefer indexed fields that aren't part of the PK
	pkSet := make(map[string]bool, len(pkFields))
	for _, f := range pkFields {
		pkSet[f] = true
	}
	for _, f := range indexedFields {
		if !pkSet[f] {
			return f
		}
	}
	// Fall back to first indexed field
	if len(indexedFields) > 0 {
		return indexedFields[0]
	}
	return "_id"
}

func generatePreSplitCmds(collName string, shardKey map[string]string, splitCount int) []string {
	if splitCount <= 1 {
		return nil
	}

	// Get the shard key field
	var keyField string
	var isHashed bool
	for k, v := range shardKey {
		keyField = k
		isHashed = (v == "hashed")
		break
	}

	var cmds []string

	if isHashed {
		// For hashed keys, use MinKey/MaxKey split points evenly distributed
		// across the hash space
		for i := 1; i < splitCount; i++ {
			// Distribute split points across the hash range
			splitPoint := (int64(i) * (1 << 62)) / int64(splitCount)
			cmds = append(cmds, fmt.Sprintf(
				`sh.splitAt("%s", {"%s": NumberLong("%d")})`,
				collName, keyField, splitPoint))
		}
	} else {
		// For ranged keys, generate placeholder split commands
		for i := 1; i < splitCount; i++ {
			cmds = append(cmds, fmt.Sprintf(
				`sh.splitAt("%s", {"%s": "split_point_%d"})`,
				collName, keyField, i))
		}
	}

	return cmds
}

// WriteYAML writes the sharding plan to a YAML file.
func (sp *ShardingPlan) WriteYAML(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	data, err := yaml.Marshal(sp)
	if err != nil {
		return fmt.Errorf("marshaling sharding plan: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}

// ShardKeyString returns a human-readable representation of the shard key.
func (cs *CollectionShard) ShardKeyString() string {
	parts := make([]string, 0, len(cs.ShardKey))
	// Sort keys for deterministic output
	keys := make([]string, 0, len(cs.ShardKey))
	for k := range cs.ShardKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %s", k, cs.ShardKey[k]))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
