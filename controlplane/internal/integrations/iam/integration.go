package iam

import (
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"k8s.io/client-go/kubernetes"
)

// Temporary implementation that returns not implemented errors
// This allows compilation while full migration to generated types is in progress

type tempIntegration struct{}

func (t *tempIntegration) CreateTaskRole(taskDefArn, roleName string, trustPolicy string) error {
	return fmt.Errorf("IAM integration not yet migrated to generated types")
}

func (t *tempIntegration) CreateTaskExecutionRole(roleName string) error {
	return fmt.Errorf("IAM integration not yet migrated to generated types")
}

func (t *tempIntegration) AttachPolicyToRole(roleName, policyArn string) error {
	return fmt.Errorf("IAM integration not yet migrated to generated types")
}

func (t *tempIntegration) CreateInlinePolicy(roleName, policyName, policyDocument string) error {
	return fmt.Errorf("IAM integration not yet migrated to generated types")
}

func (t *tempIntegration) DeleteRole(roleName string) error {
	return fmt.Errorf("IAM integration not yet migrated to generated types")
}

func (t *tempIntegration) GetServiceAccountForRole(roleName string) (string, error) {
	return "", fmt.Errorf("IAM integration not yet migrated to generated types")
}

func (t *tempIntegration) GetRoleCredentials(roleName string) (*Credentials, error) {
	return nil, fmt.Errorf("IAM integration not yet migrated to generated types")
}

// NewIntegration creates a new IAM integration instance
// TODO: Migrate to use generated types instead of AWS SDK v2
func NewIntegration(kubeClient kubernetes.Interface, localstackManager localstack.Manager, config *Config) (Integration, error) {
	return &tempIntegration{}, nil
}

// NewIntegrationWithClient creates a new IAM integration with custom clients (for testing)
func NewIntegrationWithClient(kubeClient kubernetes.Interface, iamClient IAMClient, stsClient STSClient, config *Config) Integration {
	return &tempIntegration{}
}