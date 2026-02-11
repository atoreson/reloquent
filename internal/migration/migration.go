package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/reloquent/reloquent/internal/aws"
	"github.com/reloquent/reloquent/internal/sizing"
	"github.com/reloquent/reloquent/internal/target"
)

// Status represents the current migration state.
type Status struct {
	Phase           string             `yaml:"phase"` // "preflight", "provisioning", "running", "completed", "failed", "partial_failure"
	Overall         ProgressInfo       `yaml:"overall"`
	Collections     []CollectionStatus `yaml:"collections"`
	ElapsedTime     time.Duration      `yaml:"elapsed_time"`
	EstimatedRemain time.Duration      `yaml:"estimated_remain"`
	Errors          []string           `yaml:"errors,omitempty"`
}

// ProgressInfo tracks overall progress.
type ProgressInfo struct {
	DocsWritten     int64   `yaml:"docs_written"`
	DocsTotal       int64   `yaml:"docs_total"`
	BytesRead       int64   `yaml:"bytes_read"`
	PercentComplete float64 `yaml:"percent_complete"`
	ThroughputMBps  float64 `yaml:"throughput_mbps"`
}

// CollectionStatus tracks per-collection progress.
type CollectionStatus struct {
	Name            string  `yaml:"name"`
	State           string  `yaml:"state"` // "pending", "running", "completed", "failed"
	DocsWritten     int64   `yaml:"docs_written"`
	DocsTotal       int64   `yaml:"docs_total"`
	PercentComplete float64 `yaml:"percent_complete"`
	Error           string  `yaml:"error,omitempty"`
}

// FailureAction defines what to do when a migration partially fails.
type FailureAction int

const (
	ActionRetryFailed FailureAction = iota
	ActionRestartAll
	ActionAbort
)

// StatusCallback is called when migration status updates.
type StatusCallback func(status *Status)

// Executor orchestrates the migration process.
type Executor struct {
	provisioner aws.Provisioner
	target      target.Operator
	artifacts   *aws.UploadResult
	plan        *sizing.SizingPlan
	resourceID  string
	sourceJDBC  string
	mongoURI    string
}

// NewExecutor creates a new migration executor.
func NewExecutor(prov aws.Provisioner, tgt target.Operator, arts *aws.UploadResult, plan *sizing.SizingPlan) *Executor {
	return &Executor{
		provisioner: prov,
		target:      tgt,
		artifacts:   arts,
		plan:        plan,
	}
}

// SetConnectionInfo sets the source and target connection strings for preflight checks.
func (e *Executor) SetConnectionInfo(sourceJDBC, mongoURI string) {
	e.sourceJDBC = sourceJDBC
	e.mongoURI = mongoURI
}

// Run executes the full migration.
func (e *Executor) Run(ctx context.Context, callback StatusCallback) (*Status, error) {
	startTime := time.Now()

	status := &Status{
		Phase: "preflight",
	}
	e.notify(callback, status)

	// Preflight check
	preflightResult, err := aws.RunPreflight(ctx, e.provisioner, e.resourceID, e.sourceJDBC, e.mongoURI)
	if err != nil {
		status.Phase = "failed"
		status.Errors = append(status.Errors, fmt.Sprintf("preflight failed: %v", err))
		e.notify(callback, status)
		return status, err
	}
	if len(preflightResult.Errors) > 0 {
		status.Phase = "failed"
		status.Errors = append(status.Errors, preflightResult.Errors...)
		e.notify(callback, status)
		return status, fmt.Errorf("preflight check failed: %v", preflightResult.Errors)
	}

	// Submit migration step
	status.Phase = "running"
	e.notify(callback, status)

	if err := e.provisioner.SubmitStep(ctx, e.resourceID, e.artifacts.ScriptS3URI); err != nil {
		status.Phase = "failed"
		status.Errors = append(status.Errors, fmt.Sprintf("submitting step: %v", err))
		e.notify(callback, status)
		return status, err
	}

	// Monitor progress
	monitor := NewMonitor(e.provisioner, e.resourceID)
	finalStatus, err := monitor.Poll(ctx, callback)
	if err != nil {
		status.Phase = "failed"
		status.Errors = append(status.Errors, err.Error())
		e.notify(callback, status)
		return status, err
	}

	finalStatus.ElapsedTime = time.Since(startTime)
	e.notify(callback, finalStatus)

	return finalStatus, nil
}

// RetryFailed re-runs only the failed collections.
func (e *Executor) RetryFailed(ctx context.Context, failed []string, callback StatusCallback) (*Status, error) {
	startTime := time.Now()

	status := &Status{
		Phase: "running",
		Collections: make([]CollectionStatus, len(failed)),
	}
	for i, name := range failed {
		status.Collections[i] = CollectionStatus{
			Name:  name,
			State: "pending",
		}
	}
	e.notify(callback, status)

	// Re-submit with collection filter
	if err := e.provisioner.SubmitStep(ctx, e.resourceID, e.artifacts.ScriptS3URI); err != nil {
		status.Phase = "failed"
		status.Errors = append(status.Errors, fmt.Sprintf("submitting retry step: %v", err))
		e.notify(callback, status)
		return status, err
	}

	// Monitor
	monitor := NewMonitor(e.provisioner, e.resourceID)
	finalStatus, err := monitor.Poll(ctx, callback)
	if err != nil {
		return nil, err
	}

	finalStatus.ElapsedTime = time.Since(startTime)
	return finalStatus, nil
}

// SetResourceID sets the provisioned resource ID for the executor.
func (e *Executor) SetResourceID(id string) {
	e.resourceID = id
}

func (e *Executor) notify(callback StatusCallback, status *Status) {
	if callback != nil {
		callback(status)
	}
}
