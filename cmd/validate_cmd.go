package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/postmigration"
	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/source"
	"github.com/reloquent/reloquent/internal/state"
	"github.com/reloquent/reloquent/internal/target"
)

var (
	validateSamples int
	validateFull    bool
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate migration results",
	Long:  `Compare source and target data to verify migration correctness via row counts, sampling, and aggregates.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := state.Load("")
		if err != nil {
			return fmt.Errorf("loading state: %w", err)
		}

		if st.MigrationStatus != "completed" {
			return fmt.Errorf("migration has not completed; run the migration first")
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

		// Connect to source
		srcReader, err := buildSourceReader(st.SourceConfig)
		if err != nil {
			return fmt.Errorf("connecting to source: %w", err)
		}
		defer srcReader.Close()

		// Connect to target
		tgtOp, err := target.NewMongoOperator(context.Background(),
			st.TargetConfig.ConnectionString, st.TargetConfig.Database)
		if err != nil {
			return fmt.Errorf("connecting to target: %w", err)
		}
		defer tgtOp.Close(context.Background())

		orch := &postmigration.Orchestrator{
			Source:     srcReader,
			Target:     tgtOp,
			Schema:     s,
			Mapping:    m,
			State:      st,
			StatePath:  config.ExpandHome(state.DefaultPath),
			SampleSize: validateSamples,
		}

		cb := postmigration.Callbacks{
			OnValidationCheck: func(collection, checkType string, passed bool) {
				status := "PASS"
				if !passed {
					status = "FAIL"
				}
				fmt.Printf("  [%s] %s: %s\n", status, collection, checkType)
			},
		}

		fmt.Println("Running validation...")
		result, err := orch.RunValidation(context.Background(), cb)
		if err != nil {
			return fmt.Errorf("validation: %w", err)
		}

		fmt.Printf("\nOverall: %s\n", result.Status)

		// Print detailed results
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))

		if st.ValidationReportPath != "" {
			fmt.Printf("\nReport saved to: %s\n", st.ValidationReportPath)
		}

		return nil
	},
}

func buildSourceReader(sc *config.SourceConfig) (source.Reader, error) {
	if sc == nil {
		return nil, fmt.Errorf("no source configuration")
	}
	var reader source.Reader
	switch sc.Type {
	case "postgresql":
		connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
			sc.Username, sc.Password, sc.Host, sc.Port, sc.Database)
		if sc.SSL {
			connStr += "?sslmode=require"
		} else {
			connStr += "?sslmode=disable"
		}
		reader = source.NewPostgresReader(connStr, sc.Schema)
	case "oracle":
		connStr := fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
			sc.Username, sc.Password, sc.Host, sc.Port, sc.Database)
		reader = source.NewOracleReader(connStr, sc.Schema)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", sc.Type)
	}

	if err := reader.Connect(context.Background()); err != nil {
		return nil, err
	}
	return reader, nil
}

func init() {
	validateCmd.Flags().IntVar(&validateSamples, "samples", 1000, "number of documents to sample per collection")
	validateCmd.Flags().BoolVar(&validateFull, "full", false, "full row count + aggregate validation")
	rootCmd.AddCommand(validateCmd)
}
