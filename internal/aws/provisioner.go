package aws

import (
	"context"

	"github.com/reloquent/reloquent/internal/sizing"
)

// Provisioner manages Spark infrastructure lifecycle.
type Provisioner interface {
	Provision(ctx context.Context, plan ProvisionPlan) (*ProvisionResult, error)
	Status(ctx context.Context, resourceID string) (*ProvisionStatus, error)
	SubmitStep(ctx context.Context, resourceID string, scriptS3URI string) error
	Teardown(ctx context.Context, resourceID string) error
}

// ProvisionPlan describes what infrastructure to create.
type ProvisionPlan struct {
	Platform    string           `yaml:"platform"` // "emr" or "glue"
	SparkPlan   sizing.SparkPlan `yaml:"spark_plan"`
	ScriptS3URI string           `yaml:"script_s3_uri"`
	ConfigS3URI string           `yaml:"config_s3_uri"`
	JDBCS3URI   string           `yaml:"jdbc_s3_uri,omitempty"`
	Tags        map[string]string `yaml:"tags,omitempty"`
}

// ProvisionResult holds the created resource identifiers.
type ProvisionResult struct {
	ResourceID   string `yaml:"resource_id"`
	ResourceType string `yaml:"resource_type"` // "emr_cluster" or "glue_job"
}

// ProvisionStatus describes the current state of provisioned infrastructure.
type ProvisionStatus struct {
	State   string `yaml:"state"` // "STARTING", "RUNNING", "COMPLETED", "FAILED", "TERMINATED"
	Message string `yaml:"message"`
}
