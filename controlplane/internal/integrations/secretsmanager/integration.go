package secretsmanager

import (
	"context"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"k8s.io/client-go/kubernetes"
)

// Temporary implementation that returns not implemented errors
// This allows compilation while full migration to generated types is in progress

type tempIntegration struct{}

func (t *tempIntegration) SyncSecret(ctx context.Context, secretName string, namespace string) error {
	return fmt.Errorf("Secrets Manager integration not yet migrated to generated types")
}

func (t *tempIntegration) GetSecret(ctx context.Context, secretName string) (*Secret, error) {
	return nil, fmt.Errorf("Secrets Manager integration not yet migrated to generated types")
}

func (t *tempIntegration) GetSecretWithKey(ctx context.Context, secretName, jsonKey string) (string, error) {
	return "", fmt.Errorf("Secrets Manager integration not yet migrated to generated types")
}

func (t *tempIntegration) CreateOrUpdateSecret(ctx context.Context, secret *Secret, jsonKey string, namespace string) error {
	return fmt.Errorf("Secrets Manager integration not yet migrated to generated types")
}

func (t *tempIntegration) DeleteSecret(ctx context.Context, secretName, namespace string) error {
	return fmt.Errorf("Secrets Manager integration not yet migrated to generated types")
}

func (t *tempIntegration) SyncSecrets(ctx context.Context, secrets []SecretReference, namespace string) error {
	return fmt.Errorf("Secrets Manager integration not yet migrated to generated types")
}

func (t *tempIntegration) GetSecretNameForSecret(secretName string) string {
	return DefaultSecretPrefix + secretName
}

// NewIntegration creates a new Secrets Manager integration instance
// TODO: Migrate to use generated types instead of AWS SDK v2
func NewIntegration(kubeClient kubernetes.Interface, localStackManager localstack.Manager, cfg *Config) (Integration, error) {
	return &tempIntegration{}, nil
}

// NewIntegrationWithClient creates a new Secrets Manager integration with custom clients (for testing)
func NewIntegrationWithClient(kubeClient kubernetes.Interface, smClient SecretsManagerClient, cfg *Config) Integration {
	return &tempIntegration{}
}