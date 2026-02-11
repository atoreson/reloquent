package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/mapping"
	"github.com/reloquent/reloquent/internal/state"
	"github.com/reloquent/reloquent/internal/wizard"
)

var (
	designImport     string
	designExport     string
	designWeb        bool
	designSchemaFile string
)

var designCmd = &cobra.Command{
	Use:   "design",
	Short: "Design the denormalization mapping",
	Long: `Define how relational tables map to MongoDB collections and embedded documents.

Requires a previously discovered schema file. If --schema is not provided,
looks for the schema at the default location (~/.reloquent/source-schema.yaml).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if designImport != "" {
			m, err := mapping.LoadYAML(designImport)
			if err != nil {
				return fmt.Errorf("loading mapping from %s: %w", designImport, err)
			}

			// Save to default location
			mappingPath := config.ExpandHome("~/.reloquent/mapping.yaml")
			if err := m.WriteYAML(mappingPath); err != nil {
				return fmt.Errorf("saving mapping: %w", err)
			}

			// Update state
			st, err := state.Load("")
			if err != nil {
				return fmt.Errorf("loading state: %w", err)
			}
			st.MappingPath = mappingPath
			if err := st.Save(""); err != nil {
				return fmt.Errorf("saving state: %w", err)
			}

			fmt.Printf("Imported mapping from %s (%d collections)\n", designImport, len(m.Collections))
			return nil
		}

		if designExport != "" {
			st, err := state.Load("")
			if err != nil {
				return fmt.Errorf("loading state: %w", err)
			}

			if st.MappingPath == "" {
				return fmt.Errorf("no mapping found; run the wizard or import a mapping first")
			}

			m, err := mapping.LoadYAML(st.MappingPath)
			if err != nil {
				return fmt.Errorf("loading current mapping: %w", err)
			}

			if err := m.WriteYAML(designExport); err != nil {
				return fmt.Errorf("writing mapping to %s: %w", designExport, err)
			}

			fmt.Printf("Exported mapping to %s (%d collections)\n", designExport, len(m.Collections))
			return nil
		}

		if designWeb {
			logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
			logger.Info("launching web UI for schema designer")

			// Launch serve command programmatically
			port := 8230
			url := fmt.Sprintf("http://localhost:%d/design", port)
			fmt.Printf("Opening web designer at %s\n", url)

			// Open browser
			go openBrowser(url)

			// Run serve inline (reuses the serve logic)
			servePort = port
			return serveCmd.RunE(cmd, args)
		}

		schemaPath := designSchemaFile
		if schemaPath == "" {
			schemaPath = filepath.Join(config.ExpandHome("~/.reloquent"), "source-schema.yaml")
		}

		statePath := ""
		if cfgFile != "" {
			statePath = filepath.Join(filepath.Dir(cfgFile), "state.yaml")
		}

		fmt.Println("Opening interactive denormalization designer...")
		return wizard.RunDenormStandalone(schemaPath, statePath)
	},
}

func init() {
	designCmd.Flags().StringVar(&designImport, "import", "", "import a pre-built mapping file")
	designCmd.Flags().StringVar(&designExport, "export", "", "export the current mapping")
	designCmd.Flags().BoolVar(&designWeb, "web", false, "launch browser-based visual designer")
	designCmd.Flags().StringVar(&designSchemaFile, "schema", "", "path to source schema YAML (default: ~/.reloquent/source-schema.yaml)")
	rootCmd.AddCommand(designCmd)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Run()
	}
}
