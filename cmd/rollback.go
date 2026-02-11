package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	awspkg "github.com/reloquent/reloquent/internal/aws"
	"github.com/reloquent/reloquent/internal/rollback"
	"github.com/reloquent/reloquent/internal/state"
	"github.com/reloquent/reloquent/internal/target"
)

var (
	rollbackCollections []string
	rollbackConfirm     bool
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Drop target collections and clean up",
	Long:  `Drop migrated collections from the target MongoDB cluster and release the lock file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !rollbackConfirm {
			fmt.Println("Rollback requires --confirm to proceed.")
			fmt.Println("This will DROP collections from the target MongoDB cluster.")
			return nil
		}

		st, err := state.Load("")
		if err != nil {
			return fmt.Errorf("loading state: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		// Connect to MongoDB if target config is available
		var tgt target.Operator
		if st.TargetConfig != nil {
			op, err := target.NewMongoOperator(ctx, st.TargetConfig.ConnectionString, st.TargetConfig.Database)
			if err != nil {
				fmt.Printf("Warning: could not connect to MongoDB: %v\n", err)
			} else {
				tgt = op
				defer op.Close(ctx)
			}
		}

		// Create AWS client if resource exists
		var awsClient awspkg.Client
		var prov awspkg.Provisioner
		if st.AWSResourceID != "" || st.S3ArtifactPrefix != "" {
			client, err := awspkg.NewRealClient(ctx, "", "")
			if err != nil {
				fmt.Printf("Warning: could not create AWS client: %v\n", err)
			} else {
				awsClient = client
			}

			if st.AWSResourceType == "emr_cluster" {
				p, err := awspkg.NewEMRProvisioner(ctx, "", "")
				if err == nil {
					prov = p
				}
			} else if st.AWSResourceType == "glue_job" {
				p, err := awspkg.NewGlueProvisioner(ctx, "", "")
				if err == nil {
					prov = p
				}
			}
		}

		rb := rollback.New(tgt, awsClient, prov, st)
		opts := rollback.Options{
			Collections: rollbackCollections,
			SkipAWS:     awsClient == nil && prov == nil,
			SkipMongoDB: tgt == nil,
		}

		result, err := rb.Execute(ctx, opts)
		if err != nil {
			return fmt.Errorf("rollback: %w", err)
		}

		// Report results
		if len(result.DroppedCollections) > 0 {
			fmt.Printf("Dropped collections: %v\n", result.DroppedCollections)
		}
		if result.S3ArtifactsRemoved {
			fmt.Println("S3 artifacts removed.")
		}
		if result.InfraTerminated {
			fmt.Println("Infrastructure terminated.")
		}
		if result.LockReleased {
			fmt.Println("Lock released.")
		}
		if len(result.Errors) > 0 {
			fmt.Println("Errors during rollback:")
			for _, e := range result.Errors {
				fmt.Printf("  - %s\n", e)
			}
		}

		_ = st.Save("")
		return nil
	},
}

func init() {
	rollbackCmd.Flags().StringSliceVar(&rollbackCollections, "collections", nil, "specific collections to roll back")
	rollbackCmd.Flags().BoolVar(&rollbackConfirm, "confirm", false, "skip confirmation prompt")
	rootCmd.AddCommand(rollbackCmd)
}
