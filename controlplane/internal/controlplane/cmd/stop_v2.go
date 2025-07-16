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
	stopV2ClusterName string
	stopV2DeleteData  bool
)

var stopV2Cmd = &cobra.Command{
	Use:   "stop-v2",
	Short: "Stop KECS k3d cluster (new architecture)",
	Long:  `Stop and delete the k3d cluster running KECS control plane.`,
	RunE:  runStopV2,
}

func init() {
	RootCmd.AddCommand(stopV2Cmd)

	stopV2Cmd.Flags().StringVar(&stopV2ClusterName, "name", "kecs", "Cluster name to stop")
	stopV2Cmd.Flags().BoolVar(&stopV2DeleteData, "delete-data", false, "Delete persistent data")
}

func runStopV2(cmd *cobra.Command, args []string) error {
	fmt.Printf("Stopping KECS cluster '%s'...\n", stopV2ClusterName)

	// Create k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	ctx := context.Background()

	// Check if cluster exists
	exists, err := manager.ClusterExists(ctx, stopV2ClusterName)
	if err != nil {
		return fmt.Errorf("failed to check cluster existence: %w", err)
	}

	if !exists {
		fmt.Printf("Cluster '%s' does not exist\n", stopV2ClusterName)
		return nil
	}

	// Delete the cluster
	if err := manager.DeleteCluster(ctx, stopV2ClusterName); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	fmt.Printf("Successfully stopped and deleted cluster '%s'\n", stopV2ClusterName)

	// Delete data if requested
	if stopV2DeleteData {
		home, _ := os.UserHomeDir()
		dataDir := filepath.Join(home, ".kecs", "data")
		
		fmt.Printf("Deleting data directory: %s\n", dataDir)
		if err := os.RemoveAll(dataDir); err != nil {
			fmt.Printf("Warning: Failed to delete data directory: %v\n", err)
		}
	} else {
		fmt.Println("Data directory preserved. Use --delete-data to remove it.")
	}

	return nil
}