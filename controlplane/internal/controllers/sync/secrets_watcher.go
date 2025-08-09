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
	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/secretsmanager"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/ssm"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// SecretsWatcher periodically checks for changes in LocalStack secrets and synchronizes them
type SecretsWatcher struct {
	kubeClient     kubernetes.Interface
	smIntegration  secretsmanager.Integration
	ssmIntegration ssm.Integration
	syncInterval   time.Duration
	stopCh         chan struct{}
	wg             sync.WaitGroup
	
	// Track last sync time for each secret
	lastSyncMap    map[string]time.Time
	mu             sync.RWMutex
}

// NewSecretsWatcher creates a new secrets watcher
func NewSecretsWatcher(
	kubeClient kubernetes.Interface,
	smIntegration secretsmanager.Integration,
	ssmIntegration ssm.Integration,
	syncInterval time.Duration,
) *SecretsWatcher {
	if syncInterval == 0 {
		syncInterval = 30 * time.Second // Default sync interval
	}
	
	return &SecretsWatcher{
		kubeClient:     kubeClient,
		smIntegration:  smIntegration,
		ssmIntegration: ssmIntegration,
		syncInterval:   syncInterval,
		stopCh:         make(chan struct{}),
		lastSyncMap:    make(map[string]time.Time),
	}
}

// Start begins watching for secret changes
func (w *SecretsWatcher) Start(ctx context.Context) error {
	logging.Info("Starting secrets watcher", "syncInterval", w.syncInterval)

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.watchLoop(ctx)
	}()

	return nil
}

// Stop stops the watcher
func (w *SecretsWatcher) Stop() {
	logging.Info("Stopping secrets watcher")
	close(w.stopCh)
	w.wg.Wait()
	logging.Info("Secrets watcher stopped")
}

// watchLoop is the main watch loop
func (w *SecretsWatcher) watchLoop(ctx context.Context) {
	ticker := time.NewTicker(w.syncInterval)
	defer ticker.Stop()

	// Initial sync
	w.syncAllSecrets(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.syncAllSecrets(ctx)
		}
	}
}

// syncAllSecrets synchronizes all secrets across all namespaces
func (w *SecretsWatcher) syncAllSecrets(ctx context.Context) {
	// List all namespaces that are ECS clusters (format: <cluster-name>-<region>)
	namespaces, err := w.kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: "kecs.dev/cluster-namespace=true",
	})
	if err != nil {
		logging.Error("Failed to list namespaces", "error", err)
		return
	}

	for _, ns := range namespaces.Items {
		namespace := ns.Name
		clusterName, region := w.parseNamespace(namespace)
		if clusterName == "" || region == "" {
			continue
		}

		// Sync secrets for this namespace
		w.syncNamespaceSecrets(ctx, namespace, clusterName, region)
	}
}

// parseNamespace extracts cluster name and region from namespace name
func (w *SecretsWatcher) parseNamespace(namespace string) (clusterName, region string) {
	// Namespace format: <cluster-name>-<region>
	// Example: default-us-east-1
	lastDash := strings.LastIndex(namespace, "-")
	if lastDash == -1 {
		return "", ""
	}
	
	// Check if the last part looks like a region
	possibleRegion := namespace[lastDash+1:]
	if strings.Contains(possibleRegion, "-") {
		// Complex region like us-east-1
		// Find the second to last dash
		beforeLastDash := strings.LastIndex(namespace[:lastDash], "-")
		if beforeLastDash != -1 {
			clusterName = namespace[:beforeLastDash]
			region = namespace[beforeLastDash+1:]
		}
	} else {
		// Simple format
		clusterName = namespace[:lastDash]
		region = possibleRegion
	}
	
	return clusterName, region
}

