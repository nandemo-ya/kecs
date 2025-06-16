package localstack

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// kubernetesManager implements KubernetesManager interface
type kubernetesManager struct {
	client    kubernetes.Interface
	namespace string
}

// NewKubernetesManager creates a new Kubernetes manager
func NewKubernetesManager(client kubernetes.Interface, namespace string) KubernetesManager {
	return &kubernetesManager{
		client:    client,
		namespace: namespace,
	}
}

// CreateNamespace creates the namespace for LocalStack if it doesn't exist
func (km *kubernetesManager) CreateNamespace(ctx context.Context) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: km.namespace,
			Labels: map[string]string{
				LabelManagedBy: "kecs",
				LabelComponent: "localstack",
			},
		},
	}

	_, err := km.client.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	klog.Infof("Namespace %s created or already exists", km.namespace)
	return nil
}

// DeployLocalStack creates all necessary Kubernetes resources for LocalStack
func (km *kubernetesManager) DeployLocalStack(ctx context.Context, config *Config) error {
	// Create PVC
	if err := km.createPVC(ctx, config); err != nil {
		return fmt.Errorf("failed to create PVC: %w", err)
	}

	// Create ConfigMap
	if err := km.createConfigMap(ctx, config); err != nil {
		return fmt.Errorf("failed to create ConfigMap: %w", err)
	}

	// Create Deployment
	if err := km.createDeployment(ctx, config); err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create Service
	if err := km.createService(ctx, config); err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	return nil
}

// createPVC creates a PersistentVolumeClaim for LocalStack data
func (km *kubernetesManager) createPVC(ctx context.Context, config *Config) error {
	if !config.Persistence {
		return nil
	}

	storageQuantity, err := resource.ParseQuantity(config.Resources.StorageSize)
	if err != nil {
		return fmt.Errorf("invalid storage size: %w", err)
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "localstack-data",
			Namespace: km.namespace,
			Labels: map[string]string{
				LabelApp:       "localstack",
				LabelManagedBy: "kecs",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: storageQuantity,
				},
			},
		},
	}

	_, err = km.client.CoreV1().PersistentVolumeClaims(km.namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// createConfigMap creates a ConfigMap for LocalStack configuration
func (km *kubernetesManager) createConfigMap(ctx context.Context, config *Config) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "localstack-config",
			Namespace: km.namespace,
			Labels: map[string]string{
				LabelApp:       "localstack",
				LabelManagedBy: "kecs",
			},
		},
		Data: map[string]string{
			"services": config.GetServicesString(),
		},
	}

	_, err := km.client.CoreV1().ConfigMaps(km.namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// createDeployment creates the LocalStack deployment
func (km *kubernetesManager) createDeployment(ctx context.Context, config *Config) error {
	replicas := int32(1)
	
	// Parse resource limits
	memoryQuantity, _ := resource.ParseQuantity(config.Resources.Memory)
	cpuQuantity, _ := resource.ParseQuantity(config.Resources.CPU)

	// Build environment variables
	envVars := []corev1.EnvVar{}
	for k, v := range config.GetEnvironmentVars() {
		envVars = append(envVars, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "localstack",
			Namespace: km.namespace,
			Labels: map[string]string{
				LabelApp:       "localstack",
				LabelManagedBy: "kecs",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					LabelApp: "localstack",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						LabelApp:       "localstack",
						LabelManagedBy: "kecs",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "localstack",
							Image: config.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "edge",
									ContainerPort: int32(config.EdgePort),
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: envVars,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: memoryQuantity,
									corev1.ResourceCPU:    cpuQuantity,
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: memoryQuantity,
									corev1.ResourceCPU:    cpuQuantity,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: HealthCheckPath,
										Port: intstr.FromInt(config.EdgePort),
									},
								},
								InitialDelaySeconds: 60,
								PeriodSeconds:       10,
								TimeoutSeconds:      5,
								FailureThreshold:    3,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: HealthCheckPath,
										Port: intstr.FromInt(config.EdgePort),
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       5,
								TimeoutSeconds:      5,
								FailureThreshold:    3,
							},
						},
					},
				},
			},
		},
	}

	// Add volume mounts if persistence is enabled
	if config.Persistence {
		deployment.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: "localstack-data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: "localstack-data",
					},
				},
			},
		}

		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "localstack-data",
				MountPath: config.DataDir,
			},
		}
	}

	_, err := km.client.AppsV1().Deployments(km.namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// createService creates the Kubernetes service for LocalStack
func (km *kubernetesManager) createService(ctx context.Context, config *Config) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "localstack",
			Namespace: km.namespace,
			Labels: map[string]string{
				LabelApp:       "localstack",
				LabelManagedBy: "kecs",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				LabelApp: "localstack",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "edge",
					Port:       int32(config.Port),
					TargetPort: intstr.FromInt(config.EdgePort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	_, err := km.client.CoreV1().Services(km.namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// WaitForLocalStackReady waits for LocalStack to output "Ready." in its logs
func (km *kubernetesManager) WaitForLocalStackReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	// First, wait for pod to be running
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for LocalStack to be ready")
			}
			
			pods, err := km.client.CoreV1().Pods(km.namespace).List(ctx, metav1.ListOptions{
				LabelSelector: "app=localstack",
			})
			if err != nil {
				klog.Warningf("Failed to list pods: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}
			
			if len(pods.Items) == 0 {
				klog.Info("No LocalStack pods found yet")
				time.Sleep(2 * time.Second)
				continue
			}
			
			pod := &pods.Items[0]
			if pod.Status.Phase == corev1.PodRunning {
				// Pod is running, now check logs for "Ready."
				return km.waitForReadyInLogs(ctx, pod.Name, deadline)
			}
			
			klog.Infof("Pod status: %s", pod.Status.Phase)
			time.Sleep(2 * time.Second)
		}
	}
}

// waitForReadyInLogs monitors pod logs for the "Ready." message
func (km *kubernetesManager) waitForReadyInLogs(ctx context.Context, podName string, deadline time.Time) error {
	req := km.client.CoreV1().Pods(km.namespace).GetLogs(podName, &corev1.PodLogOptions{
		Follow: true,
		Container: "localstack",
	})
	
	stream, err := req.Stream(ctx)
	if err != nil {
		return fmt.Errorf("failed to get log stream: %w", err)
	}
	defer stream.Close()
	
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for LocalStack Ready message")
			}
			
			line := scanner.Text()
			klog.V(4).Infof("LocalStack log: %s", line)
			
			// Check for "Ready." in the log line
			if strings.Contains(line, "Ready.") {
				klog.Info("LocalStack is ready!")
				return nil
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading logs: %w", err)
	}
	
	return fmt.Errorf("log stream ended without Ready message")
}

// DeleteLocalStack deletes all LocalStack resources
func (km *kubernetesManager) DeleteLocalStack(ctx context.Context) error {
	// Delete deployment
	err := km.client.AppsV1().Deployments(km.namespace).Delete(ctx, "localstack", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	// Delete service
	err = km.client.CoreV1().Services(km.namespace).Delete(ctx, "localstack", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	// Delete configmap
	err = km.client.CoreV1().ConfigMaps(km.namespace).Delete(ctx, "localstack-config", metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete configmap: %w", err)
	}

	// Note: We don't delete the PVC to preserve data

	return nil
}

// GetLocalStackPod returns the name of the LocalStack pod
func (km *kubernetesManager) GetLocalStackPod() (string, error) {
	pods, err := km.client.CoreV1().Pods(km.namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=localstack", LabelApp),
	})
	if err != nil {
		return "", fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			// Check if all containers are ready
			allReady := true
			for _, condition := range pod.Status.Conditions {
				if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
					allReady = false
					break
				}
			}
			if allReady {
				return pod.Name, nil
			}
		}
	}

	return "", fmt.Errorf("no running LocalStack pod found")
}

