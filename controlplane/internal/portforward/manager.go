package portforward

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/portforward"
)

// ForwardType represents the type of port forward
type ForwardType string

const (
	ForwardTypeService ForwardType = "service"
	ForwardTypeTask    ForwardType = "task"
)

// ForwardStatus represents the status of a port forward
type ForwardStatus string

const (
	StatusActive  ForwardStatus = "active"
	StatusStopped ForwardStatus = "stopped"
	StatusError   ForwardStatus = "error"
)

// Forward represents a port forward configuration
type Forward struct {
	ID         string        `json:"id"`
	Type       ForwardType   `json:"type"`
	Cluster    string        `json:"cluster"`
	TargetName string        `json:"targetName"`
	LocalPort  int           `json:"localPort"`
	TargetPort int           `json:"targetPort"`
	Status     ForwardStatus `json:"status"`
	CreatedAt  time.Time     `json:"createdAt"`
	UpdatedAt  time.Time     `json:"updatedAt"`
	Error      string        `json:"error,omitempty"`
}

// Manager manages port forwards for a KECS instance
type Manager struct {
	instanceName string
	k8sClient    *kubernetes.Client
	stateDir     string
	forwards     map[string]*Forward
	forwarders   map[string]*portForwarder
	mu           sync.RWMutex
}

// portForwarder holds the active port forwarding connection
type portForwarder struct {
	stopCh    chan struct{}
	readyCh   chan struct{}
	errCh     chan error
	forwarder *portforward.PortForwarder
	cmd       *exec.Cmd // For kubectl port-forward process
}

// NewManager creates a new port forward manager
func NewManager(instanceName string, k8sClient *kubernetes.Client) *Manager {
	homeDir, _ := os.UserHomeDir()
	stateDir := filepath.Join(homeDir, ".kecs", "instances", instanceName, "port-forwards")

	m := &Manager{
		instanceName: instanceName,
		k8sClient:    k8sClient,
		stateDir:     stateDir,
		forwards:     make(map[string]*Forward),
		forwarders:   make(map[string]*portForwarder),
	}

	// Ensure state directory exists
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		logging.Warn("Failed to create state directory", "path", stateDir, "error", err)
	}

	// Load existing state
	m.loadState()

	return m
}

// StartServiceForward starts port forwarding to a service
func (m *Manager) StartServiceForward(ctx context.Context, cluster, serviceName string, localPort, targetPort int) (string, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate forward ID
	forwardID := fmt.Sprintf("svc-%s-%s-%d", cluster, serviceName, time.Now().Unix())

	// Auto-assign local port if not specified
	if localPort == 0 {
		localPort = m.findAvailablePort()
	}

	// Default target port to 80 if not specified
	if targetPort == 0 {
		targetPort = 80
	}

	// Create forward configuration
	forward := &Forward{
		ID:         forwardID,
		Type:       ForwardTypeService,
		Cluster:    cluster,
		TargetName: serviceName,
		LocalPort:  localPort,
		TargetPort: targetPort,
		Status:     StatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Get the namespace for the cluster
	cfg := config.GetConfig()
	region := cfg.AWS.DefaultRegion
	if region == "" {
		region = "us-east-1" // Fallback to default
	}
	namespace := fmt.Sprintf("%s-%s", cluster, region)

	// Get the service to find NodePort
	service, err := m.k8sClient.Clientset.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return "", 0, fmt.Errorf("failed to get service: %w", err)
	}

	// Check if service has NodePort
	var nodePort int32
	if service.Spec.Type == corev1.ServiceTypeNodePort && len(service.Spec.Ports) > 0 {
		// Find the NodePort that matches our target port
		for _, port := range service.Spec.Ports {
			if port.Port == int32(targetPort) || targetPort == 0 {
				nodePort = port.NodePort
				if targetPort == 0 {
					targetPort = int(port.Port)
				}
				break
			}
		}

		if nodePort == 0 && len(service.Spec.Ports) > 0 {
			// If no matching port found, use the first one
			nodePort = service.Spec.Ports[0].NodePort
			targetPort = int(service.Spec.Ports[0].Port)
		}
	}

	if nodePort == 0 {
		return "", 0, fmt.Errorf("service %s does not have NodePort configured. Ensure assignPublicIp is enabled", serviceName)
	}

	// Map the port using k3d
	if err := m.mapPortWithK3d(ctx, localPort, int(nodePort)); err != nil {
		return "", 0, fmt.Errorf("failed to map port with k3d: %w", err)
	}

	// Start kubectl port-forward in background
	forwarder, err := m.startKubectlPortForward(ctx, namespace, fmt.Sprintf("svc/%s", serviceName), localPort, targetPort)
	if err != nil {
		// Rollback k3d port mapping on failure
		m.unmapPortWithK3d(ctx, localPort)
		return "", 0, fmt.Errorf("failed to start port forwarding: %w", err)
	}

	// Track the forwarder
	m.forwarders[forwardID] = forwarder

	logging.Info("Port forward active",
		"id", forwardID,
		"service", serviceName,
		"localPort", localPort,
		"nodePort", nodePort)

	// Save forward configuration
	m.forwards[forwardID] = forward
	m.saveState()

	return forwardID, localPort, nil
}

