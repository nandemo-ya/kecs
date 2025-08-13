package ssm

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// DEPRECATED: All SSM parameters are now stored as Secrets for consistency and simplicity.
// This code is kept for backward compatibility but is no longer used.
//
// CreateOrUpdateConfigMap creates or updates a Kubernetes ConfigMap from SSM parameter
// This is useful for non-sensitive configuration data
func (i *integration) CreateOrUpdateConfigMap(ctx context.Context, parameter *Parameter, namespace string) error {
	configMapName := i.GetConfigMapNameForParameter(parameter.Name)

	// Prepare configmap data
	configMapData := map[string]string{
		"value": parameter.Value,
	}

	// Add additional keys if the value is JSON
	if strings.HasPrefix(parameter.Value, "{") && strings.HasSuffix(parameter.Value, "}") {
		// Attempt to parse as JSON and flatten
		flatData := i.flattenJSON(parameter.Value)
		for k, v := range flatData {
			configMapData[k] = v
		}
	}

	// Prepare annotations
	annotations := map[string]string{
		ConfigMapAnnotations.ParameterName:    parameter.Name,
		ConfigMapAnnotations.ParameterVersion: strconv.FormatInt(parameter.Version, 10),
		ConfigMapAnnotations.LastSynced:       time.Now().UTC().Format(time.RFC3339),
		ConfigMapAnnotations.Source:           SourceSSM,
	}

	// Prepare labels
	labels := map[string]string{
		ConfigMapLabels.ManagedBy: "kecs",
		ConfigMapLabels.Source:    "ssm",
	}

	// Try to get existing configmap
	existingConfigMap, err := i.kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to check existing configmap: %w", err)
		}

		// Create new configmap
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:        configMapName,
				Namespace:   namespace,
				Labels:      labels,
				Annotations: annotations,
			},
			Data: configMapData,
		}

		if _, err := i.kubeClient.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create configmap: %w", err)
		}

		logging.Info("Created Kubernetes ConfigMap for SSM parameter", "namespace", namespace, "configMap", configMapName, "parameter", parameter.Name)
		return nil
	}

	// Update existing configmap
	existingConfigMap.Data = configMapData
	if existingConfigMap.Annotations == nil {
		existingConfigMap.Annotations = make(map[string]string)
	}
	for k, v := range annotations {
		existingConfigMap.Annotations[k] = v
	}
	if existingConfigMap.Labels == nil {
		existingConfigMap.Labels = make(map[string]string)
	}
	for k, v := range labels {
		existingConfigMap.Labels[k] = v
	}

	if _, err := i.kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, existingConfigMap, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update configmap: %w", err)
	}

	logging.Info("Updated Kubernetes ConfigMap for SSM parameter", "namespace", namespace, "configMap", configMapName, "parameter", parameter.Name)
	return nil
}

// GetConfigMapNameForParameter returns the Kubernetes ConfigMap name for a given parameter
func (i *integration) GetConfigMapNameForParameter(parameterName string) string {
	// Use similar logic to GetSecretNameForParameter but with "cm-" prefix for ConfigMaps
	cleanName := strings.Trim(parameterName, "/")
	cleanName = strings.ReplaceAll(cleanName, "/", "-")
	cleanName = strings.ToLower(cleanName)

	// Remove any non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, ch := range cleanName {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			result.WriteRune(ch)
		} else {
			result.WriteRune('-')
		}
	}

	cleanName = result.String()
	// Remove consecutive hyphens
	for strings.Contains(cleanName, "--") {
		cleanName = strings.ReplaceAll(cleanName, "--", "-")
	}
	cleanName = strings.Trim(cleanName, "-")

	// Use "ssm-cm-" prefix for ConfigMaps to distinguish from Secrets
	return "ssm-cm-" + cleanName
}

// SyncParameterAsConfigMap synchronizes a single SSM parameter to a Kubernetes ConfigMap
func (i *integration) SyncParameterAsConfigMap(ctx context.Context, parameterName string, namespace string) error {
	if namespace == "" {
		namespace = i.config.KubeNamespace
	}

	// Get parameter from SSM
	parameter, err := i.GetParameter(ctx, parameterName)
	if err != nil {
		return fmt.Errorf("failed to get parameter %s: %w", parameterName, err)
	}

	// Create or update Kubernetes ConfigMap
	if err := i.CreateOrUpdateConfigMap(ctx, parameter, namespace); err != nil {
		return fmt.Errorf("failed to create/update configmap for parameter %s: %w", parameterName, err)
	}

	return nil
}

// DeleteConfigMap deletes a synchronized ConfigMap
func (i *integration) DeleteConfigMap(ctx context.Context, parameterName, namespace string) error {
	if namespace == "" {
		namespace = i.config.KubeNamespace
	}

	configMapName := i.GetConfigMapNameForParameter(parameterName)

	err := i.kubeClient.CoreV1().ConfigMaps(namespace).Delete(ctx, configMapName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete configmap: %w", err)
	}

	logging.Info("Deleted Kubernetes ConfigMap for SSM parameter", "namespace", namespace, "configMap", configMapName, "parameter", parameterName)
	return nil
}

// flattenJSON attempts to flatten a JSON string into key-value pairs
func (i *integration) flattenJSON(jsonStr string) map[string]string {
	result := make(map[string]string)

	// Simple JSON flattening - in production, use a proper JSON parser
	// This is a placeholder implementation
	// TODO: Implement proper JSON flattening

	return result
}
