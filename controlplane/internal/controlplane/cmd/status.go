package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/runtime"
)

var (
	statusContainerName string
	statusAll           bool
	statusRuntime       string
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
	statusCmd.Flags().StringVar(&statusRuntime, "runtime", "", "Container runtime to use (docker, containerd, or auto)")
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get runtime manager
	manager := runtime.GetDefaultManager()
	
	// Set runtime if specified
	if statusRuntime != "" {
		switch statusRuntime {
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
			return fmt.Errorf("unknown runtime: %s", statusRuntime)
		}
	}

	// Get runtime
	rt, err := manager.GetRuntime()
	if err != nil {
		return fmt.Errorf("failed to get container runtime: %w", err)
	}

	fmt.Printf("Using container runtime: %s\n", rt.Name())

	// Set up filters
	opts := runtime.ListContainersOptions{
		All: statusAll,
		Labels: map[string]string{
			"com.kecs.managed": "true",
		},
	}

	// Add specific container name filter if provided
	if statusContainerName != "" {
		opts.Names = []string{statusContainerName}
	}

	// List containers
	containers, err := rt.ListContainers(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		if statusContainerName != "" {
			fmt.Printf("No KECS container found with name '%s'\n", statusContainerName)
		} else {
			fmt.Println("No KECS containers found")
		}
		return nil
	}

	// Display header
	fmt.Printf("%-20s %-15s %-25s %-30s %-20s\n", "NAME", "STATUS", "CREATED", "IMAGE", "PORTS")
	fmt.Println(strings.Repeat("-", 120))

	// Display containers
	for _, c := range containers {
		// Format ports
		var ports []string
		for _, p := range c.Ports {
			ports = append(ports, fmt.Sprintf("%d:%d/%s", p.HostPort, p.ContainerPort, p.Protocol))
		}
		portsStr := strings.Join(ports, ", ")
		if len(portsStr) > 20 {
			portsStr = portsStr[:17] + "..."
		}

		// Format created time
		created := time.Since(c.Created).Round(time.Second)
		var createdStr string
		if created < time.Minute {
			createdStr = fmt.Sprintf("%d seconds ago", int(created.Seconds()))
		} else if created < time.Hour {
			createdStr = fmt.Sprintf("%d minutes ago", int(created.Minutes()))
		} else if created < 24*time.Hour {
			createdStr = fmt.Sprintf("%d hours ago", int(created.Hours()))
		} else {
			createdStr = fmt.Sprintf("%d days ago", int(created.Hours()/24))
		}

		// Truncate image name if too long
		image := c.Image
		if len(image) > 30 {
			image = "..." + image[len(image)-27:]
		}

		fmt.Printf("%-20s %-15s %-25s %-30s %-20s\n",
			c.Name,
			c.State,
			createdStr,
			image,
			portsStr,
		)
	}

	return nil
}