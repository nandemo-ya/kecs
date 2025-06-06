package converters

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

func TestParseSecretARN(t *testing.T) {
	converter := NewTaskConverter("ap-northeast-1", "123456789012")

	tests := []struct {
		name     string
		arn      string
		expected *SecretInfo
	}{
		{
			name: "Secrets Manager ARN with JSON key",
			arn:  "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf:username::",
			expected: &SecretInfo{
				SecretName: "kecs-secret-my-secret",
				Key:        "username",
				Source:     "secretsmanager",
			},
		},
		{
			name: "Secrets Manager ARN without JSON key",
			arn:  "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf",
			expected: &SecretInfo{
				SecretName: "kecs-secret-my-secret",
				Key:        "value",
				Source:     "secretsmanager",
			},
		},
		{
			name: "SSM Parameter Store ARN",
			arn:  "arn:aws:ssm:us-east-1:123456789012:parameter/app/database/password",
			expected: &SecretInfo{
				SecretName: "kecs-secret-app-database-password",
				Key:        "value",
				Source:     "ssm",
			},
		},
		{
			name:     "Invalid ARN",
			arn:      "invalid-arn",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.parseSecretARN(tt.arn)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.SecretName, result.SecretName)
				assert.Equal(t, tt.expected.Key, result.Key)
				assert.Equal(t, tt.expected.Source, result.Source)
			}
		})
	}
}

func TestCollectSecrets(t *testing.T) {
	converter := NewTaskConverter("ap-northeast-1", "123456789012")

	containerDefs := []types.ContainerDefinition{
		{
			Name:  ptr.To("web"),
			Image: ptr.To("nginx:latest"),
			Secrets: []types.Secret{
				{
					Name:      ptr.To("DB_PASSWORD"),
					ValueFrom: ptr.To("arn:aws:secretsmanager:us-east-1:123456789012:secret:db-pass-XyZ123"),
				},
				{
					Name:      ptr.To("API_KEY"),
					ValueFrom: ptr.To("arn:aws:ssm:us-east-1:123456789012:parameter/api/key"),
				},
			},
		},
		{
			Name:  ptr.To("app"),
			Image: ptr.To("myapp:latest"),
			Secrets: []types.Secret{
				{
					Name:      ptr.To("DB_PASSWORD2"),
					ValueFrom: ptr.To("arn:aws:secretsmanager:us-east-1:123456789012:secret:db-pass-XyZ123"), // Same secret
				},
			},
		},
	}

	secrets := converter.CollectSecrets(containerDefs)

	// Should have 2 unique secrets (db-pass and api/key)
	assert.Len(t, secrets, 2)

	// Check that the secrets were parsed correctly
	dbSecretARN := "arn:aws:secretsmanager:us-east-1:123456789012:secret:db-pass-XyZ123"
	dbSecret, exists := secrets[dbSecretARN]
	require.True(t, exists)
	assert.Equal(t, "kecs-secret-db-pass", dbSecret.SecretName)
	assert.Equal(t, "secretsmanager", dbSecret.Source)

	apiKeyARN := "arn:aws:ssm:us-east-1:123456789012:parameter/api/key"
	apiSecret, exists := secrets[apiKeyARN]
	require.True(t, exists)
	assert.Equal(t, "kecs-secret-api-key", apiSecret.SecretName)
	assert.Equal(t, "ssm", apiSecret.Source)
}

func TestConvertSecrets(t *testing.T) {
	converter := NewTaskConverter("ap-northeast-1", "123456789012")

	secrets := []types.Secret{
		{
			Name:      ptr.To("DB_PASSWORD"),
			ValueFrom: ptr.To("arn:aws:secretsmanager:us-east-1:123456789012:secret:db-pass-XyZ123:password::"),
		},
		{
			Name:      ptr.To("API_KEY"),
			ValueFrom: ptr.To("arn:aws:ssm:us-east-1:123456789012:parameter/api/key"),
		},
		{
			Name:      ptr.To("INVALID"),
			ValueFrom: ptr.To("invalid-arn"),
		},
	}

	envVars := converter.convertSecrets(secrets)

	// Should have 2 env vars (invalid ARN is skipped)
	assert.Len(t, envVars, 2)

	// Check DB_PASSWORD
	dbEnv := envVars[0]
	assert.Equal(t, "DB_PASSWORD", dbEnv.Name)
	require.NotNil(t, dbEnv.ValueFrom)
	require.NotNil(t, dbEnv.ValueFrom.SecretKeyRef)
	assert.Equal(t, "kecs-secret-db-pass", dbEnv.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "password", dbEnv.ValueFrom.SecretKeyRef.Key)

	// Check API_KEY
	apiEnv := envVars[1]
	assert.Equal(t, "API_KEY", apiEnv.Name)
	require.NotNil(t, apiEnv.ValueFrom)
	require.NotNil(t, apiEnv.ValueFrom.SecretKeyRef)
	assert.Equal(t, "kecs-secret-api-key", apiEnv.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "value", apiEnv.ValueFrom.SecretKeyRef.Key)
}

