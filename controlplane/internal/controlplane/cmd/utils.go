package cmd

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// isPortAvailable checks if a port is available on the host
func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// getRunningKECSContainers returns all running KECS containers
func getRunningKECSContainers(ctx context.Context, cli *client.Client) ([]types.Container, error) {
	filters := filters.NewArgs()
	filters.Add("label", "app=kecs")
	filters.Add("status", "running")
	
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}
	
	return containers, nil
}

// getUsedPorts returns a map of ports currently used by KECS containers
func getUsedPorts(ctx context.Context, cli *client.Client) (map[int]string, error) {
	containers, err := getRunningKECSContainers(ctx, cli)
	if err != nil {
		return nil, err
	}
	
	usedPorts := make(map[int]string)
	for _, c := range containers {
		containerName := c.Names[0]
		if len(containerName) > 0 && containerName[0] == '/' {
			containerName = containerName[1:]
		}
		
		for _, p := range c.Ports {
			if p.PublicPort != 0 {
				usedPorts[int(p.PublicPort)] = containerName
			}
		}
	}
	
	return usedPorts, nil
}

// findAvailablePort finds an available port starting from the given port
func findAvailablePort(startPort int, usedPorts map[int]string) int {
	port := startPort
	maxAttempts := 100
	
	for i := 0; i < maxAttempts; i++ {
		if _, used := usedPorts[port]; !used && isPortAvailable(port) {
			return port
		}
		port++
	}
	
	return -1
}

// parsePortMapping parses a port mapping string (e.g., "8080:8080" or "8080")
func parsePortMapping(portStr string) (hostPort, containerPort int, err error) {
	// Check if it's a mapping (host:container)
	if parts := splitPortMapping(portStr); len(parts) == 2 {
		hostPort, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid host port: %s", parts[0])
		}
		containerPort, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid container port: %s", parts[1])
		}
		return hostPort, containerPort, nil
	}
	
	// Single port - use same for host and container
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid port: %s", portStr)
	}
	return port, port, nil
}

// splitPortMapping splits a port mapping string
func splitPortMapping(portStr string) []string {
	// Simple split by colon - could be enhanced for more complex mappings
	if colonIndex := findFirstColon(portStr); colonIndex != -1 {
		return []string{portStr[:colonIndex], portStr[colonIndex+1:]}
	}
	return []string{portStr}
}

// findFirstColon finds the first colon in a string
func findFirstColon(s string) int {
	for i, c := range s {
		if c == ':' {
			return i
		}
	}
	return -1
}