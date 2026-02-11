package postmigration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/indexes"
	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/source"
	"github.com/reloquent/reloquent/internal/state"
	"github.com/reloquent/reloquent/internal/target"
)

func makeTestOrchestrator(t *testing.T) (*Orchestrator, *source.MockReader, *target.MockOperator) {
	t.Helper()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.yaml")

	src := &source.MockReader{
		RowCounts:      map[string]int64{"users": 100},
		CountDistincts: map[string]int64{"users.user_id": 100},
	}
	tgt := &target.MockOperator{
		DocCounts:      map[string]int64{"users": 100},
		CountDistincts: map[string]int64{"users.user_id": 100},
		SampleDocs: map[string][]map[string]interface{}{
			"users": {{"_id": "1", "user_id": 1, "name": "Alice"}},
		},
		TopologyResult: &target.TopologyInfo{Type: "replica_set"},
	}

	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name:       "users",
				PrimaryKey: &schema.PrimaryKey{Name: "pk", Columns: []string{"user_id"}},
				Columns: []schema.Column{
					{Name: "user_id", DataType: "integer"},
					{Name: "name", DataType: "varchar"},
				},
			},
		},
	}
	m := &mapping.Mapping{
		Collections: []mapping.Collection{
			{Name: "users", SourceTable: "users"},
		},
	}

	st := state.New()
	st.MigrationStatus = "completed"
	st.SourceConfig = &config.SourceConfig{Type: "postgresql", Host: "localhost", Database: "testdb"}
	st.TargetConfig = &config.TargetConfig{Database: "target_db"}

	orch := &Orchestrator{
		Source:     src,
		Target:     tgt,
		Schema:     s,
		Mapping:    m,
		State:      st,
		StatePath:  statePath,
		IndexPlan:  &indexes.IndexPlan{},
		Topology:   &target.TopologyInfo{Type: "replica_set"},
		SampleSize: 10,
	}

	return orch, src, tgt
}

func TestRunValidation(t *testing.T) {
	orch, _, _ := makeTestOrchestrator(t)

	callbackCalled := false
	cb := Callbacks{
		OnValidationCheck: func(collection, checkType string, passed bool) {
			callbackCalled = true
		},
		OnStepComplete: func(step string) {
			if step != "validation" {
				t.Errorf("expected step 'validation', got %s", step)
			}
		},
	}

	result, err := orch.RunValidation(context.Background(), cb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "PASS" {
		t.Errorf("expected PASS, got %s", result.Status)
	}
	if !callbackCalled {
		t.Error("callback should have been called")
	}
	if orch.State.ValidationReportPath == "" {
		t.Error("validation report path should be set")
	}
}

func TestRunIndexBuilds(t *testing.T) {
	orch, _, _ := makeTestOrchestrator(t)
	orch.IndexPlan = &indexes.IndexPlan{
		Indexes: []target.CollectionIndex{
			{Collection: "users", Index: target.IndexDefinition{
				Keys: []target.IndexKey{{Field: "email", Order: 1}},
				Name: "idx_email",
			}},
		},
	}

	stepDone := false
	cb := Callbacks{
		OnStepComplete: func(step string) {
			stepDone = true
		},
	}

	err := orch.RunIndexBuilds(context.Background(), cb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orch.State.IndexBuildStatus != "complete" {
		t.Errorf("expected complete, got %s", orch.State.IndexBuildStatus)
	}
	if !stepDone {
		t.Error("step complete callback should fire")
	}
}

func TestRunIndexBuilds_Empty(t *testing.T) {
	orch, _, _ := makeTestOrchestrator(t)
	orch.IndexPlan = &indexes.IndexPlan{} // no indexes

	err := orch.RunIndexBuilds(context.Background(), Callbacks{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orch.State.IndexBuildStatus != "skipped" {
		t.Errorf("expected skipped, got %s", orch.State.IndexBuildStatus)
	}
}

func TestRunPostOps_Sharded(t *testing.T) {
	orch, _, tgt := makeTestOrchestrator(t)
	orch.Topology = &target.TopologyInfo{Type: "sharded"}

	err := orch.RunPostOps(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tgt.BalancerEnabled {
		t.Error("balancer should be re-enabled for sharded topology")
	}
	if !orch.State.BalancerReEnabled {
		t.Error("state should reflect balancer re-enabled")
	}
	if !tgt.WriteConcernSet {
		t.Error("write concern should be set")
	}
	if tgt.WriteConcernW != "majority" {
		t.Errorf("expected w=majority, got %s", tgt.WriteConcernW)
	}
	if !orch.State.WriteConcernRestored {
		t.Error("state should reflect write concern restored")
	}
}

func TestRunPostOps_ReplicaSet(t *testing.T) {
	orch, _, tgt := makeTestOrchestrator(t)
	orch.Topology = &target.TopologyInfo{Type: "replica_set"}

	err := orch.RunPostOps(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tgt.BalancerEnabled {
		t.Error("balancer should NOT be re-enabled for replica_set topology")
	}
	if !tgt.WriteConcernSet {
		t.Error("write concern should still be set")
	}
}

func TestCheckReadiness_AllPassed(t *testing.T) {
	orch, _, _ := makeTestOrchestrator(t)
	orch.State.MigrationStatus = "completed"
	orch.State.ValidationReportPath = "/some/path.json"
	orch.State.IndexBuildStatus = "complete"
	orch.State.WriteConcernRestored = true

	rpt, err := orch.CheckReadiness(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rpt.ProductionReady {
		t.Error("should be production ready")
	}
	if !orch.State.ProductionReady {
		t.Error("state should reflect production ready")
	}
	if orch.State.ReportPath == "" {
		t.Error("report path should be set")
	}
}

func TestCheckReadiness_NotReady(t *testing.T) {
	orch, _, _ := makeTestOrchestrator(t)
	orch.State.MigrationStatus = "failed"

	rpt, err := orch.CheckReadiness(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rpt.ProductionReady {
		t.Error("should not be production ready")
	}
}

func TestFullPipeline(t *testing.T) {
	orch, _, _ := makeTestOrchestrator(t)
	orch.IndexPlan = &indexes.IndexPlan{
		Indexes: []target.CollectionIndex{
			{Collection: "users", Index: target.IndexDefinition{
				Keys: []target.IndexKey{{Field: "name", Order: 1}},
			}},
		},
	}

	ctx := context.Background()
	cb := Callbacks{}

	// Validation
	result, err := orch.RunValidation(ctx, cb)
	if err != nil {
		t.Fatalf("validation: %v", err)
	}
	if result.Status != "PASS" {
		t.Errorf("validation: expected PASS, got %s", result.Status)
	}

	// Index builds
	if err := orch.RunIndexBuilds(ctx, cb); err != nil {
		t.Fatalf("index builds: %v", err)
	}

	// Post-ops
	if err := orch.RunPostOps(ctx); err != nil {
		t.Fatalf("post-ops: %v", err)
	}

	// Readiness
	rpt, err := orch.CheckReadiness(ctx)
	if err != nil {
		t.Fatalf("readiness: %v", err)
	}
	if !rpt.ProductionReady {
		t.Error("should be production ready after full pipeline")
	}
}
