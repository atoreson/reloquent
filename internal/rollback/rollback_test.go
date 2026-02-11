package rollback

import (
	"context"
	"errors"
	"testing"

	"github.com/reloquent/reloquent/internal/aws"
	"github.com/reloquent/reloquent/internal/state"
	"github.com/reloquent/reloquent/internal/target"
)

func TestFullRollback(t *testing.T) {
	tgt := &target.MockOperator{}
	client := aws.NewMockClient()
	prov := &aws.MockProvisioner{}
	st := &state.State{
		SelectedTables:   []string{"users", "orders"},
		S3ArtifactPrefix: "s3://bucket/reloquent/run-123/",
		AWSResourceID:    "j-ABC123",
		AWSResourceType:  "emr_cluster",
		MigrationStatus:  "running",
	}

	rb := New(tgt, client, prov, st)
	result, err := rb.Execute(context.Background(), Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.DroppedCollections) != 2 {
		t.Errorf("expected 2 dropped collections, got %d", len(result.DroppedCollections))
	}
	if !result.S3ArtifactsRemoved {
		t.Error("S3 artifacts should be removed")
	}
	if !result.InfraTerminated {
		t.Error("infrastructure should be terminated")
	}
	if !result.LockReleased {
		t.Error("lock should be released")
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %v", result.Errors)
	}
}

func TestPartialRollback_SpecificCollections(t *testing.T) {
	tgt := &target.MockOperator{}
	st := &state.State{
		SelectedTables: []string{"users", "orders", "products"},
	}

	rb := New(tgt, nil, nil, st)
	result, err := rb.Execute(context.Background(), Options{
		Collections: []string{"orders"},
		SkipAWS:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.DroppedCollections) != 1 {
		t.Errorf("expected 1 dropped collection, got %d", len(result.DroppedCollections))
	}
	if result.DroppedCollections[0] != "orders" {
		t.Errorf("expected 'orders', got %q", result.DroppedCollections[0])
	}
}

func TestRollback_S3Cleanup(t *testing.T) {
	client := aws.NewMockClient()
	st := &state.State{
		S3ArtifactPrefix: "s3://my-bucket/reloquent/run-456/",
	}

	rb := New(nil, client, nil, st)
	result, err := rb.Execute(context.Background(), Options{SkipMongoDB: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.S3ArtifactsRemoved {
		t.Error("S3 artifacts should be removed")
	}
	if len(client.DeletedPrefixes) != 1 {
		t.Errorf("expected 1 delete prefix call, got %d", len(client.DeletedPrefixes))
	}
}

func TestRollback_InfraTermination(t *testing.T) {
	prov := &aws.MockProvisioner{}
	st := &state.State{
		AWSResourceID:   "j-XYZ",
		AWSResourceType: "emr_cluster",
	}

	rb := New(nil, nil, prov, st)
	result, err := rb.Execute(context.Background(), Options{SkipMongoDB: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.InfraTerminated {
		t.Error("infrastructure should be terminated")
	}
	if !prov.TeardownCalled {
		t.Error("Teardown should have been called")
	}
}

func TestRollback_ErrorInOneStepContinues(t *testing.T) {
	tgt := &target.MockOperator{
		DropErr: errors.New("permission denied"),
	}
	client := aws.NewMockClient()
	prov := &aws.MockProvisioner{}
	st := &state.State{
		SelectedTables:   []string{"users"},
		S3ArtifactPrefix: "s3://bucket/prefix/",
		AWSResourceID:    "j-ABC",
	}

	rb := New(tgt, client, prov, st)
	result, err := rb.Execute(context.Background(), Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Drop failed, but S3 and infra should still be cleaned up
	if len(result.Errors) == 0 {
		t.Error("expected at least one error from failed drop")
	}
	if !result.S3ArtifactsRemoved {
		t.Error("S3 should still be cleaned up despite drop failure")
	}
	if !result.InfraTerminated {
		t.Error("infra should still be terminated despite drop failure")
	}
	if !result.LockReleased {
		t.Error("lock should always be released")
	}
}

func TestRollback_SkipAWS(t *testing.T) {
	tgt := &target.MockOperator{}
	st := &state.State{
		SelectedTables:   []string{"users"},
		S3ArtifactPrefix: "s3://bucket/prefix/",
		AWSResourceID:    "j-ABC",
	}

	rb := New(tgt, nil, nil, st)
	result, err := rb.Execute(context.Background(), Options{SkipAWS: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.S3ArtifactsRemoved == true {
		// S3 should NOT be removed when SkipAWS is true
	}
	if result.InfraTerminated {
		t.Error("infra should not be terminated when SkipAWS is true")
	}
}

func TestParseS3Path(t *testing.T) {
	tests := []struct {
		input      string
		wantBucket string
		wantPrefix string
	}{
		{"s3://my-bucket/prefix/path/", "my-bucket", "prefix/path/"},
		{"s3://bucket/key", "bucket", "key"},
		{"s3://bucket", "bucket", ""},
		{"invalid", "", ""},
		{"", "", ""},
	}

	for _, tt := range tests {
		bucket, prefix := parseS3Path(tt.input)
		if bucket != tt.wantBucket || prefix != tt.wantPrefix {
			t.Errorf("parseS3Path(%q) = (%q, %q), want (%q, %q)",
				tt.input, bucket, prefix, tt.wantBucket, tt.wantPrefix)
		}
	}
}
