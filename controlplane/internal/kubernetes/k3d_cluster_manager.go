package kubernetes

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/docker/go-connections/nat"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/k3d-io/k3d/v5/pkg/client"
	"github.com/k3d-io/k3d/v5/pkg/config/v1alpha5"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/k3d-io/k3d/v5/pkg/runtimes/docker"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// K3dClusterManager implements ClusterManager interface using k3d
type K3dClusterManager struct {
	runtime         runtimes.Runtime
	config          *ClusterManagerConfig
	portForwarder   *PortForwarder
	traefikPorts    map[string]int // cluster name -> traefik port mapping
	portMutex       sync.Mutex     // protects port allocation
}

// NewK3dClusterManager creates a new k3d-based cluster manager
func NewK3dClusterManager(cfg *ClusterManagerConfig) (*K3dClusterManager, error) {
	
	if cfg == nil {
		cfg = &ClusterManagerConfig{
			Provider:      "k3d",
			ContainerMode: config.GetBool("features.containerMode"),
		}
	}

	// Use the Docker runtime from k3d
	runtime := runtimes.Docker

	logging.Info("Creating K3dClusterManager with config",
		"containerMode", cfg.ContainerMode,
		"enableTraefik", cfg.EnableTraefik)

	return &K3dClusterManager{
		runtime:      runtime,
		config:       cfg,
		traefikPorts: make(map[string]int),
	}, nil
}

// CreateCluster creates a new k3d cluster with optimizations based on environment
func (k *K3dClusterManager) CreateCluster(ctx context.Context, clusterName string) error {
	// Use optimized creation for test mode or when explicitly requested
	if config.GetBool("features.testMode") || config.GetBool("kubernetes.k3dOptimized") {
		return k.CreateClusterOptimized(ctx, clusterName)
	}
	
	// Use standard creation for production-like scenarios
	return k.createClusterStandard(ctx, clusterName)
}

// createClusterStandard creates a standard k3d cluster (original implementation)
func (k *K3dClusterManager) createClusterStandard(ctx context.Context, clusterName string) error {
	normalizedName := k.normalizeClusterName(clusterName)

	// Check if cluster already exists
	exists, err := k.ClusterExists(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to check if cluster exists: %w", err)
	}

	if exists {
		logging.Info("k3d cluster already exists", "cluster", normalizedName)
		return nil
	}
	
	// Handle registry for dev mode
	var registryNode *k3d.Node
	if k.config.EnableRegistry {
		registryNode, err = k.ensureRegistry(ctx)
		if err != nil {
			return fmt.Errorf("failed to ensure registry: %w", err)
		}
		logging.Info("Using k3d registry for dev mode", "registry", registryNode.Name)
	}

	// Determine k3s image
	k3sImage := "rancher/k3s:v1.31.4-k3s1"
	if k.config.K3dImage != "" {
		k3sImage = k.config.K3dImage
	}

	// K3s args for minimal setup - disable unnecessary components
	k3sArgs := []string{
		"--disable=traefik",        // Disable Traefik ingress controller
		"--disable=servicelb",      // Disable the default service load balancer
		"--disable=metrics-server", // Disable metrics server
		"--disable-network-policy", // Disable network policy controller
	}

	// Create server node
	serverNode := &k3d.Node{
		Name:    fmt.Sprintf("k3d-%s-server-0", normalizedName),
		Role:    k3d.ServerRole,
		Image:   k3sImage,
		Restart: true,
		Args:    k3sArgs,
		K3sNodeLabels: map[string]string{
			"kecs.io/cluster": normalizedName,
		},
		Env: []string{
			"K3S_KUBECONFIG_MODE=666", // Ensure kubeconfig is readable
		},
	}
	
	// Add volume mounts if specified
	if len(k.config.VolumeMounts) > 0 {
		volumes := []string{}
		for _, mount := range k.config.VolumeMounts {
			// k3d expects volume format as "hostPath:containerPath"
			volumes = append(volumes, fmt.Sprintf("%s:%s", mount.HostPath, mount.ContainerPath))
		}
		serverNode.Volumes = volumes
		logging.Info("Adding volume mounts", "volumes", volumes)
	}
	
	// Add port mapping for HTTP access (needed regardless of Traefik deployment)
	// This maps host port to NodePort 30890 for accessing services in the cluster
	{
		// Lock for thread-safe port allocation
		k.portMutex.Lock()
		
		// Determine port for HTTP access
		httpPort := k.config.TraefikPort
		if httpPort == 0 {
			// Find available port starting from 8090
			port, err := k.findAvailablePort(8090)
			if err != nil {
				k.portMutex.Unlock()
				return fmt.Errorf("failed to find available port for HTTP access: %w", err)
			}
			httpPort = port
		}
		
		// Store the port for this cluster
		k.traefikPorts[normalizedName] = httpPort
		k.portMutex.Unlock()
		
		logging.Info("Adding port mapping for HTTP access",
			"hostPort", httpPort,
			"nodePort", 30890)
		serverNode.Ports = nat.PortMap{
			"30890/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: fmt.Sprintf("%d", httpPort),
				},
			},
		}
	}

	// Create cluster configuration with minimal required fields
	// In container mode, check if KECS network is specified
	networkName := fmt.Sprintf("k3d-%s", normalizedName)
	if k.config.ContainerMode {
		if kecsNetwork := config.GetString("docker.network"); kecsNetwork != "" {
			logging.Info("Using KECS Docker network", "network", kecsNetwork)
			networkName = kecsNetwork
		}
	}
	
	cluster := &k3d.Cluster{
		Name:  normalizedName,
		Nodes: []*k3d.Node{serverNode},
		Network: k3d.ClusterNetwork{
			Name: networkName,
			IPAM: k3d.IPAM{
				Managed: false,
			},
		},
		Token: fmt.Sprintf("kecs-%s-token", normalizedName),
		KubeAPI: &k3d.ExposureOpts{
			Host: k3d.DefaultAPIHost,
		},
	}
	
	// Registry connection will be handled after cluster creation

	// Create cluster creation options
	clusterCreateOpts := &k3d.ClusterCreateOpts{
		WaitForServer:       true,
		Timeout:             2 * time.Minute,
		DisableLoadBalancer: false,
		GlobalLabels:        make(map[string]string),
		GlobalEnv:           []string{},
		NodeHooks:           []k3d.NodeHook{},
	}

	// Create cluster config for ClusterRun
	clusterConfig := &v1alpha5.ClusterConfig{
		Cluster:           *cluster,
		ClusterCreateOpts: *clusterCreateOpts,
	}

	// Use ClusterRun to create and start the cluster
	logging.Info("Creating k3d cluster", "cluster", normalizedName)
	if err := client.ClusterRun(ctx, k.runtime, clusterConfig); err != nil {
		return fmt.Errorf("failed to create k3d cluster: %w", err)
	}
	
	// Connect registry to cluster if enabled
	if registryNode != nil && k.config.EnableRegistry {
		logging.Info("Connecting registry to cluster", "cluster", normalizedName, "registry", registryNode.Name)
		// Get the created cluster
		createdCluster, err := client.ClusterGet(ctx, k.runtime, cluster)
		if err != nil {
			return fmt.Errorf("failed to get created cluster: %w", err)
		}
		
		// Connect the registry to the cluster
		if err := client.RegistryConnectClusters(ctx, k.runtime, registryNode, []*k3d.Cluster{createdCluster}); err != nil {
			return fmt.Errorf("failed to connect registry to cluster: %w", err)
		}
		
		// Configure k3s to use the registry with HTTP
		if err := k.configureRegistryForCluster(ctx, normalizedName); err != nil {
			logging.Warn("Failed to configure registry for cluster", "error", err)
			// Continue anyway as the registry might still work
		}
	}

	// Write kubeconfig to custom path if in container mode
	if k.config.ContainerMode {
		kubeconfigPath := k.GetKubeconfigPath(clusterName)
		if err := k.writeKubeconfig(ctx, cluster, kubeconfigPath); err != nil {
			return fmt.Errorf("failed to write kubeconfig: %w", err)
		}
	}

	logging.Info("Successfully created k3d cluster", "cluster", normalizedName)
	
	// Note: Traefik deployment is now handled by start_v2.go using the new architecture
	// The old TraefikManager is deprecated and should not be used
	
	return nil
}

