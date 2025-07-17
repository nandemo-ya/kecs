package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/progress"
	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

var (
	// Start v2 flags (new architecture)
	startV2InstanceName string
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

	startV2Cmd.Flags().StringVar(&startV2InstanceName, "instance", "", "KECS instance name (auto-generated if not specified)")
	startV2Cmd.Flags().StringVar(&startV2DataDir, "data-dir", "", "Data directory (default: ~/.kecs/data)")
	startV2Cmd.Flags().IntVar(&startV2ApiPort, "api-port", 4566, "AWS API port (Traefik gateway)")
	startV2Cmd.Flags().IntVar(&startV2AdminPort, "admin-port", 8081, "Admin API port")
	startV2Cmd.Flags().StringVar(&startV2ConfigFile, "config", "", "Configuration file path")
	startV2Cmd.Flags().BoolVar(&startV2NoLocalStack, "no-localstack", false, "Disable LocalStack deployment")
	startV2Cmd.Flags().BoolVar(&startV2NoTraefik, "no-traefik", false, "Disable Traefik deployment")
	startV2Cmd.Flags().DurationVar(&startV2Timeout, "timeout", 10*time.Minute, "Timeout for cluster creation")
}

func runStartV2(cmd *cobra.Command, args []string) error {
	// Generate instance name if not provided
	if startV2InstanceName == "" {
		generatedName, err := utils.GenerateRandomName()
		if err != nil {
			return fmt.Errorf("failed to generate instance name: %w", err)
		}
		startV2InstanceName = generatedName
		progress.Info("Generated KECS instance name: %s", startV2InstanceName)
	}

	// Show header
	progress.SectionHeader(fmt.Sprintf("Creating KECS instance '%s'", startV2InstanceName))

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
		startV2DataDir = filepath.Join(home, ".kecs", "instances", startV2InstanceName, "data")
	}

	// Ensure data directory exists
	if err := os.MkdirAll(startV2DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), startV2Timeout)
	defer cancel()

	// Step 1: Create k3d cluster for KECS instance
	spinner := progress.NewSpinner("Creating k3d cluster")
	spinner.Start()
	
	if err := createK3dCluster(ctx, startV2InstanceName, cfg, startV2DataDir); err != nil {
		spinner.Fail("Failed to create k3d cluster")
		return fmt.Errorf("failed to create k3d cluster: %w", err)
	}
	spinner.Success("k3d cluster created")

	// Step 2: Create kecs-system namespace
	spinner = progress.NewSpinner("Creating kecs-system namespace")
	spinner.Start()
	if err := createKecsSystemNamespace(ctx, startV2InstanceName); err != nil {
		spinner.Fail("Failed to create namespace")
		return fmt.Errorf("failed to create kecs-system namespace: %w", err)
	}
	spinner.Success("kecs-system namespace created")

	// Step 3: Deploy KECS control plane and LocalStack in parallel
	progress.Info("Deploying KECS components")
	
	// Create parallel tracker for component deployment
	parallelTracker := progress.NewParallelTracker("Deploying components")
	
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
		if err := deployControlPlaneWithProgress(ctx, startV2InstanceName, cfg, startV2DataDir, parallelTracker); err != nil {
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
			if err := deployLocalStackWithProgress(ctx, startV2InstanceName, cfg, parallelTracker); err != nil {
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
		if err := deployTraefikGateway(ctx, startV2InstanceName, cfg, startV2ApiPort); err != nil {
			spinner.Fail("Failed to deploy Traefik")
			return fmt.Errorf("failed to deploy Traefik gateway: %w", err)
		}
		spinner.Success("Traefik gateway deployed")
	}

	// Step 5: Wait for all components to be ready
	spinner = progress.NewSpinner("Waiting for all components to be ready")
	spinner.Start()
	if err := waitForComponents(ctx, startV2InstanceName); err != nil {
		spinner.Fail("Components failed to become ready")
		return fmt.Errorf("components did not become ready: %w", err)
	}
	spinner.Success("All components are ready")

	// Show success summary
	progress.Success("KECS instance '%s' is ready!", startV2InstanceName)
	
	fmt.Println()
	progress.Info("Endpoints:")
	fmt.Printf("  AWS API: http://localhost:%d\n", startV2ApiPort)
	fmt.Printf("  Admin API: http://localhost:%d\n", startV2AdminPort)
	fmt.Printf("  Data directory: %s\n", startV2DataDir)

	if cfg.LocalStack.Enabled {
		fmt.Printf("\nLocalStack services: %v\n", cfg.LocalStack.Services)
	}

	fmt.Println()
	progress.Info("Next steps:")
	fmt.Printf("  To stop this instance: kecs stop-v2 --instance %s\n", startV2InstanceName)
	fmt.Printf("  To get kubeconfig: kecs kubeconfig get %s\n", startV2InstanceName)

	return nil
}

