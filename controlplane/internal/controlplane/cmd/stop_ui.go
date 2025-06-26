package cmd

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var (
	stopUIContainerName string
	stopUIForce         bool
)

var stopUICmd = &cobra.Command{
	Use:   "stop-ui",
	Short: "Stop and remove KECS UI container",
	Long: `Stop and remove KECS UI container.

This command stops the running KECS UI container and removes it.
The container can be restarted with 'kecs start-ui'.

Examples:
  # Stop UI container
  kecs stop-ui

  # Stop specific UI container
  kecs stop-ui --name my-kecs-ui

  # Force stop without graceful shutdown
  kecs stop-ui --force`,
	RunE: runStopUI,
}

func init() {
	RootCmd.AddCommand(stopUICmd)

	stopUICmd.Flags().StringVar(&stopUIContainerName, "name", "kecs-ui", "Container name")
	stopUICmd.Flags().BoolVarP(&stopUIForce, "force", "f", false, "Force stop without graceful shutdown")
}

func runStopUI(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// Find container by name
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("name", stopUIContainerName),
		),
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Printf("No KECS UI container found with name '%s'\n", stopUIContainerName)
		return nil
	}

	containerInfo := containers[0]

	// Stop container if running
	if containerInfo.State == "running" {
		fmt.Printf("Stopping KECS UI container '%s'...\n", stopUIContainerName)
		timeout := 10
		if stopUIForce {
			timeout = 0
		}
		if err := cli.ContainerStop(ctx, containerInfo.ID, container.StopOptions{
			Timeout: &timeout,
		}); err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}

	// Remove container
	fmt.Printf("Removing KECS UI container '%s'...\n", stopUIContainerName)
	if err := cli.ContainerRemove(ctx, containerInfo.ID, container.RemoveOptions{
		Force: stopUIForce,
	}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	fmt.Printf("KECS UI container '%s' stopped and removed successfully\n", stopUIContainerName)
	return nil
}