// CreateClusterWithPortMapping creates a new k3d cluster with specified port mappings
func (k *K3dClusterManager) CreateClusterWithPortMapping(ctx context.Context, clusterName string, portMappings map[int32]int32) error {
	// Use optimized creation for test mode or when explicitly requested
	if config.GetBool("features.testMode") || config.GetBool("kubernetes.k3dOptimized") {
		return k.CreateClusterOptimized(ctx, clusterName)
	}
	
	// Use standard creation with port mappings
	return k.createClusterStandardWithPorts(ctx, clusterName, portMappings)
}

// createClusterStandardWithPorts creates a standard k3d cluster with custom port mappings
func (k *K3dClusterManager) createClusterStandardWithPorts(ctx context.Context, clusterName string, portMappings map[int32]int32) error {
	normalizedName := k.normalizeClusterName(clusterName)

	// Check if cluster already exists
	exists, err := k.ClusterExists(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to check if cluster exists: %w", err)
	}

	if exists {
		logging.Info("k3d cluster already exists", "cluster", normalizedName)
		return nil
	}
	
	// Handle registry for dev mode
	var registryNode *k3d.Node
	if k.config.EnableRegistry {
		registryNode, err = k.ensureRegistry(ctx)
		if err != nil {
			return fmt.Errorf("failed to ensure registry: %w", err)
		}
		logging.Info("Using k3d registry for dev mode", "registry", registryNode.Name)
	}

	// Determine k3s image
	k3sImage := "rancher/k3s:v1.31.4-k3s1"
	if k.config.K3dImage != "" {
		k3sImage = k.config.K3dImage
	}

	// K3s args for minimal setup - disable unnecessary components
	k3sArgs := []string{
		"--disable=traefik",        // Disable Traefik ingress controller
		"--disable=servicelb",      // Disable the default service load balancer
		"--disable=metrics-server", // Disable metrics server
		"--disable-network-policy", // Disable network policy controller
	}

	// Create server node
	serverNode := &k3d.Node{
		Name:    fmt.Sprintf("k3d-%s-server-0", normalizedName),
		Role:    k3d.ServerRole,
		Image:   k3sImage,
		Restart: true,
		Args:    k3sArgs,
		K3sNodeLabels: map[string]string{
			"kecs.io/cluster": normalizedName,
		},
		Env: []string{
			"K3S_KUBECONFIG_MODE=666", // Ensure kubeconfig is readable
		},
	}
	
	// Add volume mounts if specified
	if len(k.config.VolumeMounts) > 0 {
		volumes := []string{}
		for _, mount := range k.config.VolumeMounts {
			// k3d expects volume format as "hostPath:containerPath"
			volumes = append(volumes, fmt.Sprintf("%s:%s", mount.HostPath, mount.ContainerPath))
		}
		serverNode.Volumes = volumes
		logging.Info("Adding volume mounts", "volumes", volumes)
	}
	
	// Add port mappings
	if len(portMappings) > 0 {
		portMap := nat.PortMap{}
		for hostPort, nodePort := range portMappings {
			logging.Info("Adding port mapping",
				"hostPort", hostPort,
				"nodePort", nodePort)
			portKey := fmt.Sprintf("%d/tcp", nodePort)
			portMap[nat.Port(portKey)] = []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: fmt.Sprintf("%d", hostPort),
				},
			}
		}
		serverNode.Ports = portMap
		
		// Store the first port mapping as the main traefik port
		for hostPort := range portMappings {
			k.portMutex.Lock()
			k.traefikPorts[normalizedName] = int(hostPort)
			k.portMutex.Unlock()
			break
		}
	} else {
		// Fallback to default behavior if no port mappings provided
		k.portMutex.Lock()
		httpPort := k.config.TraefikPort
		if httpPort == 0 {
			port, err := k.findAvailablePort(8090)
			if err != nil {
				k.portMutex.Unlock()
				return fmt.Errorf("failed to find available port for HTTP access: %w", err)
			}
			httpPort = port
		}
		k.traefikPorts[normalizedName] = httpPort
		k.portMutex.Unlock()
		
		logging.Info("Adding default port mapping",
			"hostPort", httpPort,
			"nodePort", 30890)
		serverNode.Ports = nat.PortMap{
			"30890/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: fmt.Sprintf("%d", httpPort),
				},
			},
		}
	}

	// Create cluster configuration with minimal required fields
	// In container mode, check if KECS network is specified
	networkName := fmt.Sprintf("k3d-%s", normalizedName)
	if k.config.ContainerMode {
		if kecsNetwork := config.GetString("docker.network"); kecsNetwork != "" {
			logging.Info("Using KECS Docker network", "network", kecsNetwork)
			networkName = kecsNetwork
		}
	}
	
	cluster := &k3d.Cluster{
		Name:  normalizedName,
		Nodes: []*k3d.Node{serverNode},
		Network: k3d.ClusterNetwork{
			Name: networkName,
			IPAM: k3d.IPAM{
				Managed: false,
			},
		},
		Token: fmt.Sprintf("kecs-%s-token", normalizedName),
		KubeAPI: &k3d.ExposureOpts{
			Host: k3d.DefaultAPIHost,
		},
	}
	
	// Registry connection will be handled after cluster creation

	// Create cluster creation options
	clusterCreateOpts := &k3d.ClusterCreateOpts{
		WaitForServer:       true,
		Timeout:             2 * time.Minute,
		DisableLoadBalancer: false,
		GlobalLabels:        make(map[string]string),
		GlobalEnv:           []string{},
		NodeHooks:           []k3d.NodeHook{},
	}

	// Create cluster config for ClusterRun
	clusterConfig := &v1alpha5.ClusterConfig{
		Cluster:           *cluster,
		ClusterCreateOpts: *clusterCreateOpts,
	}

	// Use ClusterRun to create and start the cluster
	logging.Info("Creating k3d cluster", "cluster", normalizedName)
	if err := client.ClusterRun(ctx, k.runtime, clusterConfig); err != nil {
		return fmt.Errorf("failed to create k3d cluster: %w", err)
	}
	
	// Connect registry to cluster if enabled
	if registryNode != nil && k.config.EnableRegistry {
		logging.Info("Connecting registry to cluster", "cluster", normalizedName, "registry", registryNode.Name)
		// Get the created cluster
		createdCluster, err := client.ClusterGet(ctx, k.runtime, cluster)
		if err != nil {
			return fmt.Errorf("failed to get created cluster: %w", err)
		}
		
		// Connect the registry to the cluster
		if err := client.RegistryConnectClusters(ctx, k.runtime, registryNode, []*k3d.Cluster{createdCluster}); err != nil {
			return fmt.Errorf("failed to connect registry to cluster: %w", err)
		}
		
		// Configure k3s to use the registry with HTTP
		if err := k.configureRegistryForCluster(ctx, normalizedName); err != nil {
			logging.Warn("Failed to configure registry for cluster", "error", err)
			// Continue anyway as the registry might still work
		}
	}

	// Write kubeconfig to custom path if in container mode
	if k.config.ContainerMode {
		kubeconfigPath := k.GetKubeconfigPath(clusterName)
		if err := k.writeKubeconfig(ctx, cluster, kubeconfigPath); err != nil {
			return fmt.Errorf("failed to write kubeconfig: %w", err)
		}
	}

	logging.Info("Successfully created k3d cluster", "cluster", normalizedName)
	
	// Note: Traefik deployment is now handled by start_v2.go using the new architecture
	// The old TraefikManager is deprecated and should not be used
	
	return nil
}

