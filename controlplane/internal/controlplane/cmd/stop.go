package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/instance"
)

var (
	stopInstanceName string
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop KECS instance",
	Long:  `Stop the KECS instance by stopping its k3d cluster. The instance can be restarted later with the start command.`,
	RunE:  runStop,
}

func init() {
	RootCmd.AddCommand(stopCmd)

	stopCmd.Flags().StringVar(&stopInstanceName, "instance", "", "KECS instance name to stop (required)")
}

func runStop(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create instance manager
	manager, err := instance.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create instance manager: %w", err)
	}

	// If instance name is not provided, list available instances
	if stopInstanceName == "" {
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

		return fmt.Errorf("please specify an instance to stop with --instance flag")
	}

	// Show header
	fmt.Printf("Stopping KECS instance '%s'\n", stopInstanceName)

	// Check instance status
	fmt.Println("Checking instance status...")

	// Stop the instance
	fmt.Println("Stopping k3d cluster...")
	if err := manager.Stop(ctx, stopInstanceName); err != nil {
		if err.Error() == fmt.Sprintf("instance '%s' does not exist", stopInstanceName) {
			fmt.Printf("KECS instance '%s' does not exist\n", stopInstanceName)
			return nil
		}
		if err.Error() == fmt.Sprintf("instance '%s' is not running", stopInstanceName) {
			fmt.Printf("KECS instance '%s' is not running\n", stopInstanceName)
			return nil
		}
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	fmt.Printf("✅ KECS instance '%s' has been stopped\n", stopInstanceName)
	fmt.Println("Instance data preserved. Use 'kecs start' to restart the instance.")

	return nil
}
