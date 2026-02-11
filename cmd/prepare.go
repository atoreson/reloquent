package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/reloquent/reloquent/internal/sizing"
	"github.com/reloquent/reloquent/internal/state"
	"github.com/reloquent/reloquent/internal/target"
)

var (
	prepareDryRun    bool
	prepareSkipShard bool
)

var prepareCmd = &cobra.Command{
	Use:   "prepare",
	Short: "Prepare the MongoDB target for migration",
	Long:  `Validate the target cluster, create collections, configure sharding, and disable the balancer.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := state.Load("")
		if err != nil {
			return fmt.Errorf("loading state: %w", err)
		}

		if st.TargetConfig == nil {
			return fmt.Errorf("no target configuration; run the wizard first")
		}

		// Load sizing plan if available
		var plan *sizing.SizingPlan
		if st.SizingPlanPath != "" {
			plan, err = sizing.LoadYAML(st.SizingPlanPath)
			if err != nil {
				fmt.Printf("Warning: could not load sizing plan: %v\n", err)
			}
		}

		// Collection names
		collections := st.SelectedTables
		if len(collections) == 0 {
			return fmt.Errorf("no tables selected; run table selection first")
		}

		if prepareDryRun {
			fmt.Println("Dry run — showing what would be prepared:")
			fmt.Printf("  Target: %s / %s\n", st.TargetConfig.ConnectionString, st.TargetConfig.Database)
			fmt.Printf("  Collections to create: %v\n", collections)
			if plan != nil && plan.ShardPlan != nil && plan.ShardPlan.Recommended && !prepareSkipShard {
				fmt.Printf("  Sharding: %d shards\n", plan.ShardPlan.ShardCount)
				for _, col := range plan.ShardPlan.Collections {
					fmt.Printf("    %s: %s\n", col.CollectionName, col.ShardKeyString())
				}
			}
			return nil
		}

		// Connect to MongoDB
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		op, err := target.NewMongoOperator(ctx, st.TargetConfig.ConnectionString, st.TargetConfig.Database)
		if err != nil {
			return fmt.Errorf("connecting to MongoDB: %w", err)
		}
		defer op.Close(ctx)

		// Detect topology
		topo, err := op.DetectTopology(ctx)
		if err != nil {
			return fmt.Errorf("detecting topology: %w", err)
		}
		fmt.Printf("Topology: %s (version %s)\n", topo.Type, topo.ServerVersion)

		// Validate
		if plan != nil {
			result, err := op.Validate(ctx, plan)
			if err != nil {
				return fmt.Errorf("validating target: %w", err)
			}
			if !result.Passed {
				for _, e := range result.Errors {
					fmt.Printf("  ERROR: [%s] %s\n    %s\n", e.Category, e.Message, e.Suggestion)
				}
				return fmt.Errorf("target validation failed")
			}
			for _, w := range result.Warnings {
				fmt.Printf("  WARNING: [%s] %s\n    %s\n", w.Category, w.Message, w.Suggestion)
			}
		}

		// Create collections
		fmt.Printf("Creating %d collections...\n", len(collections))
		if err := op.CreateCollections(ctx, collections); err != nil {
			return fmt.Errorf("creating collections: %w", err)
		}

		// Setup sharding
		if plan != nil && plan.ShardPlan != nil && plan.ShardPlan.Recommended && !prepareSkipShard {
			fmt.Println("Setting up sharding...")
			if err := op.SetupSharding(ctx, plan.ShardPlan); err != nil {
				return fmt.Errorf("setting up sharding: %w", err)
			}
			fmt.Println("Disabling balancer...")
			if err := op.DisableBalancer(ctx); err != nil {
				fmt.Printf("Warning: could not disable balancer: %v\n", err)
			}
		}

		st.CompleteStep(state.StepPreMigration, state.StepReview)
		_ = st.Save("")

		fmt.Println("Target preparation complete.")
		return nil
	},
}

func init() {
	prepareCmd.Flags().BoolVar(&prepareDryRun, "dry-run", false, "show what would be done without making changes")
	prepareCmd.Flags().BoolVar(&prepareSkipShard, "skip-shard", false, "skip sharding setup even if recommended")
	rootCmd.AddCommand(prepareCmd)
}
