# Kubernetes Integration Architecture

## Overview

KECS integrates deeply with Kubernetes to provide the underlying container orchestration while maintaining full ECS API compatibility. This document details how ECS concepts map to Kubernetes resources and how the integration layer works.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                   Kubernetes Integration Layer                   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                  Resource Converters                      │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │    Task      │  │   Service    │  │   Cluster    │  │  │
│  │  │  Converter   │  │  Converter   │  │  Converter   │  │  │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  │  │
│  └─────────┼──────────────────┼──────────────────┼──────────┘  │
│            │                  │                  │              │
│  ┌─────────▼──────────────────▼──────────────────▼──────────┐  │
│  │              Kubernetes Client Manager                    │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐  │  │
│  │  │  Client     │  │   Dynamic    │  │   Informer    │  │  │
│  │  │   Cache     │  │   Client     │  │   Factory     │  │  │
│  │  └─────────────┘  └──────────────┘  └───────────────┘  │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                  Resource Managers                        │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │     Pod      │  │  Deployment  │  │   Service    │  │  │
│  │  │   Manager    │  │   Manager    │  │   Manager    │  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │  ConfigMap   │  │    Secret    │  │  Namespace   │  │  │
│  │  │   Manager    │  │   Manager    │  │   Manager    │  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                   Status Watchers                         │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │  │
│  │  │     Pod      │  │  Deployment  │  │    Event     │  │  │
│  │  │   Watcher    │  │   Watcher    │  │   Watcher    │  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Resource Mapping

### ECS to Kubernetes Resource Mapping

| ECS Concept | Kubernetes Resource | Notes |
|-------------|-------------------|--------|
| Cluster | Namespace | Each ECS cluster maps to a K8s namespace |
| Service | Deployment + Service | Long-running services use Deployments |
| Task | Pod | One-off tasks run as standalone Pods |
| Task Definition | Pod Template | Container specs stored in ConfigMaps |
| Container Instance | Node | EC2 launch type only |
| Task Role | ServiceAccount | IAM roles map to ServiceAccounts |
| Secrets | Secret | Secrets Manager/SSM integration |
| Service Discovery | Service + Endpoints | DNS-based discovery |
| Load Balancer | Service (LoadBalancer) | Or Ingress for ALB |

### Detailed Mappings

#### Cluster to Namespace

```go
func ConvertClusterToNamespace(cluster *ecs.Cluster) *v1.Namespace {
    return &v1.Namespace{
        ObjectMeta: metav1.ObjectMeta{
            Name: sanitizeClusterName(cluster.ClusterName),
            Labels: map[string]string{
                "kecs.io/cluster-name": cluster.ClusterName,
                "kecs.io/cluster-arn":  cluster.ClusterArn,
                "kecs.io/managed":      "true",
            },
            Annotations: map[string]string{
                "kecs.io/settings":      jsonMarshal(cluster.Settings),
                "kecs.io/configuration": jsonMarshal(cluster.Configuration),
            },
        },
    }
}
```

#### Service to Deployment

```go
func ConvertServiceToDeployment(service *ecs.Service, taskDef *ecs.TaskDefinition) *appsv1.Deployment {
    return &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      service.ServiceName,
            Namespace: getNamespaceFromCluster(service.ClusterArn),
            Labels: map[string]string{
                "kecs.io/service-name": service.ServiceName,
                "kecs.io/service-arn":  service.ServiceArn,
                "kecs.io/cluster-arn":  service.ClusterArn,
            },
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: int32Ptr(service.DesiredCount),
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    "kecs.io/service": service.ServiceName,
                },
            },
            Template: v1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{
                        "kecs.io/service":     service.ServiceName,
                        "kecs.io/task-family": taskDef.Family,
                        "kecs.io/launch-type": service.LaunchType,
                    },
                },
                Spec: convertTaskDefinitionToPodSpec(taskDef, service),
            },
            Strategy: convertDeploymentStrategy(service.DeploymentConfiguration),
        },
    }
}
```

#### Task Definition to Pod Spec

