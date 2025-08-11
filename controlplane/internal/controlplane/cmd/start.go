package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/instance"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes/resources"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

var (
	// Start flags
	startInstanceName string
	startDataDir      string
	startApiPort      int
	startAdminPort    int
	startConfigFile   string
	startNoLocalStack bool
	startNoTraefik    bool
	startTimeout      time.Duration
	startVerbose      bool
	startDevMode      bool
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start KECS with control plane in k3d cluster",
	Long: `Start KECS by creating a k3d cluster and deploying the control plane inside it.
This provides a unified AWS API endpoint accessible from all containers.`,
	RunE: runStart,
}

func init() {
	RootCmd.AddCommand(startCmd)

	startCmd.Flags().StringVar(&startInstanceName, "instance", "", "KECS instance name (auto-generated if not specified)")
	startCmd.Flags().StringVar(&startDataDir, "data-dir", "", "Data directory (default: ~/.kecs/data)")
	startCmd.Flags().IntVar(&startApiPort, "api-port", 4566, "AWS API port (Traefik gateway)")
	startCmd.Flags().IntVar(&startAdminPort, "admin-port", 8081, "Admin API port")
	startCmd.Flags().StringVar(&startConfigFile, "config", "", "Configuration file path")
	startCmd.Flags().BoolVar(&startNoLocalStack, "no-localstack", false, "Disable LocalStack deployment")
	startCmd.Flags().BoolVar(&startNoTraefik, "no-traefik", false, "Disable Traefik deployment")
	startCmd.Flags().DurationVar(&startTimeout, "timeout", 10*time.Minute, "Timeout for cluster creation")
	startCmd.Flags().BoolVar(&startVerbose, "verbose", false, "Use verbose output instead of interactive progress display")
	startCmd.Flags().BoolVar(&startDevMode, "dev", false, "Enable dev mode with k3d registry for local development")
}

func runStart(cmd *cobra.Command, args []string) error {
	// Create k3d cluster manager to check existing instances
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	// If instance name is not provided, show selection
	if startInstanceName == "" {
		instanceName, isNew, err := selectOrCreateInstance(manager)
		if err != nil {
			return err
		}
		startInstanceName = instanceName
		
		// If an existing instance was selected, check if it's already running
		if !isNew {
			running, err := checkInstanceRunning(manager, startInstanceName)
			if err != nil {
				return fmt.Errorf("failed to check instance status: %w", err)
			}
			if running {
				fmt.Printf("⚠️  Instance '%s' is already running\n", startInstanceName)
				return nil
			}
			// For stopped instances, we'll restart them
			if startVerbose {
				fmt.Printf("Restarting stopped instance: %s\n", startInstanceName)
			}
		}
	} else {
		// Check if specified instance exists
		exists, err := manager.ClusterExists(context.Background(), startInstanceName)
		if err != nil {
			return fmt.Errorf("failed to check instance existence: %w", err)
		}
		if exists {
			running, err := checkInstanceRunning(manager, startInstanceName)
			if err != nil {
				return fmt.Errorf("failed to check instance status: %w", err)
			}
			if running {
				fmt.Printf("⚠️  Instance '%s' is already running\n", startInstanceName)
				return nil
			}
			// For stopped instances, we'll restart them
			if startVerbose {
				fmt.Printf("Restarting stopped instance: %s\n", startInstanceName)
			}
		}
	}

	// Only show header if using verbose output
	if startVerbose {
		fmt.Printf("\n=== Creating KECS instance '%s' ===\n", startInstanceName)
	}

	// Create instance manager
	instanceManager, err := instance.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create instance manager: %w", err)
	}

	// Set up start options
	opts := instance.StartOptions{
		InstanceName:  startInstanceName,
		DataDir:       startDataDir,
		ConfigFile:    startConfigFile,
		NoLocalStack:  startNoLocalStack,
		NoTraefik:     startNoTraefik,
		ApiPort:       startApiPort,
		AdminPort:     startAdminPort,
		DevMode:       startDevMode,
	}

	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	// Initialize logging
	logging.InitializeForProgress(nil, startVerbose)

	// Start the instance using the shared manager
	if err := instanceManager.Start(ctx, opts); err != nil {
		return err
	}
	
	// Show completion message
	fmt.Printf("\n🎉 KECS instance '%s' is ready!\n", opts.InstanceName)
	fmt.Println("\n=== Next steps ===")
	fmt.Printf("AWS API: http://localhost:%d\n", opts.ApiPort)
	fmt.Printf("Admin API: http://localhost:%d\n", opts.AdminPort)
	fmt.Printf("Data directory: %s\n", opts.DataDir)
	fmt.Println()
	fmt.Printf("To stop this instance: kecs stop --instance %s\n", opts.InstanceName)
	fmt.Printf("To get kubeconfig: kecs kubeconfig get %s\n", opts.InstanceName)
	
	return nil
}

