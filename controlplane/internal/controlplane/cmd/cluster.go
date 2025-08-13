package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

var (
	// Cluster management flags
	clusterK3dImage    string
	clusterDataDir     string
	clusterPort        int
	clusterAdminPort   int
	clusterConfigFile  string
	clusterWaitTimeout time.Duration
)

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage KECS k3d clusters",
	Long:  `Manage k3d clusters that host KECS control plane and AWS services.`,
}

var clusterCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new KECS k3d cluster",
	Long: `Create a new k3d cluster with KECS control plane, LocalStack, and Traefik.
The cluster will be configured with proper networking and volume mounts for data persistence.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runClusterCreate,
}

var clusterDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Delete a KECS k3d cluster",
	Long:  `Delete a k3d cluster and clean up associated resources.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runClusterDelete,
}

var clusterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List KECS k3d clusters",
	Long:  `List all k3d clusters managed by KECS.`,
	RunE:  runClusterList,
}

var clusterInfoCmd = &cobra.Command{
	Use:   "info [name]",
	Short: "Show information about a KECS k3d cluster",
	Long:  `Display detailed information about a k3d cluster including status and endpoints.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runClusterInfo,
}

func init() {
	// Add cluster command to root
	RootCmd.AddCommand(clusterCmd)

	// Add subcommands
	clusterCmd.AddCommand(clusterCreateCmd)
	clusterCmd.AddCommand(clusterDeleteCmd)
	clusterCmd.AddCommand(clusterListCmd)
	clusterCmd.AddCommand(clusterInfoCmd)

	// Cluster create flags
	clusterCreateCmd.Flags().StringVar(&clusterK3dImage, "k3d-image", "rancher/k3s:v1.31.4-k3s1", "K3s image to use")
	clusterCreateCmd.Flags().StringVar(&clusterDataDir, "data-dir", "", "Data directory for persistence (default: ~/.kecs/clusters/<name>/data)")
	clusterCreateCmd.Flags().IntVar(&clusterPort, "api-port", 4566, "Port to expose for AWS API access")
	clusterCreateCmd.Flags().IntVar(&clusterAdminPort, "admin-port", 8081, "Port to expose for admin API access")
	clusterCreateCmd.Flags().StringVar(&clusterConfigFile, "config", "", "Path to KECS configuration file")
	clusterCreateCmd.Flags().DurationVar(&clusterWaitTimeout, "wait-timeout", 5*time.Minute, "Timeout for waiting cluster to be ready")
}

func runClusterCreate(cmd *cobra.Command, args []string) error {
	// Get cluster name
	clusterName := "default"
	if len(args) > 0 {
		clusterName = args[0]
	}

	// Load configuration
	cfg, err := config.LoadConfig(clusterConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Set up data directory
	if clusterDataDir == "" {
		home, _ := os.UserHomeDir()
		clusterDataDir = filepath.Join(home, ".kecs", "clusters", clusterName, "data")
	}

	// Ensure data directory exists
	if err := os.MkdirAll(clusterDataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	fmt.Printf("Creating KECS k3d cluster '%s'...\n", clusterName)
	fmt.Printf("Data directory: %s\n", clusterDataDir)
	fmt.Printf("API port: %d\n", clusterPort)
	fmt.Printf("Admin port: %d\n", clusterAdminPort)

	// Create k3d cluster manager configuration
	clusterConfig := &kubernetes.ClusterManagerConfig{
		Provider:      "k3d",
		ContainerMode: false, // Running on host
		EnableTraefik: cfg.Features.Traefik,
		TraefikPort:   clusterPort, // Use API port for Traefik
	}

	// Create k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(clusterConfig)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	ctx := context.Background()

	// Create the k3d cluster
	if err := manager.CreateCluster(ctx, clusterName); err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	fmt.Printf("Successfully created k3d cluster '%s'\n", clusterName)

	// Wait for cluster to be ready
	fmt.Print("Waiting for cluster to be ready...")
	if err := manager.WaitForClusterReady(ctx, clusterName); err != nil {
		fmt.Println(" timeout!")
		return fmt.Errorf("cluster did not become ready: %w", err)
	}
	fmt.Println(" ready!")

	// Get cluster info
	info, err := manager.GetClusterInfo(ctx, clusterName)
	if err != nil {
		logging.Warn("Failed to get cluster info",
			"error", err)
	} else {
		fmt.Printf("\nCluster Information:\n")
		fmt.Printf("  Name: %s\n", info.Name)
		fmt.Printf("  Status: %s\n", info.Status)
		fmt.Printf("  Nodes: %d\n", info.NodeCount)
		fmt.Printf("  Version: %s\n", info.Version)
	}

	// Get Traefik port if enabled
	if cfg.Features.Traefik {
		port, err := manager.GetTraefikPort(ctx, clusterName)
		if err == nil {
			fmt.Printf("\nAWS API Endpoint: http://localhost:%d\n", port)
		}
	}

	fmt.Printf("\nNext steps:\n")
	fmt.Printf("1. Deploy KECS control plane: kecs deploy control-plane --cluster %s\n", clusterName)
	fmt.Printf("2. Deploy LocalStack: kecs deploy localstack --cluster %s\n", clusterName)
	fmt.Printf("3. Configure kubectl: export KUBECONFIG=%s\n", manager.GetKubeconfigPath(clusterName))

	return nil
}

func runClusterDelete(cmd *cobra.Command, args []string) error {
	// Get cluster name
	clusterName := "default"
	if len(args) > 0 {
		clusterName = args[0]
	}

	fmt.Printf("Deleting KECS k3d cluster '%s'...\n", clusterName)

	// Create k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	ctx := context.Background()

	// Delete the cluster
	if err := manager.DeleteCluster(ctx, clusterName); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	fmt.Printf("Successfully deleted k3d cluster '%s'\n", clusterName)

	// Clean up data directory if it exists
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".kecs", "clusters", clusterName)
	if _, err := os.Stat(dataDir); err == nil {
		fmt.Printf("Cleaning up data directory: %s\n", dataDir)
		if err := os.RemoveAll(dataDir); err != nil {
			logging.Warn("Failed to remove data directory",
				"error", err)
		}
	}

	return nil
}

func runClusterList(cmd *cobra.Command, args []string) error {
	// Create k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	ctx := context.Background()

	// List clusters (we need to implement a method to list all KECS clusters)
	// For now, we'll check known cluster names
	fmt.Println("KECS k3d clusters:")
	fmt.Println("NAME\tSTATUS\tNODES\tVERSION")

	// Check default cluster
	clusterNames := []string{"default"} // TODO: Implement proper cluster listing

	for _, name := range clusterNames {
		exists, err := manager.ClusterExists(ctx, name)
		if err != nil {
			logging.Warn("Failed to check cluster",
				"cluster", name,
				"error", err)
			continue
		}

		if exists {
			info, err := manager.GetClusterInfo(ctx, name)
			if err != nil {
				fmt.Printf("%s\tError\t-\t-\n", name)
			} else {
				fmt.Printf("%s\t%s\t%d\t%s\n", info.Name, info.Status, info.NodeCount, info.Version)
			}
		}
	}

	return nil
}

func runClusterInfo(cmd *cobra.Command, args []string) error {
	// Get cluster name
	clusterName := "default"
	if len(args) > 0 {
		clusterName = args[0]
	}

	// Create k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	ctx := context.Background()

	// Check if cluster exists
	exists, err := manager.ClusterExists(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to check cluster: %w", err)
	}

	if !exists {
		return fmt.Errorf("cluster '%s' does not exist", clusterName)
	}

	// Get cluster info
	info, err := manager.GetClusterInfo(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster info: %w", err)
	}

	fmt.Printf("Cluster: %s\n", info.Name)
	fmt.Printf("Status: %s\n", info.Status)
	fmt.Printf("Provider: %s\n", info.Provider)
	fmt.Printf("Nodes: %d\n", info.NodeCount)
	fmt.Printf("Version: %s\n", info.Version)
	fmt.Printf("\nMetadata:\n")
	for k, v := range info.Metadata {
		fmt.Printf("  %s: %s\n", k, v)
	}

	// Get kubeconfig path
	fmt.Printf("\nKubeconfig: %s\n", manager.GetKubeconfigPath(clusterName))

	// Get Traefik port if available
	port, err := manager.GetTraefikPort(ctx, clusterName)
	if err == nil {
		fmt.Printf("AWS API Port: %d\n", port)
	}

	return nil
}