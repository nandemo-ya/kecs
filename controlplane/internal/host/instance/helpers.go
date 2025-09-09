// Copyright 2025 The KECS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package instance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/host/k3d"
	kecs "github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes/resources"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

// generateInstanceName generates a unique instance name
func generateInstanceName() string {
	name, _ := utils.GenerateRandomName()
	return name
}

// createCluster creates the k3d cluster
func (m *Manager) createCluster(ctx context.Context, instanceName string, cfg *config.Config, opts StartOptions) error {
	clusterName := fmt.Sprintf("kecs-%s", instanceName)

	// Calculate NodePort for API access
	apiNodePort := int32(opts.ApiPort)
	if apiNodePort < 30000 {
		apiNodePort = apiNodePort + 22000
	}
	if apiNodePort < 30000 || apiNodePort > 32767 {
		apiNodePort = 30080 // fallback to default
	}

	// Calculate NodePort for Admin access
	adminNodePort := int32(opts.AdminPort)
	if adminNodePort < 30000 {
		adminNodePort = adminNodePort + 22000
	}
	if adminNodePort < 30000 || adminNodePort > 32767 {
		adminNodePort = 30081 // fallback to default
	}

	// Create port mappings for k3d cluster
	portMappings := map[int32]int32{
		int32(opts.ApiPort):   apiNodePort,   // Map host API port to NodePort for ECS API
		int32(opts.AdminPort): adminNodePort, // Map host Admin port to NodePort for Admin API
	}

	// Set up data directory for direct hostPath mounting
	// This ensures that DuckDB data persists across instance restarts
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".kecs", "instances", instanceName, "data")

	// Ensure the data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Configure volume mounts for persistent storage
	// Map the host data directory directly to container
	volumeMounts := []k3d.VolumeMount{
		{
			HostPath:      dataDir,
			ContainerPath: dataDir, // Mount to same path in container
		},
	}

	// Set volume mounts using the setter method
	m.k3dManager.SetVolumeMounts(volumeMounts)

	// Log volume mounts for debugging
	logging.Info("Setting volume mounts for k3d cluster",
		"volumeMounts", volumeMounts,
		"dataDir", dataDir)

	// Enable k3d registry
	m.k3dManager.SetEnableRegistry(true)

	// Create cluster with port mappings
	if err := m.k3dManager.CreateClusterWithPortMapping(ctx, clusterName, portMappings); err != nil {
		return err
	}

	return nil
}

// createNamespace creates the kecs-system namespace
func (m *Manager) createNamespace(ctx context.Context, instanceName string) error {
	clusterName := fmt.Sprintf("kecs-%s", instanceName)
	kubeconfig, err := m.k3dManager.GetKubeConfig(context.Background(), clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kecs-system",
		},
	}

	if _, err := client.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	return nil
}

// createOrUpdateNamespace creates the kecs-system namespace or ensures it exists
func (m *Manager) createOrUpdateNamespace(ctx context.Context, instanceName string) error {
	clusterName := fmt.Sprintf("kecs-%s", instanceName)
	kubeconfig, err := m.k3dManager.GetKubeConfig(context.Background(), clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kecs-system",
		},
	}

	// Try to get the namespace first
	_, err = client.CoreV1().Namespaces().Get(ctx, "kecs-system", metav1.GetOptions{})
	if err != nil {
		// Namespace doesn't exist, create it
		if _, err := client.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create namespace: %w", err)
		}
	}
	// Namespace already exists, nothing to do

	return nil
}