// DeleteCluster deletes a k3d cluster
// StopCluster stops a k3d cluster without deleting it
func (k *K3dClusterManager) StopCluster(ctx context.Context, clusterName string) error {
	normalizedName := k.normalizeClusterName(clusterName)

	// Check if cluster exists
	exists, err := k.ClusterExists(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to check if cluster exists: %w", err)
	}

	if !exists {
		return fmt.Errorf("cluster does not exist: %s", normalizedName)
	}

	// Get the cluster
	clusters, err := client.ClusterList(ctx, k.runtime)
	if err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	var cluster *k3d.Cluster
	for _, c := range clusters {
		if c.Name == normalizedName {
			cluster = c
			break
		}
	}

	if cluster == nil {
		return fmt.Errorf("cluster not found: %s", normalizedName)
	}

	// Stop the cluster
	logging.Info("Stopping k3d cluster", "cluster", normalizedName)
	if err := client.ClusterStop(ctx, k.runtime, cluster); err != nil {
		return fmt.Errorf("failed to stop cluster: %w", err)
	}

	logging.Info("k3d cluster stopped successfully", "cluster", normalizedName)
	return nil
}

// StartCluster starts a previously stopped k3d cluster
func (k *K3dClusterManager) StartCluster(ctx context.Context, clusterName string) error {
	normalizedName := k.normalizeClusterName(clusterName)

	// Check if cluster exists
	exists, err := k.ClusterExists(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to check if cluster exists: %w", err)
	}

	if !exists {
		return fmt.Errorf("cluster does not exist: %s", normalizedName)
	}

	// Get the cluster
	clusters, err := client.ClusterList(ctx, k.runtime)
	if err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	var cluster *k3d.Cluster
	for _, c := range clusters {
		if c.Name == normalizedName {
			cluster = c
			break
		}
	}

	if cluster == nil {
		return fmt.Errorf("cluster not found: %s", normalizedName)
	}

	// Start the cluster
	logging.Info("Starting k3d cluster", "cluster", normalizedName)
	startOpts := k3d.ClusterStartOpts{
		WaitForServer: true,
		Timeout:       60 * time.Second,
	}
	
	if err := client.ClusterStart(ctx, k.runtime, cluster, startOpts); err != nil {
		// If normal start fails due to DNS fix issues, try workaround
		if strings.Contains(err.Error(), "Host Gateway IP is missing") || 
		   strings.Contains(err.Error(), "Cannot enable DNS fix") {
			logging.Warn("Normal start failed due to DNS fix issue, attempting workaround", "error", err)
			return k.startClusterWithWorkaround(ctx, normalizedName, cluster)
		}
		return fmt.Errorf("failed to start cluster: %w", err)
	}

	logging.Info("k3d cluster started successfully", "cluster", normalizedName)
	return nil
}

// startClusterWithWorkaround handles the DNS fix issue by recreating the cluster while preserving data
func (k *K3dClusterManager) startClusterWithWorkaround(ctx context.Context, normalizedName string, cluster *k3d.Cluster) error {
	logging.Info("Using workaround: recreating cluster while preserving data", "cluster", normalizedName)
	
	// Save the original cluster configuration
	volumeMounts := k.config.VolumeMounts
	
	// Delete the problematic cluster
	logging.Info("Deleting stopped cluster", "cluster", normalizedName)
	if err := client.ClusterDelete(ctx, k.runtime, cluster, k3d.ClusterDeleteOpts{SkipRegistryCheck: false}); err != nil {
		return fmt.Errorf("failed to delete cluster for workaround: %w", err)
	}
	
	// Recreate the cluster with the same configuration
	logging.Info("Recreating cluster with preserved data", "cluster", normalizedName)
	
	// Restore the volume mounts to preserve data
	k.config.VolumeMounts = volumeMounts
	
	// Use the denormalized name for recreation (without "kecs-" prefix)
	denormalizedName := strings.TrimPrefix(normalizedName, "kecs-")
	if err := k.CreateCluster(ctx, denormalizedName); err != nil {
		return fmt.Errorf("failed to recreate cluster: %w", err)
	}
	
	logging.Info("Cluster recreated successfully with preserved data", "cluster", normalizedName)
	return nil
}

