package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/reloquent/reloquent/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default config file interactively",
	Long:  `Walk through prompts to create a Reloquent configuration file at ~/.reloquent/reloquent.yaml.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Println("Reloquent Configuration Setup")
		fmt.Println("============================")
		fmt.Println()

		// Source database
		fmt.Println("Source Database")
		fmt.Println("--------------")
		dbType := prompt(reader, "Database type (postgresql/oracle)", "postgresql")
		host := prompt(reader, "Host", "localhost")
		portStr := prompt(reader, "Port", defaultPort(dbType))
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port: %s", portStr)
		}
		database := prompt(reader, "Database name", "")
		schema := prompt(reader, "Schema (leave empty for default)", "public")
		username := prompt(reader, "Username", "")
		password := prompt(reader, "Password", "")
		fmt.Println()

		// Target MongoDB
		fmt.Println("Target MongoDB")
		fmt.Println("--------------")
		connStr := prompt(reader, "Connection string", "mongodb://localhost:27017")
		targetDB := prompt(reader, "Database name", database)
		fmt.Println()

		cfg := &config.Config{
			Version: config.CurrentVersion,
			Source: config.SourceConfig{
				Type:     dbType,
				Host:     host,
				Port:     port,
				Database: database,
				Schema:   schema,
				Username: username,
				Password: password,
			},
			Target: config.TargetConfig{
				Type:             "mongodb",
				ConnectionString: connStr,
				Database:         targetDB,
			},
		}

		cfgPath := config.ExpandHome(config.DefaultPath)
		if cfgFile != "" {
			cfgPath = cfgFile
		}

		if err := cfg.Save(cfgPath); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("Config written to %s\n", cfgPath)
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  reloquent discover   — Discover the source database schema")
		fmt.Println("  reloquent            — Launch the interactive wizard")
		fmt.Println("  reloquent serve      — Start the web UI")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func prompt(reader *bufio.Reader, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  %s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("  %s: ", label)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func defaultPort(dbType string) string {
	switch dbType {
	case "oracle":
		return "1521"
	default:
		return "5432"
	}
}
