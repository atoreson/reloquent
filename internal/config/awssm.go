package config

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// resolveAWSSecretsManager resolves an AWS Secrets Manager reference.
// Format: secret-name
func resolveAWSSecretsManager(ref string) (string, error) {
	ctx := context.Background()
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("loading AWS config: %w", err)
	}

	client := secretsmanager.NewFromConfig(cfg)
	out, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(ref),
	})
	if err != nil {
		return "", fmt.Errorf("getting secret %q: %w", ref, err)
	}

	if out.SecretString == nil {
		return "", fmt.Errorf("secret %q has no string value (binary secrets not supported)", ref)
	}

	return *out.SecretString, nil
}
