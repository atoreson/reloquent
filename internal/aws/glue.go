package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	gluetypes "github.com/aws/aws-sdk-go-v2/service/glue/types"
)

// GlueProvisioner implements Provisioner for AWS Glue.
type GlueProvisioner struct {
	client *glue.Client
}

// NewGlueProvisioner creates a new Glue provisioner.
func NewGlueProvisioner(ctx context.Context, profile, region string) (*GlueProvisioner, error) {
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

	return &GlueProvisioner{
		client: glue.NewFromConfig(cfg),
	}, nil
}

// Provision creates a Glue job for the migration.
func (p *GlueProvisioner) Provision(ctx context.Context, plan ProvisionPlan) (*ProvisionResult, error) {
	jobName := "reloquent-migration"

	// Build tags
	tags := make(map[string]string)
	for k, v := range plan.Tags {
		tags[k] = v
	}
	tags["reloquent"] = "migration"

	// Build default arguments
	defaultArgs := map[string]string{
		"--conf":       fmt.Sprintf("spark.mongodb.output.uri=%s", plan.ConfigS3URI),
		"--extra-jars": plan.JDBCS3URI,
	}

	// Create the Glue job
	_, err := p.client.CreateJob(ctx, &glue.CreateJobInput{
		Name:    aws.String(jobName),
		Role:    aws.String("AWSGlueServiceRole"),
		Tags:    tags,
		Command: &gluetypes.JobCommand{
			Name:           aws.String("glueetl"),
			ScriptLocation: aws.String(plan.ScriptS3URI),
			PythonVersion:  aws.String("3"),
		},
		GlueVersion:      aws.String("4.0"),
		NumberOfWorkers:   aws.Int32(int32(plan.SparkPlan.DPUCount)),
		WorkerType:        gluetypes.WorkerTypeG2x,
		DefaultArguments:  defaultArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("creating Glue job: %w", err)
	}

	// Start the job run
	runOut, err := p.client.StartJobRun(ctx, &glue.StartJobRunInput{
		JobName: aws.String(jobName),
	})
	if err != nil {
		return nil, fmt.Errorf("starting Glue job run: %w", err)
	}

	return &ProvisionResult{
		ResourceID:   aws.ToString(runOut.JobRunId),
		ResourceType: "glue_job",
	}, nil
}

// Status returns the current state of a Glue job run.
func (p *GlueProvisioner) Status(ctx context.Context, resourceID string) (*ProvisionStatus, error) {
	out, err := p.client.GetJobRun(ctx, &glue.GetJobRunInput{
		JobName: aws.String("reloquent-migration"),
		RunId:   aws.String(resourceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting Glue job run status: %w", err)
	}

	state := string(out.JobRun.JobRunState)
	message := aws.ToString(out.JobRun.ErrorMessage)

	mapped := mapGlueState(state)

	return &ProvisionStatus{
		State:   mapped,
		Message: message,
	}, nil
}

// SubmitStep is a no-op for Glue since the job runs immediately.
func (p *GlueProvisioner) SubmitStep(_ context.Context, _ string, _ string) error {
	return nil // Glue jobs run immediately upon creation
}

// Teardown deletes the Glue job.
func (p *GlueProvisioner) Teardown(ctx context.Context, _ string) error {
	_, err := p.client.DeleteJob(ctx, &glue.DeleteJobInput{
		JobName: aws.String("reloquent-migration"),
	})
	if err != nil {
		return fmt.Errorf("deleting Glue job: %w", err)
	}
	return nil
}

func mapGlueState(state string) string {
	switch state {
	case "STARTING":
		return "STARTING"
	case "RUNNING":
		return "RUNNING"
	case "SUCCEEDED":
		return "COMPLETED"
	case "FAILED", "ERROR":
		return "FAILED"
	case "STOPPED", "STOPPING":
		return "TERMINATED"
	default:
		return state
	}
}
