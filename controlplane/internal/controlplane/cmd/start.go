package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes/resources"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/progress"
	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

var (
	// Start flags
	startInstanceName string
	startDataDir     string
	startApiPort     int
	startAdminPort   int
	startConfigFile  string
	startNoLocalStack bool
	startNoTraefik   bool
	startTimeout     time.Duration
	startUseBubbleTea bool
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
	startCmd.Flags().BoolVar(&startUseBubbleTea, "bubbletea", false, "Use Bubble Tea for progress display (experimental)")
}

func runStart(cmd *cobra.Command, args []string) error {
	// Generate instance name if not provided
	if startInstanceName == "" {
		generatedName, err := utils.GenerateRandomName()
		if err != nil {
			return fmt.Errorf("failed to generate instance name: %w", err)
		}
		startInstanceName = generatedName
		// Only show info message if not using Bubble Tea
		if !startUseBubbleTea {
			progress.Info("Generated KECS instance name: %s", startInstanceName)
		}
	}

	// Only show header if not using Bubble Tea
	if !startUseBubbleTea {
		progress.SectionHeader(fmt.Sprintf("Creating KECS instance '%s'", startInstanceName))
	}

	// Load configuration
	cfg, err := config.LoadConfig(startConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override with flags
	if startNoLocalStack {
		cfg.LocalStack.Enabled = false
	}
	if startNoTraefik {
		cfg.Features.Traefik = false
	}

	// Set up data directory
	if startDataDir == "" {
		home, _ := os.UserHomeDir()
		startDataDir = filepath.Join(home, ".kecs", "instances", startInstanceName, "data")
	}

	// Ensure data directory exists
	if err := os.MkdirAll(startDataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	// Use Bubble Tea if flag is set
	if startUseBubbleTea {
		return runStartWithBubbleTea(ctx, startInstanceName, cfg, startDataDir)
	}

	// Step 1: Create k3d cluster for KECS instance
	spinner := progress.NewSpinner("Creating k3d cluster")
	spinner.Start()
	
	if err := createK3dCluster(ctx, startInstanceName, cfg, startDataDir); err != nil {
		spinner.Fail("Failed to create k3d cluster")
		return fmt.Errorf("failed to create k3d cluster: %w", err)
	}
	spinner.Success("k3d cluster created")

	// Step 2: Create kecs-system namespace
	spinner = progress.NewSpinner("Creating kecs-system namespace")
	spinner.Start()
	if err := createKecsSystemNamespace(ctx, startInstanceName); err != nil {
		spinner.Fail("Failed to create namespace")
		return fmt.Errorf("failed to create kecs-system namespace: %w", err)
	}
	spinner.Success("kecs-system namespace created")

	// Step 3: Deploy KECS control plane and LocalStack in parallel
	// Create log capture for deployment phase
	logCapture := progress.NewLogCapture(os.Stdout, progress.LogLevelInfo)
	
	// Redirect standard log output to our capture
	logRedirector := progress.NewLogRedirector(logCapture, progress.LogLevelInfo)
	logRedirector.RedirectStandardLog()
	defer logRedirector.Restore()
	
	// Create parallel tracker for component deployment with log capture
	parallelTracker := progress.NewParallelTracker("Deploying components").
		WithLogCapture(logCapture)
	
	// Add tasks
	parallelTracker.AddTask("controlplane", "Control Plane", 100)
	if cfg.LocalStack.Enabled {
		parallelTracker.AddTask("localstack", "LocalStack", 100)
	}
	
	var wg sync.WaitGroup
	errChan := make(chan error, 2)
	
	// Deploy Control Plane
	wg.Add(1)
	go func() {
		defer wg.Done()
		parallelTracker.StartTask("controlplane")
		if err := deployControlPlaneWithProgress(ctx, startInstanceName, cfg, startDataDir, parallelTracker); err != nil {
			parallelTracker.FailTask("controlplane", err)
			errChan <- fmt.Errorf("failed to deploy control plane: %w", err)
			return
		}
		parallelTracker.CompleteTask("controlplane")
	}()
	
	// Deploy LocalStack (if enabled)
	if cfg.LocalStack.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			parallelTracker.StartTask("localstack")
			if err := deployLocalStackWithProgress(ctx, startInstanceName, cfg, parallelTracker); err != nil {
				parallelTracker.FailTask("localstack", err)
				errChan <- fmt.Errorf("failed to deploy LocalStack: %w", err)
				return
			}
			parallelTracker.CompleteTask("localstack")
		}()
	}
	
	// Wait for parallel deployments to complete
	wg.Wait()
	parallelTracker.Stop()
	close(errChan)
	
	// Check for errors from parallel deployments
	for err := range errChan {
		return err
	}

	// Step 4: Deploy Traefik gateway (if enabled) - must be after control plane and LocalStack
	if cfg.Features.Traefik {
		spinner = progress.NewSpinner("Deploying Traefik gateway")
		spinner.Start()
		if err := deployTraefikGateway(ctx, startInstanceName, cfg, startApiPort); err != nil {
			spinner.Fail("Failed to deploy Traefik")
			return fmt.Errorf("failed to deploy Traefik gateway: %w", err)
		}
		spinner.Success("Traefik gateway deployed")
	}

	// Step 5: Wait for all components to be ready
	spinner = progress.NewSpinner("Waiting for all components to be ready")
	spinner.Start()
	if err := waitForComponents(ctx, startInstanceName); err != nil {
		spinner.Fail("Components failed to become ready")
		return fmt.Errorf("components did not become ready: %w", err)
	}
	spinner.Success("All components are ready")

	// Show success summary
	progress.Success("KECS instance '%s' is ready!", startInstanceName)
	
	fmt.Println()
	progress.Info("Endpoints:")
	fmt.Printf("  AWS API: http://localhost:%d\n", startApiPort)
	fmt.Printf("  Admin API: http://localhost:%d\n", startAdminPort)
	fmt.Printf("  Data directory: %s\n", startDataDir)

	if cfg.LocalStack.Enabled {
		fmt.Printf("\nLocalStack services: %v\n", cfg.LocalStack.Services)
	}

	fmt.Println()
	progress.Info("Next steps:")
	fmt.Printf("  To stop this instance: kecs stop --instance %s\n", startInstanceName)
	fmt.Printf("  To get kubeconfig: kecs kubeconfig get %s\n", startInstanceName)

	return nil
}

