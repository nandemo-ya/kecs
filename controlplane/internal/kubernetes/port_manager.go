package kubernetes

import (
	"fmt"
	"sync"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// PortRange defines the range of ports available for allocation
type PortRange struct {
	Start int32
	End   int32
}

// PortAllocation represents a port allocation for a task
type PortAllocation struct {
	TaskARN       string
	HostPort      int32
	ContainerPort int32
	Protocol      string
}

// PortManager manages dynamic port allocation for tasks with assignPublicIp
type PortManager struct {
	mu          sync.RWMutex
	portRange   PortRange
	allocations map[int32]*PortAllocation // hostPort -> allocation
	taskPorts   map[string][]int32        // taskARN -> []hostPort
	nextPort    int32
}

// NewPortManager creates a new port manager with the specified port range
func NewPortManager(start, end int32) *PortManager {
	return &PortManager{
		portRange:   PortRange{Start: start, End: end},
		allocations: make(map[int32]*PortAllocation),
		taskPorts:   make(map[string][]int32),
		nextPort:    start,
	}
}

// AllocatePort allocates a port for a task
func (pm *PortManager) AllocatePort(taskARN string, containerPort int32, protocol string) (int32, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Find an available port
	var allocatedPort int32
	found := false

	// Try to find an unused port starting from nextPort
	for i := pm.nextPort; i <= pm.portRange.End; i++ {
		if _, exists := pm.allocations[i]; !exists {
			allocatedPort = i
			found = true
			pm.nextPort = i + 1
			break
		}
	}

	// If not found, wrap around and search from the beginning
	if !found {
		for i := pm.portRange.Start; i < pm.nextPort; i++ {
			if _, exists := pm.allocations[i]; !exists {
				allocatedPort = i
				found = true
				pm.nextPort = i + 1
				break
			}
		}
	}

	if !found {
		return 0, fmt.Errorf("no available ports in range %d-%d", pm.portRange.Start, pm.portRange.End)
	}

	// Reset nextPort if it exceeds the range
	if pm.nextPort > pm.portRange.End {
		pm.nextPort = pm.portRange.Start
	}

	// Create allocation
	allocation := &PortAllocation{
		TaskARN:       taskARN,
		HostPort:      allocatedPort,
		ContainerPort: containerPort,
		Protocol:      protocol,
	}

	pm.allocations[allocatedPort] = allocation
	pm.taskPorts[taskARN] = append(pm.taskPorts[taskARN], allocatedPort)

	logging.Info("Allocated port for task",
		"taskARN", taskARN,
		"hostPort", allocatedPort,
		"containerPort", containerPort,
		"protocol", protocol)

	return allocatedPort, nil
}

// ReleasePort releases a specific port
func (pm *PortManager) ReleasePort(hostPort int32) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	allocation, exists := pm.allocations[hostPort]
	if !exists {
		return fmt.Errorf("port %d is not allocated", hostPort)
	}

	// Remove from allocations
	delete(pm.allocations, hostPort)

	// Remove from task ports
	if ports, exists := pm.taskPorts[allocation.TaskARN]; exists {
		newPorts := []int32{}
		for _, p := range ports {
			if p != hostPort {
				newPorts = append(newPorts, p)
			}
		}
		if len(newPorts) > 0 {
			pm.taskPorts[allocation.TaskARN] = newPorts
		} else {
			delete(pm.taskPorts, allocation.TaskARN)
		}
	}

	logging.Info("Released port",
		"hostPort", hostPort,
		"taskARN", allocation.TaskARN)

	return nil
}

// ReleaseTaskPorts releases all ports allocated to a task
func (pm *PortManager) ReleaseTaskPorts(taskARN string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	ports, exists := pm.taskPorts[taskARN]
	if !exists {
		return nil // No ports allocated for this task
	}

	for _, port := range ports {
		delete(pm.allocations, port)
		logging.Info("Released port for task",
			"taskARN", taskARN,
			"hostPort", port)
	}

	delete(pm.taskPorts, taskARN)

	return nil
}

// GetTaskPorts returns all ports allocated to a task
func (pm *PortManager) GetTaskPorts(taskARN string) []int32 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	ports := pm.taskPorts[taskARN]
	result := make([]int32, len(ports))
	copy(result, ports)
	return result
}

// GetAllocation returns the allocation for a specific port
func (pm *PortManager) GetAllocation(hostPort int32) (*PortAllocation, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	allocation, exists := pm.allocations[hostPort]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent modification
	result := *allocation
	return &result, true
}

// GetAllocatedPortsCount returns the number of allocated ports
func (pm *PortManager) GetAllocatedPortsCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return len(pm.allocations)
}

// GetAvailablePortsCount returns the number of available ports
func (pm *PortManager) GetAvailablePortsCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	total := int(pm.portRange.End - pm.portRange.Start + 1)
	return total - len(pm.allocations)
}
