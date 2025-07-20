package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes/resources"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/progress"
	"github.com/nandemo-ya/kecs/controlplane/internal/progress/bubbletea"
	"github.com/sirupsen/logrus"
)

// runStartWithBubbleTea is an alternative implementation using Bubble Tea
func runStartWithBubbleTea(ctx context.Context, instanceName string, cfg *config.Config, dataDir string) error {
	// Set environment variables to suppress external tool logs before starting
	originalK3dLogLevel := os.Getenv("K3D_LOG_LEVEL")
	os.Setenv("K3D_LOG_LEVEL", "panic") // Only show critical errors
	
	// Also suppress other potential loggers
	os.Setenv("DOCKER_CLI_HINTS", "false")
	os.Setenv("LOGRUS_LEVEL", "panic")
	
	// Configure logrus immediately to discard output
	logrus.SetLevel(logrus.PanicLevel)
	originalLogrusOut := logrus.StandardLogger().Out
	logrus.SetOutput(io.Discard)
	
	defer func() {
		if originalK3dLogLevel != "" {
			os.Setenv("K3D_LOG_LEVEL", originalK3dLogLevel)
		} else {
			os.Unsetenv("K3D_LOG_LEVEL")
		}
		os.Unsetenv("DOCKER_CLI_HINTS")
		os.Unsetenv("LOGRUS_LEVEL")
		
		// Restore logrus
		logrus.SetOutput(originalLogrusOut)
		logrus.SetLevel(logrus.InfoLevel)
	}()
	
	// Use Bubble Tea for the entire process with silent start
	return bubbletea.RunWithBubbleTeaSilent(ctx, fmt.Sprintf("Creating KECS instance '%s'", instanceName), func(tracker *bubbletea.Adapter) error {
		// Log the generated name if it was auto-generated
		if instanceName != "" {
			tracker.Log(progress.LogLevelInfo, "KECS instance name: %s", instanceName)
		}
		
		// Add all tasks upfront so they're visible immediately
		tracker.AddTask("k3d-cluster", "k3d cluster", 100)
		tracker.AddTask("namespace", "kecs-system namespace", 100)
		tracker.AddTask("controlplane", "Control Plane", 100)
		if cfg.LocalStack.Enabled {
			tracker.AddTask("localstack", "LocalStack", 100)
		}
		if cfg.Features.Traefik {
			tracker.AddTask("traefik", "Traefik gateway", 100)
		}
		tracker.AddTask("wait-ready", "Waiting for components", 100)

		// Step 1: Create k3d cluster
		tracker.StartTask("k3d-cluster")
		tracker.UpdateTask("k3d-cluster", 10, "Creating cluster...")
		if err := createK3dClusterWithProgress(ctx, instanceName, cfg, dataDir, tracker); err != nil {
			tracker.FailTask("k3d-cluster", err)
			return fmt.Errorf("failed to create k3d cluster: %w", err)
		}
		tracker.CompleteTask("k3d-cluster")

		// Step 2: Create kecs-system namespace
		tracker.StartTask("namespace")
		tracker.UpdateTask("namespace", 10, "Creating namespace...")
		if err := createKecsSystemNamespaceWithProgress(ctx, instanceName, tracker); err != nil {
			tracker.FailTask("namespace", err)
			return fmt.Errorf("failed to create kecs-system namespace: %w", err)
		}
		tracker.CompleteTask("namespace")

		// Step 3: Deploy Control Plane and LocalStack in parallel
		var wg sync.WaitGroup
		errChan := make(chan error, 2)
		
		// Deploy Control Plane
		wg.Add(1)
		go func() {
			defer wg.Done()
			tracker.StartTask("controlplane")
			if err := deployControlPlaneWithBubbleTeaProgress(ctx, instanceName, cfg, dataDir, tracker); err != nil {
				tracker.FailTask("controlplane", err)
				errChan <- fmt.Errorf("failed to deploy control plane: %w", err)
				return
			}
			tracker.CompleteTask("controlplane")
		}()
		
		// Deploy LocalStack (if enabled)
		if cfg.LocalStack.Enabled {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tracker.StartTask("localstack")
				if err := deployLocalStackWithBubbleTeaProgress(ctx, instanceName, cfg, tracker); err != nil {
					tracker.FailTask("localstack", err)
					errChan <- fmt.Errorf("failed to deploy LocalStack: %w", err)
					return
				}
				tracker.CompleteTask("localstack")
			}()
		}
		
		// Wait for parallel deployments to complete
		wg.Wait()
		close(errChan)
		
		// Check for errors from parallel deployments
		for err := range errChan {
			return err
		}

		// Step 4: Deploy Traefik gateway (if enabled)
		if cfg.Features.Traefik {
			tracker.StartTask("traefik")
			tracker.UpdateTask("traefik", 10, "Deploying gateway...")
			if err := deployTraefikGatewayWithProgress(ctx, instanceName, cfg, startApiPort, tracker); err != nil {
				tracker.FailTask("traefik", err)
				return fmt.Errorf("failed to deploy Traefik gateway: %w", err)
			}
			tracker.CompleteTask("traefik")
		}

		// Step 5: Wait for all components to be ready
		tracker.StartTask("wait-ready")
		tracker.UpdateTask("wait-ready", 10, "Checking components...")
		if err := waitForComponentsWithProgress(ctx, instanceName, tracker); err != nil {
			tracker.FailTask("wait-ready", err)
			return fmt.Errorf("components did not become ready: %w", err)
		}
		tracker.CompleteTask("wait-ready")

		// All done - show success in logs
		tracker.Log(progress.LogLevelInfo, "🎉 KECS instance '%s' is ready!", instanceName)
		tracker.Log(progress.LogLevelInfo, "")
		tracker.Log(progress.LogLevelInfo, "Endpoints:")
		tracker.Log(progress.LogLevelInfo, "  AWS API: http://localhost:%d", startApiPort)
		tracker.Log(progress.LogLevelInfo, "  Admin API: http://localhost:%d", startAdminPort)
		tracker.Log(progress.LogLevelInfo, "  Data directory: %s", dataDir)
		
		if cfg.LocalStack.Enabled {
			tracker.Log(progress.LogLevelInfo, "")
			tracker.Log(progress.LogLevelInfo, "LocalStack services: %v", cfg.LocalStack.Services)
		}
		
		tracker.Log(progress.LogLevelInfo, "")
		tracker.Log(progress.LogLevelInfo, "Next steps:")
		tracker.Log(progress.LogLevelInfo, "  To stop this instance: kecs stop --instance %s", instanceName)
		tracker.Log(progress.LogLevelInfo, "  To get kubeconfig: kecs kubeconfig get %s", instanceName)
		
		return nil
	})
}

