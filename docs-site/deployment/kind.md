# Kind Deployment

## Overview

This guide covers deploying KECS with Kind (Kubernetes in Docker) for local Kubernetes development and testing.

## Prerequisites

- Docker Desktop or Docker Engine
- Kind installed
- kubectl installed
- KECS binary or Docker image

## Installing Kind

### macOS

```bash
brew install kind
```

### Linux

```bash
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind
```

### Windows

```powershell
curl.exe -Lo kind-windows-amd64.exe https://kind.sigs.k8s.io/dl/v0.20.0/kind-windows-amd64
Move-Item .\kind-windows-amd64.exe c:\kind.exe
```

## Creating a Kind Cluster

### Basic Cluster

```bash
# Create a simple cluster
kind create cluster --name kecs-cluster

# Verify cluster is running
kubectl cluster-info --context kind-kecs-cluster
```

### Advanced Configuration

Create `kind-config.yaml`:

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: kecs-cluster
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "ingress-ready=true"
    extraPortMappings:
      - containerPort: 80
        hostPort: 80
        protocol: TCP
      - containerPort: 443
        hostPort: 443
        protocol: TCP
  - role: worker
  - role: worker
networking:
  apiServerAddress: "127.0.0.1"
  apiServerPort: 6443
```

Create cluster with configuration:

```bash
kind create cluster --config kind-config.yaml
```

## Deploying KECS to Kind

### Option 1: External KECS with Kind Backend

Run KECS outside the cluster, connecting to Kind:

```bash
# Get Kind kubeconfig
kind get kubeconfig --name kecs-cluster > ~/.kube/kind-kecs-config

# Run KECS with Kind backend
./bin/kecs server \
  --kubeconfig ~/.kube/kind-kecs-config \
  --cluster-type kind
```

### Option 2: Deploy KECS Inside Kind

#### Build and Load Docker Image

```bash
# Build KECS Docker image
docker build -t kecs:latest .

# Load image into Kind
kind load docker-image kecs:latest --name kecs-cluster
```

#### Deploy Using Kubernetes Manifests

Create `kecs-deployment.yaml`:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: kecs-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kecs-controller
  namespace: kecs-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kecs-controller
rules:
  - apiGroups: [""]
    resources: ["namespaces", "pods", "services", "configmaps", "secrets"]
    verbs: ["*"]
  - apiGroups: ["apps"]
    resources: ["deployments", "replicasets"]
    verbs: ["*"]
  - apiGroups: ["batch"]
    resources: ["jobs"]
    verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kecs-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kecs-controller
subjects:
  - kind: ServiceAccount
    name: kecs-controller
    namespace: kecs-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kecs-control-plane
  namespace: kecs-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kecs-control-plane
  template:
    metadata:
      labels:
        app: kecs-control-plane
    spec:
      serviceAccountName: kecs-controller
      containers:
        - name: kecs
          image: kecs:latest
          imagePullPolicy: Never
          command: ["/kecs", "server"]
          env:
            - name: KECS_IN_CLUSTER
              value: "true"
          ports:
            - containerPort: 8080
              name: api
            - containerPort: 8081
              name: admin
          livenessProbe:
            httpGet:
              path: /health
              port: admin
            initialDelaySeconds: 10
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /health
              port: admin
            initialDelaySeconds: 5
            periodSeconds: 5
          resources:
            requests:
              memory: "256Mi"
              cpu: "100m"
            limits:
              memory: "1Gi"
              cpu: "1000m"
---
apiVersion: v1
kind: Service
metadata:
  name: kecs-api
  namespace: kecs-system
spec:
  selector:
    app: kecs-control-plane
  ports:
    - port: 8080
      targetPort: api
      name: api
    - port: 8081
      targetPort: admin
      name: admin
  type: ClusterIP
```

Deploy to Kind:

```bash
kubectl apply -f kecs-deployment.yaml
```

#### Expose KECS API

Using port-forward:

```bash
# Forward KECS API port
kubectl port-forward -n kecs-system svc/kecs-api 8080:8080

# Forward admin port
kubectl port-forward -n kecs-system svc/kecs-api 8081:8081
```

Using NodePort:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: kecs-api-nodeport
  namespace: kecs-system
