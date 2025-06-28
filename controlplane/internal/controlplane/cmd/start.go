package cmd

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/runtime"
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
	startConfigFile    string
	startAutoPort      bool
	startRuntime       string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start KECS server in a container",
	Long: `Start KECS server in a container running in the background.
This command supports both Docker and containerd runtimes.`,
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
	startCmd.Flags().StringVar(&startConfigFile, "config", "", "Path to configuration file")
	startCmd.Flags().BoolVar(&startAutoPort, "auto-port", false, "Automatically find available ports")
	startCmd.Flags().StringVar(&startRuntime, "runtime", "", "Container runtime to use (docker, containerd, or auto)")
}

func runStart(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get runtime manager
	manager := runtime.GetDefaultManager()
	
	// Set runtime if specified
	if startRuntime != "" {
		switch startRuntime {
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
			return fmt.Errorf("unknown runtime: %s", startRuntime)
		}
	}

	// Get runtime
	rt, err := manager.GetRuntime()
	if err != nil {
		return fmt.Errorf("failed to get container runtime: %w", err)
	}

	fmt.Printf("Using container runtime: %s\n", rt.Name())

	// Check if container already exists
	containers, err := rt.ListContainers(ctx, runtime.ListContainersOptions{
		All:   true,
		Names: []string{startContainerName},
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) > 0 {
		container := containers[0]
		if container.State == "running" {
			fmt.Printf("KECS server '%s' is already running\n", startContainerName)
			fmt.Printf("API endpoint: http://localhost:%d\n", startApiPort)
			fmt.Printf("Admin endpoint: http://localhost:%d\n", startAdminPort)
			return nil
		}

		// Remove the stopped container
		fmt.Printf("Removing existing container '%s'...\n", startContainerName)
		if err := rt.RemoveContainer(ctx, container.ID, true); err != nil {
			return fmt.Errorf("failed to remove existing container: %w", err)
		}
	}

	// Set up data directory
	if startDataDir == "" {
		home, _ := os.UserHomeDir()
		startDataDir = filepath.Join(home, ".kecs", "data")
	}

	// Expand tilde in path
	if strings.HasPrefix(startDataDir, "~") {
		home, _ := os.UserHomeDir()
		startDataDir = filepath.Join(home, startDataDir[1:])
	}

	// Convert to absolute path
	absDataDir, err := filepath.Abs(startDataDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(absDataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Auto-assign ports if requested
	if startAutoPort {
		// Try to find available ports starting from the specified ones
		availablePorts, err := findAvailablePorts(startApiPort, startAdminPort, make(map[int]string))
		if err != nil {
			return fmt.Errorf("failed to find available ports: %w", err)
		}
		startApiPort = availablePorts[0]
		startAdminPort = availablePorts[1]
		fmt.Printf("Auto-assigned ports - API: %d, Admin: %d\n", startApiPort, startAdminPort)
	} else {
		// Check if ports are available
		if err := checkPortsAvailable(startApiPort, startAdminPort); err != nil {
			return err
		}
	}

	// Build local image if requested
	imageName := startImageName
	if startLocalBuild {
		fmt.Println("Building local image...")
		if err := buildLocalImage(); err != nil {
			return fmt.Errorf("failed to build local image: %w", err)
		}
		imageName = "kecs:local"
	}

	// Pull image if not local build
	if !startLocalBuild {
		fmt.Printf("Pulling image %s...\n", imageName)
		reader, err := rt.PullImage(ctx, imageName, runtime.PullImageOptions{})
		if err != nil {
			return fmt.Errorf("failed to pull image: %w", err)
		}
		defer reader.Close()
		io.Copy(io.Discard, reader)
	}

	// Prepare environment variables
	env := []string{
		"KECS_CONTAINER_MODE=true",
		fmt.Sprintf("KECS_DATA_DIR=%s", "/data"),
		"KECS_LOG_LEVEL=info",
		"KECS_SECURITY_ACKNOWLEDGED=true",  // Skip security disclaimer in container
	}

	// Add config file if specified
	configMount := []runtime.Mount{}
	if startConfigFile != "" {
		absConfigPath, err := filepath.Abs(startConfigFile)
		if err != nil {
			return fmt.Errorf("failed to get absolute config path: %w", err)
		}
		env = append(env, "KECS_CONFIG_FILE=/config/kecs.yaml")
		configMount = append(configMount, runtime.Mount{
			Type:   "bind",
			Source: absConfigPath,
			Target: "/config/kecs.yaml",
			ReadOnly: true,
		})
	}

	// Add docker socket group (typically 0/root on most systems)
	// This allows the container to access the Docker socket
	groupAdd := []string{"0"}

	// Create container configuration
	containerConfig := &runtime.ContainerConfig{
		Name:  startContainerName,
		Image: imageName,
		Cmd:   []string{"server"}, // Run the server command
		Env:   env,
		Labels: map[string]string{
			"com.kecs.managed": "true",
			"com.kecs.name":    startContainerName,
		},
		GroupAdd: groupAdd,
		Ports: []runtime.PortBinding{
			{
				ContainerPort: 8080,
				HostPort:      uint16(startApiPort),
				Protocol:      "tcp",
				HostIP:        "0.0.0.0",
			},
			{
				ContainerPort: 8081,
				HostPort:      uint16(startAdminPort),
				Protocol:      "tcp",
				HostIP:        "0.0.0.0",
			},
		},
		Mounts: append([]runtime.Mount{
			{
				Type:   "bind",
				Source: absDataDir,
				Target: "/data",
			},
			{
				Type:   "bind",
				Source: "/var/run/docker.sock",
				Target: "/var/run/docker.sock",
			},
		}, configMount...),
		RestartPolicy: runtime.RestartPolicy{
			Name: "unless-stopped",
		},
		HealthCheck: &runtime.HealthCheck{
			Test:        []string{"CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8081/health"},
			Interval:    30 * time.Second,
			Timeout:     10 * time.Second,
			StartPeriod: 10 * time.Second,
			Retries:     3,
		},
	}

	// Create the container
	fmt.Printf("Creating KECS server container...\n")
	container, err := rt.CreateContainer(ctx, containerConfig)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start the container
	fmt.Printf("Starting KECS server...\n")
	if err := rt.StartContainer(ctx, container.ID); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	fmt.Printf("KECS server '%s' started successfully\n", startContainerName)
	fmt.Printf("API endpoint: http://localhost:%d\n", startApiPort)
	fmt.Printf("Admin endpoint: http://localhost:%d\n", startAdminPort)
	fmt.Printf("Data directory: %s\n", absDataDir)

	// Wait for server to be ready
	fmt.Print("Waiting for server to be ready...")
	if err := waitForServerReady(fmt.Sprintf("http://localhost:%d/health", startAdminPort), healthCheckTimeout); err != nil {
		fmt.Println(" timeout!")
		fmt.Println("Server may still be starting. Check logs with: kecs logs")
		return nil
	}
	fmt.Println(" ready!")

	// If not detached, wait for signals
	if !startDetach {
		fmt.Println("\nRunning in foreground mode. Press Ctrl+C to stop...")

		// Set up signal handling
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Wait for signal
		<-sigChan

		fmt.Println("\nReceived interrupt signal, stopping container...")
		timeout := 10
		if err := rt.StopContainer(ctx, container.ID, &timeout); err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}

		if err := rt.RemoveContainer(ctx, container.ID, false); err != nil {
			return fmt.Errorf("failed to remove container: %w", err)
		}

		fmt.Println("Container stopped and removed")
	}

	return nil
}

func buildLocalImage() error {
	// Determine the path to the controlplane directory
	execPath, err := os.Executable()
	if err != nil {
		// Fallback: try to find the directory relative to current directory
		if _, err := os.Stat("Dockerfile"); err == nil {
			// We're already in the controlplane directory
			cmd := exec.Command("docker", "build", "-t", "kecs:local", ".")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}
		// Try parent directory
		if _, err := os.Stat("controlplane/Dockerfile"); err == nil {
			cmd := exec.Command("docker", "build", "-t", "kecs:local", "controlplane")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}
		return fmt.Errorf("cannot find Dockerfile")
	}

	// Navigate from the binary location to find the source directory
	dir := filepath.Dir(execPath)
	for {
		// Check if we found the controlplane directory
		dockerfilePath := filepath.Join(dir, "controlplane", "Dockerfile")
		if _, err := os.Stat(dockerfilePath); err == nil {
			cmd := exec.Command("docker", "build", "-t", "kecs:local", filepath.Join(dir, "controlplane"))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}

		// Check if we're in the controlplane directory
		dockerfilePath = filepath.Join(dir, "Dockerfile")
		if _, err := os.Stat(dockerfilePath); err == nil {
			cmd := exec.Command("docker", "build", "-t", "kecs:local", dir)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// We've reached the root
			break
		}
		dir = parent
	}

	return fmt.Errorf("cannot find Dockerfile in any parent directory")
}

// Helper functions

func waitForServerReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 5 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("server did not become ready within %v", timeout)
}

func checkPortsAvailable(apiPort, adminPort int) error {
	ports := []int{apiPort, adminPort}
	for _, port := range ports {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return fmt.Errorf("port %d is already in use", port)
		}
		ln.Close()
	}
	return nil
}

func findAvailablePorts(startApiPort, startAdminPort int, usedPorts map[int]string) ([]int, error) {
	var ports []int
	
	// Try to find available API port
	apiPort := startApiPort
	for i := 0; i < 100; i++ {
		if _, used := usedPorts[apiPort]; !used {
			ln, err := net.Listen("tcp", fmt.Sprintf(":%d", apiPort))
			if err == nil {
				ln.Close()
				ports = append(ports, apiPort)
				break
			}
		}
		apiPort++
	}
	
	if len(ports) == 0 {
		return nil, fmt.Errorf("no available API port found starting from %d", startApiPort)
	}
	
	// Try to find available admin port
	adminPort := startAdminPort
	for i := 0; i < 100; i++ {
		if _, used := usedPorts[adminPort]; !used && adminPort != ports[0] {
			ln, err := net.Listen("tcp", fmt.Sprintf(":%d", adminPort))
			if err == nil {
				ln.Close()
				ports = append(ports, adminPort)
				break
			}
		}
		adminPort++
	}
	
	if len(ports) == 1 {
		return nil, fmt.Errorf("no available admin port found starting from %d", startAdminPort)
	}
	
	return ports, nil
}