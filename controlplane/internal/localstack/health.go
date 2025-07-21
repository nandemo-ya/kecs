package localstack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// healthChecker implements the HealthChecker interface
type healthChecker struct {
	endpoint   string
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewHealthChecker creates a new health checker instance
func NewHealthChecker(endpoint string) HealthChecker {
	return &healthChecker{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// UpdateEndpoint updates the health check endpoint
func (hc *healthChecker) UpdateEndpoint(endpoint string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.endpoint = endpoint
	logging.Info("Updated health check endpoint", "endpoint", endpoint)
}

// CheckHealth performs a health check on LocalStack
func (hc *healthChecker) CheckHealth(ctx context.Context) (*HealthStatus, error) {
	hc.mu.RLock()
	endpoint := hc.endpoint
	hc.mu.RUnlock()
	
	healthURL := fmt.Sprintf("%s%s", endpoint, HealthCheckPath)
	logging.Debug("Performing health check on LocalStack", "url", healthURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := hc.httpClient.Do(req)
	if err != nil {
		logging.Warn("Failed to connect to LocalStack", "url", healthURL, "error", err)
		return &HealthStatus{
			Healthy:   false,
			Message:   fmt.Sprintf("failed to connect to LocalStack: %v", err),
			LastCheck: time.Now(),
		}, nil
	}
	defer resp.Body.Close()

	// Basic health check - if we get a response, LocalStack is at least running
	healthy := resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable

	// Try to get service-specific health information
	serviceHealth := hc.checkServiceHealth(ctx)

	return &HealthStatus{
		Healthy:       healthy,
		Message:       fmt.Sprintf("LocalStack responded with status %d", resp.StatusCode),
		ServiceHealth: serviceHealth,
		LastCheck:     time.Now(),
	}, nil
}

// WaitForHealthy waits for LocalStack to become healthy
func (hc *healthChecker) WaitForHealthy(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	consecutiveFails := 0
	maxConsecutiveFails := 3

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for LocalStack to become healthy")
			}

			status, err := hc.CheckHealth(ctx)
			if err != nil {
				logging.Warn("Health check error", "error", err)
				consecutiveFails++
				if consecutiveFails >= maxConsecutiveFails {
					return fmt.Errorf("health check failed %d times: %w", consecutiveFails, err)
				}
				continue
			}

			if status.Healthy {
				logging.Info("LocalStack is healthy")
				return nil
			}

			logging.Info("Waiting for LocalStack to become healthy", "message", status.Message)
			consecutiveFails = 0
		}
	}
}

// checkServiceHealth checks the health of individual AWS services
func (hc *healthChecker) checkServiceHealth(ctx context.Context) map[string]ServiceHealth {
	serviceHealthMap := make(map[string]ServiceHealth)

	// Check common services
	services := []string{"s3", "iam", "logs", "ssm", "secretsmanager", "elbv2"}

	for _, service := range services {
		start := time.Now()
		health := hc.checkIndividualService(ctx, service)
		health.Latency = time.Since(start)
		serviceHealthMap[service] = health
	}

	return serviceHealthMap
}

// checkIndividualService checks the health of a specific service
func (hc *healthChecker) checkIndividualService(ctx context.Context, service string) ServiceHealth {
	// LocalStack provides service-specific health endpoints
	// For now, we'll do a simple check by trying to access the service endpoint

	serviceHealth := ServiceHealth{
		Service: service,
		Healthy: true, // Assume healthy by default
	}

	// Service-specific health check URLs
	var checkURL string
	switch service {
	case "s3":
		checkURL = fmt.Sprintf("%s/", hc.endpoint) // S3 list buckets
	case "iam":
		checkURL = fmt.Sprintf("%s/?Action=ListUsers&Version=2010-05-08", hc.endpoint)
	case "logs":
		checkURL = fmt.Sprintf("%s/?Action=DescribeLogGroups&Version=2014-03-28", hc.endpoint)
	default:
		// Generic check
		checkURL = fmt.Sprintf("%s/", hc.endpoint)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
	if err != nil {
		serviceHealth.Healthy = false
		serviceHealth.Error = fmt.Sprintf("failed to create request: %v", err)
		return serviceHealth
	}

	// Add AWS headers for better compatibility
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20200101/us-east-1/"+service+"/aws4_request")

	resp, err := hc.httpClient.Do(req)
	if err != nil {
		serviceHealth.Healthy = false
		serviceHealth.Error = fmt.Sprintf("failed to connect: %v", err)
		return serviceHealth
	}
	defer resp.Body.Close()

	// Check if we get a valid response (even error responses indicate the service is running)
	if resp.StatusCode >= 500 {
		serviceHealth.Healthy = false
		serviceHealth.Error = fmt.Sprintf("service returned error: %d", resp.StatusCode)
	}

	return serviceHealth
}

// LocalStack health response structure
type localStackHealthResponse struct {
	Services map[string]struct {
		Available bool `json:"available"`
	} `json:"services"`
	Features struct {
		Persistence bool `json:"persistence"`
		InitScripts bool `json:"initScripts"`
	} `json:"features"`
	Version string `json:"version"`
}

// parseLocalStackHealth parses the LocalStack health endpoint response
func parseLocalStackHealth(resp *http.Response) (*localStackHealthResponse, error) {
	var health localStackHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode health response: %w", err)
	}
	return &health, nil
}