spec:
  type: NodePort
  selector:
    app: kecs-control-plane
  ports:
    - port: 8080
      targetPort: api
      nodePort: 30080
      name: api
    - port: 8081
      targetPort: admin
      nodePort: 30081
      name: admin
```

## Configuring KECS for Kind

### Dynamic Cluster Creation

KECS can dynamically create Kind clusters:

```yaml
# kecs-config.yaml
kubernetes:
  type: kind
  kind:
    provider: docker
    config:
      nodes: 2
      apiServerPort: 6443
      waitForReady: 5m
```

### Multi-Cluster Support

Configure KECS to manage multiple Kind clusters:

```yaml
clusters:
  - name: dev
    type: kind
    config:
      nodes: 1
  - name: staging
    type: kind
    config:
      nodes: 2
  - name: prod
    type: kind
    config:
      nodes: 3
      highAvailability: true
```

## Testing with Kind

### Running Integration Tests

```bash
# Set up test environment
export KECS_TEST_CLUSTER=kind-kecs-test

# Create test cluster
kind create cluster --name kecs-test

# Run integration tests
go test -tags integration ./tests/integration/...
```

### Load Testing

Create multiple nodes for load testing:

```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
  - role: worker
  - role: worker
  - role: worker
  - role: worker
```

## Monitoring and Debugging

### View KECS Logs

```bash
# If running externally
./bin/kecs server --log-level debug

# If running in Kind
kubectl logs -n kecs-system deployment/kecs-control-plane -f
```

### Access Kind Nodes

```bash
# List nodes
docker ps --filter name=kecs-cluster

# Access node
docker exec -it kecs-cluster-control-plane bash
```

### Install Debugging Tools

```bash
# Install kubectl debug plugin
kubectl krew install debug

# Debug a pod
kubectl debug -n kecs-demo pod/my-task -it --image=busybox
```

## LocalStack Integration with Kind

Deploy LocalStack in Kind cluster:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: localstack
  namespace: kecs-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: localstack
  template:
    metadata:
      labels:
        app: localstack
    spec:
      containers:
        - name: localstack
          image: localstack/localstack:latest
          ports:
            - containerPort: 4566
          env:
            - name: SERVICES
              value: "s3,dynamodb,sqs,sns,secretsmanager"
            - name: DEBUG
              value: "1"
---
apiVersion: v1
kind: Service
metadata:
  name: localstack
  namespace: kecs-system
spec:
  selector:
    app: localstack
  ports:
    - port: 4566
      targetPort: 4566
```

Configure KECS to use LocalStack:

```yaml
localstack:
  enabled: true
  endpoint: http://localstack.kecs-system.svc.cluster.local:4566
```

## Cleanup

### Delete Specific Cluster

```bash
kind delete cluster --name kecs-cluster
```

### Delete All Kind Clusters

```bash
kind delete clusters --all
```

### Clean Docker Resources

```bash
# Remove Kind node containers
docker ps -a | grep kindest/node | awk '{print $1}' | xargs docker rm -f

# Clean up volumes
docker volume prune
```

## Troubleshooting

### Cluster Creation Fails

```bash
# Check Docker daemon
docker info

# Check existing clusters
kind get clusters

# Delete and recreate
kind delete cluster --name kecs-cluster
kind create cluster --name kecs-cluster --retain
```

### Cannot Connect to Cluster

```bash
# Verify kubeconfig
kubectl config view --minify

# Set context
kubectl config use-context kind-kecs-cluster

# Test connection
kubectl get nodes
```

### Image Pull Errors

```bash
# Verify image is loaded
docker exec -it kecs-cluster-control-plane crictl images

# Reload image
kind load docker-image kecs:latest --name kecs-cluster
```

## Best Practices

1. **Resource Limits**: Always set resource limits for KECS deployment
2. **Persistent Storage**: Use local volumes for data persistence
3. **Network Policies**: Implement network policies for security
4. **Monitoring**: Deploy Prometheus and Grafana for monitoring
5. **Multi-Node**: Use multi-node clusters for testing distributed scenarios

## Next Steps

- [Production Deployment](./production) - Deploy KECS to production
- [Configuration Guide](./configuration) - Advanced configuration options
- [Troubleshooting Guide](/guides/troubleshooting) - Common issues and solutions