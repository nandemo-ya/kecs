package kubernetes

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
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
)

// K3dClusterManager implements ClusterManager interface using k3d
type K3dClusterManager struct {
	runtime         runtimes.Runtime
	config          *ClusterManagerConfig
	traefikManager  *TraefikManager
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

	log.Printf("Creating K3dClusterManager with config: ContainerMode=%v, EnableTraefik=%v", 
		cfg.ContainerMode, cfg.EnableTraefik)

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
		log.Printf("k3d cluster %s already exists", normalizedName)
		return nil
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
		log.Printf("Adding volume mounts: %v", volumes)
	}
	
	// Add port mapping for Traefik if enabled
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
		
		log.Printf("Adding port mapping for Traefik: %d/tcp", traefikPort)
		serverNode.Ports = nat.PortMap{
			"30890/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: fmt.Sprintf("%d", traefikPort),
				},
			},
		}
	}

	// Create cluster configuration with minimal required fields
	// In container mode, check if KECS network is specified
	networkName := fmt.Sprintf("k3d-%s", normalizedName)
	if k.config.ContainerMode {
		if kecsNetwork := config.GetString("docker.network"); kecsNetwork != "" {
			log.Printf("Using KECS Docker network: %s", kecsNetwork)
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
	log.Printf("Creating k3d cluster %s...", normalizedName)
	if err := client.ClusterRun(ctx, k.runtime, clusterConfig); err != nil {
		return fmt.Errorf("failed to create k3d cluster: %w", err)
	}

	// Write kubeconfig to custom path if in container mode
	if k.config.ContainerMode {
		kubeconfigPath := k.GetKubeconfigPath(clusterName)
		if err := k.writeKubeconfig(ctx, cluster, kubeconfigPath); err != nil {
			return fmt.Errorf("failed to write kubeconfig: %w", err)
		}
	}

	log.Printf("Successfully created k3d cluster %s", normalizedName)
	
	// Deploy Traefik if enabled
	log.Printf("Traefik enabled: %v", k.config.EnableTraefik)
	if k.config.EnableTraefik {
		log.Printf("Deploying Traefik to cluster %s...", normalizedName)
		if err := k.deployTraefik(ctx, clusterName); err != nil {
			log.Printf("Warning: Failed to deploy Traefik: %v", err)
			// Continue without Traefik
		} else {
			log.Printf("Successfully deployed Traefik to cluster %s", normalizedName)
		}
	} else {
		log.Printf("Traefik is disabled, skipping deployment")
	}
	
	return nil
}

// DeleteCluster deletes a k3d cluster
func (k *K3dClusterManager) DeleteCluster(ctx context.Context, clusterName string) error {
	normalizedName := k.normalizeClusterName(clusterName)

	// Check if cluster exists
	exists, err := k.ClusterExists(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to check if cluster exists: %w", err)
	}

	if !exists {
		log.Printf("k3d cluster %s does not exist", normalizedName)
		return nil
	}

	// Get cluster object
	cluster, err := client.ClusterGet(ctx, k.runtime, &k3d.Cluster{Name: normalizedName})
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Delete the cluster
	log.Printf("Deleting k3d cluster %s...", normalizedName)
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

	log.Printf("Successfully deleted k3d cluster %s", normalizedName)
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
			clusterNames = append(clusterNames, cluster.Name)
		}
	}

	return clusterNames, nil
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
			log.Printf("Failed to get loadbalancer port: %v", err)
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
					log.Printf("Container mode: using direct container connection to %s", k3dServerName)
				}
				
				// In container mode with direct connection, use the internal port 6443
				port := apiPort
				if k.config.ContainerMode {
					port = "6443" // k3d server internal port
				}
				newServer := fmt.Sprintf("https://%s:%s", host, port)
				log.Printf("Updating server URL from %s to %s", clusterConfig.Server, newServer)
				clusterConfig.Server = newServer
				kubeconfigObj.Clusters[clusterName] = clusterConfig
			}
		}
	}

	// In container mode, write kubeconfig to file for compatibility
	if k.config.ContainerMode {
		kubeconfigPath := k.GetKubeconfigPath(clusterName)
		log.Printf("Writing kubeconfig to %s", kubeconfigPath)

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

	log.Printf("Waiting for k3d cluster %s to be ready", normalizedName)

	for {
		if time.Since(startTime) > timeout {
			return fmt.Errorf("timeout waiting for cluster %s to be ready after %v", clusterName, timeout)
		}

		// Try to create a client and check connectivity
		client, err := k.GetKubeClient(clusterName)
		if err != nil {
			log.Printf("Failed to create client for cluster %s: %v, retrying...", clusterName, err)
			time.Sleep(2 * time.Second)
			continue
		}

		// Try to list nodes to verify connectivity
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err = client.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
		cancel()

		if err != nil {
			log.Printf("Failed to connect to cluster %s API: %v, retrying...", clusterName, err)
			time.Sleep(2 * time.Second)
			continue
		}

		log.Printf("k3d cluster %s is ready", clusterName)
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
			"internal_name": normalizedName,
			"image":         "rancher/k3s:v1.31.4-k3s1",
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
			log.Printf("Failed to write internal kubeconfig: %v", err)
			// Continue to try manual creation
		}
		
		// Create a host-compatible version
		hostKubeconfigPath := strings.TrimSuffix(kubeconfigPath, ".config") + ".host.config"
		if err := k.writeHostKubeconfig(ctx, cluster, kubecfg, hostKubeconfigPath); err != nil {
			log.Printf("Failed to write host kubeconfig: %v", err)
		} else {
			log.Printf("Created host-compatible kubeconfig at %s", hostKubeconfigPath)
		}
	}

	// Write kubeconfig to file
	if err := client.KubeconfigWrite(ctx, kubecfg, kubeconfigPath); err != nil {
		// In container mode, k3d might fail to create symlinks
		// Try to find the actual kubeconfig file and create a symlink manually
		if k.config.ContainerMode {
			log.Printf("k3d kubeconfig write failed, attempting to create symlink manually: %v", err)
			
			kubeconfigDir := filepath.Dir(kubeconfigPath)
			pattern := filepath.Join(kubeconfigDir, "*.config.k3d_*")
			matches, globErr := filepath.Glob(pattern)
			if globErr == nil && len(matches) > 0 {
				// Found k3d-generated kubeconfig file, create symlink
				actualFile := matches[0]
				log.Printf("Found k3d kubeconfig at %s, creating symlink to %s", actualFile, kubeconfigPath)
				
				// Remove existing file/link if it exists
				os.Remove(kubeconfigPath)
				
				// Create symlink
				if linkErr := os.Symlink(filepath.Base(actualFile), kubeconfigPath); linkErr != nil {
					log.Printf("Failed to create symlink, copying file instead: %v", linkErr)
					// If symlink fails, copy the file instead
					if copyErr := k.copyFile(actualFile, kubeconfigPath); copyErr != nil {
						return fmt.Errorf("failed to copy kubeconfig file: %w", copyErr)
					}
				}
				log.Printf("Successfully created kubeconfig at %s", kubeconfigPath)
				return nil
			}
		}
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	log.Printf("Wrote kubeconfig for cluster %s to %s", cluster.Name, kubeconfigPath)
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
			log.Printf("Failed to get loadbalancer port: %v", err)
		} else if apiPort != "" {
			// Update the server URL to use localhost
			for clusterName, clusterConfig := range hostKubeconfig.Clusters {
				newServer := fmt.Sprintf("https://localhost:%s", apiPort)
				log.Printf("Host kubeconfig: updating server URL from %s to %s", clusterConfig.Server, newServer)
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

// deployTraefik deploys Traefik reverse proxy to the cluster
func (k *K3dClusterManager) deployTraefik(ctx context.Context, clusterName string) error {
	// Get Kubernetes client for the cluster
	kubeClient, err := k.GetKubeClient(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	// Get REST config
	restConfig, err := k.getRESTConfig(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get REST config: %w", err)
	}

	// Create Traefik manager
	traefikManager, err := NewTraefikManager(kubeClient, restConfig)
	if err != nil {
		return fmt.Errorf("failed to create Traefik manager: %w", err)
	}

	// Deploy Traefik
	if err := traefikManager.Deploy(ctx); err != nil {
		return fmt.Errorf("failed to deploy Traefik: %w", err)
	}

	// Store Traefik manager for later use
	k.traefikManager = traefikManager

	return nil
}

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
			log.Printf("Failed to get loadbalancer port: %v", err)
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
					log.Printf("Container mode: using direct container connection to %s", k3dServerName)
				}
				
				// In container mode with direct connection, use the internal port 6443
				port := apiPort
				if k.config.ContainerMode {
					port = "6443" // k3d server internal port
				}
				newServer := fmt.Sprintf("https://%s:%s", host, port)
				log.Printf("Updating server URL from %s to %s", clusterConfig.Server, newServer)
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
		log.Printf("k3d cluster %s already exists", normalizedName)
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
		log.Printf("Adding volume mounts to optimized cluster: %v", volumes)
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
		
		log.Printf("Adding port mapping for Traefik: %d/tcp", traefikPort)
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
			log.Printf("Using KECS Docker network: %s", kecsNetwork)
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
		log.Printf("Creating k3d cluster asynchronously (KECS_K3D_ASYNC=true)")
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
	log.Printf("Creating optimized k3d cluster %s...", normalizedName)
	startTime := time.Now()
	
	if err := client.ClusterRun(ctx, k.runtime, clusterConfig); err != nil {
		return fmt.Errorf("failed to create k3d cluster: %w", err)
	}

	creationTime := time.Since(startTime)
	log.Printf("Successfully created k3d cluster %s in %v", normalizedName, creationTime)

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

		log.Printf("Waiting for optimized cluster %s to be ready...", normalizedName)
		if err := k.waitForClusterReadyOptimized(readyCtx, normalizedName); err != nil {
			log.Printf("Warning: cluster may not be fully ready: %v", err)
			// Don't fail here, let the caller handle readiness
		}
	} else {
		log.Printf("Cluster %s creation initiated asynchronously", normalizedName)
	}
	
	// Deploy Traefik if enabled (even in optimized mode)
	if k.config.EnableTraefik && waitForServer {
		log.Printf("Deploying Traefik to optimized cluster %s...", normalizedName)
		if err := k.deployTraefik(ctx, clusterName); err != nil {
			log.Printf("Warning: Failed to deploy Traefik: %v", err)
			// Continue without Traefik
		} else {
			log.Printf("Successfully deployed Traefik to optimized cluster %s", normalizedName)
		}
	}

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
