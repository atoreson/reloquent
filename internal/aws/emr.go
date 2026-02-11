package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/emr"
	"github.com/aws/aws-sdk-go-v2/service/emr/types"
)

// EMRProvisioner implements Provisioner for Amazon EMR.
type EMRProvisioner struct {
	client *emr.Client
}

// NewEMRProvisioner creates a new EMR provisioner.
func NewEMRProvisioner(ctx context.Context, profile, region string) (*EMRProvisioner, error) {
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

	return &EMRProvisioner{
		client: emr.NewFromConfig(cfg),
	}, nil
}

// Provision creates an EMR cluster for the migration.
func (p *EMRProvisioner) Provision(ctx context.Context, plan ProvisionPlan) (*ProvisionResult, error) {
	// Build tags
	var tags []types.Tag
	for k, v := range plan.Tags {
		tags = append(tags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	tags = append(tags, types.Tag{
		Key:   aws.String("reloquent"),
		Value: aws.String("migration"),
	})

	// Bootstrap action to install MongoDB Spark Connector
	bootstrapActions := []types.BootstrapActionConfig{
		{
			Name: aws.String("Install MongoDB Spark Connector"),
			ScriptBootstrapAction: &types.ScriptBootstrapActionConfig{
				Path: aws.String("s3://emr-bootstrap-scripts/install-mongo-spark-connector.sh"),
			},
		},
	}

	// Add JDBC bootstrap if needed
	if plan.JDBCS3URI != "" {
		bootstrapActions = append(bootstrapActions, types.BootstrapActionConfig{
			Name: aws.String("Install Oracle JDBC Driver"),
			ScriptBootstrapAction: &types.ScriptBootstrapActionConfig{
				Path: aws.String("s3://emr-bootstrap-scripts/install-jdbc.sh"),
				Args: []string{plan.JDBCS3URI},
			},
		})
	}

	input := &emr.RunJobFlowInput{
		Name:           aws.String("reloquent-migration"),
		ReleaseLabel:   aws.String("emr-7.0.0"),
		Applications:   []types.Application{{Name: aws.String("Spark")}},
		Tags:           tags,
		BootstrapActions: bootstrapActions,
		Instances: &types.JobFlowInstancesConfig{
			KeepJobFlowAliveWhenNoSteps: aws.Bool(true),
			InstanceGroups: []types.InstanceGroupConfig{
				{
					InstanceRole:  types.InstanceRoleTypeMaster,
					InstanceType:  aws.String(plan.SparkPlan.InstanceType),
					InstanceCount: aws.Int32(1),
				},
				{
					InstanceRole:  types.InstanceRoleTypeCore,
					InstanceType:  aws.String(plan.SparkPlan.InstanceType),
					InstanceCount: aws.Int32(int32(plan.SparkPlan.WorkerCount)),
				},
			},
		},
	}

	out, err := p.client.RunJobFlow(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("creating EMR cluster: %w", err)
	}

	return &ProvisionResult{
		ResourceID:   aws.ToString(out.JobFlowId),
		ResourceType: "emr_cluster",
	}, nil
}

// Status returns the current state of an EMR cluster.
func (p *EMRProvisioner) Status(ctx context.Context, resourceID string) (*ProvisionStatus, error) {
	out, err := p.client.DescribeCluster(ctx, &emr.DescribeClusterInput{
		ClusterId: aws.String(resourceID),
	})
	if err != nil {
		return nil, fmt.Errorf("describing EMR cluster: %w", err)
	}

	state := string(out.Cluster.Status.State)
	message := ""
	if out.Cluster.Status.StateChangeReason != nil {
		message = aws.ToString(out.Cluster.Status.StateChangeReason.Message)
	}

	// Map EMR states to our standard states
	mapped := mapEMRState(state)

	return &ProvisionStatus{
		State:   mapped,
		Message: message,
	}, nil
}

// SubmitStep submits a Spark step to a running EMR cluster.
func (p *EMRProvisioner) SubmitStep(ctx context.Context, resourceID string, scriptS3URI string) error {
	_, err := p.client.AddJobFlowSteps(ctx, &emr.AddJobFlowStepsInput{
		JobFlowId: aws.String(resourceID),
		Steps: []types.StepConfig{
			{
				Name:            aws.String("reloquent-migration"),
				ActionOnFailure: types.ActionOnFailureContinue,
				HadoopJarStep: &types.HadoopJarStepConfig{
					Jar:  aws.String("command-runner.jar"),
					Args: []string{"spark-submit", "--deploy-mode", "cluster", scriptS3URI},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("submitting EMR step: %w", err)
	}
	return nil
}

// Teardown terminates an EMR cluster.
func (p *EMRProvisioner) Teardown(ctx context.Context, resourceID string) error {
	_, err := p.client.TerminateJobFlows(ctx, &emr.TerminateJobFlowsInput{
		JobFlowIds: []string{resourceID},
	})
	if err != nil {
		return fmt.Errorf("terminating EMR cluster: %w", err)
	}
	return nil
}

func mapEMRState(state string) string {
	switch state {
	case "STARTING", "BOOTSTRAPPING":
		return "STARTING"
	case "RUNNING", "WAITING":
		return "RUNNING"
	case "TERMINATED":
		return "COMPLETED"
	case "TERMINATED_WITH_ERRORS":
		return "FAILED"
	case "TERMINATING":
		return "TERMINATED"
	default:
		return state
	}
}
