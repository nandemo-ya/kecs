package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/progress"
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
	ctx := context.Background()
	
	// Create k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	// If instance name is not provided, show selection
	if stopV2InstanceName == "" {
		spinner := progress.NewSpinner("Fetching KECS instances")
		spinner.Start()
		
		// Get list of clusters
		clusters, err := manager.ListClusters(ctx)
		if err != nil {
			spinner.Fail("Failed to list instances")
			return fmt.Errorf("failed to list instances: %w", err)
		}
		spinner.Stop()
		
		if len(clusters) == 0 {
			progress.Warning("No KECS instances found")
			return nil
		}
		
		// Show selection prompt
		selectedInstance, err := pterm.DefaultInteractiveSelect.
			WithOptions(clusters).
			WithDefaultText("Select KECS instance to stop").
			Show()
		if err != nil {
			return fmt.Errorf("failed to select instance: %w", err)
		}
		
		stopV2InstanceName = selectedInstance
	}

	// Show header
	progress.SectionHeader(fmt.Sprintf("Stopping KECS instance '%s'", stopV2InstanceName))

	// Check instance status
	spinner := progress.NewSpinner("Checking instance status")
	spinner.Start()

	// Check if cluster exists
	exists, err := manager.ClusterExists(ctx, stopV2InstanceName)
	if err != nil {
		spinner.Fail("Failed to check instance")
		return fmt.Errorf("failed to check cluster existence: %w", err)
	}

	if !exists {
		spinner.Stop()
		progress.Warning("KECS instance '%s' does not exist", stopV2InstanceName)
		return nil
	}
	spinner.Success("Instance found")

	// Create progress tracker for deletion
	tracker := progress.NewTracker(progress.Options{
		Description:     "Deleting k3d cluster",
		Total:           100,
		ShowElapsedTime: true,
		Width:           40,
	})

	// Start deletion in background
	errChan := make(chan error, 1)
	go func() {
		tracker.Update(30)
		if err := manager.DeleteCluster(ctx, stopV2InstanceName); err != nil {
			errChan <- err
			return
		}
		tracker.Update(100)
		errChan <- nil
	}()

	// Wait for deletion
	err = <-errChan
	if err != nil {
		tracker.FinishWithMessage("Failed to delete cluster")
		return fmt.Errorf("failed to delete cluster: %w", err)
	}
	tracker.FinishWithMessage("Cluster deleted successfully")

	progress.Success("KECS instance '%s' has been stopped", stopV2InstanceName)

	// Delete data if requested
	if stopV2DeleteData {
		home, _ := os.UserHomeDir()
		dataDir := filepath.Join(home, ".kecs", "instances", stopV2InstanceName, "data")
		
		spinner = progress.NewSpinner(fmt.Sprintf("Deleting data directory: %s", dataDir))
		spinner.Start()
		
		if err := os.RemoveAll(dataDir); err != nil {
			spinner.Fail("Failed to delete data directory")
			progress.Warning("Failed to delete data directory: %v", err)
		} else {
			spinner.Success("Data directory deleted")
		}
	} else {
		progress.Info("Instance data preserved. Use --delete-data to remove it.")
	}

	return nil
}