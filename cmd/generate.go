package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/reloquent/reloquent/internal/codegen"
	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/drivers"
	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/schema"
	"github.com/reloquent/reloquent/internal/state"
	"github.com/reloquent/reloquent/internal/typemap"
)

var generateOutput string

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate PySpark migration scripts",
	Long:  `Generate self-contained PySpark scripts based on the schema design, type mappings, and configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load state
		st, err := state.Load("")
		if err != nil {
			return fmt.Errorf("loading state: %w", err)
		}

		// Load config
		cfg, err := config.Load(cfgFile)
		if err != nil {
			// Config not strictly required — build from state
			cfg = buildConfigFromState(st)
		}

		// Load schema
		if st.SchemaPath == "" {
			return fmt.Errorf("no schema available; run `reloquent discover` first")
		}
		s, err := schema.LoadYAML(st.SchemaPath)
		if err != nil {
			return fmt.Errorf("loading schema: %w", err)
		}

		// Load mapping
		if st.MappingPath == "" {
			return fmt.Errorf("no mapping available; run the wizard through step 4 first")
		}
		m, err := mapping.LoadYAML(st.MappingPath)
		if err != nil {
			return fmt.Errorf("loading mapping: %w", err)
		}

		// Load type mapping
		var tm *typemap.TypeMap
		if st.TypeMappingPath != "" {
			tm, err = typemap.LoadYAML(st.TypeMappingPath)
			if err != nil {
				fmt.Printf("Warning: could not load type mapping: %v (using defaults)\n", err)
				tm = typemap.ForDatabase(cfg.Source.Type)
			}
		} else {
			tm = typemap.ForDatabase(cfg.Source.Type)
		}

		// Check Oracle JDBC if needed
		if cfg.Source.Type == "oracle" {
			if _, err := drivers.FindOracleJDBC(); err != nil {
				fmt.Println("Warning: Oracle JDBC driver not found.")
				fmt.Println(drivers.OracleJDBCGuidance())
			}
		}

		// Generate
		g := &codegen.Generator{
			Config:  cfg,
			Schema:  s,
			Mapping: m,
			TypeMap: tm,
		}

		result, err := g.Generate()
		if err != nil {
			return fmt.Errorf("generating migration script: %w", err)
		}

		// Write output
		outputDir := generateOutput
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
		outputPath := filepath.Join(outputDir, "migration.py")
		if err := os.WriteFile(outputPath, []byte(result.MigrationScript), 0o644); err != nil {
			return fmt.Errorf("writing migration script: %w", err)
		}

		fmt.Printf("Migration script written to %s\n", outputPath)
		return nil
	},
}

func buildConfigFromState(st *state.State) *config.Config {
	cfg := &config.Config{Version: 1}
	if st.SourceConfig != nil {
		cfg.Source = *st.SourceConfig
	}
	if st.TargetConfig != nil {
		cfg.Target = *st.TargetConfig
	}
	if cfg.Source.MaxConnections == 0 {
		cfg.Source.MaxConnections = 20
	}
	return cfg
}

func init() {
	generateCmd.Flags().StringVar(&generateOutput, "output", "output", "output directory for generated scripts")
	rootCmd.AddCommand(generateCmd)
}