// GetServiceEndpoint returns the endpoint for the LocalStack service
func (km *kubernetesManager) GetServiceEndpoint() (string, error) {
	service, err := km.client.CoreV1().Services(km.namespace).Get(context.Background(), "localstack", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get service: %w", err)
	}

	if len(service.Spec.Ports) == 0 {
		return "", fmt.Errorf("service has no ports")
	}

	return fmt.Sprintf("http://localstack.%s.svc.cluster.local:%d", km.namespace, service.Spec.Ports[0].Port), nil
}

// UpdateDeployment updates the LocalStack deployment with new configuration
func (km *kubernetesManager) UpdateDeployment(ctx context.Context, config *Config) error {
	// Get current deployment
	deployment, err := km.client.AppsV1().Deployments(km.namespace).Get(ctx, "localstack", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Update environment variables
	envVars := []corev1.EnvVar{}
	for k, v := range config.GetEnvironmentVars() {
		envVars = append(envVars, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	deployment.Spec.Template.Spec.Containers[0].Env = envVars

	// Update the deployment
	_, err = km.client.AppsV1().Deployments(km.namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	// Update configmap
	configMap, err := km.client.CoreV1().ConfigMaps(km.namespace).Get(ctx, "localstack-config", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get configmap: %w", err)
	}

	configMap.Data["services"] = config.GetServicesString()

	_, err = km.client.CoreV1().ConfigMaps(km.namespace).Update(ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update configmap: %w", err)
	}

	return nil
}