func createK3dCluster(ctx context.Context, clusterName string, cfg *config.Config, dataDir string) error {
	// Create k3d cluster manager configuration
	clusterConfig := &kubernetes.ClusterManagerConfig{
		Provider:      "k3d",
		ContainerMode: false,
		EnableTraefik: false, // Disable old Traefik deployment (we use our own)
		TraefikPort:   startApiPort, // Use the API port for Traefik
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
		log.Printf("k3d cluster '%s' already exists, using existing cluster", clusterName)
		return nil
	}

	// Create the cluster
	if err := manager.CreateCluster(ctx, clusterName); err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	// Wait for cluster to be ready
	log.Print("Waiting for cluster to be ready...")
	if err := manager.WaitForClusterReady(clusterName, 5*time.Minute); err != nil {
		return fmt.Errorf("cluster did not become ready: %w", err)
	}
	log.Println("Cluster is ready!")

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
		log.Println("kecs-system namespace already exists")
		return nil
	}

	_, err = kubeClient.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create kecs-system namespace: %w", err)
	}

	log.Println("Created kecs-system namespace")
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
	controlPlaneConfig := &resources.ControlPlaneConfig{
		Image:           "ghcr.io/nandemo-ya/kecs:latest",
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
		},
	}

	// Deploy control plane resources programmatically
	log.Println("Deploying control plane resources...")
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
		ProxyEndpoint: "", // Will be set after Traefik is deployed
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
	log.Println("Deploying LocalStack...")
	if err := lsManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start LocalStack: %w", err)
	}

	// Wait for LocalStack to be ready
	log.Print("Waiting for LocalStack to be ready...")
	for i := 0; i < 60; i++ { // Wait up to 5 minutes
		if lsManager.IsHealthy() {
			log.Println("LocalStack is ready!")
			status, err := lsManager.GetStatus()
			if err == nil {
				log.Printf("LocalStack running: %v", status.Running)
				log.Printf("LocalStack services: %v", status.EnabledServices)
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
		WebPort:         80,
		WebNodePort:     30080,
		AWSPort:         4566,
		AWSNodePort:     30890,  // Fixed NodePort in valid range (k3d maps host port to this)
		LogLevel:        "INFO",
		AccessLog:       true,
		Metrics:         false,
		Debug:           cfg.Server.LogLevel == "debug",
	}

	// Deploy Traefik resources programmatically
	log.Println("Deploying Traefik gateway resources...")
	if err := deployer.DeployTraefik(ctx, traefikConfig); err != nil {
		return fmt.Errorf("failed to deploy Traefik: %w", err)
	}

	// Wait for Traefik deployment to be ready
	log.Print("Waiting for Traefik deployment to be ready...")
	deployment := "traefik"
	namespace := "kecs-system"
	
	for i := 0; i < 60; i++ { // Wait up to 5 minutes
		deps, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, deployment, metav1.GetOptions{})
		if err == nil && deps.Status.ReadyReplicas > 0 {
			log.Println("Traefik deployment is ready!")
			break
		}
		time.Sleep(5 * time.Second)
	}

	// Wait for Traefik service to get external IP/port
	log.Print("Waiting for Traefik service to be accessible...")
	service := "traefik"
	
	for i := 0; i < 30; i++ { // Wait up to 2.5 minutes
		svc, err := kubeClient.CoreV1().Services(namespace).Get(ctx, service, metav1.GetOptions{})
		if err == nil && len(svc.Status.LoadBalancer.Ingress) > 0 {
			log.Println("Traefik service is ready!")
			log.Printf("Traefik LoadBalancer: %s", svc.Status.LoadBalancer.Ingress[0].Hostname)
			return nil
		}
		time.Sleep(5 * time.Second)
	}

	// For k3d, the LoadBalancer might not get an external IP
	// Port forwarding is handled by k3d itself
	log.Println("Traefik service is ready! (using k3d port mapping)")
	
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

	log.Println("Checking component readiness...")
	
	allReady := true
	for _, comp := range components {
		// Check deployment
		deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, comp.deployment, metav1.GetOptions{})
		if err != nil {
			if comp.required {
				log.Printf("  %s: ❌ Not found", comp.name)
				allReady = false
			} else {
				log.Printf("  %s: ⏭️  Skipped (optional)", comp.name)
			}
			continue
		}

		// Check if deployment is ready
		if deployment.Status.ReadyReplicas >= 1 {
			log.Printf("  %s: ✅ Ready (%d/%d replicas)", comp.name, deployment.Status.ReadyReplicas, *deployment.Spec.Replicas)
		} else {
			log.Printf("  %s: ⏳ Not ready (0/%d replicas)", comp.name, *deployment.Spec.Replicas)
			if comp.required {
				allReady = false
			}
		}

		// Check service endpoint
		service, err := kubeClient.CoreV1().Services(namespace).Get(ctx, comp.deployment, metav1.GetOptions{})
		if err == nil && len(service.Spec.Ports) > 0 {
			log.Printf("    Service: %s:%d", service.Name, service.Spec.Ports[0].Port)
		}
	}

	if !allReady {
		return fmt.Errorf("some required components are not ready")
	}

	// Skip external health checks for k8s deployments
	// The deployment readiness checks above are sufficient
	log.Printf("✅ All components are ready")

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