```go
func convertTaskDefinitionToPodSpec(taskDef *ecs.TaskDefinition, service *ecs.Service) v1.PodSpec {
    spec := v1.PodSpec{
        RestartPolicy: v1.RestartPolicyAlways,
        Containers:    []v1.Container{},
    }
    
    // Convert containers
    for _, containerDef := range taskDef.ContainerDefinitions {
        container := v1.Container{
            Name:  containerDef.Name,
            Image: containerDef.Image,
            Ports: convertPortMappings(containerDef.PortMappings),
            Env:   convertEnvironment(containerDef.Environment),
            Resources: v1.ResourceRequirements{
                Requests: v1.ResourceList{
                    v1.ResourceCPU:    convertCPU(containerDef.Cpu),
                    v1.ResourceMemory: convertMemory(containerDef.MemoryReservation),
                },
                Limits: v1.ResourceList{
                    v1.ResourceCPU:    convertCPU(containerDef.Cpu),
                    v1.ResourceMemory: convertMemory(containerDef.Memory),
                },
            },
            LivenessProbe:  convertHealthCheck(containerDef.HealthCheck),
            ReadinessProbe: convertHealthCheck(containerDef.HealthCheck),
        }
        
        // Handle secrets
        if len(containerDef.Secrets) > 0 {
            container.EnvFrom = append(container.EnvFrom, v1.EnvFromSource{
                SecretRef: &v1.SecretEnvSource{
                    LocalObjectReference: v1.LocalObjectReference{
                        Name: fmt.Sprintf("%s-secrets", taskDef.Family),
                    },
                },
            })
        }
        
        spec.Containers = append(spec.Containers, container)
    }
    
    // Set task role as service account
    if taskDef.TaskRoleArn != "" {
        spec.ServiceAccountName = getServiceAccountFromRole(taskDef.TaskRoleArn)
    }
    
    // Configure network mode
    if taskDef.NetworkMode == "host" {
        spec.HostNetwork = true
    }
    
    // Add volumes
    spec.Volumes = convertVolumes(taskDef.Volumes)
    
    return spec
}
```

## Client Management

### Client Factory

```go
type ClientManager interface {
    GetClient(clusterName string) (kubernetes.Interface, error)
    GetDynamicClient(clusterName string) (dynamic.Interface, error)
    GetInformerFactory(clusterName string) (informers.SharedInformerFactory, error)
}

type clientManager struct {
    clients   map[string]*clusterClient
    mu        sync.RWMutex
    config    *Config
}

type clusterClient struct {
    client         kubernetes.Interface
    dynamicClient  dynamic.Interface
    informerFactory informers.SharedInformerFactory
    lastUsed       time.Time
}

func (cm *clientManager) GetClient(clusterName string) (kubernetes.Interface, error) {
    cm.mu.RLock()
    if client, ok := cm.clients[clusterName]; ok {
        client.lastUsed = time.Now()
        cm.mu.RUnlock()
        return client.client, nil
    }
    cm.mu.RUnlock()
    
    // Create new client
    return cm.createClient(clusterName)
}
```

### Multi-Cluster Support

```go
type ClusterConfig struct {
    Name       string
    Type       string // "kind", "eks", "gke", "aks", "generic"
    Kubeconfig string
    Context    string
    Endpoint   string
}

func (cm *clientManager) createClient(clusterName string) (kubernetes.Interface, error) {
    clusterConfig, err := cm.getClusterConfig(clusterName)
    if err != nil {
        return nil, err
    }
    
    var config *rest.Config
    
    switch clusterConfig.Type {
    case "kind":
        config, err = cm.getKindConfig(clusterConfig)
    case "eks":
        config, err = cm.getEKSConfig(clusterConfig)
    case "in-cluster":
        config, err = rest.InClusterConfig()
    default:
        config, err = clientcmd.BuildConfigFromFlags("", clusterConfig.Kubeconfig)
    }
    
    if err != nil {
        return nil, err
    }
    
    // Create client with retry and timeout configuration
    config.QPS = 100
    config.Burst = 200
    config.Timeout = 30 * time.Second
    
    return kubernetes.NewForConfig(config)
}
```

## Resource Lifecycle Management

### Pod Lifecycle

