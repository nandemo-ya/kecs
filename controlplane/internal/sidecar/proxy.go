package sidecar

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// ProxyConfig contains configuration for the AWS SDK proxy sidecar
type ProxyConfig struct {
	// LocalStack endpoint URL
	LocalStackEndpoint string
	// Port to listen on
	ListenPort int
	// Services to proxy
	Services []string
	// Enable debug logging
	Debug bool
	// HTTP client timeout
	Timeout time.Duration
}

// DefaultProxyConfig returns the default proxy configuration
func DefaultProxyConfig() *ProxyConfig {
	return &ProxyConfig{
		LocalStackEndpoint: "http://localstack.localstack.svc.cluster.local:4566",
		ListenPort:         8080,
		Services:           []string{"s3", "dynamodb", "sqs", "sns", "ssm", "secretsmanager"},
		Debug:              false,
		Timeout:            30 * time.Second,
	}
}

// Proxy implements the AWS SDK proxy that redirects AWS API calls to LocalStack
type Proxy struct {
	config     *ProxyConfig
	httpClient *http.Client
	server     *http.Server
}

// NewProxy creates a new AWS SDK proxy instance
func NewProxy(config *ProxyConfig) *Proxy {
	if config == nil {
		config = DefaultProxyConfig()
	}

	return &Proxy{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Start starts the proxy server
func (p *Proxy) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", p.handleRequest)
	mux.HandleFunc("/health", p.handleHealth)

	p.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", p.config.ListenPort),
		Handler: mux,
	}

	klog.Infof("Starting AWS SDK proxy on port %d, forwarding to %s", p.config.ListenPort, p.config.LocalStackEndpoint)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := p.server.Shutdown(shutdownCtx); err != nil {
			klog.Errorf("Error shutting down proxy server: %v", err)
		}
	}()

	if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("proxy server error: %w", err)
	}

	return nil
}

// Stop stops the proxy server
func (p *Proxy) Stop(ctx context.Context) error {
	if p.server != nil {
		return p.server.Shutdown(ctx)
	}
	return nil
}

// handleHealth handles health check requests
func (p *Proxy) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// handleRequest handles AWS SDK requests and forwards them to LocalStack
func (p *Proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	if p.config.Debug {
		klog.Infof("Proxy request: %s %s", r.Method, r.URL.String())
	}

	// Extract AWS service from the request
	service := p.extractAWSService(r)
	if service == "" {
		if p.config.Debug {
			klog.Warningf("Could not determine AWS service from request")
		}
		http.Error(w, "Could not determine AWS service", http.StatusBadRequest)
		return
	}

	// Check if service is enabled for proxying
	if !p.isServiceEnabled(service) {
		if p.config.Debug {
			klog.Infof("Service %s is not enabled for proxying", service)
		}
		http.Error(w, fmt.Sprintf("Service %s is not enabled for proxying", service), http.StatusForbidden)
		return
	}

	// Create proxy request
	targetURL, err := url.Parse(p.config.LocalStackEndpoint)
	if err != nil {
		klog.Errorf("Failed to parse LocalStack endpoint: %v", err)
		http.Error(w, "Invalid LocalStack endpoint", http.StatusInternalServerError)
		return
	}

	// Preserve the original path and query
	targetURL.Path = r.URL.Path
	targetURL.RawQuery = r.URL.RawQuery

	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL.String(), r.Body)
	if err != nil {
		klog.Errorf("Failed to create proxy request: %v", err)
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for name, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(name, value)
		}
	}

	// Ensure the Host header points to LocalStack
	proxyReq.Header.Set("Host", targetURL.Host)
	proxyReq.Host = targetURL.Host

	// Forward to LocalStack
	resp, err := p.httpClient.Do(proxyReq)
	if err != nil {
		klog.Errorf("Failed to forward request to LocalStack: %v", err)
		http.Error(w, "Failed to forward request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		klog.Errorf("Failed to copy response body: %v", err)
	}

	if p.config.Debug {
		klog.Infof("Proxy response: %d for %s %s", resp.StatusCode, r.Method, r.URL.String())
	}
}

// extractAWSService extracts the AWS service name from the request
func (p *Proxy) extractAWSService(r *http.Request) string {
	// Check Authorization header for AWS service
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.Contains(auth, "AWS4-HMAC-SHA256") {
			// Extract service from "Credential=.../service/region/..."
			parts := strings.Split(auth, "/")
			for i, part := range parts {
				if strings.Contains(part, "Credential=") && i+2 < len(parts) {
					return parts[i+2]
				}
			}
		}
	}

	// Check Host header for service
	host := r.Header.Get("Host")
	if host != "" {
		// Extract service from "service.region.amazonaws.com"
		parts := strings.Split(host, ".")
		if len(parts) > 0 {
			service := parts[0]
			// Handle special cases
			if strings.HasPrefix(service, "s3-") {
				return "s3"
			}
			return service
		}
	}

	// Check X-Amz-Target header (used by some services like DynamoDB)
	if target := r.Header.Get("X-Amz-Target"); target != "" {
		parts := strings.Split(target, ".")
		if len(parts) > 0 {
			service := strings.ToLower(parts[0])
			// Map service names
			switch service {
			case "dynamodb_20120810":
				return "dynamodb"
			case "awsie":
				return "sns"
			case "amazonsqs":
				return "sqs"
			}
		}
	}

	// Check User-Agent for SDK hints
	if ua := r.Header.Get("User-Agent"); ua != "" {
		ua = strings.ToLower(ua)
		for _, service := range p.config.Services {
			if strings.Contains(ua, service) {
				return service
			}
		}
	}

	return ""
}

// isServiceEnabled checks if a service is enabled for proxying
func (p *Proxy) isServiceEnabled(service string) bool {
	for _, s := range p.config.Services {
		if s == service {
			return true
		}
	}
	return false
}