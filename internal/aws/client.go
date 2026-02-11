package aws

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// RealClient implements Client using the AWS SDK v2.
type RealClient struct {
	cfg       aws.Config
	stsClient *sts.Client
	iamClient *iam.Client
	s3Client  *s3.Client
}

// NewRealClient creates a new AWS client with the given profile and region.
func NewRealClient(ctx context.Context, profile, region string) (*RealClient, error) {
	var opts []func(*awsconfig.LoadOptions) error

	if profile != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(profile))
	}
	if region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	return &RealClient{
		cfg:       cfg,
		stsClient: sts.NewFromConfig(cfg),
		iamClient: iam.NewFromConfig(cfg),
		s3Client:  s3.NewFromConfig(cfg),
	}, nil
}

// VerifyCredentials checks the current AWS credentials using STS.
func (c *RealClient) VerifyCredentials(ctx context.Context) (*CallerIdentity, error) {
	out, err := c.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("getting caller identity: %w", err)
	}

	return &CallerIdentity{
		Account: aws.ToString(out.Account),
		ARN:     aws.ToString(out.Arn),
		UserID:  aws.ToString(out.UserId),
	}, nil
}

// CheckEMRAccess checks if the caller has permission to run EMR jobs.
func (c *RealClient) CheckEMRAccess(ctx context.Context) (bool, error) {
	return c.simulatePolicy(ctx, "elasticmapreduce:RunJobFlow", "arn:aws:elasticmapreduce:*:*:cluster/*")
}

// CheckGlueAccess checks if the caller has permission to create Glue jobs.
func (c *RealClient) CheckGlueAccess(ctx context.Context) (bool, error) {
	return c.simulatePolicy(ctx, "glue:CreateJob", "arn:aws:glue:*:*:job/*")
}

func (c *RealClient) simulatePolicy(ctx context.Context, action, resource string) (bool, error) {
	// First get the caller's ARN
	identity, err := c.VerifyCredentials(ctx)
	if err != nil {
		return false, err
	}

	out, err := c.iamClient.SimulatePrincipalPolicy(ctx, &iam.SimulatePrincipalPolicyInput{
		PolicySourceArn: aws.String(identity.ARN),
		ActionNames:     []string{action},
		ResourceArns:    []string{resource},
	})
	if err != nil {
		// If we can't simulate, assume access is not available
		return false, nil
	}

	for _, result := range out.EvaluationResults {
		if result.EvalDecision == "allowed" {
			return true, nil
		}
	}
	return false, nil
}

// UploadToS3 uploads data bytes to an S3 bucket.
func (c *RealClient) UploadToS3(ctx context.Context, bucket, key string, data []byte) error {
	_, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("uploading to s3://%s/%s: %w", bucket, key, err)
	}
	return nil
}

// UploadFileToS3 uploads a local file to an S3 bucket.
func (c *RealClient) UploadFileToS3(ctx context.Context, bucket, key, localPath string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("opening file %s: %w", localPath, err)
	}
	defer f.Close()

	_, err = c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   f,
	})
	if err != nil {
		return fmt.Errorf("uploading file to s3://%s/%s: %w", bucket, key, err)
	}
	return nil
}

// DeleteS3Prefix deletes all objects under a given prefix in an S3 bucket.
func (c *RealClient) DeleteS3Prefix(ctx context.Context, bucket, prefix string) error {
	paginator := s3.NewListObjectsV2Paginator(c.s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("listing objects under s3://%s/%s: %w", bucket, prefix, err)
		}

		if len(page.Contents) == 0 {
			continue
		}

		objects := make([]s3types.ObjectIdentifier, len(page.Contents))
		for i, obj := range page.Contents {
			objects[i] = s3types.ObjectIdentifier{Key: obj.Key}
		}

		_, err = c.s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &s3types.Delete{Objects: objects},
		})
		if err != nil {
			return fmt.Errorf("deleting objects under s3://%s/%s: %w", bucket, prefix, err)
		}
	}

	return nil
}
