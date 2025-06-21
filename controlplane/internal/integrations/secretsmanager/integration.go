package secretsmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	secretsmanagerapi "github.com/nandemo-ya/kecs/controlplane/internal/secretsmanager/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// integration implements the Secrets Manager integration
type integration struct {
	kubeClient        kubernetes.Interface
	smClient          SecretsManagerClient
	localStackManager localstack.Manager
	config            *Config
	cache             *secretCache
	mu                sync.RWMutex
}

// secretCache provides simple caching for secret values
type secretCache struct {
	entries map[string]*cacheEntry
	ttl     time.Duration
	mu      sync.RWMutex
}

type cacheEntry struct {
	secret *Secret
	expiry time.Time
}

// NewIntegration creates a new Secrets Manager integration
func NewIntegration(kubeClient kubernetes.Interface, localStackManager localstack.Manager, cfg *Config) (Integration, error) {
	if cfg == nil {
		cfg = &Config{
			SecretPrefix:  DefaultSecretPrefix,
			KubeNamespace: DefaultKubeNamespace,
			SyncRetries:   DefaultSyncRetries,
			CacheTTL:      DefaultCacheTTL,
		}
	}

	// Create Secrets Manager client configured for LocalStack
	endpoint := cfg.LocalStackEndpoint
	if endpoint == "" {
		endpoint = "http://localhost:4566"
	}
	
	smClient := newSecretsManagerClient(endpoint)

	return &integration{
		kubeClient:        kubeClient,
		smClient:          smClient,
		localStackManager: localStackManager,
		config:            cfg,
		cache: &secretCache{
			entries: make(map[string]*cacheEntry),
			ttl:     cfg.CacheTTL,
		},
	}, nil
}

// NewIntegrationWithClient creates a new Secrets Manager integration with custom clients (for testing)
func NewIntegrationWithClient(kubeClient kubernetes.Interface, smClient SecretsManagerClient, cfg *Config) Integration {
	if cfg == nil {
		cfg = &Config{
			SecretPrefix:  DefaultSecretPrefix,
			KubeNamespace: DefaultKubeNamespace,
			SyncRetries:   DefaultSyncRetries,
			CacheTTL:      DefaultCacheTTL,
		}
	}

	return &integration{
		kubeClient: kubeClient,
		smClient:   smClient,
		config:     cfg,
		cache: &secretCache{
			entries: make(map[string]*cacheEntry),
			ttl:     cfg.CacheTTL,
		},
	}
}

// SyncSecret synchronizes a single secret to a Kubernetes secret
func (i *integration) SyncSecret(ctx context.Context, secretName string, namespace string) error {
	if namespace == "" {
		namespace = i.config.KubeNamespace
	}

	// Get secret from Secrets Manager
	secret, err := i.GetSecret(ctx, secretName)
	if err != nil {
		return fmt.Errorf("failed to get secret %s: %w", secretName, err)
	}

	// Create or update Kubernetes secret
	if err := i.CreateOrUpdateSecret(ctx, secret, "", namespace); err != nil {
		return fmt.Errorf("failed to create/update secret for %s: %w", secretName, err)
	}

	return nil
}

// GetSecret retrieves a secret value from LocalStack Secrets Manager
func (i *integration) GetSecret(ctx context.Context, secretName string) (*Secret, error) {
	// Check cache first
	if cached := i.cache.get(secretName); cached != nil {
		return cached, nil
	}

	// Fetch from Secrets Manager
	input := &secretsmanagerapi.GetSecretValueRequest{
		SecretId: secretName,
	}

	result, err := i.smClient.GetSecretValue(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret from Secrets Manager: %w", err)
	}

	secret := &Secret{
		Name:         getString(result.Name),
		VersionId:    getString(result.VersionId),
		VersionStage: result.VersionStages,
		CreatedDate:  getTime(result.CreatedDate),
	}

	// Handle string vs binary secret
	if result.SecretString != nil {
		secret.Value = *result.SecretString
		secret.Type = "String"
	} else if result.SecretBinary != nil {
		// For binary secrets, we'll store as base64 encoded string
		secret.Value = string(result.SecretBinary)
		secret.Type = "Binary"
	}

	// Cache the secret
	i.cache.set(secretName, secret)

	return secret, nil
}

// GetSecretWithKey retrieves a specific JSON key from a secret
func (i *integration) GetSecretWithKey(ctx context.Context, secretName, jsonKey string) (string, error) {
	secret, err := i.GetSecret(ctx, secretName)
	if err != nil {
		return "", err
	}

	// If no JSON key specified, return the whole value
	if jsonKey == "" || jsonKey == "default" {
		return secret.Value, nil
	}

	// Parse JSON and extract key
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(secret.Value), &data); err != nil {
		return "", fmt.Errorf("failed to parse secret as JSON: %w", err)
	}

	value, exists := data[jsonKey]
	if !exists {
		return "", fmt.Errorf("key %s not found in secret", jsonKey)
	}

	// Convert value to string
	switch v := value.(type) {
	case string:
		return v, nil
	default:
		// For non-string values, marshal back to JSON
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal value for key %s: %w", jsonKey, err)
		}
		return string(jsonBytes), nil
	}
}

