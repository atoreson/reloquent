package aws

import "context"

// Client defines AWS operations needed by the migration tool.
type Client interface {
	VerifyCredentials(ctx context.Context) (*CallerIdentity, error)
	CheckEMRAccess(ctx context.Context) (bool, error)
	CheckGlueAccess(ctx context.Context) (bool, error)
	UploadToS3(ctx context.Context, bucket, key string, data []byte) error
	UploadFileToS3(ctx context.Context, bucket, key, localPath string) error
	DeleteS3Prefix(ctx context.Context, bucket, prefix string) error
}

// CallerIdentity holds AWS STS caller identity information.
type CallerIdentity struct {
	Account string
	ARN     string
	UserID  string
}

// PlatformAccess describes which Spark platforms are available.
type PlatformAccess struct {
	EMRAvailable  bool
	GlueAvailable bool
	Message       string
}

// CheckPlatformAccess determines which Spark platforms the caller can use.
func CheckPlatformAccess(ctx context.Context, client Client) (*PlatformAccess, error) {
	emr, emrErr := client.CheckEMRAccess(ctx)
	glue, glueErr := client.CheckGlueAccess(ctx)

	// If both checks fail, return an error
	if emrErr != nil && glueErr != nil {
		return nil, emrErr
	}

	access := &PlatformAccess{
		EMRAvailable:  emr,
		GlueAvailable: glue,
	}

	switch {
	case emr && glue:
		access.Message = "Both EMR and Glue are available."
	case emr:
		access.Message = "EMR is available. Glue is not accessible."
	case glue:
		access.Message = "Glue is available. EMR is not accessible."
	default:
		access.Message = "Neither EMR nor Glue is accessible. Check IAM permissions."
	}

	return access, nil
}