// deployControlPlaneWithProgress wraps deployControlPlane with progress reporting
func deployControlPlaneWithProgress(ctx context.Context, clusterName string, cfg *config.Config, dataDir string, tracker *progress.ParallelTracker) error {
	// Update progress during deployment
	tracker.UpdateTask("controlplane", 20, "Preparing resources")
	
	// Get k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	tracker.UpdateTask("controlplane", 30, "Getting Kubernetes client")
	
	// Get Kubernetes client and config
	kubeClient, err := manager.GetKubeClient(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}
	
	kubeConfig, err := manager.GetKubeConfig(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	tracker.UpdateTask("controlplane", 40, "Creating deployer")
	
	// Create resource deployer with config
	deployer, err := kubernetes.NewResourceDeployerWithConfig(kubeClient, kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create resource deployer: %w", err)
	}
	
	// Configure control plane
	controlPlaneConfig := &resources.ControlPlaneConfig{
		Image:           "ghcr.io/nandemo-ya/kecs:latest",
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
		},
	}

	tracker.UpdateTask("controlplane", 60, "Deploying resources")
	
	// Deploy control plane resources programmatically
	if err := deployer.DeployControlPlane(ctx, controlPlaneConfig); err != nil {
		return fmt.Errorf("failed to deploy control plane: %w", err)
	}

	tracker.UpdateTask("controlplane", 80, "Waiting for deployment")
	
	// Wait for deployment to be ready
	deployment := "kecs-controlplane"
	namespace := "kecs-system"
	maxWaitTime := 120 // 10 minutes (120 * 5 seconds) - increased for image pull retries
	
	for i := 0; i < maxWaitTime; i++ {
		deps, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, deployment, metav1.GetOptions{})
		if err == nil && deps.Status.ReadyReplicas > 0 {
			tracker.UpdateTask("controlplane", 100, "Ready")
			return nil
		}
		
		// Calculate progress from 80% to 99% (never reach 100% until actually ready)
		progress := 80 + ((i + 1) * 19 / maxWaitTime)
		waitTime := (i + 1) * 5
		tracker.UpdateTask("controlplane", progress, fmt.Sprintf("Waiting for pods (%ds/300s)", waitTime))
		
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("control plane deployment did not become ready in time")
}

