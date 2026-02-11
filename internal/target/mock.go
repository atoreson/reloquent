package target

import (
	"context"

	"github.com/reloquent/reloquent/internal/sizing"
)

// MockOperator is a test double for the Operator interface.
type MockOperator struct {
	TopologyResult   *TopologyInfo
	TopologyErr      error
	ValidationResult *ValidationResult
	ValidationErr    error
	CreateErr        error
	SetupShardErr    error
	DisableBalErr    error
	EnableBalErr     error
	DropErr          error
	CloseErr         error

	// Validation support
	DocCounts          map[string]int64
	DocCountErr        error
	SampleDocs         map[string][]map[string]interface{}
	SampleErr          error
	Sums               map[string]float64 // key: "collection.field"
	SumErr             error
	CountDistincts     map[string]int64 // key: "collection.field"
	CountDistinctErr   error

	// Index support
	CreateIndexErr      error
	CreateIndexesErr    error
	IndexBuildStatuses  []IndexBuildStatus
	IndexBuildErr       error
	SetWriteConcernErr  error

	// Track calls
	CreatedCollections []string
	DroppedCollections []string
	ShardingSetup      bool
	BalancerDisabled   bool
	BalancerEnabled    bool
	CreatedIndexes     []CollectionIndex
	WriteConcernSet    bool
	WriteConcernW      string
	WriteConcernJ      bool
}

func (m *MockOperator) DetectTopology(_ context.Context) (*TopologyInfo, error) {
	return m.TopologyResult, m.TopologyErr
}

func (m *MockOperator) Validate(_ context.Context, _ *sizing.SizingPlan) (*ValidationResult, error) {
	return m.ValidationResult, m.ValidationErr
}

func (m *MockOperator) CreateCollections(_ context.Context, names []string) error {
	m.CreatedCollections = append(m.CreatedCollections, names...)
	return m.CreateErr
}

func (m *MockOperator) SetupSharding(_ context.Context, _ *sizing.ShardingPlan) error {
	m.ShardingSetup = true
	return m.SetupShardErr
}

func (m *MockOperator) DisableBalancer(_ context.Context) error {
	m.BalancerDisabled = true
	return m.DisableBalErr
}

func (m *MockOperator) EnableBalancer(_ context.Context) error {
	m.BalancerEnabled = true
	return m.EnableBalErr
}

func (m *MockOperator) DropCollections(_ context.Context, names []string) error {
	m.DroppedCollections = append(m.DroppedCollections, names...)
	return m.DropErr
}

func (m *MockOperator) Close(_ context.Context) error {
	return m.CloseErr
}

func (m *MockOperator) CountDocuments(_ context.Context, collection string) (int64, error) {
	if m.DocCountErr != nil {
		return 0, m.DocCountErr
	}
	if m.DocCounts != nil {
		if c, ok := m.DocCounts[collection]; ok {
			return c, nil
		}
	}
	return 0, nil
}

func (m *MockOperator) SampleDocuments(_ context.Context, collection string, _ int) ([]map[string]interface{}, error) {
	if m.SampleErr != nil {
		return nil, m.SampleErr
	}
	if m.SampleDocs != nil {
		if s, ok := m.SampleDocs[collection]; ok {
			return s, nil
		}
	}
	return nil, nil
}

func (m *MockOperator) AggregateSum(_ context.Context, collection, field string) (float64, error) {
	if m.SumErr != nil {
		return 0, m.SumErr
	}
	key := collection + "." + field
	if m.Sums != nil {
		if s, ok := m.Sums[key]; ok {
			return s, nil
		}
	}
	return 0, nil
}

func (m *MockOperator) AggregateCountDistinct(_ context.Context, collection, field string) (int64, error) {
	if m.CountDistinctErr != nil {
		return 0, m.CountDistinctErr
	}
	key := collection + "." + field
	if m.CountDistincts != nil {
		if c, ok := m.CountDistincts[key]; ok {
			return c, nil
		}
	}
	return 0, nil
}

func (m *MockOperator) CreateIndex(_ context.Context, collection string, index IndexDefinition) error {
	m.CreatedIndexes = append(m.CreatedIndexes, CollectionIndex{Collection: collection, Index: index})
	return m.CreateIndexErr
}

func (m *MockOperator) CreateIndexes(_ context.Context, indexes []CollectionIndex) error {
	if m.CreateIndexesErr != nil {
		return m.CreateIndexesErr
	}
	m.CreatedIndexes = append(m.CreatedIndexes, indexes...)
	return nil
}

func (m *MockOperator) ListIndexBuildProgress(_ context.Context) ([]IndexBuildStatus, error) {
	return m.IndexBuildStatuses, m.IndexBuildErr
}

func (m *MockOperator) SetWriteConcern(_ context.Context, w string, journal bool) error {
	if m.SetWriteConcernErr != nil {
		return m.SetWriteConcernErr
	}
	m.WriteConcernSet = true
	m.WriteConcernW = w
	m.WriteConcernJ = journal
	return nil
}
