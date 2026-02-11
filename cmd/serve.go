package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"encoding/json"

	"github.com/reloquent/reloquent/internal/api"
	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/engine"
	"github.com/reloquent/reloquent/internal/ws"
	"github.com/reloquent/reloquent/web"
)

var servePort int
var serveDevMode bool
var serveConfig string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web UI server",
	Long:  `Start the full web UI wizard on localhost. The web UI provides the complete migration workflow in the browser.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		// Load config if provided
		var cfg *config.Config
		configPath := serveConfig
		if configPath == "" {
			configPath = cfgFile
		}
		if configPath != "" {
			var err error
			cfg, err = config.Load(configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			logger.Info("loaded config", "path", configPath)
		}

		eng := engine.New(cfg, logger)

		hub := ws.NewHub(logger)
		hub.SetStateProvider(func() ([]byte, error) {
			st, err := eng.LoadState()
			if err != nil {
				return nil, err
			}
			return json.Marshal(st)
		})
		go hub.Run()

		// Get the embedded React app filesystem
		distFS, err := fs.Sub(web.DistFS, "dist")
		if err != nil {
			return fmt.Errorf("loading embedded web UI: %w", err)
		}

		srv := api.New(eng, logger, servePort,
			api.WithStaticFS(distFS),
			api.WithHub(hub),
			api.WithDevMode(serveDevMode),
		)

		// Graceful shutdown on signals
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		errCh := make(chan error, 1)
		go func() {
			errCh <- srv.Start()
		}()

		fmt.Fprintf(os.Stderr, "Reloquent web UI: http://localhost:%d\n", servePort)

		select {
		case err := <-errCh:
			if err != nil && err != http.ErrServerClosed {
				return err
			}
		case <-ctx.Done():
			logger.Info("shutting down server")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5_000_000_000) // 5s
			defer cancel()
			if err := srv.Shutdown(shutdownCtx); err != nil {
				return fmt.Errorf("server shutdown: %w", err)
			}
		}

		return nil
	},
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 8230, "port for the web UI server")
	serveCmd.Flags().BoolVar(&serveDevMode, "dev", false, "enable CORS for development mode")
	serveCmd.Flags().StringVar(&serveConfig, "config", "", "path to config file for pre-configured connections")
	rootCmd.AddCommand(serveCmd)
}
