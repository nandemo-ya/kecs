package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/runtime"
)

var (
	stopContainerName string
	stopForce         bool
	stopRuntime       string
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop KECS server container",
	Long: `Stop and remove KECS server container.
This will stop the running KECS server and remove the container.
The data directory will be preserved.`,
	RunE: runStop,
}

func init() {
	RootCmd.AddCommand(stopCmd)

	stopCmd.Flags().StringVar(&stopContainerName, "name", defaultContainerName, "Container name")
	stopCmd.Flags().BoolVarP(&stopForce, "force", "f", false, "Force stop without graceful shutdown")
	stopCmd.Flags().StringVar(&stopRuntime, "runtime", "", "Container runtime to use (docker, containerd, or auto)")
}

func runStop(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get runtime manager
	manager := runtime.GetDefaultManager()
	
	// Set runtime if specified
	if stopRuntime != "" {
		switch stopRuntime {
		case "docker":
			if err := manager.UseDocker(); err != nil {
				return fmt.Errorf("failed to use Docker runtime: %w", err)
			}
		case "containerd":
			if err := manager.UseContainerd(""); err != nil {
				return fmt.Errorf("failed to use containerd runtime: %w", err)
			}
		case "auto":
			if err := manager.AutoDetect(); err != nil {
				return fmt.Errorf("failed to auto-detect runtime: %w", err)
			}
		default:
			return fmt.Errorf("unknown runtime: %s", stopRuntime)
		}
	}

	// Get runtime
	rt, err := manager.GetRuntime()
	if err != nil {
		return fmt.Errorf("failed to get container runtime: %w", err)
	}

	fmt.Printf("Using container runtime: %s\n", rt.Name())

	// Find container
	containers, err := rt.ListContainers(ctx, runtime.ListContainersOptions{
		All:   true,
		Names: []string{stopContainerName},
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Printf("No KECS server container found with name '%s'\n", stopContainerName)
		return nil
	}

	container := containers[0]

	// Stop container if running
	if container.State == "running" {
		fmt.Printf("Stopping KECS server '%s'...\n", stopContainerName)
		timeout := 30
		if stopForce {
			timeout = 0
		}
		if err := rt.StopContainer(ctx, container.ID, &timeout); err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}
		fmt.Println("Container stopped")
	}

	// Remove container
	fmt.Printf("Removing container '%s'...\n", stopContainerName)
	if err := rt.RemoveContainer(ctx, container.ID, stopForce); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	fmt.Printf("KECS server '%s' has been stopped and removed\n", stopContainerName)
	fmt.Println("Data directory has been preserved")

	return nil
}