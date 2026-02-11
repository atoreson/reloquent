package target

import (
	"context"

	"github.com/reloquent/reloquent/internal/sizing"
)

// Operator defines operations on the MongoDB target.
type Operator interface {
	DetectTopology(ctx context.Context) (*TopologyInfo, error)
	Validate(ctx context.Context, plan *sizing.SizingPlan) (*ValidationResult, error)
	CreateCollections(ctx context.Context, names []string) error
	SetupSharding(ctx context.Context, plan *sizing.ShardingPlan) error
	DisableBalancer(ctx context.Context) error
	EnableBalancer(ctx context.Context) error
	DropCollections(ctx context.Context, names []string) error
	Close(ctx context.Context) error

	// Validation support
	CountDocuments(ctx context.Context, collection string) (int64, error)
	SampleDocuments(ctx context.Context, collection string, n int) ([]map[string]interface{}, error)
	AggregateSum(ctx context.Context, collection, field string) (float64, error)
	AggregateCountDistinct(ctx context.Context, collection, field string) (int64, error)

	// Index operations
	CreateIndex(ctx context.Context, collection string, index IndexDefinition) error
	CreateIndexes(ctx context.Context, indexes []CollectionIndex) error
	ListIndexBuildProgress(ctx context.Context) ([]IndexBuildStatus, error)

	// Write concern
	SetWriteConcern(ctx context.Context, w string, journal bool) error
}

// TopologyInfo describes the MongoDB target topology.
type TopologyInfo struct {
	Type          string `yaml:"type"` // "atlas", "replica_set", "sharded", "standalone"
	IsAtlas       bool   `yaml:"is_atlas"`
	ShardCount    int    `yaml:"shard_count"`
	ServerVersion string `yaml:"server_version"`
	StorageBytes  int64  `yaml:"storage_bytes"`
}

// ValidationResult holds the outcome of target validation.
type ValidationResult struct {
	Passed   bool              `yaml:"passed"`
	Warnings []ValidationIssue `yaml:"warnings,omitempty"`
	Errors   []ValidationIssue `yaml:"errors,omitempty"`
}

// ValidationIssue describes a validation warning or error.
type ValidationIssue struct {
	Category   string `yaml:"category"` // "storage", "tier", "shard", "permission"
	Message    string `yaml:"message"`
	Suggestion string `yaml:"suggestion"`
}

// IndexDefinition describes a single MongoDB index.
type IndexDefinition struct {
	Keys   []IndexKey
	Name   string
	Unique bool
}

// IndexKey is a single field in a compound index.
type IndexKey struct {
	Field string
	Order int // 1 or -1
}

// CollectionIndex pairs a collection name with an index definition.
type CollectionIndex struct {
	Collection string
	Index      IndexDefinition
}

// IndexBuildStatus reports progress of a background index build.
type IndexBuildStatus struct {
	Collection string
	IndexName  string
	Phase      string  // "building", "complete", "not_started"
	Progress   float64 // 0-100
	Message    string
}
