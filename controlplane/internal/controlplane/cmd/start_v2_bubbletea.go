package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/progress"
	"github.com/nandemo-ya/kecs/controlplane/internal/progress/bubbletea"
)

// runStartV2WithBubbleTea is an alternative implementation using Bubble Tea
func runStartV2WithBubbleTea(ctx context.Context, instanceName string, cfg *config.Config, dataDir string) error {
	// Show header
	progress.SectionHeader(fmt.Sprintf("Creating KECS instance '%s'", instanceName))

	// Step 1: Create k3d cluster
	spinner := progress.NewSpinner("Creating k3d cluster")
	spinner.Start()
	
	if err := createK3dCluster(ctx, instanceName, cfg, dataDir); err != nil {
		spinner.Fail("Failed to create k3d cluster")
		return fmt.Errorf("failed to create k3d cluster: %w", err)
	}
	spinner.Success("k3d cluster created")

	// Step 2: Create kecs-system namespace
	spinner = progress.NewSpinner("Creating kecs-system namespace")
	spinner.Start()
	if err := createKecsSystemNamespace(ctx, instanceName); err != nil {
		spinner.Fail("Failed to create namespace")
		return fmt.Errorf("failed to create kecs-system namespace: %w", err)
	}
	spinner.Success("kecs-system namespace created")

	// Step 3: Deploy components using Bubble Tea
	err := bubbletea.RunWithBubbleTea(ctx, "Deploying components", func(tracker *bubbletea.Adapter) error {
		// Add tasks
		tracker.AddTask("controlplane", "Control Plane", 100)
		if cfg.LocalStack.Enabled {
			tracker.AddTask("localstack", "LocalStack", 100)
		}
		
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
		
		return nil
	})
	
	if err != nil {
		return err
	}

	// Step 4: Deploy Traefik gateway (if enabled)
	if cfg.Features.Traefik {
		spinner = progress.NewSpinner("Deploying Traefik gateway")
		spinner.Start()
		if err := deployTraefikGateway(ctx, instanceName, cfg, startV2ApiPort); err != nil {
			spinner.Fail("Failed to deploy Traefik")
			return fmt.Errorf("failed to deploy Traefik gateway: %w", err)
		}
		spinner.Success("Traefik gateway deployed")
	}

	// Step 5: Wait for all components to be ready
	spinner = progress.NewSpinner("Waiting for all components to be ready")
	spinner.Start()
	if err := waitForComponents(ctx, instanceName); err != nil {
		spinner.Fail("Components failed to become ready")
		return fmt.Errorf("components did not become ready: %w", err)
	}
	spinner.Success("All components are ready")

	// Show success summary
	progress.Success("KECS instance '%s' is ready!", instanceName)
	
	fmt.Println()
	progress.Info("Endpoints:")
	fmt.Printf("  AWS API: http://localhost:%d\n", startV2ApiPort)
	fmt.Printf("  Admin API: http://localhost:%d\n", startV2AdminPort)
	fmt.Printf("  Data directory: %s\n", dataDir)

	if cfg.LocalStack.Enabled {
		fmt.Printf("\nLocalStack services: %v\n", cfg.LocalStack.Services)
	}

	fmt.Println()
	progress.Info("Next steps:")
	fmt.Printf("  To stop this instance: kecs stop-v2 --instance %s\n", instanceName)
	fmt.Printf("  To get kubeconfig: kecs kubeconfig get %s\n", instanceName)

	return nil
}

// deployControlPlaneWithBubbleTeaProgress wraps deployControlPlane with Bubble Tea progress reporting
func deployControlPlaneWithBubbleTeaProgress(ctx context.Context, clusterName string, cfg *config.Config, dataDir string, tracker *bubbletea.Adapter) error {
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
	maxWaitTime := 60 // 5 minutes (60 * 5 seconds)
	
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

	// Give LocalStack a moment to initialize before checking health
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