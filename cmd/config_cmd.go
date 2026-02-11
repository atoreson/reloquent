package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/wizard"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `View, validate, and manage Reloquent configuration and type mappings.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current config (secrets masked)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		fmt.Println("Current configuration:")
		fmt.Println()
		fmt.Printf("  Source:\n")
		fmt.Printf("    Type:           %s\n", cfg.Source.Type)
		fmt.Printf("    Host:           %s\n", cfg.Source.Host)
		fmt.Printf("    Port:           %d\n", cfg.Source.Port)
		fmt.Printf("    Database:       %s\n", cfg.Source.Database)
		fmt.Printf("    Username:       %s\n", cfg.Source.Username)
		fmt.Printf("    Password:       %s\n", maskSecret(cfg.Source.Password))
		fmt.Printf("    Max Conns:      %d\n", cfg.Source.MaxConnections)
		fmt.Println()
		fmt.Printf("  Target:\n")
		fmt.Printf("    Connection:     %s\n", maskSecret(cfg.Target.ConnectionString))
		fmt.Printf("    Database:       %s\n", cfg.Target.Database)

		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("config invalid: %w", err)
		}

		var errors []string

		if cfg.Source.Type == "" {
			errors = append(errors, "source.type is required")
		}
		if cfg.Source.Host == "" {
			errors = append(errors, "source.host is required")
		}
		if cfg.Source.Database == "" {
			errors = append(errors, "source.database is required")
		}
		if cfg.Target.ConnectionString == "" {
			errors = append(errors, "target.connection_string is required")
		}
		if cfg.Target.Database == "" {
			errors = append(errors, "target.database is required")
		}

		if len(errors) > 0 {
			fmt.Println("Validation errors:")
			for _, e := range errors {
				fmt.Printf("  - %s\n", e)
			}
			return fmt.Errorf("%d validation error(s)", len(errors))
		}

		fmt.Println("Configuration is valid.")
		return nil
	},
}

var configTypeMappingCmd = &cobra.Command{
	Use:   "type-mapping",
	Short: "Interactive type mapping editor",
	RunE: func(cmd *cobra.Command, args []string) error {
		return wizard.RunTypeMapStandalone("")
	},
}

func maskSecret(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configTypeMappingCmd)
	rootCmd.AddCommand(configCmd)
}
