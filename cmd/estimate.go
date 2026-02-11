package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/sizing"
	"github.com/reloquent/reloquent/internal/state"
)

var estimateBenchmark bool

var estimateCmd = &cobra.Command{
	Use:   "estimate",
	Short: "Estimate cluster sizing and migration time",
	Long:  `Calculate recommended Spark cluster size, MongoDB target tier, and estimated migration duration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := state.Load("")
		if err != nil {
			return fmt.Errorf("loading state: %w", err)
		}

		if st.SchemaPath == "" {
			return fmt.Errorf("no schema available; run source discovery first (reloquent or reloquent discover)")
		}

		s, err := schema.LoadYAML(st.SchemaPath)
		if err != nil {
			return fmt.Errorf("loading schema: %w", err)
		}

		// Compute sizing input from schema
		var totalBytes int64
		var totalRows int64
		for _, t := range s.Tables {
			totalBytes += t.SizeBytes
			totalRows += t.RowCount
		}

		collCount := len(st.SelectedTables)
		if collCount == 0 {
			collCount = len(s.Tables)
		}

		input := sizing.Input{
			TotalDataBytes:        totalBytes,
			TotalRowCount:         totalRows,
			DenormExpansionFactor: 1.4,
			CollectionCount:       collCount,
		}
		if st.SourceConfig != nil {
			input.MaxSourceConnections = st.SourceConfig.MaxConnections
		}

		if estimateBenchmark {
			fmt.Println("Running source DB read benchmark...")
			fmt.Println("(Benchmark requires a live database connection — run the wizard for interactive benchmarking)")
		}

		plan := sizing.Calculate(input)

		// Display results
		fmt.Println()
		for _, exp := range plan.Explanations {
			fmt.Printf("  [%s] %s\n", exp.Category, exp.Summary)
			fmt.Printf("    %s\n\n", exp.Detail)
		}

		// Save sizing plan
		stateDir := st.SchemaPath[:len(st.SchemaPath)-len("source-schema.yaml")]
		sizingPath := stateDir + "sizing.yaml"
		if err := plan.WriteYAML(sizingPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save sizing plan: %v\n", err)
		} else {
			st.SizingPlanPath = sizingPath
			_ = st.Save("")
			fmt.Printf("Sizing plan saved to %s\n", sizingPath)
		}

		return nil
	},
}

func init() {
	estimateCmd.Flags().BoolVar(&estimateBenchmark, "benchmark", false, "run source DB read throughput benchmark")
	rootCmd.AddCommand(estimateCmd)
}