```go
type PodManager struct {
    client    kubernetes.Interface
    namespace string
}

func (pm *PodManager) CreatePod(taskDef *ecs.TaskDefinition, overrides *ecs.TaskOverride) (*v1.Pod, error) {
    pod := &v1.Pod{
        ObjectMeta: metav1.ObjectMeta{
            GenerateName: fmt.Sprintf("%s-", taskDef.Family),
            Namespace:    pm.namespace,
            Labels: map[string]string{
                "kecs.io/task-definition": fmt.Sprintf("%s:%d", taskDef.Family, taskDef.Revision),
                "kecs.io/task-type":       "standalone",
            },
            Annotations: map[string]string{
                "kecs.io/task-arn": generateTaskArn(),
            },
        },
        Spec: convertTaskDefinitionToPodSpec(taskDef, nil),
    }
    
    // Apply overrides
    if overrides != nil {
        applyTaskOverrides(pod, overrides)
    }
    
    // Create pod
    created, err := pm.client.CoreV1().Pods(pm.namespace).Create(
        context.TODO(), pod, metav1.CreateOptions{},
    )
    if err != nil {
        return nil, err
    }
    
    return created, nil
}

func (pm *PodManager) StopPod(taskArn string, reason string) error {
    pod, err := pm.getPodByTaskArn(taskArn)
    if err != nil {
        return err
    }
    
    // Add termination reason
    pod.Annotations["kecs.io/stop-reason"] = reason
    
    // Update pod first
    _, err = pm.client.CoreV1().Pods(pm.namespace).Update(
        context.TODO(), pod, metav1.UpdateOptions{},
    )
    
    // Delete pod with grace period
    deleteOptions := metav1.DeleteOptions{
        GracePeriodSeconds: int64Ptr(30),
    }
    
    return pm.client.CoreV1().Pods(pm.namespace).Delete(
        context.TODO(), pod.Name, deleteOptions,
    )
}
```

### Deployment Management

```go
type DeploymentManager struct {
    client    kubernetes.Interface
    namespace string
}

func (dm *DeploymentManager) UpdateDeployment(
    service *ecs.Service, 
    taskDef *ecs.TaskDefinition,
) (*appsv1.Deployment, error) {
    
    deployment, err := dm.getDeployment(service.ServiceName)
    if err != nil {
        return nil, err
    }
    
    // Update deployment spec
    deployment.Spec.Template = v1.PodTemplateSpec{
        ObjectMeta: metav1.ObjectMeta{
            Labels: deployment.Spec.Template.Labels,
            Annotations: map[string]string{
                "kecs.io/task-definition": fmt.Sprintf("%s:%d", taskDef.Family, taskDef.Revision),
                "kecs.io/updated-at":      time.Now().Format(time.RFC3339),
            },
        },
        Spec: convertTaskDefinitionToPodSpec(taskDef, service),
    }
    
    // Apply deployment configuration
    if service.DeploymentConfiguration != nil {
        strategy := convertDeploymentStrategy(service.DeploymentConfiguration)
        deployment.Spec.Strategy = strategy
    }
    
    // Update deployment
    updated, err := dm.client.AppsV1().Deployments(dm.namespace).Update(
        context.TODO(), deployment, metav1.UpdateOptions{},
    )
    
    return updated, err
}

func convertDeploymentStrategy(config *ecs.DeploymentConfiguration) appsv1.DeploymentStrategy {
    maxSurge := intstr.FromString(fmt.Sprintf("%d%%", config.MaximumPercent-100))
    maxUnavailable := intstr.FromString(fmt.Sprintf("%d%%", 100-config.MinimumHealthyPercent))
    
    return appsv1.DeploymentStrategy{
        Type: appsv1.RollingUpdateDeploymentStrategyType,
        RollingUpdate: &appsv1.RollingUpdateDeployment{
            MaxSurge:       &maxSurge,
            MaxUnavailable: &maxUnavailable,
        },
    }
}
```

## Status Synchronization

### Informer Pattern

