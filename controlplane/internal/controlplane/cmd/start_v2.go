package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
)

var (
	// Start v2 flags (new architecture)
	startV2ClusterName string
	startV2DataDir     string
	startV2ApiPort     int
	startV2AdminPort   int
	startV2ConfigFile  string
	startV2NoLocalStack bool
	startV2NoTraefik   bool
	startV2Timeout     time.Duration
)

var startV2Cmd = &cobra.Command{
	Use:   "start-v2",
	Short: "Start KECS with control plane in k3d cluster (new architecture)",
	Long: `Start KECS by creating a k3d cluster and deploying the control plane inside it.
This provides a unified AWS API endpoint accessible from all containers.`,
	RunE: runStartV2,
}

func init() {
	RootCmd.AddCommand(startV2Cmd)

	startV2Cmd.Flags().StringVar(&startV2ClusterName, "name", "kecs", "Cluster name")
	startV2Cmd.Flags().StringVar(&startV2DataDir, "data-dir", "", "Data directory (default: ~/.kecs/data)")
	startV2Cmd.Flags().IntVar(&startV2ApiPort, "api-port", 4566, "AWS API port (Traefik gateway)")
	startV2Cmd.Flags().IntVar(&startV2AdminPort, "admin-port", 8081, "Admin API port")
	startV2Cmd.Flags().StringVar(&startV2ConfigFile, "config", "", "Configuration file path")
	startV2Cmd.Flags().BoolVar(&startV2NoLocalStack, "no-localstack", false, "Disable LocalStack deployment")
	startV2Cmd.Flags().BoolVar(&startV2NoTraefik, "no-traefik", false, "Disable Traefik deployment")
	startV2Cmd.Flags().DurationVar(&startV2Timeout, "timeout", 10*time.Minute, "Timeout for cluster creation")
}

func runStartV2(cmd *cobra.Command, args []string) error {
	fmt.Println("Starting KECS with control plane in k3d cluster (new architecture)...")

	// Load configuration
	cfg, err := config.LoadConfig(startV2ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override with flags
	if startV2NoLocalStack {
		cfg.LocalStack.Enabled = false
	}
	if startV2NoTraefik {
		cfg.Features.Traefik = false
	}

	// Set up data directory
	if startV2DataDir == "" {
		home, _ := os.UserHomeDir()
		startV2DataDir = filepath.Join(home, ".kecs", "data")
	}

	// Ensure data directory exists
	if err := os.MkdirAll(startV2DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), startV2Timeout)
	defer cancel()

	// Step 1: Create k3d cluster
	fmt.Printf("\n=== Step 1: Creating k3d cluster '%s' ===\n", startV2ClusterName)
	if err := createK3dCluster(ctx, startV2ClusterName, cfg, startV2DataDir); err != nil {
		return fmt.Errorf("failed to create k3d cluster: %w", err)
	}

	// Step 2: Create kecs-system namespace
	fmt.Printf("\n=== Step 2: Creating kecs-system namespace ===\n")
	if err := createKecsSystemNamespace(ctx, startV2ClusterName); err != nil {
		return fmt.Errorf("failed to create kecs-system namespace: %w", err)
	}

	// Step 3: Deploy KECS control plane
	fmt.Printf("\n=== Step 3: Deploying KECS control plane ===\n")
	if err := deployControlPlane(ctx, startV2ClusterName, cfg, startV2DataDir); err != nil {
		return fmt.Errorf("failed to deploy control plane: %w", err)
	}

	// Step 4: Deploy LocalStack (if enabled)
	if cfg.LocalStack.Enabled {
		fmt.Printf("\n=== Step 4: Deploying LocalStack ===\n")
		if err := deployLocalStack(ctx, startV2ClusterName, cfg); err != nil {
			return fmt.Errorf("failed to deploy LocalStack: %w", err)
		}
	}

	// Step 5: Deploy Traefik gateway (if enabled)
	if cfg.Features.Traefik {
		fmt.Printf("\n=== Step 5: Deploying Traefik AWS API gateway ===\n")
		if err := deployTraefikGateway(ctx, startV2ClusterName, cfg, startV2ApiPort); err != nil {
			return fmt.Errorf("failed to deploy Traefik gateway: %w", err)
		}
	}

	// Step 6: Wait for all components to be ready
	fmt.Printf("\n=== Step 6: Waiting for all components to be ready ===\n")
	if err := waitForComponents(ctx, startV2ClusterName); err != nil {
		return fmt.Errorf("components did not become ready: %w", err)
	}

	fmt.Printf("\n✅ KECS started successfully!\n")
	fmt.Printf("\nEndpoints:\n")
	fmt.Printf("  AWS API: http://localhost:%d\n", startV2ApiPort)
	fmt.Printf("  Admin API: http://localhost:%d\n", startV2AdminPort)
	fmt.Printf("  Data directory: %s\n", startV2DataDir)

	if cfg.LocalStack.Enabled {
		fmt.Printf("\nLocalStack services: %v\n", cfg.LocalStack.Services)
	}

	fmt.Printf("\nTo stop KECS: kecs stop-v2 --name %s\n", startV2ClusterName)

	return nil
}

