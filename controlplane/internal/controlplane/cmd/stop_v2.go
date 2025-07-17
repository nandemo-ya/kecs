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
	stopV2InstanceName string
	stopV2DeleteData   bool
)

var stopV2Cmd = &cobra.Command{
	Use:   "stop-v2",
	Short: "Stop KECS instance (new architecture)",
	Long:  `Stop and delete the KECS instance including its k3d cluster and control plane.`,
	RunE:  runStopV2,
}

func init() {
	RootCmd.AddCommand(stopV2Cmd)

	stopV2Cmd.Flags().StringVar(&stopV2InstanceName, "instance", "", "KECS instance name to stop (required)")
	stopV2Cmd.Flags().BoolVar(&stopV2DeleteData, "delete-data", false, "Delete persistent data")
}

func runStopV2(cmd *cobra.Command, args []string) error {
	if stopV2InstanceName == "" {
		return fmt.Errorf("instance name is required. Use --instance flag to specify the instance to stop")
	}

	fmt.Printf("Stopping KECS instance '%s'...\n", stopV2InstanceName)

	// Create k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	ctx := context.Background()

	// Check if cluster exists
	exists, err := manager.ClusterExists(ctx, stopV2InstanceName)
	if err != nil {
		return fmt.Errorf("failed to check cluster existence: %w", err)
	}

	if !exists {
		fmt.Printf("KECS instance '%s' does not exist\n", stopV2InstanceName)
		return nil
	}

	// Delete the cluster
	if err := manager.DeleteCluster(ctx, stopV2InstanceName); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	fmt.Printf("Successfully stopped and deleted KECS instance '%s'\n", stopV2InstanceName)

	// Delete data if requested
	if stopV2DeleteData {
		home, _ := os.UserHomeDir()
		dataDir := filepath.Join(home, ".kecs", "instances", stopV2InstanceName, "data")
		
		fmt.Printf("Deleting data directory: %s\n", dataDir)
		if err := os.RemoveAll(dataDir); err != nil {
			fmt.Printf("Warning: Failed to delete data directory: %v\n", err)
		}
	} else {
		fmt.Println("Instance data preserved. Use --delete-data to remove it.")
	}

	return nil
}