package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/indexes"
	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/postmigration"
	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/state"
	"github.com/reloquent/reloquent/internal/target"
)

var (
	indexesDryRun  bool
	indexesMonitor bool
)

var indexesCmd = &cobra.Command{
	Use:   "indexes",
	Short: "Build indexes on target collections",
	Long:  `Create indexes on the target MongoDB collections after data insertion completes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := state.Load("")
		if err != nil {
			return fmt.Errorf("loading state: %w", err)
		}

		// Load schema and mapping
		if st.SchemaPath == "" {
			return fmt.Errorf("no schema available; run source discovery first")
		}
		s, err := schema.LoadYAML(st.SchemaPath)
		if err != nil {
			return fmt.Errorf("loading schema: %w", err)
		}

		if st.MappingPath == "" {
			return fmt.Errorf("no mapping available; run denormalization design first")
		}
		m, err := mapping.LoadYAML(st.MappingPath)
		if err != nil {
			return fmt.Errorf("loading mapping: %w", err)
		}

		// Infer indexes
		plan := indexes.Infer(s, m)

		if indexesDryRun {
			fmt.Printf("Index plan: %d indexes\n\n", len(plan.Indexes))
			for _, ci := range plan.Indexes {
				unique := ""
				if ci.Index.Unique {
					unique = " (unique)"
				}
				fields := ""
				for i, k := range ci.Index.Keys {
					if i > 0 {
						fields += ", "
					}
					dir := "asc"
					if k.Order == -1 {
						dir = "desc"
					}
					fields += fmt.Sprintf("%s:%s", k.Field, dir)
				}
				fmt.Printf("  %s.%s: {%s}%s\n", ci.Collection, ci.Index.Name, fields, unique)
			}
			fmt.Println()
			for _, e := range plan.Explanations {
				fmt.Printf("  - %s\n", e)
			}
			return nil
		}

		if indexesMonitor {
			// Just monitor existing index builds
			tgtOp, err := target.NewMongoOperator(context.Background(),
				st.TargetConfig.ConnectionString, st.TargetConfig.Database)
			if err != nil {
				return fmt.Errorf("connecting to target: %w", err)
			}
			defer tgtOp.Close(context.Background())

			fmt.Println("Monitoring index build progress...")
			for {
				statuses, err := tgtOp.ListIndexBuildProgress(context.Background())
				if err != nil {
					return fmt.Errorf("checking progress: %w", err)
				}
				if len(statuses) == 0 {
					fmt.Println("No active index builds.")
					return nil
				}
				for _, s := range statuses {
					fmt.Printf("  %s.%s: %s (%.0f%%)\n", s.Collection, s.IndexName, s.Phase, s.Progress)
				}
				time.Sleep(5 * time.Second)
			}
		}

		// Default: create indexes + run post-ops + generate report
		tgtOp, err := target.NewMongoOperator(context.Background(),
			st.TargetConfig.ConnectionString, st.TargetConfig.Database)
		if err != nil {
			return fmt.Errorf("connecting to target: %w", err)
		}
		defer tgtOp.Close(context.Background())

		topo, err := tgtOp.DetectTopology(context.Background())
		if err != nil {
			fmt.Printf("Warning: could not detect topology: %v\n", err)
		}

		orch := &postmigration.Orchestrator{
			Target:    tgtOp,
			Schema:    s,
			Mapping:   m,
			State:     st,
			StatePath: config.ExpandHome(state.DefaultPath),
			IndexPlan: plan,
			Topology:  topo,
		}

		fmt.Printf("Building %d indexes...\n", len(plan.Indexes))
		cb := postmigration.Callbacks{
			OnStepComplete: func(step string) {
				fmt.Printf("  Step complete: %s\n", step)
			},
		}

		if err := orch.RunIndexBuilds(context.Background(), cb); err != nil {
			return fmt.Errorf("building indexes: %w", err)
		}
		fmt.Println("Indexes built successfully.")

		// Post-ops
		if err := orch.RunPostOps(context.Background()); err != nil {
			return fmt.Errorf("post-ops: %w", err)
		}
		fmt.Println("Post-migration operations complete.")

		// Generate report
		rpt, err := orch.CheckReadiness(context.Background())
		if err != nil {
			return fmt.Errorf("generating report: %w", err)
		}

		if rpt.ProductionReady {
			fmt.Println("\nREADY FOR PRODUCTION")
		} else {
			fmt.Println("\nREQUIRES ATTENTION")
		}

		if st.ReportPath != "" {
			fmt.Printf("Report: %s\n", st.ReportPath)
		}

		return nil
	},
}

func init() {
	indexesCmd.Flags().BoolVar(&indexesDryRun, "dry-run", false, "show indexes without creating them")
	indexesCmd.Flags().BoolVar(&indexesMonitor, "monitor", false, "watch index build progress")
	rootCmd.AddCommand(indexesCmd)
}
