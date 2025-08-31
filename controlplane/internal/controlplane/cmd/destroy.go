package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/host/instance"
)

var (
	destroyInstanceName string
	destroyDeleteData   bool
	destroyForce        bool
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy KECS instance",
	Long:  `Destroy the KECS instance by deleting its k3d cluster and optionally its persistent data.`,
	RunE:  runDestroy,
}

func init() {
	RootCmd.AddCommand(destroyCmd)

	destroyCmd.Flags().StringVar(&destroyInstanceName, "instance", "", "KECS instance name to destroy (required)")
	destroyCmd.Flags().BoolVar(&destroyDeleteData, "delete-data", false, "Delete persistent data")
	destroyCmd.Flags().BoolVar(&destroyForce, "force", false, "Force destroy without confirmation")
}

func runDestroy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create instance manager
	manager, err := instance.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create instance manager: %w", err)
	}

	// If instance name is not provided, list available instances
	if destroyInstanceName == "" {
		fmt.Println("Fetching KECS instances...")

		// Get list of instances
		instances, err := manager.List(ctx)
		if err != nil {
			return fmt.Errorf("failed to list instances: %w", err)
		}

		if len(instances) == 0 {
			fmt.Println("No KECS instances found")
			return nil
		}

		// List available instances
		fmt.Println("\nAvailable KECS instances:")
		for i, inst := range instances {
			fmt.Printf("  %d. %s (%s)\n", i+1, inst.Name, strings.ToLower(inst.Status))
		}

		return fmt.Errorf("please specify an instance to destroy with --instance flag")
	}

	// Show header
	fmt.Printf("Destroying KECS instance '%s'\n", destroyInstanceName)

	// Check instance status
	fmt.Println("Checking instance status...")

	// Show warning if not forced
	if !destroyForce {
		fmt.Printf("\n⚠️  WARNING: You are about to destroy instance '%s'. This action cannot be undone.\n", destroyInstanceName)
		fmt.Println("Use --force flag to skip this warning.")
		return fmt.Errorf("operation cancelled (use --force to confirm)")
	}

	// Destroy the instance
	fmt.Println("Deleting k3d cluster...")
	if err := manager.Destroy(ctx, destroyInstanceName, destroyDeleteData); err != nil {
		if err.Error() == fmt.Sprintf("instance '%s' does not exist", destroyInstanceName) {
			fmt.Printf("KECS instance '%s' does not exist\n", destroyInstanceName)
			return nil
		}
		return fmt.Errorf("failed to destroy instance: %w", err)
	}

	fmt.Printf("✅ KECS instance '%s' has been destroyed\n", destroyInstanceName)

	if destroyDeleteData {
		fmt.Println("Instance data has been deleted.")
	} else {
		fmt.Println("Instance data preserved. Use --delete-data to remove it.")
	}

	return nil
}
