# Traefik Dynamic Port Allocation

## Overview

This document describes the dynamic port allocation feature for Traefik reverse proxy in KECS.

## Problem

When creating multiple k3d clusters with Traefik enabled, each cluster needs a unique host port for the Traefik proxy. Using a fixed port (e.g., 8090) causes conflicts when multiple clusters are created concurrently.

## Solution

Implemented dynamic port allocation with the following features:

1. **Thread-safe port allocation**: Uses a mutex to prevent race conditions during concurrent cluster creation
2. **Port availability checking**: Checks both in-memory allocated ports and actual system port availability
3. **Incremental port assignment**: Starts from port 8090 and increments to find available ports
4. **Per-cluster port tracking**: Maintains a map of cluster names to allocated Traefik ports

## Implementation Details

### Changes to K3dClusterManager

```go
type K3dClusterManager struct {
    // ... existing fields ...
    traefikPorts    map[string]int // cluster name -> traefik port mapping
    portMutex       sync.Mutex     // protects port allocation
}
```

### Port Allocation Logic

```go
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
```

### Port Finding Algorithm

The `findAvailablePort` function:
1. Checks if the port is already allocated to another cluster
2. Checks if the port is available on the host system
3. Tries up to 100 consecutive ports starting from the base port

## Usage

When creating clusters with Traefik enabled:

```bash
# Enable Traefik
export KECS_FEATURES_TRAEFIK=true

# Create multiple clusters
aws --endpoint-url http://localhost:8080 ecs create-cluster --cluster-name cluster1
aws --endpoint-url http://localhost:8080 ecs create-cluster --cluster-name cluster2
aws --endpoint-url http://localhost:8080 ecs create-cluster --cluster-name cluster3
```

Each cluster will automatically get a unique Traefik port:
- cluster1: port 8090
- cluster2: port 8091
- cluster3: port 8092

## LocalStack Integration

The dynamic Traefik port is automatically passed to LocalStack when deployed:

```go
if config.UseTraefik && api.clusterManager != nil {
    if port, exists := api.clusterManager.GetTraefikPort(cluster.K8sClusterName); exists {
        config.ProxyEndpoint = fmt.Sprintf("http://localhost:%d", port)
        log.Printf("Using dynamic Traefik port %d for LocalStack proxy", port)
    }
}
```

## Benefits

1. **No port conflicts**: Multiple clusters can be created concurrently without port conflicts
2. **Automatic configuration**: No manual port configuration needed
3. **LocalStack compatibility**: Proxy endpoints are automatically configured with the correct port
4. **Thread-safe**: Handles concurrent cluster creation properly