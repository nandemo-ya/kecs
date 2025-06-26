package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var (
	logsContainerName string
	logsFollow        bool
	logsTail          string
	logsTimestamps    bool
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show logs from KECS server container",
	Long: `Display logs from KECS server container.
Use -f/--follow to stream logs in real-time.`,
	RunE: runLogs,
}

func init() {
	RootCmd.AddCommand(logsCmd)

	logsCmd.Flags().StringVar(&logsContainerName, "name", defaultContainerName, "Container name")
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().StringVar(&logsTail, "tail", "all", "Number of lines to show from the end of the logs")
	logsCmd.Flags().BoolVarP(&logsTimestamps, "timestamps", "t", false, "Show timestamps")
}

func runLogs(cmd *cobra.Command, args []string) error {
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
	filters.Add("name", logsContainerName)
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		return fmt.Errorf("KECS container '%s' not found", logsContainerName)
	}

	containerInfo := containers[0]

	// Get logs
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     logsFollow,
		Tail:       logsTail,
		Timestamps: logsTimestamps,
	}

	reader, err := cli.ContainerLogs(ctx, containerInfo.ID, options)
	if err != nil {
		return fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()

	// Copy logs to stdout
	_, err = io.Copy(os.Stdout, reader)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read logs: %w", err)
	}

	return nil
}