package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/reloquent/reloquent/internal/state"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check migration readiness and current state",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := state.Load("")
		if err != nil {
			return fmt.Errorf("loading state: %w", err)
		}

		fmt.Printf("Current step: %s\n\n", st.CurrentStep)

		// Show completed steps
		steps := []state.Step{
			state.StepSourceConnection,
			state.StepTargetConnection,
			state.StepTableSelection,
			state.StepDenormalization,
			state.StepTypeMapping,
			state.StepSizing,
			state.StepAWSSetup,
			state.StepPreMigration,
			state.StepReview,
			state.StepMigration,
			state.StepValidation,
			state.StepIndexBuilds,
		}

		labels := map[state.Step]string{
			state.StepSourceConnection: "1. Source Connection",
			state.StepTargetConnection: "2. Target Connection",
			state.StepTableSelection:   "3. Table Selection",
			state.StepDenormalization:   "4. Denormalization",
			state.StepTypeMapping:       "5. Type Mapping",
			state.StepSizing:            "6. Sizing",
			state.StepAWSSetup:          "7. AWS Setup",
			state.StepPreMigration:      "8. Pre-Migration",
			state.StepReview:            "9. Review",
			state.StepMigration:         "10. Migration",
			state.StepValidation:        "11. Validation",
			state.StepIndexBuilds:       "12. Index Builds",
		}

		for _, step := range steps {
			status := "  "
			if st.IsStepComplete(step) {
				status = "OK"
			} else if st.CurrentStep == step {
				status = ">>"
			}
			fmt.Printf("  [%s] %s\n", status, labels[step])
		}

		// Additional state info
		fmt.Println()
		if st.SourceConfig != nil {
			fmt.Printf("Source: %s (%s:%d/%s)\n", st.SourceConfig.Type, st.SourceConfig.Host, st.SourceConfig.Port, st.SourceConfig.Database)
		}
		if st.TargetConfig != nil {
			fmt.Printf("Target: %s\n", st.TargetConfig.Database)
		}
		if len(st.SelectedTables) > 0 {
			fmt.Printf("Tables: %d selected\n", len(st.SelectedTables))
		}
		if st.SizingPlanPath != "" {
			fmt.Printf("Sizing: %s\n", st.SizingPlanPath)
		}
		if st.AWSResourceID != "" {
			fmt.Printf("AWS: %s (%s)\n", st.AWSResourceType, st.AWSResourceID)
		}
		if st.MigrationStatus != "" {
			fmt.Printf("Migration: %s\n", st.MigrationStatus)
		}
		if st.ValidationReportPath != "" {
			fmt.Printf("Validation: %s\n", st.ValidationReportPath)
		}
		if st.IndexBuildStatus != "" {
			fmt.Printf("Index Builds: %s\n", st.IndexBuildStatus)
		}
		if st.IndexPlanPath != "" {
			fmt.Printf("Index Plan: %s\n", st.IndexPlanPath)
		}
		if st.WriteConcernRestored {
			fmt.Println("Write Concern: restored (majority, j:true)")
		}
		if st.BalancerReEnabled {
			fmt.Println("Balancer: re-enabled")
		}
		if st.ProductionReady {
			fmt.Println("Production Ready: YES")
		}
		if st.ReportPath != "" {
			fmt.Printf("Report: %s\n", st.ReportPath)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