// createK3dClusterWithProgress wraps createK3dCluster with progress reporting
func createK3dClusterWithProgress(ctx context.Context, clusterName string, cfg *config.Config, dataDir string, tracker *bubbletea.Adapter) error {
	tracker.UpdateTask("k3d-cluster", 30, "Initializing cluster manager...")
	
	// Suppress k3d logs temporarily
	originalLogLevel := os.Getenv("K3D_LOG_LEVEL")
	os.Setenv("K3D_LOG_LEVEL", "error")
	defer func() {
		if originalLogLevel != "" {
			os.Setenv("K3D_LOG_LEVEL", originalLogLevel)
		} else {
			os.Unsetenv("K3D_LOG_LEVEL")
		}
	}()
	
	// Create k3d cluster manager configuration
	clusterConfig := &kubernetes.ClusterManagerConfig{
		Provider:      "k3d",
		ContainerMode: false,
		EnableTraefik: false, // Disable old Traefik deployment (we use our own)
		TraefikPort:   startApiPort,
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

	tracker.UpdateTask("k3d-cluster", 50, "Checking if cluster exists...")

	// Check if cluster already exists
	exists, err := manager.ClusterExists(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to check cluster existence: %w", err)
	}

	if exists {
		tracker.Log(progress.LogLevelInfo, "k3d cluster '%s' already exists, using existing cluster", clusterName)
		tracker.UpdateTask("k3d-cluster", 100, "Using existing cluster")
		return nil
	}

	tracker.UpdateTask("k3d-cluster", 70, "Creating k3d cluster...")

	// Create the cluster
	if err := manager.CreateCluster(ctx, clusterName); err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	tracker.UpdateTask("k3d-cluster", 90, "Waiting for cluster to be ready...")

	// Wait for cluster to be ready
	if err := manager.WaitForClusterReady(clusterName, 5*time.Minute); err != nil {
		return fmt.Errorf("cluster did not become ready: %w", err)
	}

	tracker.UpdateTask("k3d-cluster", 100, "Cluster is ready!")
	return nil
}

// createKecsSystemNamespaceWithProgress wraps createKecsSystemNamespace with progress reporting
func createKecsSystemNamespaceWithProgress(ctx context.Context, clusterName string, tracker *bubbletea.Adapter) error {
	tracker.UpdateTask("namespace", 30, "Getting Kubernetes client...")
	
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

	tracker.UpdateTask("namespace", 60, "Creating namespace...")

	// Create kecs-system namespace directly
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
		tracker.Log(progress.LogLevelInfo, "kecs-system namespace already exists")
		tracker.UpdateTask("namespace", 100, "Namespace already exists")
		return nil
	}

	_, err = kubeClient.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create kecs-system namespace: %w", err)
	}

	tracker.UpdateTask("namespace", 100, "Namespace created")
	tracker.Log(progress.LogLevelInfo, "Created kecs-system namespace")
	return nil
}

