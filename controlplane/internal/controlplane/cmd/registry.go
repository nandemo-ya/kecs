package cmd

import (
	"context"
	"fmt"

	"github.com/docker/go-connections/nat"
	"github.com/k3d-io/k3d/v5/pkg/client"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

var (
	registryPort int
)

// registryCmd represents the registry command
var registryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Manage k3d registry for local development",
	Long:  `Manage k3d registry for local development. This allows you to build and test images locally without pushing to external registries.`,
}

// registryStartCmd represents the registry start command
var registryStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the k3d registry",
	Long: `Start the k3d registry for local development.

This creates a local Docker registry that can be used with KECS dev mode.
Images pushed to this registry can be used by KECS clusters started with --dev flag.

Example:
  kecs registry start
  make docker-push-dev
  kecs start --dev`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return startRegistry(cmd.Context())
	},
}

// registryStopCmd represents the registry stop command
var registryStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the k3d registry",
	Long:  `Stop the k3d registry.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return stopRegistry(cmd.Context())
	},
}

// registryStatusCmd represents the registry status command
var registryStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show k3d registry status",
	Long:  `Show the status of the k3d registry.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return showRegistryStatus(cmd.Context())
	},
}

func init() {
	RootCmd.AddCommand(registryCmd)
	registryCmd.AddCommand(registryStartCmd)
	registryCmd.AddCommand(registryStopCmd)
	registryCmd.AddCommand(registryStatusCmd)

	registryStartCmd.Flags().IntVar(&registryPort, "port", 5000, "Registry port")
}

func startRegistry(ctx context.Context) error {
	registryName := "k3d-kecs-registry.localhost"
	
	// Initialize runtime
	runtime, err := runtimes.GetRuntime("docker")
	if err != nil {
		return fmt.Errorf("failed to get docker runtime: %w", err)
	}

	// Check if registry already exists
	existingRegistry, err := client.RegistryGet(ctx, runtime, registryName)
	if err == nil && existingRegistry != nil {
		// Registry exists, check if it's running
		nodes, err := runtime.GetNodesByLabel(ctx, map[string]string{k3d.LabelRole: string(k3d.RegistryRole)})
		if err == nil {
			for _, node := range nodes {
				// Check both formats: "k3d-<name>" and just "<name>"
				if node.Name == fmt.Sprintf("k3d-%s", registryName) || node.Name == registryName {
					if node.State.Running {
						logging.Info("Registry is already running", "name", registryName, "port", registryPort)
						return nil
					}
					
					// Start the registry
					logging.Info("Starting existing registry", "name", registryName)
					if err := runtime.StartNode(ctx, node); err != nil {
						return fmt.Errorf("failed to start registry: %w", err)
					}
					logging.Info("Registry started successfully", "name", registryName, "port", registryPort)
					printRegistryInfo()
					return nil
				}
			}
		}
	}

	// Create new registry
	logging.Info("Creating k3d registry", "name", registryName, "port", registryPort)
	
	registry := &k3d.Registry{
		Host:  registryName,
		Image: "docker.io/library/registry:2",
		ExposureOpts: k3d.ExposureOpts{
			Host: "0.0.0.0",
			PortMapping: nat.PortMapping{
				Port: nat.Port("5000/tcp"),
				Binding: nat.PortBinding{
					HostIP:   "0.0.0.0",
					HostPort: fmt.Sprintf("%d", registryPort),
				},
			},
		},
	}
	
	// Create the registry
	registryNode, err := client.RegistryCreate(ctx, runtime, registry)
	if err != nil {
		return fmt.Errorf("failed to create registry: %w", err)
	}
	
	// Start the registry
	logging.Info("Starting registry", "name", registryName)
	if err := runtime.StartNode(ctx, registryNode); err != nil {
		logging.Warn("Failed to start registry after creation", "error", err)
	}
	
	logging.Info("Registry created and started successfully", "name", registryName, "port", registryPort)
	printRegistryInfo()
	return nil
}

func stopRegistry(ctx context.Context) error {
	registryName := "k3d-kecs-registry.localhost"
	
	// Initialize runtime
	runtime, err := runtimes.GetRuntime("docker")
	if err != nil {
		return fmt.Errorf("failed to get docker runtime: %w", err)
	}

	// Get registry node
	nodes, err := runtime.GetNodesByLabel(ctx, map[string]string{k3d.LabelRole: string(k3d.RegistryRole)})
	if err != nil {
		return fmt.Errorf("failed to get registry nodes: %w", err)
	}

	for _, node := range nodes {
		// Check both formats: "k3d-<name>" and just "<name>"
		if node.Name == fmt.Sprintf("k3d-%s", registryName) || node.Name == registryName {
			logging.Info("Stopping registry", "name", registryName)
			if err := runtime.StopNode(ctx, node); err != nil {
				return fmt.Errorf("failed to stop registry: %w", err)
			}
			logging.Info("Registry stopped successfully", "name", registryName)
			return nil
		}
	}

	logging.Warn("Registry not found", "name", registryName)
	return nil
}

func showRegistryStatus(ctx context.Context) error {
	registryName := "k3d-kecs-registry.localhost"
	
	// Initialize runtime
	runtime, err := runtimes.GetRuntime("docker")
	if err != nil {
		return fmt.Errorf("failed to get docker runtime: %w", err)
	}

	// Check if registry exists
	existingRegistry, err := client.RegistryGet(ctx, runtime, registryName)
	if err != nil || existingRegistry == nil {
		fmt.Println("Registry Status: Not Created")
		fmt.Println("\nTo create the registry, run:")
		fmt.Println("  kecs registry start")
		return nil
	}

	// Get registry node status
	nodes, err := runtime.GetNodesByLabel(ctx, map[string]string{k3d.LabelRole: string(k3d.RegistryRole)})
	if err != nil {
		return fmt.Errorf("failed to get registry nodes: %w", err)
	}

	for _, node := range nodes {
		// Check both formats: "k3d-<name>" and just "<name>"
		if node.Name == fmt.Sprintf("k3d-%s", registryName) || node.Name == registryName {
			status := "Stopped"
			if node.State.Running {
				status = "Running"
			}
			
			fmt.Printf("Registry Status: %s\n", status)
			fmt.Printf("Registry Name: %s\n", registryName)
			fmt.Printf("Registry Port: %d\n", registryPort)
			
			if status == "Running" {
				fmt.Println("\nRegistry is ready for use!")
				fmt.Println("Push images with: make docker-push-dev")
				fmt.Println("Start KECS with: kecs start --dev")
			} else {
				fmt.Println("\nTo start the registry, run:")
				fmt.Println("  kecs registry start")
			}
			return nil
		}
	}

	fmt.Println("Registry Status: Unknown")
	return nil
}

func printRegistryInfo() {
	fmt.Println("\n✅ Registry is ready!")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Ensure /etc/hosts contains:")
	fmt.Printf("   127.0.0.1 k3d-kecs-registry.localhost\n")
	fmt.Println("\n2. Build and push images:")
	fmt.Println("   make docker-push-dev")
	fmt.Println("\n3. Start KECS in dev mode:")
	fmt.Println("   kecs start --dev")
}