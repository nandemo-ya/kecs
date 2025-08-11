package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
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
	
	// Create k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	// If instance name is not provided, list available instances
	if stopInstanceName == "" {
		fmt.Println("Fetching KECS instances...")
		
		// Get list of clusters
		clusters, err := manager.ListClusters(ctx)
		if err != nil {
			return fmt.Errorf("failed to list instances: %w", err)
		}
		
		if len(clusters) == 0 {
			fmt.Println("No KECS instances found")
			return nil
		}
		
		// List available instances
		fmt.Println("\nAvailable KECS instances:")
		for i, cluster := range clusters {
			// Check if cluster is running
			running, _ := manager.IsClusterRunning(ctx, cluster)
			status := "stopped"
			if running {
				status = "running"
			}
			fmt.Printf("  %d. %s (%s)\n", i+1, cluster, status)
		}
		
		return fmt.Errorf("please specify an instance to stop with --instance flag")
	}

	// Show header
	fmt.Printf("Stopping KECS instance '%s'\n", stopInstanceName)

	// Check instance status
	fmt.Println("Checking instance status...")

	// Check if cluster exists
	exists, err := manager.ClusterExists(ctx, stopInstanceName)
	if err != nil {
		return fmt.Errorf("failed to check cluster existence: %w", err)
	}

	if !exists {
		fmt.Printf("KECS instance '%s' does not exist\n", stopInstanceName)
		return nil
	}
	fmt.Println("Instance found")

	// Stop the cluster
	fmt.Println("Stopping k3d cluster...")
	if err := manager.StopCluster(ctx, stopInstanceName); err != nil {
		return fmt.Errorf("failed to stop cluster: %w", err)
	}
	
	fmt.Printf("✅ KECS instance '%s' has been stopped\n", stopInstanceName)
	fmt.Println("Instance data preserved. Use 'kecs start' to restart the instance.")

	return nil
}