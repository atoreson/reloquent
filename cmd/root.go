package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/reloquent/reloquent/internal/wizard"
)

var (
	cfgFile  string
	logLevel string
	version  = "dev"
	commit   = "none"
	date     = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "reloquent",
	Short: "Reloquent — Relational to MongoDB migration tool",
	Long: `Reloquent automates offline migrations from relational databases
(Oracle, PostgreSQL) to MongoDB using Apache Spark.

Running without a subcommand launches the interactive wizard.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Launching interactive wizard...")
		w, err := wizard.New("")
		if err != nil {
			return err
		}
		return w.Run()
	},
}

func Execute() {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.reloquent/reloquent.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
}
