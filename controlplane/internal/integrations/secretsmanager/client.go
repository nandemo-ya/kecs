package secretsmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	secretsmanagerapi "github.com/nandemo-ya/kecs/controlplane/internal/secretsmanager/generated"
)

// secretsManagerClient implements SecretsManagerClient interface using HTTP calls
type secretsManagerClient struct {
	endpoint   string
	httpClient *http.Client
}

// newSecretsManagerClient creates a new Secrets Manager client
func newSecretsManagerClient(endpoint string) SecretsManagerClient {
	if endpoint == "" {
		endpoint = "http://localhost:4566"
	}
	
	return &secretsManagerClient{
		endpoint:   endpoint,
		httpClient: &http.Client{},
	}
}

// GetSecretValue retrieves a secret value
func (c *secretsManagerClient) GetSecretValue(ctx context.Context, params *secretsmanagerapi.GetSecretValueRequest) (*secretsmanagerapi.GetSecretValueResponse, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "secretsmanager.GetSecretValue")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Check for specific error types
		if strings.Contains(string(body), "ResourceNotFoundException") {
			return nil, fmt.Errorf("secret not found: %s", params.SecretId)
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result secretsmanagerapi.GetSecretValueResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// CreateSecret creates a new secret
func (c *secretsManagerClient) CreateSecret(ctx context.Context, params *secretsmanagerapi.CreateSecretRequest) (*secretsmanagerapi.CreateSecretResponse, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "secretsmanager.CreateSecret")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result secretsmanagerapi.CreateSecretResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// UpdateSecret updates an existing secret
func (c *secretsManagerClient) UpdateSecret(ctx context.Context, params *secretsmanagerapi.UpdateSecretRequest) (*secretsmanagerapi.UpdateSecretResponse, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "secretsmanager.UpdateSecret")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result secretsmanagerapi.UpdateSecretResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DeleteSecret deletes a secret
func (c *secretsManagerClient) DeleteSecret(ctx context.Context, params *secretsmanagerapi.DeleteSecretRequest) (*secretsmanagerapi.DeleteSecretResponse, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "secretsmanager.DeleteSecret")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Check for specific error types
		if strings.Contains(string(body), "ResourceNotFoundException") {
			return nil, fmt.Errorf("secret not found")
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result secretsmanagerapi.DeleteSecretResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}