```go
type StatusWatcher struct {
    informerFactory informers.SharedInformerFactory
    taskManager     *TaskManager
    serviceManager  *ServiceManager
}

func (sw *StatusWatcher) Start(stopCh <-chan struct{}) {
    // Watch pods for task status
    podInformer := sw.informerFactory.Core().V1().Pods()
    podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc:    sw.handlePodAdd,
        UpdateFunc: sw.handlePodUpdate,
        DeleteFunc: sw.handlePodDelete,
    })
    
    // Watch deployments for service status
    deploymentInformer := sw.informerFactory.Apps().V1().Deployments()
    deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc:    sw.handleDeploymentAdd,
        UpdateFunc: sw.handleDeploymentUpdate,
        DeleteFunc: sw.handleDeploymentDelete,
    })
    
    // Start informers
    sw.informerFactory.Start(stopCh)
    sw.informerFactory.WaitForCacheSync(stopCh)
}

func (sw *StatusWatcher) handlePodUpdate(oldObj, newObj interface{}) {
    oldPod := oldObj.(*v1.Pod)
    newPod := newObj.(*v1.Pod)
    
    // Skip if no status change
    if oldPod.Status.Phase == newPod.Status.Phase {
        return
    }
    
    // Extract task ARN
    taskArn, ok := newPod.Annotations["kecs.io/task-arn"]
    if !ok {
        return
    }
    
    // Convert pod status to ECS task status
    taskStatus := convertPodStatusToTaskStatus(newPod.Status)
    
    // Update task in storage
    err := sw.taskManager.UpdateTaskStatus(taskArn, taskStatus)
    if err != nil {
        log.Error("Failed to update task status", 
            zap.String("taskArn", taskArn),
            zap.Error(err))
    }
    
    // Publish event
    sw.publishTaskEvent(taskArn, taskStatus)
}
```

### Status Mapping

```go
func convertPodStatusToTaskStatus(podStatus v1.PodStatus) *ecs.TaskStatus {
    status := &ecs.TaskStatus{
        LastStatus: "PENDING",
        Containers: []ecs.ContainerStatus{},
    }
    
    switch podStatus.Phase {
    case v1.PodPending:
        if len(podStatus.ContainerStatuses) > 0 {
            // Check if containers are being created
            for _, cs := range podStatus.ContainerStatuses {
                if cs.State.Waiting != nil && 
                   cs.State.Waiting.Reason == "ContainerCreating" {
                    status.LastStatus = "PROVISIONING"
                    break
                }
            }
        }
        
    case v1.PodRunning:
        status.LastStatus = "RUNNING"
        
    case v1.PodSucceeded:
        status.LastStatus = "STOPPED"
        status.StopCode = "EssentialContainerExited"
        status.StoppedReason = "Essential container exited"
        
    case v1.PodFailed:
        status.LastStatus = "STOPPED"
        status.StopCode = "TaskFailedToStart"
        status.StoppedReason = podStatus.Message
    }
    
    // Convert container statuses
    for _, containerStatus := range podStatus.ContainerStatuses {
        status.Containers = append(status.Containers, convertContainerStatus(containerStatus))
    }
    
    return status
}
```

## Secret and Configuration Management

### Secret Integration

```go
type SecretManager struct {
    client         kubernetes.Interface
    secretsManager secretsmanager.Client
    ssmClient      ssm.Client
}

func (sm *SecretManager) CreateTaskSecrets(
    namespace string,
    taskDef *ecs.TaskDefinition,
) error {
    secrets := make(map[string]string)
    
    // Collect all secrets from containers
    for _, container := range taskDef.ContainerDefinitions {
        for _, secret := range container.Secrets {
            value, err := sm.fetchSecretValue(secret.ValueFrom)
            if err != nil {
                return err
            }
            secrets[secret.Name] = value
        }
    }
    
    if len(secrets) == 0 {
        return nil
    }
    
    // Create Kubernetes secret
    k8sSecret := &v1.Secret{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("%s-secrets", taskDef.Family),
            Namespace: namespace,
            Labels: map[string]string{
                "kecs.io/task-family": taskDef.Family,
                "kecs.io/managed":     "true",
            },
        },
        Type:       v1.SecretTypeOpaque,
        StringData: secrets,
    }
    
    _, err := sm.client.CoreV1().Secrets(namespace).Create(
        context.TODO(), k8sSecret, metav1.CreateOptions{},
    )
    
    return err
}

func (sm *SecretManager) fetchSecretValue(arn string) (string, error) {
    if strings.Contains(arn, ":secretsmanager:") {
        return sm.fetchFromSecretsManager(arn)
    } else if strings.Contains(arn, ":ssm:") {
        return sm.fetchFromSSM(arn)
    }
    return "", fmt.Errorf("unsupported secret type: %s", arn)
}
```