// deployControlPlane deploys the KECS control plane
func (m *Manager) deployControlPlane(ctx context.Context, instanceName string, cfg *config.Config, opts StartOptions) error {

	clusterName := fmt.Sprintf("kecs-%s", instanceName)
	kubeconfig, err := m.k3dManager.GetKubeConfig(context.Background(), clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Set up data directory path for hostPath volume
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".kecs", "instances", instanceName, "data")

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Calculate NodePort for API access
	apiNodePort := int32(opts.ApiPort)
	if apiNodePort < 30000 {
		apiNodePort = apiNodePort + 22000
	}
	if apiNodePort < 30000 || apiNodePort > 32767 {
		apiNodePort = 30080 // fallback to default
	}

	// Calculate NodePort for Admin access
	adminNodePort := int32(opts.AdminPort)
	if adminNodePort < 30000 {
		adminNodePort = adminNodePort + 22000
	}
	if adminNodePort < 30000 || adminNodePort > 32767 {
		adminNodePort = 30081 // fallback to default
	}

	// Create control plane config
	controlPlaneConfig := &resources.ControlPlaneConfig{
		Image:           cfg.Server.ControlPlaneImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		CPURequest:      "100m",
		MemoryRequest:   "128Mi",
		CPULimit:        "1000m",
		MemoryLimit:     "1Gi",
		StorageSize:     "10Gi",
		DataHostPath:    dataDir,                                 // Use hostPath for data persistence
		APIPort:         80,                                      // Service port (external facing)
		AdminPort:       resources.ControlPlaneInternalAdminPort, // Admin service port
		APINodePort:     apiNodePort,                             // NodePort for API access
		AdminNodePort:   adminNodePort,                           // NodePort for Admin access
		LogLevel:        cfg.Server.LogLevel,
	}

	// Create control plane resources
	controlPlaneResources := resources.CreateControlPlaneResources(controlPlaneConfig)

	// Create service account
	if controlPlaneResources.ServiceAccount != nil {
		if _, err := client.CoreV1().ServiceAccounts("kecs-system").Create(ctx, controlPlaneResources.ServiceAccount, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create service account: %w", err)
		}
	}

	// Create cluster role
	if controlPlaneResources.ClusterRole != nil {
		if _, err := client.RbacV1().ClusterRoles().Create(ctx, controlPlaneResources.ClusterRole, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create cluster role: %w", err)
		}
	}

	// Create cluster role binding
	if controlPlaneResources.ClusterRoleBinding != nil {
		if _, err := client.RbacV1().ClusterRoleBindings().Create(ctx, controlPlaneResources.ClusterRoleBinding, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create cluster role binding: %w", err)
		}
	}

	// Create config map
	if controlPlaneResources.ConfigMap != nil {
		if _, err := client.CoreV1().ConfigMaps("kecs-system").Create(ctx, controlPlaneResources.ConfigMap, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create config map: %w", err)
		}
	}

	// Create PVC (only if not using hostPath)
	if controlPlaneResources.PVC != nil {
		if _, err := client.CoreV1().PersistentVolumeClaims("kecs-system").Create(ctx, controlPlaneResources.PVC, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create PVC: %w", err)
		}
	}

	// Deploy deployment
	if _, err := client.AppsV1().Deployments("kecs-system").Create(ctx, controlPlaneResources.Deployment, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create services
	for _, service := range controlPlaneResources.Services {
		if _, err := client.CoreV1().Services("kecs-system").Create(ctx, service, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create service %s: %w", service.Name, err)
		}
	}

	return nil
}

// deployLocalStack deploys LocalStack
func (m *Manager) deployLocalStack(ctx context.Context, instanceName string, cfg *config.Config) error {

	clusterName := fmt.Sprintf("kecs-%s", instanceName)
	kubeconfig, err := m.k3dManager.GetKubeConfig(context.Background(), clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Create LocalStack config
	localstackConfig := &localstack.Config{
		Enabled:   true,
		Namespace: "kecs-system",
		Services:  cfg.LocalStack.Services,
		Port:      4566,
		EdgePort:  4566,
		Image:     cfg.LocalStack.Image,
		Version:   cfg.LocalStack.Version,
	}

	manager, err := localstack.NewManager(localstackConfig, client, kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create LocalStack manager: %w", err)
	}

	if err := manager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start LocalStack: %w", err)
	}

	return nil
}

// deployTraefik is deprecated and does nothing
func (m *Manager) deployTraefik(ctx context.Context, instanceName string, cfg *config.Config, apiPort int) error {
	return nil
}

// waitForReady waits for all components to be ready
func (m *Manager) waitForReady(ctx context.Context, instanceName string, cfg *config.Config) error {

	clusterName := fmt.Sprintf("kecs-%s", instanceName)
	kubeconfig, err := m.k3dManager.GetKubeConfig(context.Background(), clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Wait for control plane
	if err := waitForDeployment(ctx, client, "kecs-system", "kecs-controlplane"); err != nil {
		return fmt.Errorf("control plane failed to become ready: %w", err)
	}

	// Wait for LocalStack if enabled
	if cfg.LocalStack.Enabled {
		if err := waitForDeployment(ctx, client, "kecs-system", "localstack"); err != nil {
			return fmt.Errorf("LocalStack failed to become ready: %w", err)
		}
	}

	return nil
}

// waitForDeployment waits for a deployment to be ready
func waitForDeployment(ctx context.Context, client kubernetes.Interface, namespace, name string) error {
	// Implementation would check deployment status
	// For now, just wait a bit
	select {
	case <-time.After(5 * time.Second):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// deployVector deploys Vector DaemonSet for log aggregation
func (m *Manager) deployVector(ctx context.Context, instanceName string, cfg *config.Config) error {
	clusterName := fmt.Sprintf("kecs-%s", instanceName)
	kubeconfig, err := m.k3dManager.GetKubeConfig(context.Background(), clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	client, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Get LocalStack endpoint if available
	localstackEndpoint := ""
	if cfg.LocalStack.Enabled {
		// LocalStack endpoint is always the cluster-internal endpoint
		localstackEndpoint = "http://localstack.kecs-system.svc.cluster.local:4566"
	}

	// Get region from config
	region := cfg.AWS.DefaultRegion
	if region == "" {
		region = "us-east-1"
	}

	// Deploy Vector using singleton pattern
	// This ensures Vector is only deployed once per KECS instance
	if err := kecs.DeployVectorOnce(ctx, client, localstackEndpoint, region); err != nil {
		return fmt.Errorf("failed to deploy Vector: %w", err)
	}

	return nil
}
