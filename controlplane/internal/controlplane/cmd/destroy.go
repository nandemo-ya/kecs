package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
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
	
	// Create k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	// If instance name is not provided, list available instances
	if destroyInstanceName == "" {
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
		
		return fmt.Errorf("please specify an instance to destroy with --instance flag")
	}

	// Show header
	fmt.Printf("Destroying KECS instance '%s'\n", destroyInstanceName)

	// Check instance status
	fmt.Println("Checking instance status...")

	// Check if cluster exists
	exists, err := manager.ClusterExists(ctx, destroyInstanceName)
	if err != nil {
		return fmt.Errorf("failed to check cluster existence: %w", err)
	}

	if !exists {
		fmt.Printf("KECS instance '%s' does not exist\n", destroyInstanceName)
		return nil
	}
	fmt.Println("Instance found")

	// Show warning if not forced
	if !destroyForce {
		fmt.Printf("\n⚠️  WARNING: You are about to destroy instance '%s'. This action cannot be undone.\n", destroyInstanceName)
		fmt.Println("Use --force flag to skip this warning.")
		return fmt.Errorf("operation cancelled (use --force to confirm)")
	}

	// Delete the cluster
	fmt.Println("Deleting k3d cluster...")
	if err := manager.DeleteCluster(ctx, destroyInstanceName); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}
	
	fmt.Printf("✅ KECS instance '%s' has been destroyed\n", destroyInstanceName)

	// Delete data if requested
	if destroyDeleteData {
		home, _ := os.UserHomeDir()
		dataDir := filepath.Join(home, ".kecs", "instances", destroyInstanceName, "data")
		
		fmt.Printf("Deleting data directory: %s\n", dataDir)
		
		if err := os.RemoveAll(dataDir); err != nil {
			fmt.Printf("⚠️  Failed to delete data directory: %v\n", err)
		} else {
			fmt.Println("Data directory deleted")
		}
		
		// Also delete the instance directory if it's empty
		instanceDir := filepath.Join(home, ".kecs", "instances", destroyInstanceName)
		os.Remove(instanceDir) // This will only succeed if directory is empty
	} else {
		fmt.Println("Instance data preserved. Use --delete-data to remove it.")
	}

	return nil
}