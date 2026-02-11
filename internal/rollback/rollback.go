package rollback

import (
	"context"
	"fmt"

	"github.com/reloquent/reloquent/internal/aws"
	"github.com/reloquent/reloquent/internal/state"
	"github.com/reloquent/reloquent/internal/target"
)

// Rollback orchestrates cleanup of a migration.
type Rollback struct {
	target      target.Operator
	awsClient   aws.Client
	provisioner aws.Provisioner
	state       *state.State
}

// Options controls what gets rolled back.
type Options struct {
	Collections []string // empty = all from mapping
	SkipAWS     bool
	SkipMongoDB bool
}

// Result holds the outcome of a rollback.
type Result struct {
	DroppedCollections []string `yaml:"dropped_collections"`
	S3ArtifactsRemoved bool     `yaml:"s3_artifacts_removed"`
	InfraTerminated    bool     `yaml:"infra_terminated"`
	LockReleased       bool     `yaml:"lock_released"`
	Errors             []string `yaml:"errors,omitempty"`
}

// New creates a new Rollback orchestrator.
func New(tgt target.Operator, client aws.Client, prov aws.Provisioner, st *state.State) *Rollback {
	return &Rollback{
		target:      tgt,
		awsClient:   client,
		provisioner: prov,
		state:       st,
	}
}

// Execute performs the rollback. Each step continues even if a prior step fails.
func (r *Rollback) Execute(ctx context.Context, opts Options) (*Result, error) {
	result := &Result{}

	// Step 1: Drop MongoDB collections
	if !opts.SkipMongoDB && r.target != nil {
		collections := opts.Collections
		if len(collections) == 0 {
			collections = r.collectionsFromState()
		}

		if len(collections) > 0 {
			if err := r.target.DropCollections(ctx, collections); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("dropping collections: %v", err))
			} else {
				result.DroppedCollections = collections
			}
		}
	}

	// Step 2: Remove S3 artifacts
	if !opts.SkipAWS && r.awsClient != nil && r.state.S3ArtifactPrefix != "" {
		// Parse bucket and prefix from the S3 prefix path
		bucket, prefix := parseS3Path(r.state.S3ArtifactPrefix)
		if bucket != "" && prefix != "" {
			if err := r.awsClient.DeleteS3Prefix(ctx, bucket, prefix); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("removing S3 artifacts: %v", err))
			} else {
				result.S3ArtifactsRemoved = true
			}
		}
	}

	// Step 3: Terminate EMR/Glue infrastructure
	if !opts.SkipAWS && r.provisioner != nil && r.state.AWSResourceID != "" {
		if err := r.provisioner.Teardown(ctx, r.state.AWSResourceID); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("terminating infrastructure: %v", err))
		} else {
			result.InfraTerminated = true
		}
	}

	// Step 4: Release lock (reset state)
	r.state.AWSResourceID = ""
	r.state.AWSResourceType = ""
	r.state.MigrationStatus = ""
	r.state.S3ArtifactPrefix = ""
	result.LockReleased = true

	return result, nil
}

func (r *Rollback) collectionsFromState() []string {
	// Collections would come from the mapping; for now return selected tables
	// since collection names match table names by default
	return r.state.SelectedTables
}

// parseS3Path splits "s3://bucket/prefix" into bucket and prefix.
func parseS3Path(s3Path string) (string, string) {
	if len(s3Path) < 6 || s3Path[:5] != "s3://" {
		return "", ""
	}
	rest := s3Path[5:]
	for i, c := range rest {
		if c == '/' {
			return rest[:i], rest[i+1:]
		}
	}
	return rest, ""
}
