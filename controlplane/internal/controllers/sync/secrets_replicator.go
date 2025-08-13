package sync

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// SecretsReplicator replicates secrets from kecs-system to user namespaces as needed
type SecretsReplicator struct {
	kubeClient kubernetes.Interface
}

// NewSecretsReplicator creates a new secrets replicator
func NewSecretsReplicator(kubeClient kubernetes.Interface) *SecretsReplicator {
	return &SecretsReplicator{
		kubeClient: kubeClient,
	}
}

// ReplicateSecretToNamespace replicates a secret from kecs-system to target namespace
func (r *SecretsReplicator) ReplicateSecretToNamespace(ctx context.Context, secretName, targetNamespace string) error {
	// Get the secret from kecs-system
	sourceSecret, err := r.kubeClient.CoreV1().Secrets("kecs-system").Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret %s from kecs-system: %w", secretName, err)
	}

	// Create a copy for the target namespace
	targetSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: targetNamespace,
			Labels: map[string]string{
				"kecs.io/managed-by":      "kecs",
				"kecs.io/replicated-from": "kecs-system",
				"kecs.io/source":          sourceSecret.Labels["kecs.io/source"],
			},
			Annotations: map[string]string{
				"kecs.io/last-replicated":  time.Now().UTC().Format(time.RFC3339),
				"kecs.io/source-namespace": "kecs-system",
			},
		},
		Type: sourceSecret.Type,
		Data: sourceSecret.Data,
	}

	// Copy relevant annotations from source
	if sourceSecret.Annotations != nil {
		for k, v := range sourceSecret.Annotations {
			if strings.HasPrefix(k, "kecs.io/") {
				if targetSecret.Annotations == nil {
					targetSecret.Annotations = make(map[string]string)
				}
				targetSecret.Annotations[k] = v
			}
		}
	}

	// Check if secret already exists in target namespace
	existing, err := r.kubeClient.CoreV1().Secrets(targetNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new secret
			_, err = r.kubeClient.CoreV1().Secrets(targetNamespace).Create(ctx, targetSecret, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create secret %s in namespace %s: %w", secretName, targetNamespace, err)
			}
			logging.Info("Replicated secret to namespace", "secret", secretName, "namespace", targetNamespace)
			return nil
		}
		return fmt.Errorf("failed to check existing secret: %w", err)
	}

	// Update existing secret
	existing.Data = targetSecret.Data
	existing.Labels = targetSecret.Labels
	existing.Annotations = targetSecret.Annotations

	_, err = r.kubeClient.CoreV1().Secrets(targetNamespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update secret %s in namespace %s: %w", secretName, targetNamespace, err)
	}

	logging.Info("Updated replicated secret in namespace", "secret", secretName, "namespace", targetNamespace)
	return nil
}

// ReplicateConfigMapToNamespace replicates a ConfigMap from kecs-system to target namespace
// DEPRECATED: All SSM parameters are now stored as Secrets. This function is kept for backward compatibility.
func (r *SecretsReplicator) ReplicateConfigMapToNamespace(ctx context.Context, configMapName, targetNamespace string) error {
	// Get the ConfigMap from kecs-system
	sourceConfigMap, err := r.kubeClient.CoreV1().ConfigMaps("kecs-system").Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get configmap %s from kecs-system: %w", configMapName, err)
	}

	// Create a copy for the target namespace
	targetConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: targetNamespace,
			Labels: map[string]string{
				"kecs.io/managed-by":      "kecs",
				"kecs.io/replicated-from": "kecs-system",
				"kecs.io/source":          sourceConfigMap.Labels["kecs.io/source"],
			},
			Annotations: map[string]string{
				"kecs.io/last-replicated":  time.Now().UTC().Format(time.RFC3339),
				"kecs.io/source-namespace": "kecs-system",
			},
		},
		Data: sourceConfigMap.Data,
	}

	// Copy relevant annotations from source
	if sourceConfigMap.Annotations != nil {
		for k, v := range sourceConfigMap.Annotations {
			if strings.HasPrefix(k, "kecs.io/") {
				if targetConfigMap.Annotations == nil {
					targetConfigMap.Annotations = make(map[string]string)
				}
				targetConfigMap.Annotations[k] = v
			}
		}
	}

	// Check if ConfigMap already exists in target namespace
	existing, err := r.kubeClient.CoreV1().ConfigMaps(targetNamespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new ConfigMap
			_, err = r.kubeClient.CoreV1().ConfigMaps(targetNamespace).Create(ctx, targetConfigMap, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create configmap %s in namespace %s: %w", configMapName, targetNamespace, err)
			}
			logging.Info("Replicated configmap to namespace", "configmap", configMapName, "namespace", targetNamespace)
			return nil
		}
		return fmt.Errorf("failed to check existing configmap: %w", err)
	}

	// Update existing ConfigMap
	existing.Data = targetConfigMap.Data
	existing.Labels = targetConfigMap.Labels
	existing.Annotations = targetConfigMap.Annotations

	_, err = r.kubeClient.CoreV1().ConfigMaps(targetNamespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update configmap %s in namespace %s: %w", configMapName, targetNamespace, err)
	}

	logging.Info("Updated replicated configmap in namespace", "configmap", configMapName, "namespace", targetNamespace)
	return nil
}

// CleanupOrphanedReplicas removes replicated secrets/configmaps that are no longer in kecs-system
func (r *SecretsReplicator) CleanupOrphanedReplicas(ctx context.Context, namespace string) error {
	// List all replicated secrets in the namespace
	secrets, err := r.kubeClient.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "kecs.io/replicated-from=kecs-system",
	})
	if err != nil {
		return fmt.Errorf("failed to list replicated secrets: %w", err)
	}

	for _, secret := range secrets.Items {
		// Check if the source still exists in kecs-system
		_, err := r.kubeClient.CoreV1().Secrets("kecs-system").Get(ctx, secret.Name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// Source no longer exists, delete the replica
				err = r.kubeClient.CoreV1().Secrets(namespace).Delete(ctx, secret.Name, metav1.DeleteOptions{})
				if err != nil && !errors.IsNotFound(err) {
					logging.Error("Failed to delete orphaned secret replica", "secret", secret.Name, "namespace", namespace, "error", err)
				} else {
					logging.Info("Deleted orphaned secret replica", "secret", secret.Name, "namespace", namespace)
				}
			}
		}
	}

	// List all replicated ConfigMaps in the namespace
	configMaps, err := r.kubeClient.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "kecs.io/replicated-from=kecs-system",
	})
	if err != nil {
		return fmt.Errorf("failed to list replicated configmaps: %w", err)
	}

	for _, cm := range configMaps.Items {
		// Check if the source still exists in kecs-system
		_, err := r.kubeClient.CoreV1().ConfigMaps("kecs-system").Get(ctx, cm.Name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// Source no longer exists, delete the replica
				err = r.kubeClient.CoreV1().ConfigMaps(namespace).Delete(ctx, cm.Name, metav1.DeleteOptions{})
				if err != nil && !errors.IsNotFound(err) {
					logging.Error("Failed to delete orphaned configmap replica", "configmap", cm.Name, "namespace", namespace, "error", err)
				} else {
					logging.Info("Deleted orphaned configmap replica", "configmap", cm.Name, "namespace", namespace)
				}
			}
		}
	}

	return nil
}
