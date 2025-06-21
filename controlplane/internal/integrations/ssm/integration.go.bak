package ssm

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// integration implements the SSM Parameter Store integration
type integration struct {
	kubeClient        kubernetes.Interface
	ssmClient         SSMClient
	localStackManager localstack.Manager
	config            *Config
	cache             *parameterCache
	mu                sync.RWMutex
}

// parameterCache provides simple caching for parameter values
type parameterCache struct {
	entries map[string]*cacheEntry
	ttl     time.Duration
	mu      sync.RWMutex
}

type cacheEntry struct {
	parameter *Parameter
	expiry    time.Time
}

// NewIntegration creates a new SSM integration
func NewIntegration(kubeClient kubernetes.Interface, localStackManager localstack.Manager, cfg *Config) (Integration, error) {
	if cfg == nil {
		cfg = &Config{
			SecretPrefix:  DefaultSecretPrefix,
			KubeNamespace: DefaultKubeNamespace,
			SyncRetries:   DefaultSyncRetries,
			CacheTTL:      DefaultCacheTTL,
		}
	}

	// Create AWS config for LocalStack
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				if cfg.LocalStackEndpoint != "" {
					return aws.Endpoint{
						URL:               cfg.LocalStackEndpoint,
						HostnameImmutable: true,
					}, nil
				}
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			})),
		config.WithCredentialsProvider(aws.AnonymousCredentials{}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	// Create SSM client
	ssmClient := ssm.NewFromConfig(awsCfg)

	return &integration{
		kubeClient:        kubeClient,
		ssmClient:         ssmClient,
		localStackManager: localStackManager,
		config:            cfg,
		cache: &parameterCache{
			entries: make(map[string]*cacheEntry),
			ttl:     cfg.CacheTTL,
		},
	}, nil
}

// NewIntegrationWithClient creates a new SSM integration with custom clients (for testing)
func NewIntegrationWithClient(kubeClient kubernetes.Interface, ssmClient SSMClient, cfg *Config) Integration {
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
		ssmClient:  ssmClient,
		config:     cfg,
		cache: &parameterCache{
			entries: make(map[string]*cacheEntry),
			ttl:     cfg.CacheTTL,
		},
	}
}

// SyncParameter synchronizes a single SSM parameter to a Kubernetes secret
func (i *integration) SyncParameter(ctx context.Context, parameterName string, namespace string) error {
	if namespace == "" {
		namespace = i.config.KubeNamespace
	}

	// Get parameter from SSM
	parameter, err := i.GetParameter(ctx, parameterName)
	if err != nil {
		return fmt.Errorf("failed to get parameter %s: %w", parameterName, err)
	}

	// Create or update Kubernetes secret
	if err := i.CreateOrUpdateSecret(ctx, parameter, namespace); err != nil {
		return fmt.Errorf("failed to create/update secret for parameter %s: %w", parameterName, err)
	}

	return nil
}

// GetParameter retrieves a parameter value from LocalStack SSM
func (i *integration) GetParameter(ctx context.Context, parameterName string) (*Parameter, error) {
	// Check cache first
	if cached := i.cache.get(parameterName); cached != nil {
		return cached, nil
	}

	// Fetch from SSM
	input := &ssm.GetParameterInput{
		Name:           aws.String(parameterName),
		WithDecryption: aws.Bool(true),
	}

	result, err := i.ssmClient.GetParameter(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get parameter from SSM: %w", err)
	}

	if result.Parameter == nil {
		return nil, fmt.Errorf("parameter not found: %s", parameterName)
	}

	parameter := &Parameter{
		Name:         aws.ToString(result.Parameter.Name),
		Value:        aws.ToString(result.Parameter.Value),
		Type:         string(result.Parameter.Type),
		Version:      aws.ToInt64(&result.Parameter.Version),
		LastModified: aws.ToTime(result.Parameter.LastModifiedDate),
	}

	// Cache the parameter
	i.cache.set(parameterName, parameter)

	return parameter, nil
}

// CreateOrUpdateSecret creates or updates a Kubernetes secret from SSM parameter
func (i *integration) CreateOrUpdateSecret(ctx context.Context, parameter *Parameter, namespace string) error {
	secretName := i.GetSecretNameForParameter(parameter.Name)

	// Prepare secret data
	secretData := map[string][]byte{
		"value": []byte(parameter.Value),
	}

	// Prepare annotations
	annotations := map[string]string{
		SecretAnnotations.ParameterName:    parameter.Name,
		SecretAnnotations.ParameterVersion: strconv.FormatInt(parameter.Version, 10),
		SecretAnnotations.LastSynced:       time.Now().UTC().Format(time.RFC3339),
		SecretAnnotations.Source:           SourceSSM,
	}

	// Prepare labels
	labels := map[string]string{
		SecretLabels.ManagedBy: "kecs",
		SecretLabels.Source:    "ssm",
	}

	// Try to get existing secret
	existingSecret, err := i.kubeClient.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to check existing secret: %w", err)
		}

		// Create new secret
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        secretName,
				Namespace:   namespace,
				Labels:      labels,
				Annotations: annotations,
			},
			Type: corev1.SecretTypeOpaque,
			Data: secretData,
		}

		_, err = i.kubeClient.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create secret: %w", err)
		}

		log.Printf("Created Kubernetes secret %s/%s for SSM parameter %s", namespace, secretName, parameter.Name)
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

	_, err = i.kubeClient.CoreV1().Secrets(namespace).Update(ctx, existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	log.Printf("Updated Kubernetes secret %s/%s for SSM parameter %s", namespace, secretName, parameter.Name)
	return nil
}

// DeleteSecret deletes a synchronized secret
func (i *integration) DeleteSecret(ctx context.Context, parameterName, namespace string) error {
	secretName := i.GetSecretNameForParameter(parameterName)

	err := i.kubeClient.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	// Clear from cache
	i.cache.delete(parameterName)

	return nil
}

// SyncParameters batch synchronizes multiple parameters
func (i *integration) SyncParameters(ctx context.Context, parameters []string, namespace string) error {
	if namespace == "" {
		namespace = i.config.KubeNamespace
	}

	var syncErrors []error
	for _, paramName := range parameters {
		if err := i.SyncParameter(ctx, paramName, namespace); err != nil {
			syncErrors = append(syncErrors, fmt.Errorf("parameter %s: %w", paramName, err))
			// Continue with other parameters
		}
	}

	if len(syncErrors) > 0 {
		return fmt.Errorf("failed to sync %d parameters: %v", len(syncErrors), syncErrors)
	}

	return nil
}

// GetSecretNameForParameter returns the Kubernetes secret name for a given parameter
func (i *integration) GetSecretNameForParameter(parameterName string) string {
	// Remove leading slash if present
	name := strings.TrimPrefix(parameterName, "/")
	
	// Replace slashes and other non-alphanumeric characters with hyphens
	re := regexp.MustCompile(`[^a-zA-Z0-9-]+`)
	name = re.ReplaceAllString(name, "-")
	
	// Remove leading and trailing hyphens
	name = strings.Trim(name, "-")
	
	// Convert to lowercase
	name = strings.ToLower(name)
	
	// Add prefix
	return i.config.SecretPrefix + name
}

// Cache implementation

func (c *parameterCache) get(key string) *Parameter {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists || time.Now().After(entry.expiry) {
		return nil
	}
	return entry.parameter
}

func (c *parameterCache) set(key string, parameter *Parameter) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &cacheEntry{
		parameter: parameter,
		expiry:    time.Now().Add(c.ttl),
	}
}

func (c *parameterCache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}