// syncNamespaceSecrets synchronizes secrets for a specific namespace
func (w *SecretsWatcher) syncNamespaceSecrets(ctx context.Context, namespace, clusterName, region string) {
	// List all K8s secrets and configmaps managed by KECS in this namespace
	secrets, err := w.kubeClient.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "kecs.io/managed-by=kecs",
	})
	if err != nil {
		logging.Error("Failed to list secrets", "namespace", namespace, "error", err)
		return
	}

	configMaps, err := w.kubeClient.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "kecs.io/managed-by=kecs",
	})
	if err != nil {
		logging.Error("Failed to list configmaps", "namespace", namespace, "error", err)
		return
	}

	// Check each secret for updates
	for _, secret := range secrets.Items {
		source := secret.Labels["kecs.io/source"]
		switch source {
		case "secretsmanager":
			w.syncSecretsManagerSecret(ctx, &secret, namespace, clusterName)
		case "ssm":
			w.syncSSMSecret(ctx, &secret, namespace, clusterName)
		}
	}

	// Check each configmap for updates
	for _, cm := range configMaps.Items {
		source := cm.Labels["kecs.io/source"]
		if source == "ssm" {
			w.syncSSMConfigMap(ctx, &cm, namespace, clusterName)
		}
	}
}

// syncSecretsManagerSecret checks and syncs a Secrets Manager secret
func (w *SecretsWatcher) syncSecretsManagerSecret(ctx context.Context, k8sSecret *corev1.Secret, namespace, clusterName string) {
	annotations := k8sSecret.GetAnnotations()
	originalSecretName := annotations["kecs.io/secret-name"]
	if originalSecretName == "" {
		return
	}

	// Build the namespace-aware secret name in LocalStack
	// Format: <cluster>/<namespace>/<secret-name>
	localStackSecretName := fmt.Sprintf("%s/%s/%s", clusterName, namespace, originalSecretName)

	// Check if the secret exists and has been updated in LocalStack
	secret, err := w.smIntegration.GetSecret(ctx, localStackSecretName)
	if err != nil {
		if isNotFoundError(err) {
			// Secret was deleted in LocalStack, delete from K8s
			logging.Info("Secret deleted in LocalStack, removing from K8s", 
				"namespace", namespace, 
				"secret", k8sSecret.GetName(),
				"localStackSecret", localStackSecretName)
			
			err = w.kubeClient.CoreV1().Secrets(namespace).Delete(ctx, k8sSecret.GetName(), metav1.DeleteOptions{})
			if err != nil && !errors.IsNotFound(err) {
				logging.Error("Failed to delete secret", "namespace", namespace, "secret", (*k8sSecret).GetName(), "error", err)
			}
			return
		}
		logging.Error("Failed to get secret from Secrets Manager", "secret", localStackSecretName, "error", err)
		return
	}

	// Check if update is needed
	lastSyncedStr := annotations["kecs.io/sm-last-synced"]
	if lastSyncedStr != "" {
		lastSynced, _ := time.Parse(time.RFC3339, lastSyncedStr)
		if !secret.CreatedDate.After(lastSynced) {
			// No update needed
			return
		}
	}

	// Update the secret
	logging.Info("Updating secret from Secrets Manager", 
		"namespace", namespace, 
		"k8sSecret", k8sSecret.GetName(),
		"localStackSecret", localStackSecretName)
	
	// Extract JSON key if specified
	jsonKey := annotations["kecs.io/sm-json-key"]
	err = w.smIntegration.CreateOrUpdateSecret(ctx, secret, jsonKey, namespace)
	if err != nil {
		logging.Error("Failed to update secret", "namespace", namespace, "secret", k8sSecret.GetName(), "error", err)
	}
}

