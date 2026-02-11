package aws

import (
	"context"
	"testing"

	"github.com/reloquent/reloquent/internal/sizing"
)

func TestGlueProvisioner_MockProvision(t *testing.T) {
	mock := &MockProvisioner{
		ProvisionResult: &ProvisionResult{
			ResourceID:   "jr-GLUE123",
			ResourceType: "glue_job",
		},
	}

	plan := ProvisionPlan{
		Platform:  "glue",
		SparkPlan: sizing.SparkPlan{Platform: "glue", DPUCount: 50},
	}

	result, err := mock.Provision(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ResourceType != "glue_job" {
		t.Errorf("ResourceType = %q, want %q", result.ResourceType, "glue_job")
	}
}

func TestGlueProvisioner_DPUCount(t *testing.T) {
	mock := &MockProvisioner{
		ProvisionResult: &ProvisionResult{ResourceID: "jr-1", ResourceType: "glue_job"},
	}

	plan := ProvisionPlan{
		Platform:  "glue",
		SparkPlan: sizing.SparkPlan{Platform: "glue", DPUCount: 100},
	}

	_, err := mock.Provision(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.ProvisionedPlan.SparkPlan.DPUCount != 100 {
		t.Errorf("DPU count = %d, want 100", mock.ProvisionedPlan.SparkPlan.DPUCount)
	}
}

func TestGlueProvisioner_Teardown(t *testing.T) {
	mock := &MockProvisioner{}
	err := mock.Teardown(context.Background(), "jr-GLUE123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.TeardownCalled {
		t.Error("Teardown should have been called")
	}
}

func TestMapGlueState(t *testing.T) {
	tests := []struct {
		glueState string
		want      string
	}{
		{"STARTING", "STARTING"},
		{"RUNNING", "RUNNING"},
		{"SUCCEEDED", "COMPLETED"},
		{"FAILED", "FAILED"},
		{"ERROR", "FAILED"},
		{"STOPPED", "TERMINATED"},
		{"STOPPING", "TERMINATED"},
	}

	for _, tt := range tests {
		t.Run(tt.glueState, func(t *testing.T) {
			if got := mapGlueState(tt.glueState); got != tt.want {
				t.Errorf("mapGlueState(%q) = %q, want %q", tt.glueState, got, tt.want)
			}
		})
	}
}
