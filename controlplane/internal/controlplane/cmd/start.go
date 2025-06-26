package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/cobra"
)

const (
	defaultContainerName = "kecs-server"
	defaultImage         = "ghcr.io/nandemo-ya/kecs:latest"
	healthCheckTimeout   = 30 * time.Second
)

var (
	startContainerName string
	startImageName     string
	startApiPort       int
	startAdminPort     int
	startDataDir       string
	startDetach        bool
	startLocalBuild    bool
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start KECS server in a Docker container",
	Long: `Start KECS server in a Docker container running in the background.
This allows you to run KECS without keeping a terminal session open.`,
	RunE: runStart,
}

func init() {
	RootCmd.AddCommand(startCmd)

	startCmd.Flags().StringVar(&startContainerName, "name", defaultContainerName, "Container name")
	startCmd.Flags().StringVar(&startImageName, "image", defaultImage, "Docker image to use")
	startCmd.Flags().IntVar(&startApiPort, "api-port", 8080, "API server port")
	startCmd.Flags().IntVar(&startAdminPort, "admin-port", 8081, "Admin server port")
	startCmd.Flags().StringVar(&startDataDir, "data-dir", "", "Data directory (default: ~/.kecs/data)")
	startCmd.Flags().BoolVarP(&startDetach, "detach", "d", true, "Run container in background")
	startCmd.Flags().BoolVar(&startLocalBuild, "local-build", false, "Build and use local image")
}

func runStart(cmd *cobra.Command, args []string) error {
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

	// Check if container already exists
	filters := filters.NewArgs()
	filters.Add("name", startContainerName)
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) > 0 {
		containerInfo := containers[0]
		if containerInfo.State == "running" {
			fmt.Printf("KECS container '%s' is already running on ports %d (API) and %d (admin)\n", 
				startContainerName, startApiPort, startAdminPort)
			return nil
		}

		// Start stopped container
		fmt.Printf("Starting existing KECS container '%s'...\n", startContainerName)
		if err := cli.ContainerStart(ctx, containerInfo.ID, container.StartOptions{}); err != nil {
			return fmt.Errorf("failed to start container: %w", err)
		}
	} else {
		// Build local image if requested
		if startLocalBuild {
			fmt.Println("Building local KECS image...")
			if err := buildLocalImage(); err != nil {
				return fmt.Errorf("failed to build local image: %w", err)
			}
			startImageName = "kecs:local"
		}

		// Pull image if not using local build
		if !startLocalBuild {
			fmt.Printf("Pulling image %s...\n", startImageName)
			reader, err := cli.ImagePull(ctx, startImageName, image.PullOptions{})
			if err != nil {
				return fmt.Errorf("failed to pull image: %w", err)
			}
			defer reader.Close()
			io.Copy(io.Discard, reader)
		}

		// Create container
		if err := createAndStartContainer(ctx, cli); err != nil {
			return err
		}
	}

	// Wait for health check
	fmt.Printf("Waiting for KECS to be ready...\n")
	if err := waitForHealthCheck(ctx, cli); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	fmt.Printf("KECS started successfully!\n")
	fmt.Printf("API endpoint: http://localhost:%d\n", startApiPort)
	fmt.Printf("Admin endpoint: http://localhost:%d\n", startAdminPort)
	fmt.Printf("Web UI: http://localhost:%d\n", startApiPort)
	
	return nil
}

func createAndStartContainer(ctx context.Context, cli *client.Client) error {
	// Set up data directory
	if startDataDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		startDataDir = filepath.Join(homeDir, ".kecs", "data")
	}

	// Ensure data directory exists
	if err := os.MkdirAll(startDataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Container configuration
	config := &container.Config{
		Image: startImageName,
		ExposedPorts: nat.PortSet{
			nat.Port(fmt.Sprintf("%d/tcp", startApiPort)):   struct{}{},
			nat.Port(fmt.Sprintf("%d/tcp", startAdminPort)): struct{}{},
		},
		Env: []string{
			"KECS_CONTAINER_MODE=true",
			fmt.Sprintf("KECS_DATA_DIR=/data"),
		},
	}

	// Host configuration
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			nat.Port(fmt.Sprintf("%d/tcp", 8080)): []nat.PortBinding{{HostPort: fmt.Sprintf("%d", startApiPort)}},
			nat.Port(fmt.Sprintf("%d/tcp", 8081)): []nat.PortBinding{{HostPort: fmt.Sprintf("%d", startAdminPort)}},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: startDataDir,
				Target: "/data",
			},
		},
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}

	// Create container
	fmt.Printf("Creating KECS container '%s'...\n", startContainerName)
	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, startContainerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	fmt.Printf("Starting KECS container...\n")
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return nil
}

func waitForHealthCheck(ctx context.Context, cli *client.Client) error {
	start := time.Now()
	for {
		if time.Since(start) > healthCheckTimeout {
			return fmt.Errorf("timeout waiting for KECS to be ready")
		}

		// Get container info
		filters := filters.NewArgs()
		filters.Add("name", startContainerName)
		containers, err := cli.ContainerList(ctx, container.ListOptions{
			Filters: filters,
		})
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		if len(containers) == 0 {
			return fmt.Errorf("container not found")
		}

		// Check if container is running
		if containers[0].State != "running" {
			return fmt.Errorf("container is not running")
		}

		// Check admin health endpoint
		cmd := exec.Command("curl", "-s", "-f", fmt.Sprintf("http://localhost:%d/health", startAdminPort))
		if err := cmd.Run(); err == nil {
			return nil
		}

		time.Sleep(1 * time.Second)
	}
}

func buildLocalImage() error {
	// Change to the project root directory
	projectRoot := filepath.Join("..", "..", "..", "..")
	if err := os.Chdir(projectRoot); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

	// Run docker build
	cmd := exec.Command("docker", "build", "-t", "kecs:local", "-f", "Dockerfile", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	return nil
}