package kubernetes

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// K3dPortManager manages port mappings for k3d clusters
type K3dPortManager struct {
	clusterName string
	portManager *PortManager
}

// NewK3dPortManager creates a new K3d port manager
func NewK3dPortManager(clusterName string) *K3dPortManager {
	// Default port range for dynamic allocation (32000-32999)
	portManager := NewPortManager(32000, 32999)

	return &K3dPortManager{
		clusterName: clusterName,
		portManager: portManager,
	}
}

// AddPortMapping adds a port mapping to the k3d cluster's load balancer
func (k *K3dPortManager) AddPortMapping(ctx context.Context, hostPort, nodePort int32) error {
	// Format: k3d node edit k3d-<cluster>-serverlb --port-add <hostPort>:<nodePort>
	nodeName := fmt.Sprintf("k3d-%s-serverlb", k.clusterName)
	portMapping := fmt.Sprintf("%d:%d", hostPort, nodePort)

	logging.Info("Port mapping configured (k3d command skipped - running in container)",
		"node", nodeName,
		"mapping", portMapping,
		"note", "Ensure k3d cluster was started with port range 32000-32999:30000-30999")

	// Skip actual k3d command execution since we're running inside the container
	// The k3d cluster should be pre-configured with the necessary port range
	// This requires the cluster to be started with:
	// k3d cluster create --port "32000-32999:30000-30999@server:0"

	// We still track the port allocation but don't execute the k3d command
	logging.Info("Port allocation tracked successfully",
		"hostPort", hostPort,
		"nodePort", nodePort)

	return nil
}

// AllocateAndMapPort allocates a host port and maps it to a container port
func (k *K3dPortManager) AllocateAndMapPort(ctx context.Context, taskARN string, containerPort int32, protocol string) (int32, error) {
	// Allocate a host port
	hostPort, err := k.portManager.AllocatePort(taskARN, containerPort, protocol)
	if err != nil {
		return 0, fmt.Errorf("failed to allocate port: %w", err)
	}

	// For k3d, we map the host port to a NodePort
	// NodePort range in Kubernetes is typically 30000-32767
	// We'll use a simple mapping: hostPort -> 30000 + (hostPort - 32000)
	// This maps our range 32000-32999 to NodePort range 30000-30999
	nodePort := 30000 + (hostPort - 32000)

	// Add the port mapping to k3d
	if err := k.AddPortMapping(ctx, hostPort, nodePort); err != nil {
		// If mapping fails, release the allocated port
		k.portManager.ReleasePort(hostPort)
		return 0, fmt.Errorf("failed to add k3d port mapping: %w", err)
	}

	return hostPort, nil
}

// ReleaseTaskPorts releases all ports allocated to a task
// Note: k3d doesn't support removing individual port mappings after cluster creation,
// so we only release the port allocation tracking
func (k *K3dPortManager) ReleaseTaskPorts(ctx context.Context, taskARN string) error {
	// Get the allocated ports before releasing
	ports := k.portManager.GetTaskPorts(taskARN)

	// Release the port allocations
	if err := k.portManager.ReleaseTaskPorts(taskARN); err != nil {
		return fmt.Errorf("failed to release task ports: %w", err)
	}

	// Note: We cannot remove port mappings from a running k3d cluster
	// The ports will remain mapped but unused until the cluster is recreated
	if len(ports) > 0 {
		logging.Warn("Released port allocations for task, but k3d port mappings remain",
			"taskARN", taskARN,
			"ports", ports,
			"note", "Port mappings will be cleaned up on cluster recreation")
	}

	return nil
}

// GetNodePortForHost returns the NodePort mapped to a host port
func (k *K3dPortManager) GetNodePortForHost(hostPort int32) int32 {
	// Using our mapping scheme
	return 30000 + (hostPort - 32000)
}

// GetPortManager returns the underlying port manager
func (k *K3dPortManager) GetPortManager() *PortManager {
	return k.portManager
}

// IsPortAvailable checks if a port is available for allocation
func (k *K3dPortManager) IsPortAvailable(hostPort int32) bool {
	_, allocated := k.portManager.GetAllocation(hostPort)
	return !allocated
}

// GetClusterPortMappings retrieves current port mappings from k3d cluster
// This is useful for syncing state after a restart
func (k *K3dPortManager) GetClusterPortMappings(ctx context.Context) (map[int32]int32, error) {
	// Get load balancer node info
	nodeName := fmt.Sprintf("k3d-%s-serverlb", k.clusterName)

	cmd := exec.CommandContext(ctx, "docker", "inspect", nodeName, "--format",
		"{{range $p, $conf := .NetworkSettings.Ports}}{{$p}} -> {{(index $conf 0).HostPort}}{{println}}{{end}}")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get port mappings: %w", err)
	}

	// Parse the output to extract port mappings
	mappings := make(map[int32]int32)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse format: "30080/tcp -> 32000"
		var containerPort, hostPort int32
		var proto string
		_, err := fmt.Sscanf(line, "%d/%s -> %d", &containerPort, &proto, &hostPort)
		if err == nil {
			mappings[hostPort] = containerPort
		}
	}

	return mappings, nil
}
