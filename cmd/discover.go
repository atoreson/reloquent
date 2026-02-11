package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/discovery"
)

var (
	discoverDirect bool
	discoverScript bool
	discoverOutput string
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover source database schema",
	Long:  `Connect to the source database and extract schema metadata including tables, columns, types, constraints, foreign keys, and sizes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if discoverScript {
			return runDiscoverScript()
		}

		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		d, err := discovery.New(&cfg.Source)
		if err != nil {
			return fmt.Errorf("initializing discoverer: %w", err)
		}
		defer d.Close()

		ctx := context.Background()

		fmt.Printf("Connecting to %s at %s:%d/%s...\n",
			cfg.Source.Type, cfg.Source.Host, cfg.Source.Port, cfg.Source.Database)
		if err := d.Connect(ctx); err != nil {
			return fmt.Errorf("connecting to source: %w", err)
		}

		fmt.Println("Discovering schema...")
		schema, err := d.Discover(ctx)
		if err != nil {
			return fmt.Errorf("discovering schema: %w", err)
		}

		fmt.Println(schema.Summary())

		outputPath := discoverOutput
		if outputPath == "" {
			outputPath = filepath.Join("output", "config", "source-schema.yaml")
		}
		if err := schema.WriteYAML(outputPath); err != nil {
			return fmt.Errorf("writing schema: %w", err)
		}
		fmt.Printf("\nSchema written to %s\n", outputPath)

		return nil
	},
}

func runDiscoverScript() error {
	// Determine DB type and schema from config if available, or use defaults
	dbType := "postgresql"
	schemaName := ""

	cfg, err := config.Load(cfgFile)
	if err == nil {
		dbType = cfg.Source.Type
		schemaName = cfg.Source.Schema
	}

	sg := &discovery.ScriptGenerator{
		DBType: dbType,
		Schema: schemaName,
	}

	script := sg.GenerateScript()

	if discoverOutput != "" {
		// Write SQL script to file
		if err := os.MkdirAll(filepath.Dir(discoverOutput), 0o755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
		if err := os.WriteFile(discoverOutput, []byte(script), 0o644); err != nil {
			return fmt.Errorf("writing script: %w", err)
		}
		fmt.Printf("Discovery script written to %s\n", discoverOutput)

		// Also write the shell wrapper
		wrapperPath := discoverOutput[:len(discoverOutput)-len(filepath.Ext(discoverOutput))] + ".sh"
		wrapper := sg.GenerateShellWrapper()
		if err := os.WriteFile(wrapperPath, []byte(wrapper), 0o755); err != nil {
			return fmt.Errorf("writing wrapper: %w", err)
		}
		fmt.Printf("Shell wrapper written to %s\n", wrapperPath)
	} else {
		fmt.Print(script)
	}

	return nil
}

func init() {
	discoverCmd.Flags().BoolVar(&discoverDirect, "direct", true, "connect to source DB directly")
	discoverCmd.Flags().BoolVar(&discoverScript, "script", false, "generate offline discovery script")
	discoverCmd.Flags().StringVarP(&discoverOutput, "output", "o", "", "output path for schema YAML (default: output/config/source-schema.yaml)")
	rootCmd.AddCommand(discoverCmd)
}
