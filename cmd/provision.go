package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	awspkg "github.com/reloquent/reloquent/internal/aws"
	"github.com/reloquent/reloquent/internal/state"
)

var (
	provisionDryRun   bool
	provisionTeardown bool
	provisionCheck    bool
)

var provisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Provision AWS infrastructure for the migration",
	Long:  `Provision the Spark cluster (EMR or Glue) and upload migration artifacts to S3.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := state.Load("")
		if err != nil {
			return fmt.Errorf("loading state: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if provisionCheck {
			fmt.Println("Checking AWS credentials and permissions...")

			client, err := awspkg.NewRealClient(ctx, "", "")
			if err != nil {
				return fmt.Errorf("creating AWS client: %w", err)
			}

			identity, err := client.VerifyCredentials(ctx)
			if err != nil {
				return fmt.Errorf("verifying credentials: %w", err)
			}
			fmt.Printf("  Account: %s\n  ARN: %s\n", identity.Account, identity.ARN)

			access, err := awspkg.CheckPlatformAccess(ctx, client)
			if err != nil {
				return fmt.Errorf("checking platform access: %w", err)
			}
			fmt.Printf("  EMR: %v  Glue: %v\n", access.EMRAvailable, access.GlueAvailable)
			fmt.Printf("  %s\n", access.Message)
			return nil
		}

		if provisionTeardown {
			if st.AWSResourceID == "" {
				return fmt.Errorf("no provisioned resources to tear down")
			}
			fmt.Printf("Tearing down %s (%s)...\n", st.AWSResourceType, st.AWSResourceID)

			// Create appropriate provisioner based on type
			if st.AWSResourceType == "emr_cluster" {
				prov, err := awspkg.NewEMRProvisioner(ctx, "", "")
				if err != nil {
					return fmt.Errorf("creating EMR provisioner: %w", err)
				}
				if err := prov.Teardown(ctx, st.AWSResourceID); err != nil {
					return fmt.Errorf("tearing down: %w", err)
				}
			} else {
				prov, err := awspkg.NewGlueProvisioner(ctx, "", "")
				if err != nil {
					return fmt.Errorf("creating Glue provisioner: %w", err)
				}
				if err := prov.Teardown(ctx, st.AWSResourceID); err != nil {
					return fmt.Errorf("tearing down: %w", err)
				}
			}

			st.AWSResourceID = ""
			st.AWSResourceType = ""
			_ = st.Save("")
			fmt.Println("Infrastructure terminated.")
			return nil
		}

		if provisionDryRun {
			fmt.Println("Dry run — showing what would be provisioned:")
			if st.SizingPlanPath != "" {
				fmt.Printf("  Sizing plan: %s\n", st.SizingPlanPath)
			}
			return nil
		}

		fmt.Println("Provisioning AWS infrastructure...")
		fmt.Println("(Full provisioning requires AWS credentials and network access)")
		return nil
	},
}

func init() {
	provisionCmd.Flags().BoolVar(&provisionDryRun, "dry-run", false, "show what would be provisioned")
	provisionCmd.Flags().BoolVar(&provisionTeardown, "teardown", false, "destroy provisioned resources")
	provisionCmd.Flags().BoolVar(&provisionCheck, "check", false, "verify AWS credentials and platform permissions")
	rootCmd.AddCommand(provisionCmd)
}
