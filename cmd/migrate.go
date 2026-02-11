package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/reloquent/reloquent/internal/aws"
	"github.com/reloquent/reloquent/internal/codegen"
	"github.com/reloquent/reloquent/internal/config"
	"github.com/reloquent/reloquent/internal/engine"
	"github.com/reloquent/reloquent/internal/migration"
)

var (
	migrateSkipProvision bool
	migrateCollection    string
	migrateDryRun        bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run the migration",
	Long:  `Execute the PySpark migration job on the Spark cluster, monitor progress, and report results.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

		cfgPath := cfgFile
		if cfgPath == "" {
			cfgPath = config.ExpandHome(config.DefaultPath)
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		eng := engine.New(cfg, logger)
		st, err := eng.LoadState()
		if err != nil {
			return fmt.Errorf("loading state: %w", err)
		}

		if st.AWSResourceID == "" && !migrateSkipProvision && !migrateDryRun {
			return fmt.Errorf("no AWS infrastructure provisioned; run `reloquent provision` first or use --dry-run")
		}

		// Dry run: show what would happen
		if migrateDryRun {
			fmt.Println("Dry run — showing migration plan:")
			fmt.Println()
			fmt.Printf("Source: %s://%s:%d/%s\n", cfg.Source.Type, cfg.Source.Host, cfg.Source.Port, cfg.Source.Database)
			fmt.Printf("Target: %s (%s)\n", cfg.Target.ConnectionString, cfg.Target.Database)
			fmt.Println()

			if eng.Schema != nil && eng.Mapping != nil {
				gen := &codegen.Generator{
					Config:  cfg,
					Schema:  eng.Schema,
					Mapping: eng.Mapping,
					TypeMap: eng.GetTypeMap(),
				}
				result, err := gen.Generate()
				if err != nil {
					return fmt.Errorf("generating code: %w", err)
				}
				fmt.Println("Generated PySpark script:")
				fmt.Println("========================")
				fmt.Println(result.MigrationScript)
			} else {
				fmt.Println("Run `reloquent discover` and `reloquent design` first to see the generated migration script.")
			}

			if cfg.AWS.Platform != "" {
				fmt.Printf("Platform: %s\n", cfg.AWS.Platform)
				fmt.Printf("S3 Bucket: %s\n", cfg.AWS.S3Bucket)
				fmt.Printf("Region: %s\n", cfg.AWS.Region)
			}
			return nil
		}

		// Real migration
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		callback := func(status *migration.Status) {
			switch status.Phase {
			case "preflight":
				fmt.Println("Running pre-flight checks...")
			case "provisioning":
				fmt.Println("Waiting for infrastructure...")
			case "running":
				pct := status.Overall.PercentComplete
				if pct > 0 {
					fmt.Printf("\rProgress: %.1f%% (%d/%d docs)",
						pct, status.Overall.DocsWritten, status.Overall.DocsTotal)
				}
			case "completed":
				fmt.Printf("\nMigration completed in %s\n", status.ElapsedTime)
			case "failed":
				fmt.Printf("\nMigration failed: %v\n", status.Errors)
			case "partial_failure":
				fmt.Println("\nPartial failure — some collections failed:")
				for _, col := range status.Collections {
					if col.State == "failed" {
						fmt.Printf("  %s: %s\n", col.Name, col.Error)
					}
				}
				fmt.Println("Use `reloquent migrate --collection <name>` to retry failed collections.")
			}
		}

		st.MigrationStatus = "running"
		_ = eng.SaveState()

		// Build provisioner
		awsClient, err := aws.NewRealClient(ctx, cfg.AWS.Profile, cfg.AWS.Region)
		if err != nil {
			return fmt.Errorf("creating AWS client: %w", err)
		}

		// Upload artifacts
		uploader := aws.NewArtifactUploader(awsClient, cfg.AWS.S3Bucket, "reloquent/"+cfg.Target.Database)

		var script []byte
		if eng.Schema != nil && eng.Mapping != nil {
			gen := &codegen.Generator{
				Config:  cfg,
				Schema:  eng.Schema,
				Mapping: eng.Mapping,
				TypeMap: eng.GetTypeMap(),
			}
			result, err := gen.Generate()
			if err != nil {
				return fmt.Errorf("generating migration script: %w", err)
			}
			script = []byte(result.MigrationScript)
		} else {
			return fmt.Errorf("run `reloquent discover` and `reloquent design` before migrating")
		}

		artifacts, err := uploader.UploadArtifacts(ctx, aws.ArtifactSet{
			MigrationScript: script,
		})
		if err != nil {
			return fmt.Errorf("uploading artifacts: %w", err)
		}

		// Create executor
		var prov aws.Provisioner
		switch cfg.AWS.Platform {
		case "emr":
			p, provErr := aws.NewEMRProvisioner(ctx, cfg.AWS.Profile, cfg.AWS.Region)
			if provErr != nil {
				return fmt.Errorf("creating EMR provisioner: %w", provErr)
			}
			prov = p
		case "glue":
			p, provErr := aws.NewGlueProvisioner(ctx, cfg.AWS.Profile, cfg.AWS.Region)
			if provErr != nil {
				return fmt.Errorf("creating Glue provisioner: %w", provErr)
			}
			prov = p
		default:
			return fmt.Errorf("unsupported platform: %s", cfg.AWS.Platform)
		}

		executor := migration.NewExecutor(prov, nil, artifacts, nil)
		executor.SetResourceID(st.AWSResourceID)

		if migrateCollection != "" {
			fmt.Printf("Retrying migration for collection: %s\n", migrateCollection)
			_, err = executor.RetryFailed(ctx, []string{migrateCollection}, callback)
		} else {
			fmt.Println("Running full migration...")
			_, err = executor.Run(ctx, callback)
		}

		if err != nil {
			st.MigrationStatus = "failed"
			eng.SaveState()
			return err
		}

		st.MigrationStatus = "completed"
		eng.SaveState()
		return nil
	},
}

func init() {
	migrateCmd.Flags().BoolVar(&migrateSkipProvision, "skip-provision", false, "use existing cluster")
	migrateCmd.Flags().StringVar(&migrateCollection, "collection", "", "retry a specific failed collection")
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "show what would happen without executing")
	rootCmd.AddCommand(migrateCmd)
}