func createK3dCluster(ctx context.Context, clusterName string, cfg *config.Config, dataDir string) error {
	// Create k3d cluster manager configuration
	clusterConfig := &kubernetes.ClusterManagerConfig{
		Provider:       "k3d",
		ContainerMode:  false,
		EnableTraefik:  false,        // Disable old Traefik deployment (we use our own)
		TraefikPort:    startApiPort, // Use the API port for Traefik
		EnableRegistry: startDevMode,  // Enable k3d registry in dev mode
		RegistryPort:   5000,         // Default registry port
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
		logging.Info("k3d cluster already exists, checking if it's running", "cluster", clusterName)
		
		// Check if the cluster is running
		running, err := manager.IsClusterRunning(ctx, clusterName)
		if err != nil {
			return fmt.Errorf("failed to check cluster status: %w", err)
		}
		
		if !running {
			logging.Info("k3d cluster is stopped, starting it", "cluster", clusterName)
			if err := manager.StartCluster(ctx, clusterName); err != nil {
				return fmt.Errorf("failed to start cluster: %w", err)
			}
			
			// Wait for cluster to be ready after starting
			logging.Info("Waiting for cluster to be ready after start")
			if err := manager.WaitForClusterReady(clusterName, 5*time.Minute); err != nil {
				return fmt.Errorf("cluster did not become ready after start: %w", err)
			}
			logging.Info("Cluster is ready")
		} else {
			logging.Info("k3d cluster is already running")
		}
		
		return nil
	}

	// Create the cluster
	if err := manager.CreateCluster(ctx, clusterName); err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	// Wait for cluster to be ready
	logging.Info("Waiting for cluster to be ready")
	if err := manager.WaitForClusterReady(clusterName, 5*time.Minute); err != nil {
		return fmt.Errorf("cluster did not become ready: %w", err)
	}
	logging.Info("Cluster is ready")

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
		logging.Info("kecs-system namespace already exists")
		return nil
	}

	_, err = kubeClient.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create kecs-system namespace: %w", err)
	}

	logging.Info("Created kecs-system namespace")
	return nil
}