func (k *K3dClusterManager) DeleteCluster(ctx context.Context, clusterName string) error {
	normalizedName := k.normalizeClusterName(clusterName)

	// Check if cluster exists
	exists, err := k.ClusterExists(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to check if cluster exists: %w", err)
	}

	if !exists {
		logging.Info("k3d cluster does not exist", "cluster", normalizedName)
		return nil
	}

	// Get cluster object
	cluster, err := client.ClusterGet(ctx, k.runtime, &k3d.Cluster{Name: normalizedName})
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Delete the cluster
	logging.Info("Deleting k3d cluster", "cluster", normalizedName)
	deleteOpts := k3d.ClusterDeleteOpts{
		SkipRegistryCheck: true,
	}

	if err := client.ClusterDelete(ctx, k.runtime, cluster, deleteOpts); err != nil {
		return fmt.Errorf("failed to delete k3d cluster: %w", err)
	}

	// Clean up kubeconfig files
	kubeconfigPath := k.GetKubeconfigPath(clusterName)
	if kubeconfigPath != "" {
		os.Remove(kubeconfigPath)
	}
	
	// Also remove host kubeconfig if in container mode
	if k.config.ContainerMode {
		hostKubeconfigPath := k.GetHostKubeconfigPath(clusterName)
		if hostKubeconfigPath != "" {
			os.Remove(hostKubeconfigPath)
		}
	}

	logging.Info("Successfully deleted k3d cluster", "cluster", normalizedName)
	return nil
}

// ClusterExists checks if a k3d cluster exists
func (k *K3dClusterManager) ClusterExists(ctx context.Context, clusterName string) (bool, error) {
	normalizedName := k.normalizeClusterName(clusterName)

	clusters, err := client.ClusterList(ctx, k.runtime)
	if err != nil {
		return false, fmt.Errorf("failed to list clusters: %w", err)
	}

	for _, cluster := range clusters {
		if cluster.Name == normalizedName {
			return true, nil
		}
	}

	return false, nil
}

// ListClusters returns a list of all k3d clusters
func (k *K3dClusterManager) ListClusters(ctx context.Context) ([]string, error) {
	clusters, err := client.ClusterList(ctx, k.runtime)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	var clusterNames []string
	for _, cluster := range clusters {
		// Only include KECS clusters (those with kecs- prefix)
		if strings.HasPrefix(cluster.Name, "kecs-") {
			// Return the instance name without the kecs- prefix
			instanceName := strings.TrimPrefix(cluster.Name, "kecs-")
			clusterNames = append(clusterNames, instanceName)
		}
	}

	return clusterNames, nil
}