// deployLocalStackWithProgress wraps deployLocalStack with progress reporting
func deployLocalStackWithProgress(ctx context.Context, clusterName string, cfg *config.Config, tracker *progress.ParallelTracker) error {
	tracker.UpdateTask("localstack", 10, "Initializing")
	
	// Get k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	tracker.UpdateTask("localstack", 20, "Getting Kubernetes client")
	
	// Get Kubernetes client and config
	kubeClient, err := manager.GetKubeClient(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	kubeConfig, err := manager.GetKubeConfig(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	tracker.UpdateTask("localstack", 30, "Configuring LocalStack")
	
	// Configure LocalStack
	localstackConfig := &localstack.Config{
		Enabled:       true,
		UseTraefik:    false, // Don't use Traefik during initial deployment
		Namespace:     "kecs-system",
		Services:      cfg.LocalStack.Services,
		Port:          4566,
		EdgePort:      4566,
		ProxyEndpoint: "", // Will be set after Traefik is deployed
		ContainerMode: false,
		Image:         cfg.LocalStack.Image,
		Version:       cfg.LocalStack.Version,
		Debug:         cfg.Server.LogLevel == "debug",
	}

	tracker.UpdateTask("localstack", 40, "Creating LocalStack manager")
	
	// Create LocalStack manager
	lsManager, err := localstack.NewManager(localstackConfig, kubeClient, kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create LocalStack manager: %w", err)
	}

	tracker.UpdateTask("localstack", 50, "Starting LocalStack")
	
	// Deploy LocalStack
	if err := lsManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start LocalStack: %w", err)
	}

	// Give LocalStack a moment to initialize before checking health
	// This prevents the progress from jumping due to the initial health status
	tracker.UpdateTask("localstack", 60, "LocalStack started, initializing...")
	time.Sleep(3 * time.Second)

	tracker.UpdateTask("localstack", 70, "Waiting for LocalStack to be ready")
	
	// Wait for LocalStack to be ready
	maxWaitTime := 60 // 5 minutes (60 * 5 seconds)
	for i := 0; i < maxWaitTime; i++ {
		// Check if LocalStack deployment is ready
		status, err := lsManager.GetStatus()
		if err == nil && status.Running && status.Healthy {
			tracker.UpdateTask("localstack", 100, "Ready")
			return nil
		}
		
		// Calculate progress from 70% to 99% (never reach 100% until actually ready)
		progress := 70 + ((i + 1) * 29 / maxWaitTime)
		waitTime := (i + 1) * 5
		
		// Provide more detailed status message
		statusMsg := fmt.Sprintf("Health check (%ds/300s)", waitTime)
		if status != nil {
			if !status.Running {
				statusMsg = fmt.Sprintf("Starting LocalStack pod (%ds/300s)", waitTime)
			} else if !status.Healthy {
				statusMsg = fmt.Sprintf("Waiting for LocalStack services (%ds/300s)", waitTime)
			}
		}
		
		tracker.UpdateTask("localstack", progress, statusMsg)
		
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("LocalStack did not become ready in time")
}