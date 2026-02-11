package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/wizard"
)

var selectSchemaFile string

var selectCmd = &cobra.Command{
	Use:   "select",
	Short: "Select tables to migrate",
	Long: `Interactively select which source tables to include in the migration.

Requires a previously discovered schema file. If --schema is not provided,
looks for the schema at the default location (~/.reloquent/source-schema.yaml).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		schemaPath := selectSchemaFile
		if schemaPath == "" {
			schemaPath = filepath.Join(config.ExpandHome("~/.reloquent"), "source-schema.yaml")
		}

		statePath := ""
		if cfgFile != "" {
			statePath = filepath.Join(filepath.Dir(cfgFile), "state.yaml")
		}

		fmt.Println("Opening table selection...")
		return wizard.RunTableSelectStandalone(schemaPath, statePath)
	},
}

func init() {
	selectCmd.Flags().StringVar(&selectSchemaFile, "schema", "", "path to source schema YAML (default: ~/.reloquent/source-schema.yaml)")
	rootCmd.AddCommand(selectCmd)
}
