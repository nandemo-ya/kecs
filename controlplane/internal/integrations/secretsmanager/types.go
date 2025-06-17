package secretsmanager

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// Integration represents the Secrets Manager integration
type Integration interface {
	// SyncSecret synchronizes a single secret to a Kubernetes secret
	SyncSecret(ctx context.Context, secretName string, namespace string) error

	// GetSecret retrieves a secret value from LocalStack Secrets Manager
	GetSecret(ctx context.Context, secretName string) (*Secret, error)

	// GetSecretWithKey retrieves a specific JSON key from a secret
	GetSecretWithKey(ctx context.Context, secretName, jsonKey string) (string, error)

	// CreateOrUpdateSecret creates or updates a Kubernetes secret from Secrets Manager
	CreateOrUpdateSecret(ctx context.Context, secret *Secret, jsonKey string, namespace string) error

	// DeleteSecret deletes a synchronized secret
	DeleteSecret(ctx context.Context, secretName, namespace string) error

	// SyncSecrets batch synchronizes multiple secrets
	SyncSecrets(ctx context.Context, secrets []SecretReference, namespace string) error

	// GetSecretNameForSecret returns the Kubernetes secret name for a given Secrets Manager secret
	GetSecretNameForSecret(secretName string) string
}

// Secret represents a Secrets Manager secret
type Secret struct {
	Name         string
	Value        string // The actual secret value (plain text or JSON)
	Type         string // Binary or String
	VersionId    string
	VersionStage []string
	CreatedDate  time.Time
}

// SecretReference represents a reference to a secret with optional JSON key
type SecretReference struct {
	SecretName   string
	JSONKey      string // Optional: specific key to extract from JSON secret
	VersionStage string // Optional: version stage (e.g., AWSCURRENT, AWSPENDING)
	VersionId    string // Optional: specific version ID
}

// Config represents Secrets Manager integration configuration
type Config struct {
	LocalStackEndpoint string
	SecretPrefix       string        // Prefix for created secrets (e.g., "sm-")
	KubeNamespace      string        // Default namespace for secrets
	SyncRetries        int           // Number of retries for sync operations
	CacheTTL           time.Duration // Cache duration for secret values
}

// SecretAnnotations defines annotations added to Kubernetes secrets
var SecretAnnotations = struct {
	SecretName    string
	VersionId     string
	VersionStage  string
	LastSynced    string
	Source        string
	JSONKey       string
}{
	SecretName:    "kecs.io/secretsmanager-secret-name",
	VersionId:     "kecs.io/secretsmanager-version-id",
	VersionStage:  "kecs.io/secretsmanager-version-stage",
	LastSynced:    "kecs.io/secretsmanager-last-synced",
	Source:        "kecs.io/secret-source",
	JSONKey:       "kecs.io/secretsmanager-json-key",
}

// SecretLabels defines labels added to Kubernetes secrets
var SecretLabels = struct {
	ManagedBy string
	Source    string
}{
	ManagedBy: "kecs.io/managed-by",
	Source:    "kecs.io/source",
}

// Default configuration values
const (
	DefaultSecretPrefix  = "sm-"
	DefaultKubeNamespace = "default"
	DefaultSyncRetries   = 3
	DefaultCacheTTL      = 5 * time.Minute
	SourceSecretsManager = "secrets-manager"
)

// SecretsManagerClient interface for Secrets Manager operations (for testing)
type SecretsManagerClient interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
	UpdateSecret(ctx context.Context, params *secretsmanager.UpdateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error)
	DeleteSecret(ctx context.Context, params *secretsmanager.DeleteSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error)
}