### ConfigMap for Task Definitions

```go
func (cm *ConfigMapManager) StoreTaskDefinition(
    namespace string,
    taskDef *ecs.TaskDefinition,
) error {
    configMap := &v1.ConfigMap{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("taskdef-%s-%d", taskDef.Family, taskDef.Revision),
            Namespace: namespace,
            Labels: map[string]string{
                "kecs.io/task-family":   taskDef.Family,
                "kecs.io/task-revision": fmt.Sprintf("%d", taskDef.Revision),
            },
        },
        Data: map[string]string{
            "definition.json": jsonMarshal(taskDef),
        },
    }
    
    _, err := cm.client.CoreV1().ConfigMaps(namespace).Create(
        context.TODO(), configMap, metav1.CreateOptions{},
    )
    
    return err
}
```

## Service Discovery

### Kubernetes Service Creation

```go
func (sm *ServiceManager) CreateKubernetesService(
    ecsService *ecs.Service,
    taskDef *ecs.TaskDefinition,
) (*v1.Service, error) {
    
    // Find target port from task definition
    targetPort := getTargetPort(ecsService, taskDef)
    
    k8sService := &v1.Service{
        ObjectMeta: metav1.ObjectMeta{
            Name:      ecsService.ServiceName,
            Namespace: getNamespaceFromCluster(ecsService.ClusterArn),
            Labels: map[string]string{
                "kecs.io/service-name": ecsService.ServiceName,
                "kecs.io/service-arn":  ecsService.ServiceArn,
            },
        },
        Spec: v1.ServiceSpec{
            Selector: map[string]string{
                "kecs.io/service": ecsService.ServiceName,
            },
            Ports: []v1.ServicePort{
                {
                    Name:       "primary",
                    Port:       targetPort,
                    TargetPort: intstr.FromInt(int(targetPort)),
                    Protocol:   v1.ProtocolTCP,
                },
            },
            Type: v1.ServiceTypeClusterIP,
        },
    }
    
    // Configure load balancer if needed
    if len(ecsService.LoadBalancers) > 0 {
        k8sService.Spec.Type = v1.ServiceTypeLoadBalancer
        k8sService.Annotations = map[string]string{
            "service.beta.kubernetes.io/aws-load-balancer-type": "nlb",
        }
    }
    
    return sm.client.CoreV1().Services(k8sService.Namespace).Create(
        context.TODO(), k8sService, metav1.CreateOptions{},
    )
}
```

### DNS Integration

```go
type ServiceDiscoveryManager struct {
    client kubernetes.Interface
}

func (sdm *ServiceDiscoveryManager) RegisterService(
    service *ecs.Service,
    registry *ecs.ServiceRegistry,
) error {
    // Create headless service for DNS
    headlessService := &v1.Service{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("%s-discovery", service.ServiceName),
            Namespace: getNamespaceFromCluster(service.ClusterArn),
            Labels: map[string]string{
                "kecs.io/service-discovery": "true",
                "kecs.io/service-name":      service.ServiceName,
            },
        },
        Spec: v1.ServiceSpec{
            ClusterIP: "None", // Headless service
            Selector: map[string]string{
                "kecs.io/service": service.ServiceName,
            },
            Ports: []v1.ServicePort{
                {
                    Name:       "discovery",
                    Port:       registry.Port,
                    TargetPort: intstr.FromInt(int(registry.ContainerPort)),
                },
            },
        },
    }
    
    _, err := sdm.client.CoreV1().Services(headlessService.Namespace).Create(
        context.TODO(), headlessService, metav1.CreateOptions{},
    )
    
    return err
}
```

## Kind Integration

### Dynamic Cluster Creation