// IsClusterRunning checks if a cluster is running by examining container states
func (k *K3dClusterManager) IsClusterRunning(ctx context.Context, clusterName string) (bool, error) {
	normalizedName := k.normalizeClusterName(clusterName)

	// Get cluster
	_, err := client.ClusterGet(ctx, k.runtime, &k3d.Cluster{Name: normalizedName})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, fmt.Errorf("failed to get cluster: %w", err)
	}

	// Check if any nodes are running
	nodes, err := k.runtime.GetNodesByLabel(ctx, map[string]string{k3d.LabelClusterName: normalizedName})
	if err != nil {
		return false, fmt.Errorf("failed to get nodes: %w", err)
	}

	// If we have nodes and cluster exists, check container states
	if len(nodes) > 0 {
		// Check if at least one node is running
		for _, node := range nodes {
			// Get node status from runtime
			nodeStatus, err := k.runtime.GetNode(ctx, node)
			if err != nil {
				continue
			}
			// If we find a running node, cluster is considered running
			if nodeStatus != nil && nodeStatus.State.Running {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetKubeClient returns a Kubernetes client for the specified cluster
func (k *K3dClusterManager) GetKubeClient(clusterName string) (kubernetes.Interface, error) {
	normalizedName := k.normalizeClusterName(clusterName)
	ctx := context.Background()

	// Get the k3d cluster to retrieve kubeconfig
	cluster, err := client.ClusterGet(ctx, k.runtime, &k3d.Cluster{Name: normalizedName})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	// Get all nodes including the loadbalancer
	nodes, err := k.runtime.GetNodesByLabel(ctx, map[string]string{k3d.LabelClusterName: normalizedName})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster nodes: %w", err)
	}

	// Find the loadbalancer node
	var loadbalancerNode *k3d.Node
	for i := range nodes {
		if nodes[i].Role == k3d.LoadBalancerRole {
			loadbalancerNode = nodes[i]
			cluster.ServerLoadBalancer = &k3d.Loadbalancer{
				Node: loadbalancerNode,
			}
			break
		}
	}

	// Get kubeconfig from k3d
	kubeconfigObj, err := client.KubeconfigGet(ctx, k.runtime, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Fix the server URL by getting the actual port from Docker inspect
	if loadbalancerNode != nil {
		// Get the actual port mapping from Docker
		apiPort, err := k.getLoadBalancerAPIPort(ctx, loadbalancerNode.Name)
		if err != nil {
			logging.Warn("Failed to get loadbalancer port", "error", err)
		} else if apiPort != "" {
			// Update the server URL with the correct port
			for clusterName, clusterConfig := range kubeconfigObj.Clusters {
				// When running in container mode, we need to connect to k3d containers directly
				host := "127.0.0.1"
				if k.config.ContainerMode {
					// In container mode, connect directly to the k3d server container
					// using its container name within the same Docker network
					k3dServerName := fmt.Sprintf("k3d-%s-server-0", normalizedName)
					host = k3dServerName
					logging.Debug("Container mode: using direct container connection", "server", k3dServerName)
				}
				
				// In container mode with direct connection, use the internal port 6443
				port := apiPort
				if k.config.ContainerMode {
					port = "6443" // k3d server internal port
				}
				newServer := fmt.Sprintf("https://%s:%s", host, port)
				logging.Debug("Updating server URL",
					"oldServer", clusterConfig.Server,
					"newServer", newServer)
				clusterConfig.Server = newServer
				kubeconfigObj.Clusters[clusterName] = clusterConfig
			}
		}
	}

	// In container mode, write kubeconfig to file for compatibility
	if k.config.ContainerMode {
		kubeconfigPath := k.GetKubeconfigPath(clusterName)
		logging.Debug("Writing kubeconfig", "path", kubeconfigPath)

		// Ensure directory exists
		kubeconfigDir := filepath.Dir(kubeconfigPath)
		if err := os.MkdirAll(kubeconfigDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create kubeconfig directory: %w", err)
		}

		// Write kubeconfig to file
		kubeconfigBytes, err := clientcmd.Write(*kubeconfigObj)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize kubeconfig: %w", err)
		}

		if err := os.WriteFile(kubeconfigPath, kubeconfigBytes, 0600); err != nil {
			return nil, fmt.Errorf("failed to write kubeconfig file: %w", err)
		}
	}

	// Convert the kubeconfig object to REST config
	config, err := clientcmd.NewDefaultClientConfig(
		*kubeconfigObj,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get client config: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return clientset, nil
}

// GetKubeConfig returns the REST config for the specified cluster
func (k *K3dClusterManager) GetKubeConfig(clusterName string) (*rest.Config, error) {
	return k.getRESTConfig(clusterName)
}

// WaitForClusterReady waits for a k3d cluster to be ready
func (k *K3dClusterManager) WaitForClusterReady(clusterName string, timeout time.Duration) error {
	startTime := time.Now()
	normalizedName := k.normalizeClusterName(clusterName)

	logging.Info("Waiting for k3d cluster to be ready", "cluster", normalizedName)

	for {
		if time.Since(startTime) > timeout {
			return fmt.Errorf("timeout waiting for cluster %s to be ready after %v", clusterName, timeout)
		}

		// Try to create a client and check connectivity
		client, err := k.GetKubeClient(clusterName)
		if err != nil {
			logging.Debug("Failed to create client for cluster, retrying",
				"cluster", clusterName,
				"error", err)
			time.Sleep(2 * time.Second)
			continue
		}

		// Try to list nodes to verify connectivity
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err = client.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
		cancel()

		if err != nil {
			logging.Debug("Failed to connect to cluster API, retrying",
				"cluster", clusterName,
				"error", err)
			time.Sleep(2 * time.Second)
			continue
		}

		logging.Info("k3d cluster is ready", "cluster", clusterName)
		return nil
	}
}

// GetKubeconfigPath returns the path to the kubeconfig file for the cluster
func (k *K3dClusterManager) GetKubeconfigPath(clusterName string) string {
	normalizedName := k.normalizeClusterName(clusterName)
	
	if k.config.ContainerMode {
		kubeconfigPath := k.config.KubeconfigPath
		if kubeconfigPath == "" {
			kubeconfigPath = config.GetString("kubernetes.kubeconfigPath")
			if kubeconfigPath == "" {
				// Use a temporary directory that's writable in container mode
				kubeconfigPath = filepath.Join(os.TempDir(), "kecs", "kubeconfig")
			}
		}
		os.MkdirAll(kubeconfigPath, 0755)
		return filepath.Join(kubeconfigPath, fmt.Sprintf("%s.config", normalizedName))
	}

	// For non-container mode (including new architecture), check multiple locations
	homeDir, _ := os.UserHomeDir()
	
	// Try ~/.config/kubeconfig-<cluster>.yaml (k3d v5 default)
	configPath := filepath.Join(homeDir, ".config", fmt.Sprintf("kubeconfig-%s.yaml", normalizedName))
	if _, err := os.Stat(configPath); err == nil {
		return configPath
	}
	
	// Try ~/.k3d/kubeconfig-<cluster>.yaml (older k3d versions)
	k3dConfigPath := filepath.Join(homeDir, ".k3d", fmt.Sprintf("kubeconfig-%s.yaml", normalizedName))
	if _, err := os.Stat(k3dConfigPath); err == nil {
		return k3dConfigPath
	}
	
	// Try default kubeconfig location
	defaultConfig := filepath.Join(homeDir, ".kube", "config")
	if _, err := os.Stat(defaultConfig); err == nil {
		return defaultConfig
	}
	
	// Return the expected path even if it doesn't exist yet
	return configPath
}

// GetHostKubeconfigPath returns the path to the host-compatible kubeconfig file
func (k *K3dClusterManager) GetHostKubeconfigPath(clusterName string) string {
	if k.config.ContainerMode {
		kubeconfigPath := k.config.KubeconfigPath
		if kubeconfigPath == "" {
			kubeconfigPath = config.GetString("kubernetes.kubeconfigPath")
			if kubeconfigPath == "" {
				// Use a temporary directory that's writable in container mode
				kubeconfigPath = filepath.Join(os.TempDir(), "kecs", "kubeconfig")
			}
		}
		os.MkdirAll(kubeconfigPath, 0755)
		normalizedName := k.normalizeClusterName(clusterName)
		return filepath.Join(kubeconfigPath, fmt.Sprintf("%s.host.config", normalizedName))
	}
	// For non-container mode, there's no separate host kubeconfig
	return k.GetKubeconfigPath(clusterName)
}

// GetClusterInfo returns information about the cluster
func (k *K3dClusterManager) GetClusterInfo(ctx context.Context, clusterName string) (*ClusterInfo, error) {
	normalizedName := k.normalizeClusterName(clusterName)

	exists, err := k.ClusterExists(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("cluster %s does not exist", clusterName)
	}

	// Get cluster details
	cluster, err := client.ClusterGet(ctx, k.runtime, &k3d.Cluster{Name: normalizedName})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster details: %w", err)
	}

	// Count nodes
	nodeCount := len(cluster.Nodes)

	// Try to get Kubernetes version
	version := "unknown"
	if kubeClient, err := k.GetKubeClient(clusterName); err == nil {
		if serverVersion, err := kubeClient.Discovery().ServerVersion(); err == nil {
			version = serverVersion.GitVersion
		}
	}

	return &ClusterInfo{
		Name:      clusterName,
		Status:    "Running", // k3d clusters are either running or don't exist
		Provider:  "k3d",
		NodeCount: nodeCount,
		Version:   version,
		Metadata: map[string]string{
			"k3d_cluster_name": normalizedName,
			"image":            "rancher/k3s:v1.31.4-k3s1",
		},
	}, nil
}

// normalizeClusterName ensures cluster name has the kecs- prefix for k3d
func (k *K3dClusterManager) normalizeClusterName(clusterName string) string {
	if !strings.HasPrefix(clusterName, "kecs-") {
		return fmt.Sprintf("kecs-%s", clusterName)
	}
	return clusterName
}

// writeKubeconfig writes the kubeconfig for the cluster to the specified path
func (k *K3dClusterManager) writeKubeconfig(ctx context.Context, cluster *k3d.Cluster, kubeconfigPath string) error {
	// Get kubeconfig from k3d
	kubecfg, err := client.KubeconfigGet(ctx, k.runtime, cluster)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// In container mode, also write a host-compatible kubeconfig
	if k.config.ContainerMode {
		// First write the internal kubeconfig
		if err := client.KubeconfigWrite(ctx, kubecfg, kubeconfigPath); err != nil {
			logging.Warn("Failed to write internal kubeconfig", "error", err)
			// Continue to try manual creation
		}
		
		// Create a host-compatible version
		hostKubeconfigPath := strings.TrimSuffix(kubeconfigPath, ".config") + ".host.config"
		if err := k.writeHostKubeconfig(ctx, cluster, kubecfg, hostKubeconfigPath); err != nil {
			logging.Warn("Failed to write host kubeconfig", "error", err)
		} else {
			logging.Info("Created host-compatible kubeconfig", "path", hostKubeconfigPath)
		}
	}

	// Write kubeconfig to file
	if err := client.KubeconfigWrite(ctx, kubecfg, kubeconfigPath); err != nil {
		// In container mode, k3d might fail to create symlinks
		// Try to find the actual kubeconfig file and create a symlink manually
		if k.config.ContainerMode {
			logging.Warn("k3d kubeconfig write failed, attempting to create symlink manually", "error", err)
			
			kubeconfigDir := filepath.Dir(kubeconfigPath)
			pattern := filepath.Join(kubeconfigDir, "*.config.k3d_*")
			matches, globErr := filepath.Glob(pattern)
			if globErr == nil && len(matches) > 0 {
				// Found k3d-generated kubeconfig file, create symlink
				actualFile := matches[0]
				logging.Debug("Found k3d kubeconfig, creating symlink",
					"actualFile", actualFile,
					"targetPath", kubeconfigPath)
				
				// Remove existing file/link if it exists
				os.Remove(kubeconfigPath)
				
				// Create symlink
				if linkErr := os.Symlink(filepath.Base(actualFile), kubeconfigPath); linkErr != nil {
					logging.Warn("Failed to create symlink, copying file instead", "error", linkErr)
					// If symlink fails, copy the file instead
					if copyErr := k.copyFile(actualFile, kubeconfigPath); copyErr != nil {
						return fmt.Errorf("failed to copy kubeconfig file: %w", copyErr)
					}
				}
				logging.Info("Successfully created kubeconfig", "path", kubeconfigPath)
				return nil
			}
		}
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	logging.Info("Wrote kubeconfig for cluster",
		"cluster", cluster.Name,
		"path", kubeconfigPath)
	return nil
}

// copyFile copies a file from src to dst
func (k *K3dClusterManager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}

// writeHostKubeconfig writes a host-compatible kubeconfig file
func (k *K3dClusterManager) writeHostKubeconfig(ctx context.Context, cluster *k3d.Cluster, kubecfg *clientcmdapi.Config, path string) error {
	// Create a copy of the kubeconfig
	hostKubeconfig := kubecfg.DeepCopy()
	
	// Get the loadbalancer node to find the exposed port
	var loadbalancerNode *k3d.Node
	nodes, err := client.NodeList(ctx, k.runtime)
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}
	
	for _, node := range nodes {
		if node.Role == k3d.LoadBalancerRole && strings.HasPrefix(node.Name, fmt.Sprintf("k3d-%s-", cluster.Name)) {
			loadbalancerNode = node
			break
		}
	}
	
	if loadbalancerNode != nil {
		// Get the actual port mapping from Docker
		apiPort, err := k.getLoadBalancerAPIPort(ctx, loadbalancerNode.Name)
		if err != nil {
			logging.Warn("Failed to get loadbalancer port", "error", err)
		} else if apiPort != "" {
			// Update the server URL to use localhost
			for clusterName, clusterConfig := range hostKubeconfig.Clusters {
				newServer := fmt.Sprintf("https://localhost:%s", apiPort)
				logging.Debug("Host kubeconfig: updating server URL",
					"oldServer", clusterConfig.Server,
					"newServer", newServer)
				clusterConfig.Server = newServer
				hostKubeconfig.Clusters[clusterName] = clusterConfig
			}
		}
	}
	
	// Write the host kubeconfig file
	return clientcmd.WriteToFile(*hostKubeconfig, path)
}

// getLoadBalancerAPIPort gets the host port for the API from the loadbalancer container
func (k *K3dClusterManager) getLoadBalancerAPIPort(ctx context.Context, containerName string) (string, error) {
	// Get Docker client
	dockerClient, err := docker.GetDockerClient()
	if err != nil {
		return "", fmt.Errorf("failed to get docker client: %w", err)
	}

	// Get the container inspect data
	containerJSON, err := dockerClient.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	// Look for port 6443/tcp mapping
	if containerJSON.NetworkSettings != nil && containerJSON.NetworkSettings.Ports != nil {
		if bindings, ok := containerJSON.NetworkSettings.Ports["6443/tcp"]; ok && len(bindings) > 0 {
			for _, binding := range bindings {
				if binding.HostPort != "" {
					return binding.HostPort, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no port mapping found for 6443/tcp")
}

// deployTraefik method has been removed - Traefik deployment is now handled by ResourceDeployer

// getRESTConfig returns the REST config for a cluster
func (k *K3dClusterManager) getRESTConfig(clusterName string) (*rest.Config, error) {
	normalizedName := k.normalizeClusterName(clusterName)
	ctx := context.Background()

	// Get cluster
	cluster, err := client.ClusterGet(ctx, k.runtime, &k3d.Cluster{Name: normalizedName})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	// Get all nodes including the loadbalancer
	nodes, err := k.runtime.GetNodesByLabel(ctx, map[string]string{k3d.LabelClusterName: normalizedName})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster nodes: %w", err)
	}

	// Find the loadbalancer node
	var loadbalancerNode *k3d.Node
	for i := range nodes {
		if nodes[i].Role == k3d.LoadBalancerRole {
			loadbalancerNode = nodes[i]
			cluster.ServerLoadBalancer = &k3d.Loadbalancer{
				Node: loadbalancerNode,
			}
			break
		}
	}

	// Get kubeconfig
	kubeconfigObj, err := client.KubeconfigGet(ctx, k.runtime, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Fix the server URL by getting the actual port from Docker inspect
	if loadbalancerNode != nil {
		// Get the actual port mapping from Docker
		apiPort, err := k.getLoadBalancerAPIPort(ctx, loadbalancerNode.Name)
		if err != nil {
			logging.Warn("Failed to get loadbalancer port", "error", err)
		} else if apiPort != "" {
			// Update the server URL with the correct port
			for clusterName, clusterConfig := range kubeconfigObj.Clusters {
				// When running in container mode, we need to connect to k3d containers directly
				host := "127.0.0.1"
				if k.config.ContainerMode {
					// In container mode, connect directly to the k3d server container
					// using its container name within the same Docker network
					k3dServerName := fmt.Sprintf("k3d-%s-server-0", normalizedName)
					host = k3dServerName
					logging.Debug("Container mode: using direct container connection", "server", k3dServerName)
				}
				
				// In container mode with direct connection, use the internal port 6443
				port := apiPort
				if k.config.ContainerMode {
					port = "6443" // k3d server internal port
				}
				newServer := fmt.Sprintf("https://%s:%s", host, port)
				logging.Debug("Updating server URL",
					"oldServer", clusterConfig.Server,
					"newServer", newServer)
				clusterConfig.Server = newServer
				kubeconfigObj.Clusters[clusterName] = clusterConfig
			}
		}
	}

	// Convert to REST config
	config, err := clientcmd.NewDefaultClientConfig(
		*kubeconfigObj,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get client config: %w", err)
	}


	return config, nil
}

// CreateClusterOptimized creates a new k3d cluster with optimizations for faster startup
func (k *K3dClusterManager) CreateClusterOptimized(ctx context.Context, clusterName string) error {
	normalizedName := k.normalizeClusterName(clusterName)

	// Check if cluster already exists
	exists, err := k.ClusterExists(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to check if cluster exists: %w", err)
	}

	if exists {
		logging.Info("k3d cluster already exists", "cluster", normalizedName)
		return nil
	}

	// K3s args for minimal setup - disable unnecessary components
	k3sArgs := []string{
		"--disable=traefik",        // Disable Traefik ingress controller
		"--disable=servicelb",      // Disable the default service load balancer
		"--disable=metrics-server", // Disable metrics server
		"--disable-network-policy", // Disable network policy controller
	}
	
	// Optionally disable CoreDNS based on configuration
	// Some tests might need DNS resolution
	if config.GetBool("kubernetes.disableCoreDNS") {
		k3sArgs = append(k3sArgs, "--disable=coredns")
	}

	// Determine k3s image
	k3sImage := "rancher/k3s:v1.31.4-k3s1"
	if k.config.K3dImage != "" {
		k3sImage = k.config.K3dImage
	}

	// Create server node with optimizations
	serverNode := &k3d.Node{
		Name:  fmt.Sprintf("k3d-%s-server-0", normalizedName),
		Role:  k3d.ServerRole,
		Image: k3sImage,
		Restart: false, // Don't restart automatically in test scenarios
		K3sNodeLabels: map[string]string{
			"kecs.io/cluster": normalizedName,
		},
		Args: k3sArgs,
		Env: []string{
			"K3S_KUBECONFIG_MODE=666", // Ensure kubeconfig is readable
		},
		Memory: "512M", // Limit memory usage for faster startup
	}
	
	// Add volume mounts if specified
	if len(k.config.VolumeMounts) > 0 {
		volumes := []string{}
		for _, mount := range k.config.VolumeMounts {
			// k3d expects volume format as "hostPath:containerPath"
			volumes = append(volumes, fmt.Sprintf("%s:%s", mount.HostPath, mount.ContainerPath))
		}
		serverNode.Volumes = volumes
		logging.Info("Adding volume mounts to optimized cluster", "volumes", volumes)
	}
	
	// Add port mapping for Traefik if enabled (even in optimized mode)
	if k.config.EnableTraefik {
		// Lock for thread-safe port allocation
		k.portMutex.Lock()
		
		// Determine Traefik port
		traefikPort := k.config.TraefikPort
		if traefikPort == 0 {
			// Find available port starting from 8090
			port, err := k.findAvailablePort(8090)
			if err != nil {
				k.portMutex.Unlock()
				return fmt.Errorf("failed to find available port for Traefik: %w", err)
			}
			traefikPort = port
		}
		
		// Store the port for this cluster
		k.traefikPorts[normalizedName] = traefikPort
		k.portMutex.Unlock()
		
		logging.Info("Adding port mapping for Traefik", "port", traefikPort)
		serverNode.Ports = nat.PortMap{
			"30890/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: fmt.Sprintf("%d", traefikPort),
				},
			},
		}
	}

	// Create minimal cluster configuration
	// In container mode, check if KECS network is specified
	networkName := fmt.Sprintf("k3d-%s", normalizedName)
	if k.config.ContainerMode {
		if kecsNetwork := config.GetString("docker.network"); kecsNetwork != "" {
			logging.Info("Using KECS Docker network", "network", kecsNetwork)
			networkName = kecsNetwork
		}
	}
	
	cluster := &k3d.Cluster{
		Name:  normalizedName,
		Nodes: []*k3d.Node{serverNode},
		Network: k3d.ClusterNetwork{
			Name: networkName,
			IPAM: k3d.IPAM{
				Managed: false, // Don't manage IPAM
			},
		},
		Token: fmt.Sprintf("kecs-%s-token", normalizedName),
		KubeAPI: &k3d.ExposureOpts{
			Host: k3d.DefaultAPIHost,
		},
	}

	// For single-node clusters, we'll disable load balancer in create opts

	// Determine if we should wait for server based on configuration
	waitForServer := true
	if config.GetBool("kubernetes.k3dAsync") {
		waitForServer = false
		logging.Info("Creating k3d cluster asynchronously", "async", true)
	}
	
	// Create cluster creation options with shorter timeout
	clusterCreateOpts := &k3d.ClusterCreateOpts{
		WaitForServer:       waitForServer,
		Timeout:             30 * time.Second, // Reduced from 2 minutes
		DisableLoadBalancer: len(cluster.Nodes) == 1, // Disable for single-node
		DisableImageVolume:  true,             // Don't create image volume
		GlobalLabels: map[string]string{
			"kecs.io/optimized": "true",
		},
		GlobalEnv: []string{},
		NodeHooks: []k3d.NodeHook{},
	}

	// Create cluster config for ClusterRun
	clusterConfig := &v1alpha5.ClusterConfig{
		Cluster:           *cluster,
		ClusterCreateOpts: *clusterCreateOpts,
	}

	// Use ClusterRun to create and start the cluster
	logging.Info("Creating optimized k3d cluster", "cluster", normalizedName)
	startTime := time.Now()
	
	if err := client.ClusterRun(ctx, k.runtime, clusterConfig); err != nil {
		return fmt.Errorf("failed to create k3d cluster: %w", err)
	}

	creationTime := time.Since(startTime)
	logging.Info("Successfully created k3d cluster",
		"cluster", normalizedName,
		"duration", creationTime)

	// Write kubeconfig to custom path if in container mode
	if k.config.ContainerMode {
		kubeconfigPath := k.GetKubeconfigPath(clusterName)
		if err := k.writeKubeconfig(ctx, cluster, kubeconfigPath); err != nil {
			return fmt.Errorf("failed to write kubeconfig: %w", err)
		}
	}

	// Quick readiness check with shorter timeout - only if we waited for server
	if waitForServer {
		readyCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()

		logging.Info("Waiting for optimized cluster to be ready", "cluster", normalizedName)
		if err := k.waitForClusterReadyOptimized(readyCtx, normalizedName); err != nil {
			logging.Warn("Cluster may not be fully ready", "error", err)
			// Don't fail here, let the caller handle readiness
		}
	} else {
		logging.Info("Cluster creation initiated asynchronously", "cluster", normalizedName)
	}
	
	// Traefik deployment is now handled by ResourceDeployer in the start command

	return nil
}

// waitForClusterReadyOptimized performs a quick readiness check for optimized clusters
func (k *K3dClusterManager) waitForClusterReadyOptimized(ctx context.Context, clusterName string) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for cluster readiness")
		case <-ticker.C:
			// Try to get cluster info to check if API is ready
			cluster, err := client.ClusterGet(ctx, k.runtime, &k3d.Cluster{Name: clusterName})
			if err != nil {
				continue
			}

			// Check if at least one node is present
			if len(cluster.Nodes) > 0 {
				// Try to create a kube client
				if _, err := k.GetKubeClient(clusterName); err == nil {
					return nil
				}
			}
		}
	}
}

// findAvailablePort finds an available port starting from the given port
func (k *K3dClusterManager) findAvailablePort(startPort int) (int, error) {
	// Try up to 100 ports
	for i := 0; i < 100; i++ {
		port := startPort + i
		
		// Check if port is already allocated to another cluster
		portInUse := false
		for _, allocatedPort := range k.traefikPorts {
			if allocatedPort == port {
				portInUse = true
				break
			}
		}
		
		// Skip if already allocated
		if portInUse {
			continue
		}
		
		// Check if port is available on the host
		if IsPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found starting from %d", startPort)
}

// GetTraefikPort returns the Traefik port for a given cluster
func (k *K3dClusterManager) GetTraefikPort(clusterName string) (int, bool) {
	k.portMutex.Lock()
	defer k.portMutex.Unlock()
	
	normalizedName := k.normalizeClusterName(clusterName)
	port, exists := k.traefikPorts[normalizedName]
	return port, exists
}

// ensureRegistry gets the k3d registry for dev mode (does not create)
func (k *K3dClusterManager) ensureRegistry(ctx context.Context) (*k3d.Node, error) {
	registryName := "k3d-kecs-registry.localhost"
	
	// Try to get the registry
	existingRegistry, err := client.RegistryGet(ctx, k.runtime, registryName)
	if err != nil || existingRegistry == nil {
		return nil, fmt.Errorf("k3d registry not found. Please run 'kecs registry start' first")
	}
	
	logging.Info("Found existing k3d registry", "name", registryName)
	
	// Get the registry node
	nodes, err := k.runtime.GetNodesByLabel(ctx, map[string]string{k3d.LabelRole: string(k3d.RegistryRole)})
	if err != nil {
		return nil, fmt.Errorf("failed to get registry nodes: %w", err)
	}
	
	for _, node := range nodes {
		// Check both formats: "k3d-<name>" and just "<name>"
		if node.Name == fmt.Sprintf("k3d-%s", registryName) || node.Name == registryName {
			// Check if registry is running
			if !node.State.Running {
				return nil, fmt.Errorf("k3d registry exists but is not running. Please run 'kecs registry start'")
			}
			
			logging.Info("Using existing running k3d registry", "name", registryName, "nodeName", node.Name)
			return node, nil
		}
	}
	
	return nil, fmt.Errorf("k3d registry node not found")
}

// configureRegistryForCluster configures k3s to use the registry with HTTP
func (k *K3dClusterManager) configureRegistryForCluster(ctx context.Context, clusterName string) error {
	logging.Info("Configuring k3s to use registry with HTTP", "cluster", clusterName)
	
	// Create registry configuration for k3s
	// Use k3d-kecs-registry as the hostname (without .localhost) for cluster-internal access
	registryConfig := `mirrors:
  "k3d-kecs-registry.localhost:5000":
    endpoint:
      - "http://k3d-kecs-registry:5000"
  "localhost:5000":
    endpoint:
      - "http://k3d-kecs-registry:5000"
  "k3d-kecs-registry:5000":
    endpoint:
      - "http://k3d-kecs-registry:5000"
`
	
	// Get the server node name
	serverNodeName := fmt.Sprintf("k3d-%s-server-0", clusterName)
	
	// Get the server node
	nodes, err := k.runtime.GetNodesByLabel(ctx, map[string]string{
		k3d.LabelClusterName: clusterName,
		k3d.LabelRole:        string(k3d.ServerRole),
	})
	if err != nil || len(nodes) == 0 {
		return fmt.Errorf("failed to find server node for cluster %s", clusterName)
	}
	
	serverNode := nodes[0]
	
	// Write registry config using runtime's WriteToNode method
	if err := k.runtime.WriteToNode(ctx, []byte(registryConfig), "/etc/rancher/k3s/registries.yaml", 0644, serverNode); err != nil {
		logging.Warn("Failed to write registry config via runtime, trying alternative method", "error", err)
		
		// Alternative: use docker exec directly
		cmd := fmt.Sprintf(`docker exec %s sh -c "mkdir -p /etc/rancher/k3s && echo '%s' > /etc/rancher/k3s/registries.yaml"`, serverNodeName, registryConfig)
		if output, err := exec.CommandContext(ctx, "sh", "-c", cmd).CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create registry config: %w, output: %s", err, string(output))
		}
	}
	
	logging.Info("Successfully configured k3s registry", "cluster", clusterName)
	
	// Restart the node to apply the configuration
	logging.Info("Restarting k3s to apply registry configuration", "cluster", clusterName)
	
	// Stop and start the node
	if err := k.runtime.StopNode(ctx, serverNode); err != nil {
		logging.Warn("Failed to stop node via runtime, trying docker restart", "error", err)
		
		// Alternative: use docker restart directly
		cmd := fmt.Sprintf("docker restart %s", serverNodeName)
		if output, err := exec.CommandContext(ctx, "sh", "-c", cmd).CombinedOutput(); err != nil {
			return fmt.Errorf("failed to restart k3s container: %w, output: %s", err, string(output))
		}
	} else {
		// Successfully stopped, now start it
		if err := k.runtime.StartNode(ctx, serverNode); err != nil {
			return fmt.Errorf("failed to start node after stop: %w", err)
		}
	}
	
	// Wait a bit for k3s to come back up
	time.Sleep(5 * time.Second)
	
	logging.Info("k3s restarted successfully", "cluster", clusterName)
	return nil
}
