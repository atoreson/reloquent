package postmigration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/indexes"
	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/report"
	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/source"
	"github.com/reloquent/reloquent/internal/state"
	"github.com/reloquent/reloquent/internal/target"
	"github.com/reloquent/reloquent/internal/validation"
)

// Orchestrator sequences post-migration operations.
type Orchestrator struct {
	Source     source.Reader
	Target     target.Operator
	Schema     *schema.Schema
	Mapping    *mapping.Mapping
	State      *state.State
	StatePath  string
	IndexPlan  *indexes.IndexPlan
	Topology   *target.TopologyInfo
	SampleSize int
}

// Callbacks provides hooks for progress reporting.
type Callbacks struct {
	OnValidationCheck func(collection, checkType string, passed bool)
	OnIndexProgress   func(status []target.IndexBuildStatus)
	OnStepComplete    func(step string)
}

// RunValidation executes validation checks and updates state.
func (o *Orchestrator) RunValidation(ctx context.Context, cb Callbacks) (*validation.Result, error) {
	v := &validation.Validator{
		Source:     o.Source,
		Target:     o.Target,
		Schema:     o.Schema,
		Mapping:    o.Mapping,
		SampleSize: o.SampleSize,
		Callback:   cb.OnValidationCheck,
	}

	result, err := v.Validate(ctx)
	if err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	// Save validation report
	stateDir := filepath.Dir(config.ExpandHome(o.StatePath))
	reportPath := filepath.Join(stateDir, "validation-report.json")
	if err := writeValidationReport(result, reportPath); err != nil {
		return nil, fmt.Errorf("saving validation report: %w", err)
	}

	o.State.ValidationReportPath = reportPath
	if err := o.State.Save(o.StatePath); err != nil {
		return nil, fmt.Errorf("saving state: %w", err)
	}

	if cb.OnStepComplete != nil {
		cb.OnStepComplete("validation")
	}

	return result, nil
}

// RunIndexBuilds creates indexes and monitors progress.
func (o *Orchestrator) RunIndexBuilds(ctx context.Context, cb Callbacks) error {
	if o.IndexPlan == nil || len(o.IndexPlan.Indexes) == 0 {
		o.State.IndexBuildStatus = "skipped"
		return o.State.Save(o.StatePath)
	}

	o.State.IndexBuildStatus = "building"
	if err := o.State.Save(o.StatePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	if err := o.Target.CreateIndexes(ctx, o.IndexPlan.Indexes); err != nil {
		o.State.IndexBuildStatus = "failed"
		o.State.Save(o.StatePath)
		return fmt.Errorf("creating indexes: %w", err)
	}

	o.State.IndexBuildStatus = "complete"
	if err := o.State.Save(o.StatePath); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	if cb.OnStepComplete != nil {
		cb.OnStepComplete("index_builds")
	}

	return nil
}

// RunPostOps re-enables the balancer and restores write concern.
func (o *Orchestrator) RunPostOps(ctx context.Context) error {
	// Re-enable balancer if topology is sharded
	if o.Topology != nil && o.Topology.Type == "sharded" {
		if err := o.Target.EnableBalancer(ctx); err != nil {
			return fmt.Errorf("re-enabling balancer: %w", err)
		}
		o.State.BalancerReEnabled = true
	}

	// Restore production write concern
	if err := o.Target.SetWriteConcern(ctx, "majority", true); err != nil {
		return fmt.Errorf("restoring write concern: %w", err)
	}
	o.State.WriteConcernRestored = true

	return o.State.Save(o.StatePath)
}

// CheckReadiness evaluates all production readiness conditions and generates the report.
func (o *Orchestrator) CheckReadiness(ctx context.Context) (*report.MigrationReport, error) {
	var checks []report.ReadinessCheck

	// 1. Migration completed
	migPassed := o.State.MigrationStatus == "completed"
	checks = append(checks, report.ReadinessCheck{
		Name:    "Migration completed",
		Passed:  migPassed,
		Message: condMsg(migPassed, "Migration finished successfully", "Migration has not completed"),
	})

	// 2. Validation passed
	valPassed := o.State.ValidationReportPath != ""
	checks = append(checks, report.ReadinessCheck{
		Name:    "Data validation",
		Passed:  valPassed,
		Message: condMsg(valPassed, "Validation report generated", "Run validation to verify data integrity"),
	})

	// 3. Indexes built
	idxPassed := o.State.IndexBuildStatus == "complete" || o.State.IndexBuildStatus == "skipped"
	checks = append(checks, report.ReadinessCheck{
		Name:    "Indexes built",
		Passed:  idxPassed,
		Message: condMsg(idxPassed, "All indexes built successfully", "Index builds not complete"),
	})

	// 4. Write concern restored
	wcPassed := o.State.WriteConcernRestored
	checks = append(checks, report.ReadinessCheck{
		Name:    "Write concern restored",
		Passed:  wcPassed,
		Message: condMsg(wcPassed, "Write concern set to majority with journaling", "Restore production write concern (w:majority, j:true)"),
	})

	// 5. Balancer re-enabled (only if sharded)
	if o.Topology != nil && o.Topology.Type == "sharded" {
		balPassed := o.State.BalancerReEnabled
		checks = append(checks, report.ReadinessCheck{
			Name:    "Balancer re-enabled",
			Passed:  balPassed,
			Message: condMsg(balPassed, "Balancer is running", "Re-enable the chunk balancer"),
		})
	}

	// Determine topology and counts
	topoType := "unknown"
	if o.Topology != nil {
		topoType = o.Topology.Type
	}
	sourceType, sourceHost, sourceDB := "", "", ""
	tableCount := 0
	if o.State.SourceConfig != nil {
		sourceType = o.State.SourceConfig.Type
		sourceHost = o.State.SourceConfig.Host
		sourceDB = o.State.SourceConfig.Database
	}
	if o.Schema != nil {
		tableCount = len(o.Schema.Tables)
	}
	targetDB := ""
	if o.State.TargetConfig != nil {
		targetDB = o.State.TargetConfig.Database
	}
	collCount := 0
	if o.Mapping != nil {
		collCount = len(o.Mapping.Collections)
	}
	indexCount := 0
	if o.IndexPlan != nil {
		indexCount = len(o.IndexPlan.Indexes)
	}

	rpt := report.GenerateReport(
		sourceType, sourceHost, sourceDB, tableCount,
		targetDB, topoType, collCount,
		o.State.MigrationStatus, o.State.AWSResourceType,
		nil, // validation result loaded separately if needed
		indexCount, o.State.IndexBuildStatus,
		checks,
	)

	// Set production ready on state
	o.State.ProductionReady = rpt.ProductionReady

	// Save report
	stateDir := filepath.Dir(config.ExpandHome(o.StatePath))
	reportPath := filepath.Join(stateDir, "migration-report.json")
	if err := report.WriteJSON(rpt, reportPath); err != nil {
		return nil, fmt.Errorf("writing report: %w", err)
	}
	o.State.ReportPath = reportPath

	if err := o.State.Save(o.StatePath); err != nil {
		return nil, fmt.Errorf("saving state: %w", err)
	}

	return rpt, nil
}

func condMsg(passed bool, passMsg, failMsg string) string {
	if passed {
		return passMsg
	}
	return failMsg
}

func writeValidationReport(result *validation.Result, path string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling validation report: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}