// StartTaskForward starts port forwarding to a task
func (m *Manager) StartTaskForward(ctx context.Context, cluster, taskID string, localPort, targetPort int) (string, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate forward ID - use full task ID to avoid collisions
	forwardID := fmt.Sprintf("task-%s-%s-%d", cluster, taskID, time.Now().Unix())

	// Auto-assign local port if not specified
	if localPort == 0 {
		localPort = m.findAvailablePort()
	}

	// Default target port to 80 if not specified
	if targetPort == 0 {
		targetPort = 80
	}

	// Create forward configuration
	forward := &Forward{
		ID:         forwardID,
		Type:       ForwardTypeTask,
		Cluster:    cluster,
		TargetName: taskID,
		LocalPort:  localPort,
		TargetPort: targetPort,
		Status:     StatusActive,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Get the namespace for the cluster
	cfg := config.GetConfig()
	region := cfg.AWS.DefaultRegion
	if region == "" {
		region = "us-east-1" // Fallback to default
	}
	namespace := fmt.Sprintf("%s-%s", cluster, region)

	// Find the pod for this task
	pods, err := m.k8sClient.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("kecs.dev/task-id=%s", taskID),
	})
	if err != nil {
		return "", 0, fmt.Errorf("failed to find pod for task: %w", err)
	}

	if len(pods.Items) == 0 {
		return "", 0, fmt.Errorf("no pod found for task %s", taskID)
	}

	pod := &pods.Items[0]

	// Check if the pod's service has NodePort
	// Start kubectl port-forward to the pod
	forwarder, err := m.startKubectlPortForward(ctx, namespace, fmt.Sprintf("pod/%s", pod.Name), localPort, targetPort)
	if err != nil {
		return "", 0, fmt.Errorf("failed to start port forwarding: %w", err)
	}

	// Track the forwarder
	m.forwarders[forwardID] = forwarder

	logging.Info("Port forward active for task",
		"id", forwardID,
		"task", taskID,
		"pod", pod.Name,
		"localPort", localPort,
		"targetPort", targetPort)

	// Save forward configuration
	m.forwards[forwardID] = forward
	m.saveState()

	return forwardID, localPort, nil
}

// ListForwards lists all port forwards
func (m *Manager) ListForwards() ([]*Forward, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var forwards []*Forward
	for _, fwd := range m.forwards {
		forwards = append(forwards, fwd)
	}

	return forwards, nil
}

// StopForward stops a specific port forward
func (m *Manager) StopForward(forwardID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	forward, exists := m.forwards[forwardID]
	if !exists {
		return fmt.Errorf("forward %s not found", forwardID)
	}

	// Stop the forwarder if it exists
	if forwarder, ok := m.forwarders[forwardID]; ok {
		if forwarder.cmd != nil {
			// Kill the kubectl process
			if err := forwarder.cmd.Process.Kill(); err != nil {
				logging.Warn("Failed to kill port-forward process", "error", err)
			}
		}
		if forwarder.stopCh != nil {
			close(forwarder.stopCh)
		}
		delete(m.forwarders, forwardID)
	}

	// If this was a service forward with k3d mapping, unmap it
	if forward.Type == ForwardTypeService {
		if err := m.unmapPortWithK3d(context.Background(), forward.LocalPort); err != nil {
			logging.Warn("Failed to unmap port with k3d", "port", forward.LocalPort, "error", err)
		}
	}

	// Update status
	forward.Status = StatusStopped
	forward.UpdatedAt = time.Now()

	// Remove from active forwards
	delete(m.forwards, forwardID)

	m.saveState()

	logging.Info("Port forward stopped", "id", forwardID)

	return nil
}

// StopAllForwards stops all port forwards
func (m *Manager) StopAllForwards() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for forwardID, forwarder := range m.forwarders {
		if forwarder.cmd != nil {
			// Kill the kubectl process
			if err := forwarder.cmd.Process.Kill(); err != nil {
				logging.Warn("Failed to kill port-forward process", "error", err)
			}
		}
		if forwarder.stopCh != nil {
			close(forwarder.stopCh)
		}
		delete(m.forwarders, forwardID)
	}

	// Unmap all k3d ports for service forwards
	for _, forward := range m.forwards {
		if forward.Type == ForwardTypeService {
			if err := m.unmapPortWithK3d(context.Background(), forward.LocalPort); err != nil {
				logging.Warn("Failed to unmap port with k3d", "port", forward.LocalPort, "error", err)
			}
		}
	}

	// Clear all forwards
	m.forwards = make(map[string]*Forward)

	m.saveState()

	logging.Info("All port forwards stopped")

	return nil
}

