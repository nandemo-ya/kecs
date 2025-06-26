package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var (
	statusContainerName string
	statusAll           bool
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show KECS server container status",
	Long:  `Show the status of KECS server container(s).`,
	RunE:  runStatus,
}

func init() {
	RootCmd.AddCommand(statusCmd)

	statusCmd.Flags().StringVar(&statusContainerName, "name", "", "Container name (empty for all KECS containers)")
	statusCmd.Flags().BoolVarP(&statusAll, "all", "a", false, "Show all containers including stopped ones")
}

func runStatus(cmd *cobra.Command, args []string) error {
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

	// Set up filters
	filters := filters.NewArgs()
	if statusContainerName != "" {
		filters.Add("name", statusContainerName)
	} else {
		// Look for containers with names starting with "kecs"
		filters.Add("name", "kecs")
	}

	// List containers
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     statusAll,
		Filters: filters,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Filter to only KECS containers if no specific name was provided
	if statusContainerName == "" {
		var kecsContainers []types.Container
		for _, c := range containers {
			for _, name := range c.Names {
				if strings.HasPrefix(strings.TrimPrefix(name, "/"), "kecs") {
					kecsContainers = append(kecsContainers, c)
					break
				}
			}
		}
		containers = kecsContainers
	}

	if len(containers) == 0 {
		if statusContainerName != "" {
			fmt.Printf("KECS container '%s' not found\n", statusContainerName)
		} else {
			fmt.Println("No KECS containers found")
		}
		return nil
	}

	// Print header
	fmt.Printf("%-20s %-10s %-15s %-20s %-30s\n", "NAME", "STATUS", "CREATED", "PORTS", "IMAGE")
	fmt.Println(strings.Repeat("-", 95))

	// Print container info
	for _, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")
		status := c.State
		if c.Status != "" {
			status = c.Status
		}
		
		created := time.Unix(c.Created, 0).Format("2006-01-02 15:04")
		
		// Format ports
		var ports []string
		for _, p := range c.Ports {
			if p.PublicPort != 0 {
				ports = append(ports, fmt.Sprintf("%d->%d", p.PublicPort, p.PrivatePort))
			}
		}
		portsStr := strings.Join(ports, ", ")
		if portsStr == "" {
			portsStr = "-"
		}

		fmt.Printf("%-20s %-10s %-15s %-20s %-30s\n",
			truncate(name, 20),
			truncate(status, 10),
			created,
			truncate(portsStr, 20),
			truncate(c.Image, 30))
	}

	// Show additional details for single container
	if statusContainerName != "" && len(containers) == 1 {
		c := containers[0]
		fmt.Println("\nDetails:")
		fmt.Printf("  Container ID: %s\n", c.ID[:12])
		fmt.Printf("  State: %s\n", c.State)
		
		// Get more details
		inspect, err := cli.ContainerInspect(ctx, c.ID)
		if err == nil {
			if inspect.State.Running {
				fmt.Printf("  Started: %s\n", inspect.State.StartedAt)
				if startedAt, err := time.Parse(time.RFC3339, inspect.State.StartedAt); err == nil {
				fmt.Printf("  Uptime: %s\n", time.Since(startedAt).Round(time.Second))
			}
			}
			
			// Show mounts
			if len(inspect.Mounts) > 0 {
				fmt.Println("  Mounts:")
				for _, m := range inspect.Mounts {
					fmt.Printf("    %s -> %s\n", m.Source, m.Destination)
				}
			}
			
			// Show environment variables
			envVars := []string{}
			for _, env := range inspect.Config.Env {
				if strings.HasPrefix(env, "KECS_") {
					envVars = append(envVars, env)
				}
			}
			if len(envVars) > 0 {
				fmt.Println("  Environment:")
				for _, env := range envVars {
					fmt.Printf("    %s\n", env)
				}
			}
		}
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}