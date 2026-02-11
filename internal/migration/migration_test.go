package migration

import (
	"context"
	"errors"
	"testing"

	"github.com/reloquent/reloquent/internal/aws"
	"github.com/reloquent/reloquent/internal/sizing"
	"github.com/reloquent/reloquent/internal/target"
)

func TestNewExecutor(t *testing.T) {
	prov := &aws.MockProvisioner{
		StatusResult: &aws.ProvisionStatus{State: "RUNNING"},
	}
	tgt := &target.MockOperator{}
	arts := &aws.UploadResult{
		ScriptS3URI: "s3://bucket/migration.py",
		ConfigS3URI: "s3://bucket/config.yaml",
	}
	plan := &sizing.SizingPlan{}

	exec := NewExecutor(prov, tgt, arts, plan)
	if exec == nil {
		t.Fatal("expected non-nil executor")
	}
}

func TestExecutor_Run_Success(t *testing.T) {
	// Create a provisioner that transitions from RUNNING to COMPLETED
	callCount := 0
	prov := &statusSequenceProvisioner{
		states: []string{"RUNNING", "RUNNING", "COMPLETED"},
	}
	_ = callCount

	arts := &aws.UploadResult{
		ScriptS3URI: "s3://bucket/migration.py",
	}
	plan := &sizing.SizingPlan{}
	tgt := &target.MockOperator{}

	exec := NewExecutor(prov, tgt, arts, plan)
	exec.SetResourceID("j-ABC")
	exec.SetConnectionInfo("jdbc:postgresql://host/db", "mongodb://host/db")

	var callbacks int
	status, err := exec.Run(context.Background(), func(s *Status) {
		callbacks++
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Phase != "completed" {
		t.Errorf("phase = %q, want completed", status.Phase)
	}
	if callbacks == 0 {
		t.Error("expected callback to be called at least once")
	}
}

func TestExecutor_Run_Failure(t *testing.T) {
	prov := &statusSequenceProvisioner{
		states:   []string{"RUNNING", "FAILED"},
		messages: []string{"", "OOM error"},
	}

	arts := &aws.UploadResult{
		ScriptS3URI: "s3://bucket/migration.py",
	}
	plan := &sizing.SizingPlan{}
	tgt := &target.MockOperator{}

	exec := NewExecutor(prov, tgt, arts, plan)
	exec.SetResourceID("j-ABC")
	exec.SetConnectionInfo("jdbc:postgresql://host/db", "mongodb://host/db")

	status, err := exec.Run(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for failed migration")
	}
	if status.Phase != "failed" {
		t.Errorf("phase = %q, want failed", status.Phase)
	}
}

func TestExecutor_Run_ContextCancellation(t *testing.T) {
	prov := &statusSequenceProvisioner{
		states: []string{"RUNNING", "RUNNING", "RUNNING"}, // never completes
	}

	arts := &aws.UploadResult{ScriptS3URI: "s3://bucket/script.py"}
	plan := &sizing.SizingPlan{}
	tgt := &target.MockOperator{}

	exec := NewExecutor(prov, tgt, arts, plan)
	exec.SetResourceID("j-ABC")
	exec.SetConnectionInfo("jdbc:postgresql://host/db", "mongodb://host/db")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := exec.Run(ctx, nil)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestRetryFailed(t *testing.T) {
	prov := &statusSequenceProvisioner{
		states: []string{"RUNNING", "COMPLETED"},
	}

	arts := &aws.UploadResult{ScriptS3URI: "s3://bucket/script.py"}
	plan := &sizing.SizingPlan{}
	tgt := &target.MockOperator{}

	exec := NewExecutor(prov, tgt, arts, plan)
	exec.SetResourceID("j-ABC")

	var callbacks int
	status, err := exec.RetryFailed(context.Background(), []string{"orders"}, func(s *Status) {
		callbacks++
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Phase != "completed" {
		t.Errorf("phase = %q, want completed", status.Phase)
	}
}

func TestCallbackFiring(t *testing.T) {
	prov := &statusSequenceProvisioner{
		states: []string{"RUNNING", "RUNNING", "COMPLETED"},
	}

	arts := &aws.UploadResult{ScriptS3URI: "s3://bucket/script.py"}
	plan := &sizing.SizingPlan{}
	tgt := &target.MockOperator{}

	exec := NewExecutor(prov, tgt, arts, plan)
	exec.SetResourceID("j-ABC")
	exec.SetConnectionInfo("jdbc:postgresql://host/db", "mongodb://host/db")

	var phases []string
	_, err := exec.Run(context.Background(), func(s *Status) {
		phases = append(phases, s.Phase)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(phases) < 2 {
		t.Errorf("expected at least 2 callbacks, got %d", len(phases))
	}
}

// statusSequenceProvisioner is a mock that returns a sequence of states.
type statusSequenceProvisioner struct {
	states   []string
	messages []string
	index    int
}

func (p *statusSequenceProvisioner) Provision(_ context.Context, _ aws.ProvisionPlan) (*aws.ProvisionResult, error) {
	return &aws.ProvisionResult{ResourceID: "j-TEST"}, nil
}

func (p *statusSequenceProvisioner) Status(_ context.Context, _ string) (*aws.ProvisionStatus, error) {
	if p.index >= len(p.states) {
		return &aws.ProvisionStatus{State: "COMPLETED"}, nil
	}
	state := p.states[p.index]
	msg := ""
	if p.index < len(p.messages) {
		msg = p.messages[p.index]
	}
	p.index++
	return &aws.ProvisionStatus{State: state, Message: msg}, nil
}

func (p *statusSequenceProvisioner) SubmitStep(_ context.Context, _ string, _ string) error {
	return nil
}

func (p *statusSequenceProvisioner) Teardown(_ context.Context, _ string) error {
	return nil
}

func TestExecutor_PreflightStatusError(t *testing.T) {
	prov := &aws.MockProvisioner{
		StatusErr: errors.New("connection timeout"),
	}
	arts := &aws.UploadResult{ScriptS3URI: "s3://bucket/script.py"}
	plan := &sizing.SizingPlan{}
	tgt := &target.MockOperator{}

	exec := NewExecutor(prov, tgt, arts, plan)
	exec.SetResourceID("j-ABC")
	exec.SetConnectionInfo("jdbc:postgresql://host/db", "mongodb://host/db")

	_, err := exec.Run(context.Background(), nil)
	if err == nil {
		t.Error("expected error when preflight status check fails")
	}
}
