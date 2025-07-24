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

	// If instance name is not provided, show selection
	if destroyInstanceName == "" {
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
			WithDefaultText("Select KECS instance to destroy").
			Show()
		if err != nil {
			return fmt.Errorf("failed to select instance: %w", err)
		}
		
		destroyInstanceName = selectedInstance
	}

	// Show header
	progress.SectionHeader(fmt.Sprintf("Destroying KECS instance '%s'", destroyInstanceName))

	// Check instance status
	spinner := progress.NewSpinner("Checking instance status")
	spinner.Start()

	// Check if cluster exists
	exists, err := manager.ClusterExists(ctx, destroyInstanceName)
	if err != nil {
		spinner.Fail("Failed to check instance")
		return fmt.Errorf("failed to check cluster existence: %w", err)
	}

	if !exists {
		spinner.Stop()
		progress.Warning("KECS instance '%s' does not exist", destroyInstanceName)
		return nil
	}
	spinner.Success("Instance found")

	// Show confirmation prompt if not forced
	if !destroyForce {
		confirmed, err := pterm.DefaultInteractiveConfirm.
			WithDefaultText(fmt.Sprintf("Are you sure you want to destroy instance '%s'? This action cannot be undone.", destroyInstanceName)).
			Show()
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirmed {
			progress.Info("Destroy operation cancelled")
			return nil
		}
	}

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
		if err := manager.DeleteCluster(ctx, destroyInstanceName); err != nil {
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

	progress.Success("KECS instance '%s' has been destroyed", destroyInstanceName)

	// Delete data if requested
	if destroyDeleteData {
		home, _ := os.UserHomeDir()
		dataDir := filepath.Join(home, ".kecs", "instances", destroyInstanceName, "data")
		
		spinner = progress.NewSpinner(fmt.Sprintf("Deleting data directory: %s", dataDir))
		spinner.Start()
		
		if err := os.RemoveAll(dataDir); err != nil {
			spinner.Fail("Failed to delete data directory")
			progress.Warning("Failed to delete data directory: %v", err)
		} else {
			spinner.Success("Data directory deleted")
		}
		
		// Also delete the instance directory if it's empty
		instanceDir := filepath.Join(home, ".kecs", "instances", destroyInstanceName)
		os.Remove(instanceDir) // This will only succeed if directory is empty
	} else {
		progress.Info("Instance data preserved. Use --delete-data to remove it.")
	}

	return nil
}