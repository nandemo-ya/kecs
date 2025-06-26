package cmd

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/spf13/cobra"
)

var (
	startUIContainerName string
	startUIPort          int
	startUIAPIEndpoint   string
	startUIImage         string
	startUIDetach        bool
	startUIDataDir       string
	startUIAutoPort      bool
	startUIEnv           []string
	startUILabels        []string
)

var startUICmd = &cobra.Command{
	Use:   "start-ui",
	Short: "Start KECS Web UI in a Docker container",
	Long: `Start KECS Web UI in a Docker container.

The Web UI runs separately from the API server and connects to it via the API endpoint.
This allows for flexible deployment options and better resource management.

Examples:
  # Start UI with default settings
  kecs start-ui

  # Start UI with custom API endpoint
  kecs start-ui --api-endpoint http://localhost:8080

  # Start UI on custom port
  kecs start-ui --port 3000

  # Start with auto port assignment
  kecs start-ui --auto-port`,
	RunE: runStartUI,
}

func init() {
	RootCmd.AddCommand(startUICmd)

	startUICmd.Flags().StringVar(&startUIContainerName, "name", "kecs-ui", "Container name")
	startUICmd.Flags().IntVar(&startUIPort, "port", 3000, "UI server port")
	startUICmd.Flags().StringVar(&startUIAPIEndpoint, "api-endpoint", "http://localhost:8080", "KECS API endpoint URL")
	startUICmd.Flags().StringVar(&startUIImage, "image", "ghcr.io/nandemo-ya/kecs-ui:latest", "Docker image to use")
	startUICmd.Flags().BoolVarP(&startUIDetach, "detach", "d", true, "Run container in background")
	startUICmd.Flags().StringVar(&startUIDataDir, "data-dir", "", "Data directory for UI assets (optional)")
	startUICmd.Flags().BoolVar(&startUIAutoPort, "auto-port", false, "Automatically find available ports")
	startUICmd.Flags().StringArrayVar(&startUIEnv, "env", []string{}, "Set environment variables")
	startUICmd.Flags().StringArrayVar(&startUILabels, "label", []string{}, "Set metadata labels")
}

func runStartUI(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// Check if container already exists
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("name", startUIContainerName),
		),
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) > 0 {
		// Container exists
		containerInfo := containers[0]
		if containerInfo.State == "running" {
			fmt.Printf("KECS UI container '%s' is already running\n", startUIContainerName)
			fmt.Printf("UI available at: http://localhost:%d\n", startUIPort)
			return nil
		}

		// Remove the stopped container
		fmt.Printf("Removing existing container '%s'...\n", startUIContainerName)
		if err := cli.ContainerRemove(ctx, containerInfo.ID, container.RemoveOptions{
			Force: true,
		}); err != nil {
			return fmt.Errorf("failed to remove existing container: %w", err)
		}
	}

	// Auto-assign port if requested
	if startUIAutoPort {
		port, err := findAvailableUIPort(startUIPort)
		if err != nil {
			return fmt.Errorf("failed to find available port: %w", err)
		}
		startUIPort = port
		fmt.Printf("Auto-assigned UI port: %d\n", startUIPort)
	} else {
		// Check if port is available
		if !isUIPortAvailable(startUIPort) {
			return fmt.Errorf("port %d is already in use", startUIPort)
		}
	}

	// Pull image if not exists
	fmt.Printf("Pulling image %s...\n", startUIImage)
	reader, err := cli.ImagePull(ctx, startUIImage, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()
	io.Copy(io.Discard, reader)

	// Prepare environment variables
	env := []string{
		fmt.Sprintf("KECS_API_ENDPOINT=%s", startUIAPIEndpoint),
		"NODE_ENV=production",
	}
	env = append(env, startUIEnv...)

	// Prepare labels
	labels := map[string]string{
		"com.kecs.component": "ui",
		"com.kecs.name":      startUIContainerName,
	}
	for _, label := range startUILabels {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		}
	}

	// Create container configuration
	containerConfig := &container.Config{
		Image: startUIImage,
		Env:   env,
		ExposedPorts: nat.PortSet{
			"80/tcp": struct{}{},
		},
		Labels: labels,
	}

	// Prepare mounts
	var mounts []mount.Mount
	if startUIDataDir != "" {
		// Expand path
		if strings.HasPrefix(startUIDataDir, "~") {
			home, _ := os.UserHomeDir()
			startUIDataDir = filepath.Join(home, startUIDataDir[1:])
		}
		absDataDir, _ := filepath.Abs(startUIDataDir)

		// Create directory if it doesn't exist
		if err := os.MkdirAll(absDataDir, 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}

		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: absDataDir,
			Target: "/usr/share/nginx/html/assets",
		})
	}

	// Create host configuration
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"80/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: strconv.Itoa(startUIPort),
				},
			},
		},
		Mounts:      mounts,
		AutoRemove:  false,
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
	}

	// Create the container
	fmt.Printf("Creating KECS UI container...\n")
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, startUIContainerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start the container
	fmt.Printf("Starting KECS UI container...\n")
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	fmt.Printf("KECS UI container '%s' started successfully\n", startUIContainerName)
	fmt.Printf("UI available at: http://localhost:%d\n", startUIPort)
	fmt.Printf("Connected to API: %s\n", startUIAPIEndpoint)

	// If not detached, wait for container and handle signals
	if !startUIDetach {
		fmt.Println("\nRunning in foreground mode. Press Ctrl+C to stop...")

		// Set up signal handling
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Wait for container or signal
		statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("error waiting for container: %w", err)
			}
		case status := <-statusCh:
			return fmt.Errorf("container exited with status %d", status.StatusCode)
		case <-sigChan:
			fmt.Println("\nReceived interrupt signal, stopping container...")
			timeout := 10
			if err := cli.ContainerStop(ctx, resp.ID, container.StopOptions{
				Timeout: &timeout,
			}); err != nil {
				return fmt.Errorf("failed to stop container: %w", err)
			}
			fmt.Println("Container stopped")
		}
	}

	return nil
}

// waitForUIReady waits for the UI server to be ready
func waitForUIReady(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf("http://localhost:%d", port)

	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("UI server did not become ready within %v", timeout)
}

// Helper function to check if port is available
func isUIPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// Helper function to find available port starting from base
func findAvailableUIPort(basePort int) (int, error) {
	for port := basePort; port < basePort+100; port++ {
		if isUIPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found in range %d-%d", basePort, basePort+99)
}