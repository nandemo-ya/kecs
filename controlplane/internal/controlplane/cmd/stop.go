package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var (
	stopContainerName string
	stopForce         bool
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
}

func runStop(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// Check if Docker daemon is running
	if _, err := cli.Ping(ctx); err != nil {
		return fmt.Errorf("Docker daemon is not running: %w", err)
	}

	// Find container
	filters := filters.NewArgs()
	filters.Add("name", stopContainerName)
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Printf("KECS container '%s' not found\n", stopContainerName)
		return nil
	}

	containerInfo := containers[0]

	// Stop container if running
	if containerInfo.State == "running" {
		fmt.Printf("Stopping KECS container '%s'...\n", stopContainerName)
		
		// Graceful shutdown timeout
		timeout := 10
		if stopForce {
			timeout = 0
		}
		
		if err := cli.ContainerStop(ctx, containerInfo.ID, container.StopOptions{
			Timeout: &timeout,
		}); err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}

		// Wait a bit for container to fully stop
		time.Sleep(1 * time.Second)
	}

	// Remove container
	fmt.Printf("Removing KECS container '%s'...\n", stopContainerName)
	if err := cli.ContainerRemove(ctx, containerInfo.ID, container.RemoveOptions{
		Force: stopForce,
	}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	fmt.Printf("KECS container '%s' stopped and removed successfully\n", stopContainerName)
	fmt.Println("Note: Data directory has been preserved")
	
	return nil
}