// CreateOrUpdateSecret creates or updates a Kubernetes secret from Secrets Manager
func (i *integration) CreateOrUpdateSecret(ctx context.Context, secret *Secret, jsonKey string, namespace string) error {
	secretName := i.GetSecretNameForSecret(secret.Name)

	// Prepare secret data
	secretData := make(map[string][]byte)
	
	if jsonKey != "" && jsonKey != "default" {
		// Extract specific JSON key
		value, err := i.extractJSONKey(secret.Value, jsonKey)
		if err != nil {
			return fmt.Errorf("failed to extract JSON key %s: %w", jsonKey, err)
		}
		secretData[jsonKey] = []byte(value)
	} else {
		// Store entire secret value
		secretData["value"] = []byte(secret.Value)
	}

	// Prepare annotations
	annotations := map[string]string{
		SecretAnnotations.SecretName:   secret.Name,
		SecretAnnotations.VersionId:    secret.VersionId,
		SecretAnnotations.LastSynced:   time.Now().UTC().Format(time.RFC3339),
		SecretAnnotations.Source:       SourceSecretsManager,
	}
	
	if len(secret.VersionStage) > 0 {
		annotations[SecretAnnotations.VersionStage] = strings.Join(secret.VersionStage, ",")
	}
	
	if jsonKey != "" {
		annotations[SecretAnnotations.JSONKey] = jsonKey
	}

	// Prepare labels
	labels := map[string]string{
		SecretLabels.ManagedBy: "kecs",
		SecretLabels.Source:    "secretsmanager",
	}

	// Try to get existing secret
	existingSecret, err := i.kubeClient.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to check existing secret: %w", err)
		}

		// Create new secret
		k8sSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        secretName,
				Namespace:   namespace,
				Labels:      labels,
				Annotations: annotations,
			},
			Type: corev1.SecretTypeOpaque,
			Data: secretData,
		}

		if _, err := i.kubeClient.CoreV1().Secrets(namespace).Create(ctx, k8sSecret, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create secret: %w", err)
		}

		log.Printf("Created Kubernetes secret %s/%s for Secrets Manager secret %s", namespace, secretName, secret.Name)
		return nil
	}

	// Update existing secret
	existingSecret.Data = secretData
	if existingSecret.Annotations == nil {
		existingSecret.Annotations = make(map[string]string)
	}
	for k, v := range annotations {
		existingSecret.Annotations[k] = v
	}
	if existingSecret.Labels == nil {
		existingSecret.Labels = make(map[string]string)
	}
	for k, v := range labels {
		existingSecret.Labels[k] = v
	}

	if _, err := i.kubeClient.CoreV1().Secrets(namespace).Update(ctx, existingSecret, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	log.Printf("Updated Kubernetes secret %s/%s for Secrets Manager secret %s", namespace, secretName, secret.Name)
	return nil
}

// DeleteSecret deletes a synchronized secret
func (i *integration) DeleteSecret(ctx context.Context, secretName, namespace string) error {
	if namespace == "" {
		namespace = i.config.KubeNamespace
	}

	k8sSecretName := i.GetSecretNameForSecret(secretName)

	err := i.kubeClient.CoreV1().Secrets(namespace).Delete(ctx, k8sSecretName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	// Clear from cache
	i.cache.delete(secretName)

	log.Printf("Deleted Kubernetes secret %s/%s for Secrets Manager secret %s", namespace, k8sSecretName, secretName)
	return nil
}

// SyncSecrets batch synchronizes multiple secrets
func (i *integration) SyncSecrets(ctx context.Context, secrets []SecretReference, namespace string) error {
	if namespace == "" {
		namespace = i.config.KubeNamespace
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(secrets))

	for _, secretRef := range secrets {
		wg.Add(1)
		go func(ref SecretReference) {
			defer wg.Done()
			
			// Get the secret
			secret, err := i.GetSecret(ctx, ref.SecretName)
			if err != nil {
				errCh <- fmt.Errorf("secret %s: %w", ref.SecretName, err)
				return
			}

			// Create or update the Kubernetes secret
			if err := i.CreateOrUpdateSecret(ctx, secret, ref.JSONKey, namespace); err != nil {
				errCh <- fmt.Errorf("secret %s: %w", ref.SecretName, err)
			}
		}(secretRef)
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

	return nil
}

// GetSecretNameForSecret returns the Kubernetes secret name for a given Secrets Manager secret
func (i *integration) GetSecretNameForSecret(secretName string) string {
	// Remove the random suffix that Secrets Manager adds (e.g., -AbCdEf)
	re := regexp.MustCompile(`-[A-Za-z0-9]{6}$`)
	cleanName := re.ReplaceAllString(secretName, "")
	
	// Replace slashes and other non-alphanumeric characters with hyphens
	re = regexp.MustCompile(`[^a-zA-Z0-9\-]`)
	cleanName = re.ReplaceAllString(cleanName, "-")
	
	// Remove consecutive hyphens
	re = regexp.MustCompile(`-+`)
	cleanName = re.ReplaceAllString(cleanName, "-")
	
	// Remove leading and trailing hyphens
	cleanName = strings.Trim(cleanName, "-")
	
	// Convert to lowercase
	cleanName = strings.ToLower(cleanName)
	
	// Add prefix
	return i.config.SecretPrefix + cleanName
}

// extractJSONKey extracts a specific key from a JSON string
func (i *integration) extractJSONKey(jsonStr, key string) (string, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	value, exists := data[key]
	if !exists {
		return "", fmt.Errorf("key %s not found in JSON", key)
	}

	// Convert value to string
	switch v := value.(type) {
	case string:
		return v, nil
	default:
		// For non-string values, marshal back to JSON
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal value: %w", err)
		}
		return string(jsonBytes), nil
	}
}

// Cache methods
func (c *secretCache) get(key string) *Secret {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists || time.Now().After(entry.expiry) {
		return nil
	}
	return entry.secret
}

func (c *secretCache) set(key string, secret *Secret) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &cacheEntry{
		secret: secret,
		expiry: time.Now().Add(c.ttl),
	}
}

func (c *secretCache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

// Helper functions
func getString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}