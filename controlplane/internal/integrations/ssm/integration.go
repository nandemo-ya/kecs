package ssm

import (
	"context"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"k8s.io/client-go/kubernetes"
)

// Temporary implementation that returns not implemented errors
// This allows compilation while full migration to generated types is in progress

type tempIntegration struct{}

func (t *tempIntegration) SyncParameter(ctx context.Context, parameterName string, namespace string) error {
	return fmt.Errorf("SSM integration not yet migrated to generated types")
}

func (t *tempIntegration) GetParameter(ctx context.Context, parameterName string) (*Parameter, error) {
	return nil, fmt.Errorf("SSM integration not yet migrated to generated types")
}

func (t *tempIntegration) CreateOrUpdateSecret(ctx context.Context, parameter *Parameter, namespace string) error {
	return fmt.Errorf("SSM integration not yet migrated to generated types")
}

func (t *tempIntegration) DeleteSecret(ctx context.Context, parameterName, namespace string) error {
	return fmt.Errorf("SSM integration not yet migrated to generated types")
}

func (t *tempIntegration) SyncParameters(ctx context.Context, parameters []string, namespace string) error {
	return fmt.Errorf("SSM integration not yet migrated to generated types")
}

func (t *tempIntegration) GetSecretNameForParameter(parameterName string) string {
	return DefaultSecretPrefix + parameterName
}

// NewIntegration creates a new SSM integration instance
// TODO: Migrate to use generated types instead of AWS SDK v2
func NewIntegration(kubeClient kubernetes.Interface, localStackManager localstack.Manager, cfg *Config) (Integration, error) {
	return &tempIntegration{}, nil
}

// NewIntegrationWithClient creates a new SSM integration with custom client (for testing)
func NewIntegrationWithClient(kubeClient kubernetes.Interface, ssmClient SSMClient, cfg *Config) Integration {
	return &tempIntegration{}
}