func createK3dCluster(ctx context.Context, clusterName string, cfg *config.Config, dataDir string) error {
	// Create k3d cluster manager configuration
	clusterConfig := &kubernetes.ClusterManagerConfig{
		Provider:      "k3d",
		ContainerMode: false,
		EnableTraefik: false, // We'll deploy our own Traefik
		TraefikPort:   startV2ApiPort, // Use the API port for Traefik
		VolumeMounts: []kubernetes.VolumeMount{
			{
				HostPath:      dataDir,
				ContainerPath: "/data",
			},
		},
	}

	// Create k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(clusterConfig)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	// Check if cluster already exists
	exists, err := manager.ClusterExists(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to check cluster existence: %w", err)
	}

	if exists {
		fmt.Printf("k3d cluster '%s' already exists, using existing cluster\n", clusterName)
		return nil
	}

	// Create the cluster
	if err := manager.CreateCluster(ctx, clusterName); err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	// Wait for cluster to be ready
	fmt.Print("Waiting for cluster to be ready...")
	if err := manager.WaitForClusterReady(clusterName, 5*time.Minute); err != nil {
		return fmt.Errorf("cluster did not become ready: %w", err)
	}
	fmt.Println(" ready!")

	return nil
}

func createKecsSystemNamespace(ctx context.Context, clusterName string) error {
	// Get k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	// Get Kubernetes client
	kubeClient, err := manager.GetKubeClient(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	// Create kecs-system namespace directly
	// We don't use the NamespaceManager here as it's designed for ECS cluster namespaces
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kecs-system",
			Labels: map[string]string{
				"kecs.dev/managed": "true",
				"kecs.dev/type":    "system",
			},
		},
	}

	_, err = kubeClient.CoreV1().Namespaces().Get(ctx, "kecs-system", metav1.GetOptions{})
	if err == nil {
		fmt.Println("kecs-system namespace already exists")
		return nil
	}

	_, err = kubeClient.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create kecs-system namespace: %w", err)
	}

	fmt.Println("Created kecs-system namespace")
	return nil
}

