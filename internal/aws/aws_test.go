package aws

import (
	"context"
	"errors"
	"testing"
)

func TestCheckPlatformAccess_BothAvailable(t *testing.T) {
	mock := NewMockClient()
	mock.EMRAccess = true
	mock.GlueAccess = true

	access, err := CheckPlatformAccess(context.Background(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !access.EMRAvailable {
		t.Error("EMR should be available")
	}
	if !access.GlueAvailable {
		t.Error("Glue should be available")
	}
	if access.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestCheckPlatformAccess_OnlyEMR(t *testing.T) {
	mock := NewMockClient()
	mock.EMRAccess = true
	mock.GlueAccess = false

	access, err := CheckPlatformAccess(context.Background(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !access.EMRAvailable {
		t.Error("EMR should be available")
	}
	if access.GlueAvailable {
		t.Error("Glue should not be available")
	}
}

func TestCheckPlatformAccess_OnlyGlue(t *testing.T) {
	mock := NewMockClient()
	mock.EMRAccess = false
	mock.GlueAccess = true

	access, err := CheckPlatformAccess(context.Background(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if access.EMRAvailable {
		t.Error("EMR should not be available")
	}
	if !access.GlueAvailable {
		t.Error("Glue should be available")
	}
}

func TestCheckPlatformAccess_NeitherAvailable(t *testing.T) {
	mock := NewMockClient()
	mock.EMRAccess = false
	mock.GlueAccess = false

	access, err := CheckPlatformAccess(context.Background(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if access.EMRAvailable || access.GlueAvailable {
		t.Error("neither should be available")
	}
}

func TestCheckPlatformAccess_BothFail(t *testing.T) {
	mock := NewMockClient()
	mock.EMRErr = errors.New("access denied")
	mock.GlueErr = errors.New("access denied")

	_, err := CheckPlatformAccess(context.Background(), mock)
	if err == nil {
		t.Error("expected error when both checks fail")
	}
}

func TestArtifactUpload_S3URIs(t *testing.T) {
	mock := NewMockClient()
	uploader := NewArtifactUploader(mock, "my-bucket", "reloquent/run-123")

	artifacts := ArtifactSet{
		MigrationScript: []byte("# pyspark script"),
		ConfigYAML:      []byte("version: 1"),
	}

	result, err := uploader.UploadArtifacts(context.Background(), artifacts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ScriptS3URI != "s3://my-bucket/reloquent/run-123/migration.py" {
		t.Errorf("script URI = %q", result.ScriptS3URI)
	}
	if result.ConfigS3URI != "s3://my-bucket/reloquent/run-123/config.yaml" {
		t.Errorf("config URI = %q", result.ConfigS3URI)
	}
	if result.JDBCS3URI != "" {
		t.Errorf("JDBC URI should be empty, got %q", result.JDBCS3URI)
	}

	// Verify mock tracked uploads
	if len(mock.UploadedObjects) != 2 {
		t.Errorf("expected 2 uploads, got %d", len(mock.UploadedObjects))
	}
}

func TestArtifactUpload_WithJDBC(t *testing.T) {
	mock := NewMockClient()
	uploader := NewArtifactUploader(mock, "bucket", "prefix")

	artifacts := ArtifactSet{
		MigrationScript: []byte("script"),
		ConfigYAML:      []byte("config"),
		OracleJDBCPath:  "/path/to/ojdbc11.jar",
	}

	result, err := uploader.UploadArtifacts(context.Background(), artifacts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.JDBCS3URI == "" {
		t.Error("JDBC URI should not be empty when path provided")
	}
	if len(mock.UploadedFiles) != 1 {
		t.Errorf("expected 1 file upload, got %d", len(mock.UploadedFiles))
	}
}

func TestDeleteS3Prefix(t *testing.T) {
	mock := NewMockClient()
	err := mock.DeleteS3Prefix(context.Background(), "my-bucket", "reloquent/run-123/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.DeletedPrefixes) != 1 {
		t.Errorf("expected 1 deleted prefix, got %d", len(mock.DeletedPrefixes))
	}
}
