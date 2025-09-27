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
	"strings"
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
	ID              string        `json:"id"`
	Type            ForwardType   `json:"type"`
	Cluster         string        `json:"cluster"`
	TargetName      string        `json:"targetName"`
	LocalPort       int           `json:"localPort"`
	TargetPort      int           `json:"targetPort"`
	Status          ForwardStatus `json:"status"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
	Error           string        `json:"error,omitempty"`
	AutoReconnect   bool          `json:"autoReconnect"`
	RetryCount      int           `json:"retryCount"`
	LastHealthCheck time.Time     `json:"lastHealthCheck,omitempty"`
}

// Manager manages port forwards for a KECS instance
type Manager struct {
	instanceName string
	k8sClient    *kubernetes.Client
	stateDir     string
	forwards     map[string]*Forward
	forwarders   map[string]*portForwarder
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// portForwarder holds the active port forwarding connection
type portForwarder struct {
	stopCh     chan struct{}
	readyCh    chan struct{}
	errCh      chan error
	forwarder  *portforward.PortForwarder
	cmd        *exec.Cmd    // For kubectl port-forward process
	forward    *Forward     // Reference to the forward configuration
	healthTick *time.Ticker // Health check ticker
}

// NewManager creates a new port forward manager
func NewManager(instanceName string, k8sClient *kubernetes.Client) *Manager {
	homeDir, _ := os.UserHomeDir()
	stateDir := filepath.Join(homeDir, ".kecs", "instances", instanceName, "port-forwards")

	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		instanceName: instanceName,
		k8sClient:    k8sClient,
		stateDir:     stateDir,
		forwards:     make(map[string]*Forward),
		forwarders:   make(map[string]*portForwarder),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Ensure state directory exists
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		logging.Warn("Failed to create state directory", "path", stateDir, "error", err)
	}

	// Load existing state
	m.loadState()

	// Start background monitoring
	m.startBackgroundTasks()

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
		ID:            forwardID,
		Type:          ForwardTypeService,
		Cluster:       cluster,
		TargetName:    serviceName,
		LocalPort:     localPort,
		TargetPort:    targetPort,
		Status:        StatusActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		AutoReconnect: true, // Enable auto-reconnect by default
		RetryCount:    0,
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
	forwarder, err := m.startKubectlPortForward(ctx, namespace, fmt.Sprintf("svc/%s", serviceName), localPort, targetPort, forward)
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
		ID:            forwardID,
		Type:          ForwardTypeTask,
		Cluster:       cluster,
		TargetName:    taskID,
		LocalPort:     localPort,
		TargetPort:    targetPort,
		Status:        StatusActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		AutoReconnect: true, // Enable auto-reconnect by default
		RetryCount:    0,
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
	forwarder, err := m.startKubectlPortForward(ctx, namespace, fmt.Sprintf("pod/%s", pod.Name), localPort, targetPort, forward)
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
		if forwarder.cmd != nil && forwarder.cmd.Process != nil {
			// First try graceful termination
			if err := forwarder.cmd.Process.Signal(os.Interrupt); err != nil {
				logging.Debug("Failed to send interrupt signal", "error", err)
				// If interrupt fails, force kill
				if err := forwarder.cmd.Process.Kill(); err != nil {
					logging.Warn("Failed to kill port-forward process", "error", err)
				}
			} else {
				// Wait briefly for graceful shutdown
				time.Sleep(500 * time.Millisecond)
				// Ensure process is terminated
				if forwarder.cmd.ProcessState == nil {
					if err := forwarder.cmd.Process.Kill(); err != nil {
						logging.Debug("Process already terminated", "error", err)
					}
				}
			}
		}
		if forwarder.stopCh != nil {
			select {
			case <-forwarder.stopCh:
				// Already closed
			default:
				close(forwarder.stopCh)
			}
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

	var wg sync.WaitGroup
	for forwardID, forwarder := range m.forwarders {
		wg.Add(1)
		go func(id string, fwd *portForwarder) {
			defer wg.Done()
			if fwd.cmd != nil && fwd.cmd.Process != nil {
				// Try graceful termination first
				if err := fwd.cmd.Process.Signal(os.Interrupt); err == nil {
					time.Sleep(500 * time.Millisecond)
				}
				// Force kill if still running
				if fwd.cmd.ProcessState == nil {
					if err := fwd.cmd.Process.Kill(); err != nil {
						logging.Debug("Failed to kill port-forward process", "id", id, "error", err)
					}
				}
			}
			if fwd.stopCh != nil {
				select {
				case <-fwd.stopCh:
					// Already closed
				default:
					close(fwd.stopCh)
				}
			}
		}(forwardID, forwarder)
	}

	// Wait for all forwarders to stop with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All forwarders stopped
	case <-time.After(5 * time.Second):
		logging.Warn("Timeout waiting for all port-forwards to stop")
	}

	// Clear the forwarders map
	for forwardID := range m.forwarders {
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
	// Format: [HOST:][HOSTPORT:]CONTAINERPORT[/PROTOCOL][@NODEFILTER]
	// We map localPort on host to nodePort in the container
	cmd := exec.Command("k3d", "node", "edit",
		fmt.Sprintf("k3d-%s-serverlb", clusterName),
		"--port-add", fmt.Sprintf("%d:%d", localPort, nodePort))

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if error is due to port already being mapped
		outputStr := string(output)
		if strings.Contains(outputStr, "already exists") ||
			strings.Contains(outputStr, "port is already allocated") ||
			strings.Contains(outputStr, "already in use") {
			logging.Warn("k3d port mapping already exists",
				"localPort", localPort,
				"nodePort", nodePort)
			// Continue as the mapping exists
		} else {
			return fmt.Errorf("failed to add port mapping: %w, output: %s", err, outputStr)
		}
	}

	logging.Info("Added k3d port mapping",
		"localPort", localPort,
		"nodePort", nodePort,
		"cluster", clusterName)

	// Wait for serverlb to restart with timeout
	return m.waitForPortMapping(ctx, localPort, 30*time.Second)
}

// unmapPortWithK3d removes a port mapping from k3d
func (m *Manager) unmapPortWithK3d(ctx context.Context, localPort int) error {
	// Track port mappings for future cleanup
	// k3d doesn't support removing individual port mappings without recreating the node
	// Store unmapped ports in a file for cleanup during instance restart
	unmappedFile := filepath.Join(m.stateDir, "unmapped_ports.json")

	var unmappedPorts []int
	if data, err := os.ReadFile(unmappedFile); err == nil {
		json.Unmarshal(data, &unmappedPorts)
	}

	unmappedPorts = append(unmappedPorts, localPort)
	data, _ := json.Marshal(unmappedPorts)
	os.WriteFile(unmappedFile, data, 0644)

	logging.Info("Port unmap tracked for cleanup",
		"localPort", localPort,
		"instance", m.instanceName,
		"note", "Port will be cleaned up on next instance restart")

	return nil
}

// startKubectlPortForward starts a kubectl port-forward process
func (m *Manager) startKubectlPortForward(ctx context.Context, namespace, target string, localPort, targetPort int, forward *Forward) (*portForwarder, error) {
	// Get kubeconfig path
	kubeconfigPath := filepath.Join(os.TempDir(), fmt.Sprintf("kecs-%s.config", m.instanceName))

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
		forward: forward,
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start kubectl port-forward: %w", err)
	}

	// Monitor the process in a goroutine
	go func() {
		err := cmd.Wait()
		if err != nil {
			// Only log error if it wasn't a deliberate termination
			if !strings.Contains(err.Error(), "signal: interrupt") &&
				!strings.Contains(err.Error(), "signal: killed") {
				logging.Error("kubectl port-forward exited with error",
					"namespace", namespace,
					"target", target,
					"error", err)
			}
			select {
			case forwarder.errCh <- err:
			default:
			}
		}
	}()

	// Wait for kubectl port-forward to become ready
	if err := m.waitForKubectlReady(ctx, cmd, localPort, 10*time.Second); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("kubectl port-forward failed to become ready: %w", err)
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

// waitForPortMapping waits for the k3d port mapping to become available
func (m *Manager) waitForPortMapping(ctx context.Context, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	logging.Debug("Waiting for port mapping to become available", "port", port, "timeout", timeout)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for port mapping: %w", ctx.Err())
		case <-ticker.C:
			// Try to connect to the port to check if it's available
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 1*time.Second)
			if err == nil {
				conn.Close()
				logging.Info("Port mapping ready", "port", port)
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for port %d to become available after %v: %w", port, timeout, err)
			}
			logging.Debug("Port not yet available, retrying", "port", port, "error", err)
		}
	}
}

// waitForKubectlReady waits for kubectl port-forward to become ready
func (m *Manager) waitForKubectlReady(ctx context.Context, cmd *exec.Cmd, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check if process is still running
			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				return fmt.Errorf("kubectl port-forward exited unexpectedly")
			}

			// Try to connect to the port to check if it's ready
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 500*time.Millisecond)
			if err == nil {
				conn.Close()
				return nil
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for kubectl port-forward on port %d", port)
			}
		}
	}
}

// startBackgroundTasks starts background monitoring and management tasks
func (m *Manager) startBackgroundTasks() {
	// Start health monitoring
	m.wg.Add(1)
	go m.healthMonitor()

	// Start auto-reconnection monitor
	m.wg.Add(1)
	go m.reconnectionMonitor()
}

// Stop gracefully shuts down the manager
func (m *Manager) Stop() {
	logging.Info("Shutting down port forward manager")

	// Cancel context to stop all background tasks
	m.cancel()

	// Stop all active port forwards
	m.StopAllForwards()

	// Wait for all background tasks to complete
	m.wg.Wait()
}

// healthMonitor periodically checks the health of active port forwards
func (m *Manager) healthMonitor() {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkHealthAll()
		}
	}
}

// checkHealthAll checks health of all active port forwards
func (m *Manager) checkHealthAll() {
	m.mu.RLock()
	forwardIDs := make([]string, 0, len(m.forwarders))
	for id := range m.forwarders {
		forwardIDs = append(forwardIDs, id)
	}
	m.mu.RUnlock()

	for _, id := range forwardIDs {
		m.checkHealth(id)
	}
}

// checkHealth checks health of a single port forward
func (m *Manager) checkHealth(forwardID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	forward, exists := m.forwards[forwardID]
	if !exists {
		return
	}

	forwarder, active := m.forwarders[forwardID]
	if !active {
		if forward.Status == StatusActive && forward.AutoReconnect {
			// Port forward should be active but isn't, mark for reconnection
			forward.Status = StatusError
			forward.Error = "Port forward process not found"
			forward.UpdatedAt = time.Now()
			m.saveState()
		}
		return
	}

	// Check if process is still alive
	if forwarder.cmd != nil && forwarder.cmd.Process != nil {
		// Check if process has exited
		if forwarder.cmd.ProcessState != nil && forwarder.cmd.ProcessState.Exited() {
			forward.Status = StatusError
			forward.Error = "Port forward process exited"
			forward.UpdatedAt = time.Now()
			delete(m.forwarders, forwardID)
			m.saveState()
			return
		}
	}

	// Try to connect to the port to verify it's still working
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", forward.LocalPort), 2*time.Second)
	if err != nil {
		logging.Warn("Port forward health check failed",
			"id", forwardID,
			"port", forward.LocalPort,
			"error", err)
		forward.Status = StatusError
		forward.Error = fmt.Sprintf("Health check failed: %v", err)
	} else {
		conn.Close()
		// Update last health check time
		forward.LastHealthCheck = time.Now()
		if forward.Status == StatusError {
			// Recovered from error state
			forward.Status = StatusActive
			forward.Error = ""
			logging.Info("Port forward recovered", "id", forwardID)
		}
	}

	forward.UpdatedAt = time.Now()
	m.saveState()
}

// reconnectionMonitor monitors for failed port forwards and attempts to reconnect
func (m *Manager) reconnectionMonitor() {
	defer m.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.attemptReconnections()
		}
	}
}

// attemptReconnections attempts to reconnect failed port forwards
func (m *Manager) attemptReconnections() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, forward := range m.forwards {
		// Skip if not auto-reconnect enabled or already active
		if !forward.AutoReconnect || forward.Status == StatusActive {
			continue
		}

		// Skip if forwarder already exists
		if _, exists := m.forwarders[id]; exists {
			continue
		}

		// Limit retry attempts
		if forward.RetryCount >= 5 {
			if forward.Status != StatusStopped {
				logging.Warn("Max retries reached for port forward",
					"id", id,
					"retryCount", forward.RetryCount)
				forward.Status = StatusStopped
				forward.Error = "Max retry attempts reached"
				forward.UpdatedAt = time.Now()
				m.saveState()
			}
			continue
		}

		// Attempt to reconnect
		logging.Info("Attempting to reconnect port forward",
			"id", id,
			"type", forward.Type,
			"target", forward.TargetName,
			"retryCount", forward.RetryCount)

		go m.reconnectForward(forward)
	}
}

// reconnectForward attempts to reconnect a single port forward
func (m *Manager) reconnectForward(forward *Forward) {
	// Increment retry count
	m.mu.Lock()
	forward.RetryCount++
	forward.UpdatedAt = time.Now()
	m.saveState()
	m.mu.Unlock()

	// Recreate the port forward based on type
	var err error
	switch forward.Type {
	case ForwardTypeService:
		_, _, err = m.StartServiceForward(context.Background(),
			forward.Cluster, forward.TargetName,
			forward.LocalPort, forward.TargetPort)
	case ForwardTypeTask:
		_, _, err = m.StartTaskForward(context.Background(),
			forward.Cluster, forward.TargetName,
			forward.LocalPort, forward.TargetPort)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err != nil {
		logging.Error("Failed to reconnect port forward",
			"id", forward.ID,
			"error", err)
		forward.Status = StatusError
		forward.Error = fmt.Sprintf("Reconnection failed: %v", err)
	} else {
		logging.Info("Successfully reconnected port forward",
			"id", forward.ID,
			"localPort", forward.LocalPort)
		forward.Status = StatusActive
		forward.Error = ""
		forward.RetryCount = 0 // Reset retry count on success
	}

	forward.UpdatedAt = time.Now()
	m.saveState()
}
