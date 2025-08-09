package sync

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/secretsmanager"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/ssm"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// SecretsController watches for pods with secret annotations and synchronizes secrets
type SecretsController struct {
	kubeClient    kubernetes.Interface
	smIntegration secretsmanager.Integration
	ssmIntegration ssm.Integration
	namespace     string
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// NewSecretsController creates a new secrets synchronization controller
func NewSecretsController(
	kubeClient kubernetes.Interface,
	smIntegration secretsmanager.Integration,
	ssmIntegration ssm.Integration,
	namespace string,
) *SecretsController {
	return &SecretsController{
		kubeClient:    kubeClient,
		smIntegration: smIntegration,
		ssmIntegration: ssmIntegration,
		namespace:     namespace,
		stopCh:        make(chan struct{}),
	}
}

// Start begins watching for pods and synchronizing secrets
func (c *SecretsController) Start(ctx context.Context) error {
	logging.Info("Starting secrets synchronization controller", "namespace", c.namespace)

	// Create pod informer
	podInformer := cache.NewSharedInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.LabelSelector = "kecs.dev/managed-by=kecs"
				return c.kubeClient.CoreV1().Pods(c.namespace).List(ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = "kecs.dev/managed-by=kecs"
				return c.kubeClient.CoreV1().Pods(c.namespace).Watch(ctx, options)
			},
		},
		&corev1.Pod{},
		time.Minute*5, // Resync period
	)

	// Add event handlers
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if pod, ok := obj.(*corev1.Pod); ok {
				c.handlePodCreate(ctx, pod)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if pod, ok := newObj.(*corev1.Pod); ok {
				c.handlePodUpdate(ctx, pod)
			}
		},
	})

	// Start the informer
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		podInformer.Run(c.stopCh)
	}()

	// Wait for cache sync
	if !cache.WaitForCacheSync(ctx.Done(), podInformer.HasSynced) {
		return fmt.Errorf("failed to sync pod cache")
	}

	logging.Info("Secrets synchronization controller started successfully")
	return nil
}

// Stop stops the controller
func (c *SecretsController) Stop() {
	logging.Info("Stopping secrets synchronization controller")
	close(c.stopCh)
	c.wg.Wait()
	logging.Info("Secrets synchronization controller stopped")
}

// handlePodCreate handles new pod creation
func (c *SecretsController) handlePodCreate(ctx context.Context, pod *corev1.Pod) {
	// Check if pod has secret annotations
	secretCount := c.getSecretCount(pod)
	if secretCount == 0 {
		return
	}

	logging.Info("Processing new pod with secrets", "pod", pod.Name, "secretCount", secretCount)
	
	// Extract and sync secrets
	if err := c.syncPodsSecrets(ctx, pod); err != nil {
		logging.Error("Failed to sync secrets for pod", "pod", pod.Name, "error", err)
	}
}

// handlePodUpdate handles pod updates
func (c *SecretsController) handlePodUpdate(ctx context.Context, pod *corev1.Pod) {
	// Only process if the pod is still pending (waiting for secrets)
	if pod.Status.Phase != corev1.PodPending {
		return
	}

	// Check if pod has secret annotations
	secretCount := c.getSecretCount(pod)
	if secretCount == 0 {
		return
	}

	// Check if secrets are already synced
	if c.areSecretssSynced(ctx, pod) {
		return
	}

	logging.Info("Re-syncing secrets for pending pod", "pod", pod.Name, "secretCount", secretCount)
	
	// Retry syncing secrets
	if err := c.syncPodsSecrets(ctx, pod); err != nil {
		logging.Error("Failed to sync secrets for pod", "pod", pod.Name, "error", err)
	}
}

// getSecretCount returns the number of secrets in the pod
func (c *SecretsController) getSecretCount(pod *corev1.Pod) int {
	if pod.Annotations == nil {
		return 0
	}

	countStr, exists := pod.Annotations["kecs.dev/secret-count"]
	if !exists {
		return 0
	}

	var count int
	fmt.Sscanf(countStr, "%d", &count)
	return count
}

