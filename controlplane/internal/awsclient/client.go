package awsclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Config holds configuration for AWS client
type Config struct {
	// Credentials for AWS authentication
	Credentials Credentials

	// Region is the AWS region
	Region string

	// Endpoint is the API endpoint (optional, for custom endpoints like LocalStack)
	Endpoint string

	// InsecureSkipVerify skips TLS certificate verification
	InsecureSkipVerify bool

	// HTTPClient is a custom HTTP client (optional)
	HTTPClient *http.Client

	// Timeout for requests
	Timeout time.Duration

	// MaxRetries is the maximum number of retries
	MaxRetries int

	// RetryDelay is the initial retry delay
	RetryDelay time.Duration
}

// Client is a generic AWS API client
type Client struct {
	config     Config
	httpClient *http.Client
	signer     *Signer
}

// NewClient creates a new AWS client
func NewClient(config Config) *Client {
	// Set defaults
	if config.Region == "" {
		config.Region = "us-east-1"
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	if config.RetryDelay == 0 {
		config.RetryDelay = 100 * time.Millisecond
	}

	// Create HTTP client if not provided
	httpClient := config.HTTPClient
	if httpClient == nil {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.InsecureSkipVerify,
			},
		}

		timeout := config.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		httpClient = &http.Client{
			Transport: transport,
			Timeout:   timeout,
		}
	}

	return &Client{
		config:     config,
		httpClient: httpClient,
	}
}

// DoRequest performs an AWS API request with signing and retries
func (c *Client) DoRequest(ctx context.Context, req *http.Request, service string) (*http.Response, error) {
	// Set up signer if we have credentials
	if c.config.Credentials.AccessKeyID != "" && c.config.Credentials.SecretAccessKey != "" {
		if c.signer == nil {
			c.signer = NewSigner(c.config.Credentials, SignerOptions{
				Service: service,
				Region:  c.config.Region,
			})
		}

		// Sign the request
		if err := c.signer.SignRequest(req); err != nil {
			return nil, fmt.Errorf("failed to sign request: %w", err)
		}
	}

	// Perform request with retries
	var lastErr error
	delay := c.config.RetryDelay

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		// Clone the request for retry
		reqClone := req.Clone(ctx)

		// Perform the request
		resp, err := c.httpClient.Do(reqClone)
		if err != nil {
			lastErr = err
			if attempt < c.config.MaxRetries {
				time.Sleep(delay)
				delay *= 2 // Exponential backoff
				continue
			}
			return nil, fmt.Errorf("request failed after %d attempts: %w", attempt+1, err)
		}

		// Check if we should retry based on status code
		if shouldRetry(resp.StatusCode) && attempt < c.config.MaxRetries {
			resp.Body.Close()
			time.Sleep(delay)
			delay *= 2
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", c.config.MaxRetries+1, lastErr)
}

// BuildEndpoint builds the full endpoint URL for a service
func (c *Client) BuildEndpoint(service string) string {
	if c.config.Endpoint != "" {
		// Use custom endpoint
		endpoint := c.config.Endpoint
		if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			endpoint = "https://" + endpoint
		}
		return strings.TrimSuffix(endpoint, "/")
	}

	// Build standard AWS endpoint
	// Special handling for certain services
	switch service {
	case "s3":
		if c.config.Region == "us-east-1" {
			return "https://s3.amazonaws.com"
		}
		return fmt.Sprintf("https://s3.%s.amazonaws.com", c.config.Region)
	default:
		return fmt.Sprintf("https://%s.%s.amazonaws.com", service, c.config.Region)
	}
}

// shouldRetry determines if a request should be retried based on status code
func shouldRetry(statusCode int) bool {
	switch statusCode {
	case 500, 502, 503, 504: // Server errors
		return true
	case 429: // Too many requests
		return true
	default:
		return false
	}
}

// GetCredentials returns the client's credentials
func (c *Client) GetCredentials() Credentials {
	return c.config.Credentials
}