```go
type KindManager struct {
    provider *cluster.Provider
}

func (km *KindManager) CreateCluster(clusterName string) error {
    config := &v1alpha4.Cluster{
        TypeMeta: v1alpha4.TypeMeta{
            Kind:       "Cluster",
            APIVersion: "kind.x-k8s.io/v1alpha4",
        },
        Nodes: []v1alpha4.Node{
            {
                Role: v1alpha4.ControlPlaneRole,
                ExtraPortMappings: []v1alpha4.PortMapping{
                    {
                        ContainerPort: 80,
                        HostPort:      80,
                        Protocol:      v1alpha4.PortMappingProtocolTCP,
                    },
                    {
                        ContainerPort: 443,
                        HostPort:      443,
                        Protocol:      v1alpha4.PortMappingProtocolTCP,
                    },
                },
            },
        },
    }
    
    return km.provider.Create(
        clusterName,
        cluster.CreateWithV1Alpha4Config(config),
        cluster.CreateWithWaitForReady(5*time.Minute),
    )
}

func (km *KindManager) DeleteCluster(clusterName string) error {
    return km.provider.Delete(clusterName)
}

func (km *KindManager) GetKubeconfig(clusterName string) (string, error) {
    return km.provider.KubeConfig(clusterName, false)
}
```

## Performance Optimizations

### Client Caching

```go
type CachedKubernetesClient struct {
    client       kubernetes.Interface
    lister       cache.GenericLister
    resyncPeriod time.Duration
}

func (c *CachedKubernetesClient) GetPod(namespace, name string) (*v1.Pod, error) {
    // Try cache first
    obj, err := c.lister.ByNamespace(namespace).Get(name)
    if err == nil {
        return obj.(*v1.Pod), nil
    }
    
    // Fall back to API call
    return c.client.CoreV1().Pods(namespace).Get(
        context.TODO(), name, metav1.GetOptions{},
    )
}
```

### Batch Operations

```go
func (pm *PodManager) CreatePodsBatch(pods []*v1.Pod) error {
    var wg sync.WaitGroup
    errors := make(chan error, len(pods))
    
    // Limit concurrent operations
    semaphore := make(chan struct{}, 10)
    
    for _, pod := range pods {
        wg.Add(1)
        go func(p *v1.Pod) {
            defer wg.Done()
            
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            
            _, err := pm.client.CoreV1().Pods(p.Namespace).Create(
                context.TODO(), p, metav1.CreateOptions{},
            )
            if err != nil {
                errors <- err
            }
        }(pod)
    }
    
    wg.Wait()
    close(errors)
    
    // Collect errors
    var errs []error
    for err := range errors {
        errs = append(errs, err)
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("batch creation failed: %v", errs)
    }
    
    return nil
}
```

## Error Handling

### Kubernetes Error Mapping

```go
func mapKubernetesError(err error) error {
    if errors.IsNotFound(err) {
        return &ecs.ResourceNotFoundException{
            Message: "Resource not found",
        }
    }
    
    if errors.IsAlreadyExists(err) {
        return &ecs.ResourceInUseException{
            Message: "Resource already exists",
        }
    }
    
    if errors.IsForbidden(err) {
        return &ecs.AccessDeniedException{
            Message: "Access denied",
        }
    }
    
    if errors.IsTimeout(err) {
        return &ecs.ServerException{
            Message: "Operation timed out",
        }
    }
    
    return &ecs.ServerException{
        Message: fmt.Sprintf("Kubernetes error: %v", err),
    }
}
```

## Monitoring and Metrics

### Kubernetes Metrics

```go
var (
    k8sAPICallDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "kecs_k8s_api_duration_seconds",
            Help: "Kubernetes API call duration",
        },
        []string{"operation", "resource", "namespace"},
    )
    
    k8sResourceCount = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "kecs_k8s_resource_count",
            Help: "Count of Kubernetes resources",
        },
        []string{"type", "namespace"},
    )
    
    k8sErrors = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "kecs_k8s_errors_total",
            Help: "Total Kubernetes API errors",
        },
        []string{"operation", "error_type"},
    )
)
```

## Future Enhancements

1. **Advanced Scheduling**
   - Pod topology spread constraints
   - Inter-pod affinity/anti-affinity
   - Custom scheduler integration

2. **Network Policies**
   - Automatic NetworkPolicy generation
   - Service mesh integration
   - Multi-cluster networking

3. **Resource Optimization**
   - Vertical Pod Autoscaling
   - Resource recommendation
   - Cost optimization

4. **Enhanced Observability**
   - Distributed tracing
   - Custom metrics
   - Log aggregation