// syncPodsSecrets synchronizes all secrets required by a pod
func (c *SecretsController) syncPodsSecrets(ctx context.Context, pod *corev1.Pod) error {
	secretCount := c.getSecretCount(pod)
	if secretCount == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, secretCount)

	// Process each secret annotation
	for i := 0; i < secretCount; i++ {
		annotationKey := fmt.Sprintf("kecs.dev/secret-%d-arn", i)
		annotationValue, exists := pod.Annotations[annotationKey]
		if !exists {
			continue
		}

		// Parse annotation value: containerName:envVarName:arn
		parts := strings.SplitN(annotationValue, ":", 3)
		if len(parts) != 3 {
			logging.Warn("Invalid secret annotation format", "annotation", annotationKey, "value", annotationValue)
			continue
		}

		containerName := parts[0]
		envVarName := parts[1]
		arn := parts[2]

		wg.Add(1)
		go func(container, envVar, secretArn string) {
			defer wg.Done()
			if err := c.syncSecret(ctx, secretArn, pod.Namespace); err != nil {
				errCh <- fmt.Errorf("failed to sync secret %s for container %s env %s: %w", 
					secretArn, container, envVar, err)
			}
		}(containerName, envVarName, arn)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var errs []string
	for err := range errCh {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to sync %d secrets: %s", len(errs), strings.Join(errs, "; "))
	}

	// Update pod containers to use the synced secrets
	return c.updatePodContainers(ctx, pod)
}

// syncSecret synchronizes a single secret based on its ARN
func (c *SecretsController) syncSecret(ctx context.Context, arn string, namespace string) error {
	// Parse ARN to determine service type
	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return fmt.Errorf("invalid ARN format: %s", arn)
	}

	service := parts[2]
	
	switch service {
	case "secretsmanager":
		return c.syncSecretsManagerSecret(ctx, arn, namespace)
	case "ssm":
		return c.syncSSMParameter(ctx, arn, namespace)
	default:
		return fmt.Errorf("unsupported secret service: %s", service)
	}
}

// syncSecretsManagerSecret syncs a Secrets Manager secret
func (c *SecretsController) syncSecretsManagerSecret(ctx context.Context, arn string, namespace string) error {
	// Extract secret name and key from ARN
	// Format: arn:aws:secretsmanager:region:account-id:secret:name-6RandomChars:key::
	parts := strings.Split(arn, ":")
	if len(parts) < 7 {
		return fmt.Errorf("invalid Secrets Manager ARN: %s", arn)
	}

	secretName := parts[6]
	jsonKey := ""
	if len(parts) > 7 && parts[7] != "" && parts[7] != "*" {
		jsonKey = parts[7]
	}

	// Get the secret from Secrets Manager
	secret, err := c.smIntegration.GetSecret(ctx, secretName)
	if err != nil {
		return fmt.Errorf("failed to get secret %s: %w", secretName, err)
	}

	// Create or update Kubernetes secret
	return c.smIntegration.CreateOrUpdateSecret(ctx, secret, jsonKey, namespace)
}

// syncSSMParameter syncs an SSM parameter
func (c *SecretsController) syncSSMParameter(ctx context.Context, arn string, namespace string) error {
	// Extract parameter name from ARN
	// Format: arn:aws:ssm:region:account-id:parameter/path/to/param
	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return fmt.Errorf("invalid SSM ARN: %s", arn)
	}

	resourcePart := parts[5]
	parameterName := ""
	
	if strings.HasPrefix(resourcePart, "parameter/") {
		parameterName = strings.TrimPrefix(resourcePart, "parameter/")
	} else if strings.HasPrefix(resourcePart, "parameter") && len(parts) > 6 {
		parameterName = parts[6]
	} else {
		parameterName = resourcePart
	}

	// Sync the parameter
	return c.ssmIntegration.SyncParameter(ctx, parameterName, namespace)
}