// deployControlPlaneWithBubbleTeaProgress wraps deployControlPlane with Bubble Tea progress reporting
func deployControlPlaneWithBubbleTeaProgress(ctx context.Context, clusterName string, cfg *config.Config, dataDir string, tracker *bubbletea.Adapter) error {
	// Update progress during deployment
	tracker.UpdateTask("controlplane", 20, "Preparing resources")
	
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

	tracker.UpdateTask("controlplane", 40, "Creating deployer")
	
	// Create resource deployer
	deployer := kubernetes.NewResourceDeployer(kubeClient)
	
	// Configure control plane
	controlPlaneConfig := &resources.ControlPlaneConfig{
		Image:           cfg.Server.ControlPlaneImage,
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

// deployLocalStackWithBubbleTeaProgress wraps deployLocalStack with Bubble Tea progress reporting
func deployLocalStackWithBubbleTeaProgress(ctx context.Context, clusterName string, cfg *config.Config, tracker *bubbletea.Adapter) error {
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
	tracker.UpdateTask("localstack", 60, "LocalStack started, initializing...")
	time.Sleep(3 * time.Second)

	tracker.UpdateTask("localstack", 70, "Waiting for LocalStack to be ready")
	
	// Wait for LocalStack to be ready
	maxWaitTime := 120 // 10 minutes (120 * 5 seconds) - increased for image pull retries
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

// deployTraefikGatewayWithProgress wraps deployTraefikGateway with progress reporting
func deployTraefikGatewayWithProgress(ctx context.Context, clusterName string, cfg *config.Config, apiPort int, tracker *bubbletea.Adapter) error {
	tracker.UpdateTask("traefik", 20, "Getting cluster manager")
	
	// Get k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	tracker.UpdateTask("traefik", 30, "Getting Kubernetes client")
	
	// Get Kubernetes client and config
	kubeClient, err := manager.GetKubeClient(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}
	
	kubeConfig, err := manager.GetKubeConfig(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes config: %w", err)
	}
	
	tracker.UpdateTask("traefik", 40, "Creating deployer")
	
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

	tracker.UpdateTask("traefik", 60, "Deploying Traefik resources")
	
	// Deploy Traefik resources programmatically
	if err := deployer.DeployTraefik(ctx, traefikConfig); err != nil {
		return fmt.Errorf("failed to deploy Traefik: %w", err)
	}

	tracker.UpdateTask("traefik", 80, "Waiting for Traefik deployment")
	
	// Wait for Traefik deployment to be ready
	deployment := "traefik"
	namespace := "kecs-system"
	
	for i := 0; i < 60; i++ { // Wait up to 5 minutes
		deps, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, deployment, metav1.GetOptions{})
		if err == nil && deps.Status.ReadyReplicas > 0 {
			tracker.Log(progress.LogLevelInfo, "Traefik deployment is ready!")
			break
		}
		time.Sleep(5 * time.Second)
		
		progress := 80 + (i * 15 / 60) // Progress from 80% to 95%
		tracker.UpdateTask("traefik", progress, fmt.Sprintf("Waiting for pods (%ds/300s)", (i+1)*5))
	}

	tracker.UpdateTask("traefik", 95, "Waiting for Traefik service")
	
	// Wait for Traefik service to get external IP/port
	service := "traefik"
	
	for i := 0; i < 30; i++ { // Wait up to 2.5 minutes
		svc, err := kubeClient.CoreV1().Services(namespace).Get(ctx, service, metav1.GetOptions{})
		if err == nil && len(svc.Status.LoadBalancer.Ingress) > 0 {
			tracker.Log(progress.LogLevelInfo, "Traefik service is ready!")
			tracker.Log(progress.LogLevelInfo, "Traefik LoadBalancer: %s", svc.Status.LoadBalancer.Ingress[0].Hostname)
			tracker.UpdateTask("traefik", 100, "Service ready")
			return nil
		}
		time.Sleep(5 * time.Second)
	}

	// For k3d, the LoadBalancer might not get an external IP
	tracker.Log(progress.LogLevelInfo, "Traefik service is ready! (using k3d port mapping)")
	tracker.UpdateTask("traefik", 100, "Ready with k3d port mapping")
	
	return nil
}

// waitForComponentsWithProgress wraps waitForComponents with progress reporting
func waitForComponentsWithProgress(ctx context.Context, clusterName string, tracker *bubbletea.Adapter) error {
	tracker.UpdateTask("wait-ready", 20, "Getting cluster manager")
	
	// Get k3d cluster manager
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf("failed to create cluster manager: %w", err)
	}

	tracker.UpdateTask("wait-ready", 30, "Getting Kubernetes client")
	
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

	tracker.UpdateTask("wait-ready", 50, "Checking component readiness")
	tracker.Log(progress.LogLevelInfo, "Checking component readiness...")
	
	allReady := true
	for _, comp := range components {
		// Check deployment
		deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, comp.deployment, metav1.GetOptions{})
		if err != nil {
			if comp.required {
				tracker.Log(progress.LogLevelError, "  %s: ❌ Not found", comp.name)
				allReady = false
			} else {
				tracker.Log(progress.LogLevelInfo, "  %s: ⏭️  Skipped (optional)", comp.name)
			}
			continue
		}

		// Check if deployment is ready
		if deployment.Status.ReadyReplicas >= 1 {
			tracker.Log(progress.LogLevelInfo, "  %s: ✅ Ready (%d/%d replicas)", comp.name, deployment.Status.ReadyReplicas, *deployment.Spec.Replicas)
		} else {
			tracker.Log(progress.LogLevelWarning, "  %s: ⏳ Not ready (0/%d replicas)", comp.name, *deployment.Spec.Replicas)
			if comp.required {
				allReady = false
			}
		}

		// Check service endpoint
		service, err := kubeClient.CoreV1().Services(namespace).Get(ctx, comp.deployment, metav1.GetOptions{})
		if err == nil && len(service.Spec.Ports) > 0 {
			tracker.Log(progress.LogLevelInfo, "    Service: %s:%d", service.Name, service.Spec.Ports[0].Port)
		}
	}

	if !allReady {
		return fmt.Errorf("some required components are not ready")
	}

	// Skip external health checks for k8s deployments
	// The deployment readiness checks above are sufficient
	tracker.Log(progress.LogLevelInfo, "✅ All components are ready")
	
	tracker.UpdateTask("wait-ready", 100, "All components ready")
	return nil
}