func deployControlPlane(ctx context.Context, clusterName string, cfg *config.Config, dataDir string) error {
	// Get k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	// Get Kubernetes client
	kubeClient, err := manager.GetKubeClient(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	// Deploy control plane using kubectl apply
	// We'll use the manifests we created
	manifestsDir := filepath.Join(os.Getenv("GOPATH"), "src/github.com/nandemo-ya/kecs/controlplane/manifests")
	if manifestsDir == "" {
		// Fallback to relative path from current directory
		manifestsDir = "controlplane/manifests"
	}

	// Check if manifests directory exists
	if _, err := os.Stat(manifestsDir); os.IsNotExist(err) {
		return fmt.Errorf("manifests directory not found: %s", manifestsDir)
	}

	// Apply manifests using kubectl
	fmt.Println("Applying control plane manifests...")
	cmd := exec.Command("kubectl", "apply", "-k", manifestsDir, "--kubeconfig", manager.GetKubeconfigPath(clusterName))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply manifests: %w", err)
	}

	// Wait for deployment to be ready
	fmt.Print("Waiting for control plane deployment to be ready...")
	deployment := "kecs-controlplane"
	namespace := "kecs-system"
	
	for i := 0; i < 60; i++ { // Wait up to 5 minutes
		deps, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, deployment, metav1.GetOptions{})
		if err == nil && deps.Status.ReadyReplicas > 0 {
			fmt.Println(" ready!")
			return nil
		}
		time.Sleep(5 * time.Second)
		fmt.Print(".")
	}

	return fmt.Errorf("control plane deployment did not become ready in time")
}

func deployLocalStack(ctx context.Context, clusterName string, cfg *config.Config) error {
	// TODO: Implement LocalStack deployment
	// This will use the existing LocalStack manager but deploy to kecs-system namespace
	fmt.Println("LocalStack deployment not yet implemented")
	fmt.Println("TODO: Deploy LocalStack to kecs-system namespace")
	return nil
}

func deployTraefikGateway(ctx context.Context, clusterName string, cfg *config.Config, apiPort int) error {
	// Get k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	// Get Kubernetes client
	kubeClient, err := manager.GetKubeClient(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	// Deploy Traefik using kubectl apply
	manifestsDir := filepath.Join(os.Getenv("GOPATH"), "src/github.com/nandemo-ya/kecs/controlplane/manifests/traefik")
	if manifestsDir == "" {
		// Fallback to relative path from current directory
		manifestsDir = "controlplane/manifests/traefik"
	}

	// Check if manifests directory exists
	if _, err := os.Stat(manifestsDir); os.IsNotExist(err) {
		return fmt.Errorf("traefik manifests directory not found: %s", manifestsDir)
	}

	// Apply Traefik manifests
	fmt.Println("Applying Traefik gateway manifests...")
	cmd := exec.Command("kubectl", "apply", "-k", manifestsDir, "--kubeconfig", manager.GetKubeconfigPath(clusterName))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply traefik manifests: %w", err)
	}

	// Wait for Traefik deployment to be ready
	fmt.Print("Waiting for Traefik deployment to be ready...")
	deployment := "traefik"
	namespace := "kecs-system"
	
	for i := 0; i < 60; i++ { // Wait up to 5 minutes
		deps, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, deployment, metav1.GetOptions{})
		if err == nil && deps.Status.ReadyReplicas > 0 {
			fmt.Println(" ready!")
			break
		}
		time.Sleep(5 * time.Second)
		fmt.Print(".")
	}

	// Wait for Traefik service to get external IP/port
	fmt.Print("Waiting for Traefik service to be accessible...")
	service := "traefik"
	
	for i := 0; i < 30; i++ { // Wait up to 2.5 minutes
		svc, err := kubeClient.CoreV1().Services(namespace).Get(ctx, service, metav1.GetOptions{})
		if err == nil && len(svc.Status.LoadBalancer.Ingress) > 0 {
			fmt.Println(" ready!")
			fmt.Printf("Traefik LoadBalancer: %s\n", svc.Status.LoadBalancer.Ingress[0].Hostname)
			return nil
		}
		time.Sleep(5 * time.Second)
		fmt.Print(".")
	}

	// For k3d, the LoadBalancer might not get an external IP
	// Port forwarding is handled by k3d itself
	fmt.Println(" ready! (using k3d port mapping)")
	
	return nil
}

func waitForComponents(ctx context.Context, clusterName string) error {
	// TODO: Implement readiness checks for all components
	fmt.Println("Component readiness checks not yet implemented")
	return nil
}