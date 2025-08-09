package ssm

import (
	"context"
	"time"

	ssmapi "github.com/nandemo-ya/kecs/controlplane/internal/ssm/generated"
)

// Integration represents the SSM Parameter Store integration
type Integration interface {
	// SyncParameter synchronizes a single SSM parameter to a Kubernetes secret
	SyncParameter(ctx context.Context, parameterName string, namespace string) error

	// GetParameter retrieves a parameter value from LocalStack SSM
	GetParameter(ctx context.Context, parameterName string) (*Parameter, error)

	// CreateOrUpdateSecret creates or updates a Kubernetes secret from SSM parameter
	CreateOrUpdateSecret(ctx context.Context, parameter *Parameter, namespace string) error

	// DeleteSecret deletes a synchronized secret
	DeleteSecret(ctx context.Context, parameterName, namespace string) error

	// SyncParameters batch synchronizes multiple parameters
	SyncParameters(ctx context.Context, parameters []string, namespace string) error

	// GetSecretNameForParameter returns the Kubernetes secret name for a given parameter
	GetSecretNameForParameter(parameterName string) string

	// ConfigMap methods for non-sensitive configuration
	// SyncParameterAsConfigMap synchronizes a single SSM parameter to a Kubernetes ConfigMap
	SyncParameterAsConfigMap(ctx context.Context, parameterName string, namespace string) error

	// CreateOrUpdateConfigMap creates or updates a Kubernetes ConfigMap from SSM parameter
	CreateOrUpdateConfigMap(ctx context.Context, parameter *Parameter, namespace string) error

	// DeleteConfigMap deletes a synchronized ConfigMap
	DeleteConfigMap(ctx context.Context, parameterName, namespace string) error

	// GetConfigMapNameForParameter returns the Kubernetes ConfigMap name for a given parameter
	GetConfigMapNameForParameter(parameterName string) string
}

// Parameter represents an SSM parameter
type Parameter struct {
	Name         string
	Value        string
	Type         string // String, StringList, SecureString
	Version      int64
	LastModified time.Time
}

// Config represents SSM integration configuration
type Config struct {
	LocalStackEndpoint string
	SecretPrefix       string        // Prefix for created secrets (e.g., "ssm-")
	KubeNamespace      string        // Default namespace for secrets
	SyncRetries        int           // Number of retries for sync operations
	CacheTTL           time.Duration // Cache duration for parameter values
}

// SecretAnnotations defines annotations added to Kubernetes secrets
var SecretAnnotations = struct {
	ParameterName    string
	ParameterVersion string
	LastSynced       string
	Source           string
}{
	ParameterName:    "kecs.io/ssm-parameter-name",
	ParameterVersion: "kecs.io/ssm-parameter-version",
	LastSynced:       "kecs.io/ssm-last-synced",
	Source:           "kecs.io/secret-source",
}

// SecretLabels defines labels added to Kubernetes secrets
var SecretLabels = struct {
	ManagedBy string
	Source    string
}{
	ManagedBy: "kecs.io/managed-by",
	Source:    "kecs.io/source",
}

// ConfigMapAnnotations defines annotations added to Kubernetes ConfigMaps
var ConfigMapAnnotations = struct {
	ParameterName    string
	ParameterVersion string
	LastSynced       string
	Source           string
}{
	ParameterName:    "kecs.io/ssm-parameter-name",
	ParameterVersion: "kecs.io/ssm-parameter-version",
	LastSynced:       "kecs.io/ssm-last-synced",
	Source:           "kecs.io/config-source",
}

// ConfigMapLabels defines labels added to Kubernetes ConfigMaps
var ConfigMapLabels = struct {
	ManagedBy string
	Source    string
}{
	ManagedBy: "kecs.io/managed-by",
	Source:    "kecs.io/source",
}

// Default configuration values
const (
	DefaultSecretPrefix  = "ssm-"
	DefaultKubeNamespace = "default"
	DefaultSyncRetries   = 3
	DefaultCacheTTL      = 5 * time.Minute
	SourceSSM            = "ssm-parameter-store"
)

// SSMClient interface for SSM operations (for testing)
type SSMClient interface {
	GetParameter(ctx context.Context, params *ssmapi.GetParameterRequest) (*ssmapi.GetParameterResult, error)
	GetParameters(ctx context.Context, params *ssmapi.GetParametersRequest) (*ssmapi.GetParametersResult, error)
	PutParameter(ctx context.Context, params *ssmapi.PutParameterRequest) (*ssmapi.PutParameterResult, error)
	DeleteParameter(ctx context.Context, params *ssmapi.DeleteParameterRequest) (*ssmapi.DeleteParameterResult, error)
}