func TestSanitizeSecretName(t *testing.T) {
	converter := NewTaskConverter("ap-northeast-1", "123456789012")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple name",
			input:    "my-secret",
			expected: "kecs-secret-my-secret",
		},
		{
			name:     "Name with special characters",
			input:    "my_secret@123",
			expected: "kecs-secret-my-secret-123",
		},
		{
			name:     "Name with uppercase",
			input:    "MySecret",
			expected: "kecs-secret-mysecret",
		},
		{
			name:     "Name with slashes",
			input:    "/app/db/password",
			expected: "kecs-secret-app-db-password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.sanitizeSecretName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTaskConverterWithSecrets(t *testing.T) {
	converter := NewTaskConverter("ap-northeast-1", "123456789012")

	// Create a task definition with secrets
	containerDefs := []types.ContainerDefinition{
		{
			Name:  ptr.To("app"),
			Image: ptr.To("myapp:latest"),
			Environment: []types.KeyValuePair{
				{Name: ptr.To("ENV"), Value: ptr.To("production")},
			},
			Secrets: []types.Secret{
				{
					Name:      ptr.To("DB_PASSWORD"),
					ValueFrom: ptr.To("arn:aws:secretsmanager:us-east-1:123456789012:secret:db-pass-XyZ123"),
				},
			},
			PortMappings: []types.PortMapping{
				{ContainerPort: ptr.To(8080), Protocol: ptr.To("tcp")},
			},
		},
	}

	containerDefsJSON, err := json.Marshal(containerDefs)
	require.NoError(t, err)

	taskDef := &storage.TaskDefinition{
		ARN:                  "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/app:1",
		Family:               "app",
		Revision:             1,
		ContainerDefinitions: string(containerDefsJSON),
		NetworkMode:          "bridge",
		CPU:                  "256",
		Memory:               "512",
	}

	runTaskReq := types.RunTaskRequest{
		TaskDefinition: ptr.To("app:1"),
		Cluster:        ptr.To("default"),
		Count:          ptr.To(1),
		LaunchType:     ptr.To("FARGATE"),
	}
	runTaskReqJSON, err := json.Marshal(runTaskReq)
	require.NoError(t, err)

	cluster := &storage.Cluster{
		Name:   "default",
		ARN:    "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default",
		Region: "ap-northeast-1",
	}

	// Convert to pod
	pod, err := converter.ConvertTaskToPod(taskDef, runTaskReqJSON, cluster, "task-123")
	require.NoError(t, err)
	require.NotNil(t, pod)

	// Check pod basics
	assert.Equal(t, "ecs-task-task-123", pod.Name)
	assert.Equal(t, "default-ap-northeast-1", pod.Namespace)

	// Check container
	require.Len(t, pod.Spec.Containers, 1)
	container := pod.Spec.Containers[0]
	assert.Equal(t, "app", container.Name)
	assert.Equal(t, "myapp:latest", container.Image)

	// Check environment variables
	require.Len(t, container.Env, 2) // 1 regular env + 1 secret
	
	// Regular env var
	assert.Equal(t, "ENV", container.Env[0].Name)
	assert.Equal(t, "production", container.Env[0].Value)
	
	// Secret env var
	assert.Equal(t, "DB_PASSWORD", container.Env[1].Name)
	require.NotNil(t, container.Env[1].ValueFrom)
	require.NotNil(t, container.Env[1].ValueFrom.SecretKeyRef)
	assert.Equal(t, "kecs-secret-db-pass", container.Env[1].ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "value", container.Env[1].ValueFrom.SecretKeyRef.Key)
}