func deployControlPlane(ctx context.Context, clusterName string, cfg *config.Config, dataDir string) error {
	// Get k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	// Get Kubernetes client and config
	kubeClient, err := manager.GetKubeClient(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	kubeConfig, err := manager.GetKubeConfig(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	// Create resource deployer with config
	deployer, err := kubernetes.NewResourceDeployerWithConfig(kubeClient, kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create resource deployer: %w", err)
	}

	// Configure control plane
	controlPlaneImage := cfg.Server.ControlPlaneImage
	if startDevMode {
		// Use local registry image in dev mode
		controlPlaneImage = "k3d-kecs-registry.localhost:5000/nandemo-ya/kecs-controlplane:latest"
		logging.Info("Dev mode enabled, using local registry image", "image", controlPlaneImage)
	}
	
	controlPlaneConfig := &resources.ControlPlaneConfig{
		Image:           controlPlaneImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		CPURequest:      "100m",
		MemoryRequest:   "128Mi",
		CPULimit:        "1000m",
		MemoryLimit:     "1Gi",
		StorageSize:     "10Gi",
		APIPort:         80,
		AdminPort:       int32(startAdminPort),
		LogLevel:        cfg.Server.LogLevel,
		ExtraEnvVars: []corev1.EnvVar{
			{
				Name:  "KECS_SKIP_SECURITY_DISCLAIMER",
				Value: "true",
			},
			{
				Name:  "KECS_INSTANCE_NAME",
				Value: clusterName,
			},
		},
	}

	// Deploy control plane resources programmatically
	logging.Info("Deploying control plane resources")
	if err := deployer.DeployControlPlane(ctx, controlPlaneConfig); err != nil {
		return fmt.Errorf("failed to deploy control plane: %w", err)
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
	// Get k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	// Get Kubernetes client and config
	kubeClient, err := manager.GetKubeClient(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	kubeConfig, err := manager.GetKubeConfig(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	// Configure LocalStack for in-cluster deployment
	localstackConfig := &localstack.Config{
		Enabled:       true,
		UseTraefik:    false, // Don't use Traefik during initial deployment
		Namespace:     "kecs-system",
		Services:      cfg.LocalStack.Services,
		Port:          4566,
		EdgePort:      4566,
		ProxyEndpoint: "",    // Will be set after Traefik is deployed
		ContainerMode: false, // We're deploying in k8s, not standalone container
		Image:         cfg.LocalStack.Image,
		Version:       cfg.LocalStack.Version,
		Debug:         cfg.Server.LogLevel == "debug",
	}

	// Create LocalStack manager
	lsManager, err := localstack.NewManager(localstackConfig, kubeClient, kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create LocalStack manager: %w", err)
	}

	// Deploy LocalStack
	logging.Info("Deploying LocalStack")
	if err := lsManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start LocalStack: %w", err)
	}

	// Wait for LocalStack to be ready
	logging.Info("Waiting for LocalStack to be ready")
	for i := 0; i < 60; i++ { // Wait up to 5 minutes
		if lsManager.IsHealthy() {
			logging.Info("LocalStack is ready")
			status, err := lsManager.GetStatus()
			if err == nil {
				logging.Info("LocalStack status", "running", status.Running)
				logging.Info("LocalStack services", "services", status.EnabledServices)
			}
			return nil
		}
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("LocalStack did not become ready in time")
}

func deployTraefikGateway(ctx context.Context, clusterName string, cfg *config.Config, apiPort int) error {
	// Get k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	// Get Kubernetes client and config
	kubeClient, err := manager.GetKubeClient(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	kubeConfig, err := manager.GetKubeConfig(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	// Create resource deployer with config
	deployer, err := kubernetes.NewResourceDeployerWithConfig(kubeClient, kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create resource deployer: %w", err)
	}

	// Configure Traefik
	traefikConfig := &resources.TraefikConfig{
		Image:           "traefik:v3.2",
		ImagePullPolicy: corev1.PullIfNotPresent,
		CPURequest:      "100m",
		MemoryRequest:   "128Mi",
		CPULimit:        "500m",
		MemoryLimit:     "512Mi",
		APIPort:         80,
		APINodePort:     30080,
		AWSPort:         4566,
		AWSNodePort:     30890, // Fixed NodePort in valid range (k3d maps host port to this)
		LogLevel:        "INFO",
		AccessLog:       true,
		Metrics:         false,
		Debug:           cfg.Server.LogLevel == "debug",
	}

	// Deploy Traefik resources programmatically
	logging.Info("Deploying Traefik gateway resources")
	if err := deployer.DeployTraefik(ctx, traefikConfig); err != nil {
		return fmt.Errorf("failed to deploy Traefik: %w", err)
	}

	// Wait for Traefik deployment to be ready
	logging.Info("Waiting for Traefik deployment to be ready")
	deployment := "traefik"
	namespace := "kecs-system"

	for i := 0; i < 60; i++ { // Wait up to 5 minutes
		deps, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, deployment, metav1.GetOptions{})
		if err == nil && deps.Status.ReadyReplicas > 0 {
			logging.Info("Traefik deployment is ready")
			break
		}
		time.Sleep(5 * time.Second)
	}

	// Wait for Traefik service to get external IP/port
	logging.Info("Waiting for Traefik service to be accessible")
	service := "traefik"

	for i := 0; i < 30; i++ { // Wait up to 2.5 minutes
		svc, err := kubeClient.CoreV1().Services(namespace).Get(ctx, service, metav1.GetOptions{})
		if err == nil && len(svc.Status.LoadBalancer.Ingress) > 0 {
			logging.Info("Traefik service is ready")
			logging.Info("Traefik LoadBalancer configured", "hostname", svc.Status.LoadBalancer.Ingress[0].Hostname)
			return nil
		}
		time.Sleep(5 * time.Second)
	}

	// For k3d, the LoadBalancer might not get an external IP
	// Port forwarding is handled by k3d itself
	logging.Info("Traefik service is ready (using k3d port mapping)")

	return nil
}

func waitForComponents(ctx context.Context, clusterName string) error {
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

	namespace := "kecs-system"
	components := []struct {
		name       string
		deployment string
		required   bool
	}{
		{"KECS Control Plane", "kecs-controlplane", true},
		{"Traefik Gateway", "traefik", true},
		{"LocalStack", "localstack", false}, // Optional based on config
	}

	logging.Info("Checking component readiness")

	allReady := true
	for _, comp := range components {
		// Check deployment
		deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, comp.deployment, metav1.GetOptions{})
		if err != nil {
			if comp.required {
				logging.Info("Component not found", "component", comp.name, "status", "❌")
				allReady = false
			} else {
				logging.Info("Component skipped (optional)", "component", comp.name, "status", "⏭️")
			}
			continue
		}

		// Check if deployment is ready
		if deployment.Status.ReadyReplicas >= 1 {
			logging.Info("Component ready", "component", comp.name, "status", "✅", "readyReplicas", deployment.Status.ReadyReplicas, "totalReplicas", *deployment.Spec.Replicas)
		} else {
			logging.Info("Component not ready", "component", comp.name, "status", "⏳", "readyReplicas", 0, "totalReplicas", *deployment.Spec.Replicas)
			if comp.required {
				allReady = false
			}
		}

		// Check service endpoint
		service, err := kubeClient.CoreV1().Services(namespace).Get(ctx, comp.deployment, metav1.GetOptions{})
		if err == nil && len(service.Spec.Ports) > 0 {
			logging.Info("Service endpoint", "service", service.Name, "port", service.Spec.Ports[0].Port)
		}
	}

	if !allReady {
		return fmt.Errorf("some required components are not ready")
	}

	// Skip external health checks for k8s deployments
	// The deployment readiness checks above are sufficient
	logging.Info("All components are ready", "status", "✅")

	return nil
}

func checkEndpointHealth(endpoint string, timeout time.Duration) error {
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(endpoint)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("endpoint %s did not become healthy within %v", endpoint, timeout)
}





// selectOrCreateInstance shows an interactive selection for existing instances or creates a new one
func selectOrCreateInstance(manager *kubernetes.K3dClusterManager) (string, bool, error) {
	ctx := context.Background()
	
	fmt.Println("Fetching KECS instances...")
	
	// Get list of clusters
	clusters, err := manager.ListClusters(ctx)
	if err != nil {
		return "", false, fmt.Errorf("failed to list instances: %w", err)
	}
	
	if len(clusters) == 0 {
		// No existing instances, create a new one
		generatedName, err := utils.GenerateRandomName()
		if err != nil {
			return "", false, fmt.Errorf("failed to generate instance name: %w", err)
		}
		fmt.Printf("No existing instances found. Creating new KECS instance: %s\n", generatedName)
		return generatedName, true, nil
	}
	
	// List existing instances
	fmt.Println("\nExisting KECS instances:")
	for i, cluster := range clusters {
		status := "stopped"
		// Check if cluster is running
		running, _ := checkInstanceRunning(manager, cluster)
		if running {
			status = "running"
		}
		
		// Check for data directory
		home, _ := os.UserHomeDir()
		dataDir := filepath.Join(home, ".kecs", "instances", cluster, "data")
		if _, err := os.Stat(dataDir); err == nil {
			fmt.Printf("  %d. %s (%s, has data)\n", i+1, cluster, status)
		} else {
			fmt.Printf("  %d. %s (%s)\n", i+1, cluster, status)
		}
	}
	
	// Since we can't do interactive selection without pterm, 
	// we'll auto-generate a new instance name
	generatedName, err := utils.GenerateRandomName()
	if err != nil {
		return "", false, fmt.Errorf("failed to generate instance name: %w", err)
	}
	
	fmt.Printf("\nCreating new KECS instance: %s\n", generatedName)
	fmt.Println("To use an existing instance, specify it with --instance flag")
	
	return generatedName, true, nil
}

// checkInstanceRunning checks if a KECS instance is currently running
func checkInstanceRunning(manager *kubernetes.K3dClusterManager, instanceName string) (bool, error) {
	ctx := context.Background()
	
	// Use the new IsClusterRunning method to check status without triggering warnings
	return manager.IsClusterRunning(ctx, instanceName)
}

