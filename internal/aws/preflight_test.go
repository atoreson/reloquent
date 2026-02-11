package aws

import (
	"context"
	"errors"
	"testing"
)

func TestRunPreflight_AllPass(t *testing.T) {
	mock := &MockProvisioner{
		StatusResult: &ProvisionStatus{State: "RUNNING"},
	}

	result, err := RunPreflight(context.Background(), mock, "j-ABC", "jdbc:postgresql://host/db", "mongodb://host/db")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.SourceDBReachable {
		t.Error("source should be reachable")
	}
	if !result.MongoDBReachable {
		t.Error("MongoDB should be reachable")
	}
	if !result.ConnectorAvailable {
		t.Error("connector should be available")
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %v", result.Errors)
	}
}

func TestRunPreflight_SourceUnreachable(t *testing.T) {
	mock := &MockProvisioner{
		StatusResult: &ProvisionStatus{State: "RUNNING"},
	}

	result, err := RunPreflight(context.Background(), mock, "j-ABC", "", "mongodb://host/db")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SourceDBReachable {
		t.Error("source should not be reachable with empty JDBC")
	}
	if len(result.Errors) == 0 {
		t.Error("expected error for empty source JDBC")
	}
}

func TestRunPreflight_MongoDBUnreachable(t *testing.T) {
	mock := &MockProvisioner{
		StatusResult: &ProvisionStatus{State: "RUNNING"},
	}

	result, err := RunPreflight(context.Background(), mock, "j-ABC", "jdbc:postgresql://host/db", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.MongoDBReachable {
		t.Error("MongoDB should not be reachable with empty URI")
	}
	if len(result.Errors) == 0 {
		t.Error("expected error for empty MongoDB URI")
	}
}

func TestRunPreflight_InfraNotRunning(t *testing.T) {
	mock := &MockProvisioner{
		StatusResult: &ProvisionStatus{State: "STARTING", Message: "Cluster is bootstrapping"},
	}

	result, err := RunPreflight(context.Background(), mock, "j-ABC", "jdbc:pg://host/db", "mongodb://host/db")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Errors) == 0 {
		t.Error("expected error when infra is not running")
	}
}

func TestRunPreflight_StatusError(t *testing.T) {
	mock := &MockProvisioner{
		StatusErr: errors.New("connection timeout"),
	}

	_, err := RunPreflight(context.Background(), mock, "j-ABC", "jdbc:pg://host/db", "mongodb://host/db")
	if err == nil {
		t.Error("expected error when status check fails")
	}
}
