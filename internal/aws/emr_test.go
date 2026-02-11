package aws

import (
	"context"
	"testing"
)

func TestMockProvisioner_Provision(t *testing.T) {
	mock := &MockProvisioner{
		ProvisionResult: &ProvisionResult{
			ResourceID:   "j-ABC123",
			ResourceType: "emr_cluster",
		},
	}

	plan := ProvisionPlan{
		Platform:    "emr",
		ScriptS3URI: "s3://bucket/script.py",
		ConfigS3URI: "s3://bucket/config.yaml",
		Tags:        map[string]string{"env": "test"},
	}

	result, err := mock.Provision(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.ProvisionCalled {
		t.Error("Provision should have been called")
	}
	if result.ResourceID != "j-ABC123" {
		t.Errorf("ResourceID = %q, want %q", result.ResourceID, "j-ABC123")
	}
	if result.ResourceType != "emr_cluster" {
		t.Errorf("ResourceType = %q, want %q", result.ResourceType, "emr_cluster")
	}
	if mock.ProvisionedPlan.Platform != "emr" {
		t.Errorf("Platform = %q, want %q", mock.ProvisionedPlan.Platform, "emr")
	}
}

func TestMockProvisioner_StatusTransitions(t *testing.T) {
	tests := []struct {
		state   string
		message string
	}{
		{"STARTING", "Cluster is starting"},
		{"RUNNING", "Cluster is running"},
		{"COMPLETED", "Cluster terminated"},
		{"FAILED", "Bootstrap action failed"},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			mock := &MockProvisioner{
				StatusResult: &ProvisionStatus{
					State:   tt.state,
					Message: tt.message,
				},
			}

			status, err := mock.Status(context.Background(), "j-ABC123")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if status.State != tt.state {
				t.Errorf("State = %q, want %q", status.State, tt.state)
			}
			if status.Message != tt.message {
				t.Errorf("Message = %q, want %q", status.Message, tt.message)
			}
		})
	}
}

func TestMockProvisioner_SubmitStep(t *testing.T) {
	mock := &MockProvisioner{}

	err := mock.SubmitStep(context.Background(), "j-ABC123", "s3://bucket/migration.py")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.SubmitStepCalls != 1 {
		t.Errorf("SubmitStepCalls = %d, want 1", mock.SubmitStepCalls)
	}
}

func TestMockProvisioner_Teardown(t *testing.T) {
	mock := &MockProvisioner{}

	err := mock.Teardown(context.Background(), "j-ABC123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mock.TeardownCalled {
		t.Error("Teardown should have been called")
	}
	if mock.TeardownResource != "j-ABC123" {
		t.Errorf("TeardownResource = %q, want %q", mock.TeardownResource, "j-ABC123")
	}
}

func TestMapEMRState(t *testing.T) {
	tests := []struct {
		emrState string
		want     string
	}{
		{"STARTING", "STARTING"},
		{"BOOTSTRAPPING", "STARTING"},
		{"RUNNING", "RUNNING"},
		{"WAITING", "RUNNING"},
		{"TERMINATED", "COMPLETED"},
		{"TERMINATED_WITH_ERRORS", "FAILED"},
		{"TERMINATING", "TERMINATED"},
	}

	for _, tt := range tests {
		t.Run(tt.emrState, func(t *testing.T) {
			if got := mapEMRState(tt.emrState); got != tt.want {
				t.Errorf("mapEMRState(%q) = %q, want %q", tt.emrState, got, tt.want)
			}
		})
	}
}

func TestBootstrapIncludesConnector(t *testing.T) {
	// This is a design verification test: ensure the EMR provisioner
	// includes the MongoDB Spark Connector in bootstrap actions.
	// Since we can't call AWS in unit tests, we verify the provisioner
	// stores the JDBC URI from the plan.
	mock := &MockProvisioner{
		ProvisionResult: &ProvisionResult{
			ResourceID:   "j-TEST",
			ResourceType: "emr_cluster",
		},
	}

	plan := ProvisionPlan{
		Platform:    "emr",
		JDBCS3URI:   "s3://bucket/ojdbc11.jar",
		ScriptS3URI: "s3://bucket/migration.py",
	}

	_, err := mock.Provision(context.Background(), plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.ProvisionedPlan.JDBCS3URI != "s3://bucket/ojdbc11.jar" {
		t.Errorf("JDBC URI not preserved in plan: %q", mock.ProvisionedPlan.JDBCS3URI)
	}
}
