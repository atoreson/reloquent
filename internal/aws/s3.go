package aws

import (
	"context"
	"fmt"
	"path"
)

// ArtifactUploader manages uploading migration artifacts to S3.
type ArtifactUploader struct {
	client Client
	bucket string
	prefix string
}

// NewArtifactUploader creates a new artifact uploader.
func NewArtifactUploader(client Client, bucket, prefix string) *ArtifactUploader {
	return &ArtifactUploader{
		client: client,
		bucket: bucket,
		prefix: prefix,
	}
}

// ArtifactSet holds the migration artifacts to upload.
type ArtifactSet struct {
	MigrationScript []byte
	ConfigYAML      []byte
	OracleJDBCPath  string // empty if not Oracle
}

// UploadResult holds the S3 URIs of uploaded artifacts.
type UploadResult struct {
	ScriptS3URI string
	ConfigS3URI string
	JDBCS3URI   string
}

// UploadArtifacts uploads migration artifacts to S3.
func (u *ArtifactUploader) UploadArtifacts(ctx context.Context, artifacts ArtifactSet) (*UploadResult, error) {
	result := &UploadResult{}

	// Upload migration script
	scriptKey := path.Join(u.prefix, "migration.py")
	if err := u.client.UploadToS3(ctx, u.bucket, scriptKey, artifacts.MigrationScript); err != nil {
		return nil, fmt.Errorf("uploading migration script: %w", err)
	}
	result.ScriptS3URI = fmt.Sprintf("s3://%s/%s", u.bucket, scriptKey)

	// Upload config
	configKey := path.Join(u.prefix, "config.yaml")
	if err := u.client.UploadToS3(ctx, u.bucket, configKey, artifacts.ConfigYAML); err != nil {
		return nil, fmt.Errorf("uploading config: %w", err)
	}
	result.ConfigS3URI = fmt.Sprintf("s3://%s/%s", u.bucket, configKey)

	// Upload Oracle JDBC driver if provided
	if artifacts.OracleJDBCPath != "" {
		jdbcKey := path.Join(u.prefix, "ojdbc11.jar")
		if err := u.client.UploadFileToS3(ctx, u.bucket, jdbcKey, artifacts.OracleJDBCPath); err != nil {
			return nil, fmt.Errorf("uploading JDBC driver: %w", err)
		}
		result.JDBCS3URI = fmt.Sprintf("s3://%s/%s", u.bucket, jdbcKey)
	}

	return result, nil
}