func createK3dCluster(ctx context.Context, clusterName string, cfg *config.Config, dataDir string) error {
	// Create k3d cluster manager configuration
	clusterConfig := &kubernetes.ClusterManagerConfig{
		Provider:      "k3d",
		ContainerMode: false,
		EnableTraefik: true, // Enable Traefik for the new architecture
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
	manifestsDir := ""
	
	// Try to find manifests directory
	// 1. Try relative path from current directory
	if _, err := os.Stat("manifests"); err == nil {
		manifestsDir = "manifests"
	} else if _, err := os.Stat("controlplane/manifests"); err == nil {
		manifestsDir = "controlplane/manifests"
	} else if gopath := os.Getenv("GOPATH"); gopath != "" {
		// 2. Try GOPATH
		gopathManifests := filepath.Join(gopath, "src/github.com/nandemo-ya/kecs/controlplane/manifests")
		if _, err := os.Stat(gopathManifests); err == nil {
			manifestsDir = gopathManifests
		}
	}
	
	// 3. Try to find the executable path and work from there
	if manifestsDir == "" {
		execPath, err := os.Executable()
		if err == nil {
			execDir := filepath.Dir(execPath)
			// Check if we're in bin directory
			if filepath.Base(execDir) == "bin" {
				// Go up one level and look for controlplane/manifests
				parentDir := filepath.Dir(execDir)
				possiblePath := filepath.Join(parentDir, "controlplane/manifests")
				if _, err := os.Stat(possiblePath); err == nil {
					manifestsDir = possiblePath
				}
			}
		}
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
		UseTraefik:    cfg.Features.Traefik,
		Namespace:     "kecs-system",
		Services:      cfg.LocalStack.Services,
		Port:          4566,
		EdgePort:      4566,
		ProxyEndpoint: "http://traefik.kecs-system.svc.cluster.local:4566",
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
	fmt.Println("Deploying LocalStack...")
	if err := lsManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start LocalStack: %w", err)
	}

	// Wait for LocalStack to be ready
	fmt.Print("Waiting for LocalStack to be ready...")
	for i := 0; i < 60; i++ { // Wait up to 5 minutes
		if lsManager.IsHealthy() {
			fmt.Println(" ready!")
			status, err := lsManager.GetStatus()
			if err == nil {
				fmt.Printf("LocalStack running: %v\n", status.Running)
				fmt.Printf("LocalStack services: %v\n", status.EnabledServices)
			}
			return nil
		}
		time.Sleep(5 * time.Second)
		fmt.Print(".")
	}

	return fmt.Errorf("LocalStack did not become ready in time")
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
	traefikManifestsDir := ""
	
	// Try to find Traefik manifests directory
	// 1. Try relative path from current directory
	if _, err := os.Stat("manifests/traefik"); err == nil {
		traefikManifestsDir = "manifests/traefik"
	} else if _, err := os.Stat("controlplane/manifests/traefik"); err == nil {
		traefikManifestsDir = "controlplane/manifests/traefik"
	} else if gopath := os.Getenv("GOPATH"); gopath != "" {
		// 2. Try GOPATH
		gopathManifests := filepath.Join(gopath, "src/github.com/nandemo-ya/kecs/controlplane/manifests/traefik")
		if _, err := os.Stat(gopathManifests); err == nil {
			traefikManifestsDir = gopathManifests
		}
	}
	
	// 3. Try to find the executable path and work from there
	if traefikManifestsDir == "" {
		execPath, err := os.Executable()
		if err == nil {
			execDir := filepath.Dir(execPath)
			// Check if we're in bin directory
			if filepath.Base(execDir) == "bin" {
				// Go up one level and look for controlplane/manifests/traefik
				parentDir := filepath.Dir(execDir)
				possiblePath := filepath.Join(parentDir, "controlplane/manifests/traefik")
				if _, err := os.Stat(possiblePath); err == nil {
					traefikManifestsDir = possiblePath
				}
			}
		}
	}

	// Check if manifests directory exists
	if _, err := os.Stat(traefikManifestsDir); os.IsNotExist(err) {
		return fmt.Errorf("traefik manifests directory not found: %s", traefikManifestsDir)
	}

	// Apply Traefik manifests
	fmt.Println("Applying Traefik gateway manifests...")
	cmd := exec.Command("kubectl", "apply", "-k", traefikManifestsDir, "--kubeconfig", manager.GetKubeconfigPath(clusterName))
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

	fmt.Println("Checking component readiness...")
	
	allReady := true
	for _, comp := range components {
		fmt.Printf("  %s: ", comp.name)
		
		// Check deployment
		deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, comp.deployment, metav1.GetOptions{})
		if err != nil {
			if comp.required {
				fmt.Printf("❌ Not found\n")
				allReady = false
			} else {
				fmt.Printf("⏭️  Skipped (optional)\n")
			}
			continue
		}

		// Check if deployment is ready
		if deployment.Status.ReadyReplicas >= 1 {
			fmt.Printf("✅ Ready (%d/%d replicas)\n", deployment.Status.ReadyReplicas, *deployment.Spec.Replicas)
		} else {
			fmt.Printf("⏳ Not ready (0/%d replicas)\n", *deployment.Spec.Replicas)
			if comp.required {
				allReady = false
			}
		}

		// Check service endpoint
		service, err := kubeClient.CoreV1().Services(namespace).Get(ctx, comp.deployment, metav1.GetOptions{})
		if err == nil && len(service.Spec.Ports) > 0 {
			fmt.Printf("    Service: %s:%d\n", service.Name, service.Spec.Ports[0].Port)
		}
	}

	if !allReady {
		return fmt.Errorf("some required components are not ready")
	}

	// Check API connectivity
	fmt.Print("\nChecking API connectivity...")
	
	// Test KECS control plane health endpoint
	adminEndpoint := fmt.Sprintf("http://localhost:%d/health", startV2AdminPort)
	if err := checkEndpointHealth(adminEndpoint, 30*time.Second); err != nil {
		fmt.Printf(" ❌\n")
		return fmt.Errorf("KECS admin API not accessible: %w", err)
	}
	fmt.Printf(" ✅\n")

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
	tracker.UpdateTask("controlplane", 20, "Preparing manifests")
	
	// Get k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	tracker.UpdateTask("controlplane", 30, "Getting Kubernetes client")
	
	// Get Kubernetes client
	kubeClient, err := manager.GetKubeClient(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	tracker.UpdateTask("controlplane", 40, "Locating manifests")
	
	// Find manifests directory
	manifestsDir := ""
	if _, err := os.Stat("manifests"); err == nil {
		manifestsDir = "manifests"
	} else if _, err := os.Stat("controlplane/manifests"); err == nil {
		manifestsDir = "controlplane/manifests"
	} else if gopath := os.Getenv("GOPATH"); gopath != "" {
		gopathManifests := filepath.Join(gopath, "src/github.com/nandemo-ya/kecs/controlplane/manifests")
		if _, err := os.Stat(gopathManifests); err == nil {
			manifestsDir = gopathManifests
		}
	}
	
	// Try to find the executable path
	if manifestsDir == "" {
		execPath, err := os.Executable()
		if err == nil {
			execDir := filepath.Dir(execPath)
			if filepath.Base(execDir) == "bin" {
				parentDir := filepath.Dir(execDir)
				possiblePath := filepath.Join(parentDir, "controlplane/manifests")
				if _, err := os.Stat(possiblePath); err == nil {
					manifestsDir = possiblePath
				}
			}
		}
	}

	if _, err := os.Stat(manifestsDir); os.IsNotExist(err) {
		return fmt.Errorf("manifests directory not found: %s", manifestsDir)
	}

	tracker.UpdateTask("controlplane", 60, "Applying manifests")
	
	// Apply manifests
	cmd := exec.Command("kubectl", "apply", "-k", manifestsDir, "--kubeconfig", manager.GetKubeconfigPath(clusterName))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to apply manifests: %w", err)
	}

	tracker.UpdateTask("controlplane", 80, "Waiting for deployment")
	
	// Wait for deployment to be ready
	deployment := "kecs-controlplane"
	namespace := "kecs-system"
	
	for i := 0; i < 60; i++ { // Wait up to 5 minutes
		deps, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, deployment, metav1.GetOptions{})
		if err == nil && deps.Status.ReadyReplicas > 0 {
			tracker.UpdateTask("controlplane", 100, "Ready")
			return nil
		}
		time.Sleep(5 * time.Second)
		progress := 80 + (i * 20 / 60)
		tracker.UpdateTask("controlplane", progress, fmt.Sprintf("Waiting for pods (%d/60s)", i*5))
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
		UseTraefik:    cfg.Features.Traefik,
		Namespace:     "kecs-system",
		Services:      cfg.LocalStack.Services,
		Port:          4566,
		EdgePort:      4566,
		ProxyEndpoint: "http://traefik.kecs-system.svc.cluster.local:4566",
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

	tracker.UpdateTask("localstack", 70, "Waiting for LocalStack to be ready")
	
	// Wait for LocalStack to be ready
	for i := 0; i < 60; i++ { // Wait up to 5 minutes
		if lsManager.IsHealthy() {
			tracker.UpdateTask("localstack", 100, "Ready")
			return nil
		}
		time.Sleep(5 * time.Second)
		progress := 70 + (i * 30 / 60)
		tracker.UpdateTask("localstack", progress, fmt.Sprintf("Health check (%d/60s)", i*5))
	}

	return fmt.Errorf("LocalStack did not become ready in time")
}