// areSecretssSynced checks if all secrets for a pod are already synced
func (c *SecretsController) areSecretssSynced(ctx context.Context, pod *corev1.Pod) bool {
	secretCount := c.getSecretCount(pod)
	
	for i := 0; i < secretCount; i++ {
		annotationKey := fmt.Sprintf("kecs.dev/secret-%d-arn", i)
		annotationValue, exists := pod.Annotations[annotationKey]
		if !exists {
			continue
		}

		// Parse annotation value
		parts := strings.SplitN(annotationValue, ":", 3)
		if len(parts) != 3 {
			continue
		}

		arn := parts[2]
		
		// Check if the corresponding Kubernetes secret exists
		secretName := c.getK8sSecretName(arn)
		_, err := c.kubeClient.CoreV1().Secrets(pod.Namespace).Get(ctx, secretName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return false // Secret doesn't exist yet
			}
			// Error checking secret, assume not synced
			return false
		}
	}

	return true
}

// getK8sSecretName returns the Kubernetes secret name for a given ARN
func (c *SecretsController) getK8sSecretName(arn string) string {
	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return ""
	}

	service := parts[2]
	
	switch service {
	case "secretsmanager":
		if len(parts) >= 7 {
			secretName := parts[6]
			return c.smIntegration.GetSecretNameForSecret(secretName)
		}
	case "ssm":
		resourcePart := parts[5]
		parameterName := ""
		if strings.HasPrefix(resourcePart, "parameter/") {
			parameterName = strings.TrimPrefix(resourcePart, "parameter/")
		} else if len(parts) > 6 {
			parameterName = parts[6]
		} else {
			parameterName = resourcePart
		}
		return c.ssmIntegration.GetSecretNameForParameter(parameterName)
	}

	return ""
}

// updatePodContainers updates pod containers to use the synced secrets
func (c *SecretsController) updatePodContainers(ctx context.Context, pod *corev1.Pod) error {
	// Parse secret annotations and build a map of container -> env vars
	containerEnvMap := make(map[string][]corev1.EnvVar)
	
	secretCount := c.getSecretCount(pod)
	for i := 0; i < secretCount; i++ {
		annotationKey := fmt.Sprintf("kecs.dev/secret-%d-arn", i)
		annotationValue, exists := pod.Annotations[annotationKey]
		if !exists {
			continue
		}

		// Parse annotation value: containerName:envVarName:arn
		parts := strings.SplitN(annotationValue, ":", 3)
		if len(parts) != 3 {
			continue
		}

		containerName := parts[0]
		envVarName := parts[1]
		arn := parts[2]

		// Get the Kubernetes secret name
		secretName := c.getK8sSecretName(arn)
		if secretName == "" {
			continue
		}

		// Determine the key within the secret
		key := "value" // Default key
		if service := c.getServiceFromARN(arn); service == "secretsmanager" {
			// Check if a specific JSON key is specified
			arnParts := strings.Split(arn, ":")
			if len(arnParts) > 7 && arnParts[7] != "" && arnParts[7] != "*" {
				key = arnParts[7]
			}
		}

		// Create environment variable referencing the secret
		envVar := corev1.EnvVar{
			Name: envVarName,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: key,
				},
			},
		}

		if _, exists := containerEnvMap[containerName]; !exists {
			containerEnvMap[containerName] = []corev1.EnvVar{}
		}
		containerEnvMap[containerName] = append(containerEnvMap[containerName], envVar)
	}

	// Note: We cannot directly update a running pod's spec.
	// The pod needs to be recreated with the updated environment variables.
	// This is typically handled by the deployment/service controller.
	// For now, we'll just log that secrets are ready.

	if len(containerEnvMap) > 0 {
		logging.Info("Secrets synchronized for pod", "pod", pod.Name, "containers", len(containerEnvMap))
		
		// Add an annotation to indicate secrets are ready
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations["kecs.dev/secrets-synced"] = "true"
		pod.Annotations["kecs.dev/secrets-synced-at"] = time.Now().UTC().Format(time.RFC3339)
		
		// Update the pod annotations
		_, err := c.kubeClient.CoreV1().Pods(pod.Namespace).Update(ctx, pod, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update pod annotations: %w", err)
		}
	}

	return nil
}

// getServiceFromARN extracts the service name from an ARN
func (c *SecretsController) getServiceFromARN(arn string) string {
	parts := strings.Split(arn, ":")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}