// findAvailablePort finds an available local port
func (m *Manager) findAvailablePort() int {
	// Start from a random port in the dynamic range
	basePort := 30000 + rand.Intn(10000)

	for i := 0; i < 100; i++ {
		port := basePort + i

		// Check if port is already in use by our forwards
		inUse := false
		for _, fwd := range m.forwards {
			if fwd.LocalPort == port {
				inUse = true
				break
			}
		}
		if inUse {
			continue
		}

		// Check if port is available on the system
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			listener.Close()
			return port
		}
	}

	// Fallback to any available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port
}

// loadState loads the saved state from disk
func (m *Manager) loadState() error {
	stateFile := filepath.Join(m.stateDir, "state.json")

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No state file yet
		}
		return err
	}

	var forwards []*Forward
	if err := json.Unmarshal(data, &forwards); err != nil {
		return err
	}

	for _, fwd := range forwards {
		// Only restore active forwards
		if fwd.Status == StatusActive {
			m.forwards[fwd.ID] = fwd
		}
	}

	return nil
}

// saveState saves the current state to disk
func (m *Manager) saveState() error {
	var forwards []*Forward
	for _, fwd := range m.forwards {
		forwards = append(forwards, fwd)
	}

	data, err := json.MarshalIndent(forwards, "", "  ")
	if err != nil {
		return err
	}

	stateFile := filepath.Join(m.stateDir, "state.json")
	return os.WriteFile(stateFile, data, 0644)
}

// mapPortWithK3d uses k3d to map a local port to a NodePort
func (m *Manager) mapPortWithK3d(ctx context.Context, localPort int, nodePort int) error {
	// Get the cluster name
	clusterName := fmt.Sprintf("kecs-%s", m.instanceName)

	// For services, we actually need to map to NodePort
	// k3d exposes NodePorts through the serverlb automatically
	// But we need to use k3d node edit to add the port mapping

	// Using k3d CLI directly as the Go API is complex
	cmd := exec.Command("k3d", "node", "edit",
		fmt.Sprintf("k3d-%s-serverlb", clusterName),
		"--port-add", fmt.Sprintf("%d:%d@loadbalancer", localPort, nodePort))

	output, err := cmd.CombinedOutput()
	if err != nil {
		// If the port mapping already exists, k3d will return an error but it's OK
		if string(output) != "" && err != nil {
			logging.Warn("k3d port mapping may already exist",
				"localPort", localPort,
				"nodePort", nodePort,
				"output", string(output))
			// Continue anyway as the mapping might be working
		} else {
			return fmt.Errorf("failed to add port mapping: %w, output: %s", err, string(output))
		}
	}

	logging.Info("Added k3d port mapping",
		"localPort", localPort,
		"nodePort", nodePort,
		"cluster", clusterName)

	// Wait for serverlb to restart (typically takes ~10 seconds)
	time.Sleep(10 * time.Second)

	return nil
}

// unmapPortWithK3d removes a port mapping from k3d
func (m *Manager) unmapPortWithK3d(ctx context.Context, localPort int) error {
	// Note: k3d doesn't have a direct API to remove port mappings
	// We would need to recreate the node or cluster
	// For now, we'll just log that the port should be unmapped
	logging.Info("Port unmap requested (manual cleanup may be needed)",
		"localPort", localPort,
		"instance", m.instanceName)

	// In production, you might want to track all port mappings
	// and recreate the serverlb with updated mappings
	return nil
}

// startKubectlPortForward starts a kubectl port-forward process
func (m *Manager) startKubectlPortForward(ctx context.Context, namespace, target string, localPort, targetPort int) (*portForwarder, error) {
	// Get kubeconfig path
	kubeconfigPath := fmt.Sprintf("/tmp/kecs-%s.config", m.instanceName)

	// Check if kubeconfig exists
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("kubeconfig not found at %s. Run 'kecs kubeconfig get %s > %s' first",
			kubeconfigPath, m.instanceName, kubeconfigPath)
	}

	// Create the kubectl port-forward command
	cmd := exec.Command("kubectl",
		"--kubeconfig", kubeconfigPath,
		"port-forward",
		"-n", namespace,
		target,
		fmt.Sprintf("%d:%d", localPort, targetPort))

	// Create forwarder
	forwarder := &portForwarder{
		stopCh:  make(chan struct{}),
		readyCh: make(chan struct{}),
		errCh:   make(chan error, 1),
		cmd:     cmd,
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start kubectl port-forward: %w", err)
	}

	// Monitor the process in a goroutine
	go func() {
		err := cmd.Wait()
		if err != nil {
			select {
			case forwarder.errCh <- err:
			default:
			}
		}
	}()

	// Give it a moment to establish connection
	time.Sleep(2 * time.Second)

	// Check if process is still running
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return nil, fmt.Errorf("kubectl port-forward exited immediately")
	}

	// Signal ready
	close(forwarder.readyCh)

	logging.Info("Started kubectl port-forward",
		"namespace", namespace,
		"target", target,
		"localPort", localPort,
		"targetPort", targetPort)

	return forwarder, nil
}