// syncSSMSecret checks and syncs an SSM parameter stored as a secret
func (w *SecretsWatcher) syncSSMSecret(ctx context.Context, k8sSecret *corev1.Secret, namespace, clusterName string) {
	annotations := k8sSecret.GetAnnotations()
	originalParamName := annotations["kecs.io/ssm-parameter-name"]
	if originalParamName == "" {
		return
	}

	// Build the namespace-aware parameter name in LocalStack
	// Format: /<cluster>/<namespace>/<param-path>
	localStackParamName := fmt.Sprintf("/%s/%s/%s", clusterName, namespace, strings.TrimPrefix(originalParamName, "/"))

	// Check if the parameter exists and has been updated in LocalStack
	parameter, err := w.ssmIntegration.GetParameter(ctx, localStackParamName)
	if err != nil {
		if isNotFoundError(err) {
			// Parameter was deleted in LocalStack, delete from K8s
			logging.Info("Parameter deleted in LocalStack, removing secret from K8s", 
				"namespace", namespace, 
				"secret", k8sSecret.GetName(),
				"localStackParam", localStackParamName)
			
			err = w.kubeClient.CoreV1().Secrets(namespace).Delete(ctx, k8sSecret.GetName(), metav1.DeleteOptions{})
			if err != nil && !errors.IsNotFound(err) {
				logging.Error("Failed to delete secret", "namespace", namespace, "secret", (*k8sSecret).GetName(), "error", err)
			}
			return
		}
		logging.Error("Failed to get parameter from SSM", "parameter", localStackParamName, "error", err)
		return
	}

	// Check if update is needed
	lastSyncedStr := annotations["kecs.io/ssm-last-synced"]
	if lastSyncedStr != "" {
		lastSynced, _ := time.Parse(time.RFC3339, lastSyncedStr)
		if !parameter.LastModified.After(lastSynced) {
			// No update needed
			return
		}
	}

	// Update the secret
	logging.Info("Updating secret from SSM parameter", 
		"namespace", namespace, 
		"k8sSecret", k8sSecret.GetName(),
		"localStackParam", localStackParamName)
	
	err = w.ssmIntegration.CreateOrUpdateSecret(ctx, parameter, namespace)
	if err != nil {
		logging.Error("Failed to update secret", "namespace", namespace, "secret", k8sSecret.GetName(), "error", err)
	}
}

// syncSSMConfigMap checks and syncs an SSM parameter stored as a ConfigMap
func (w *SecretsWatcher) syncSSMConfigMap(ctx context.Context, k8sCM *corev1.ConfigMap, namespace, clusterName string) {
	annotations := k8sCM.GetAnnotations()
	originalParamName := annotations["kecs.io/ssm-parameter-name"]
	if originalParamName == "" {
		return
	}

	// Build the namespace-aware parameter name in LocalStack
	// Format: /<cluster>/<namespace>/<param-path>
	localStackParamName := fmt.Sprintf("/%s/%s/%s", clusterName, namespace, strings.TrimPrefix(originalParamName, "/"))

	// Check if the parameter exists and has been updated in LocalStack
	parameter, err := w.ssmIntegration.GetParameter(ctx, localStackParamName)
	if err != nil {
		if isNotFoundError(err) {
			// Parameter was deleted in LocalStack, delete from K8s
			logging.Info("Parameter deleted in LocalStack, removing ConfigMap from K8s", 
				"namespace", namespace, 
				"configmap", k8sCM.GetName(),
				"localStackParam", localStackParamName)
			
			err = w.kubeClient.CoreV1().ConfigMaps(namespace).Delete(ctx, k8sCM.GetName(), metav1.DeleteOptions{})
			if err != nil && !errors.IsNotFound(err) {
				logging.Error("Failed to delete ConfigMap", "namespace", namespace, "configmap", (*k8sCM).GetName(), "error", err)
			}
			return
		}
		logging.Error("Failed to get parameter from SSM", "parameter", localStackParamName, "error", err)
		return
	}

	// Check if update is needed
	lastSyncedStr := annotations["kecs.io/ssm-last-synced"]
	if lastSyncedStr != "" {
		lastSynced, _ := time.Parse(time.RFC3339, lastSyncedStr)
		if !parameter.LastModified.After(lastSynced) {
			// No update needed
			return
		}
	}

	// Update the ConfigMap
	logging.Info("Updating ConfigMap from SSM parameter", 
		"namespace", namespace, 
		"configMap", k8sCM.GetName(),
		"localStackParam", localStackParamName)
	
	err = w.ssmIntegration.CreateOrUpdateConfigMap(ctx, parameter, namespace)
	if err != nil {
		logging.Error("Failed to update ConfigMap", "namespace", namespace, "configmap", (*k8sCM).GetName(), "error", err)
	}
}

// isNotFoundError checks if the error indicates a resource was not found
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check for common "not found" error patterns
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "not found") || 
		strings.Contains(errStr, "does not exist") ||
		strings.Contains(errStr, "resourcenotfoundexception")
}