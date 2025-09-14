package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// ProxyHandler handles routing requests to appropriate backends
type ProxyHandler struct {
	localStackURL   *url.URL
	localStackProxy *httputil.ReverseProxy
	ecsHandler      http.Handler
	elbv2Handler    http.Handler
	sdHandler       http.Handler // Service Discovery handler
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(localStackURL string, ecsHandler, elbv2Handler, sdHandler http.Handler) (*ProxyHandler, error) {
	parsedURL, err := url.Parse(localStackURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LocalStack URL: %w", err)
	}

	// Create reverse proxy for LocalStack
	proxy := httputil.NewSingleHostReverseProxy(parsedURL)

	// Customize the director to preserve headers
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Preserve original host header if needed
		if req.Header.Get("X-Forwarded-Host") == "" {
			req.Header.Set("X-Forwarded-Host", req.Host)
		}
	}

	// Add error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logging.Error("Proxy error", "path", r.URL.Path, "error", err)
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}

	// Configure transport for better performance
	proxy.Transport = &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	}

	return &ProxyHandler{
		localStackURL:   parsedURL,
		localStackProxy: proxy,
		ecsHandler:      ecsHandler,
		elbv2Handler:    elbv2Handler,
		sdHandler:       sdHandler,
	}, nil
}

// ServeHTTP implements http.Handler interface with simplified routing logic
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Log incoming request
	logging.Info("ProxyHandler incoming request",
		"method", r.Method,
		"path", r.URL.Path,
		"x-amz-target", r.Header.Get("X-Amz-Target"),
		"content-type", r.Header.Get("Content-Type"),
	)

	// Simplified routing based on headers only (no body reading)

	// Step 1: Check Content-Type for form-encoded APIs (ELBv2, EC2, RDS, etc.)
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		logging.Info("Routing to ELBv2 handler (form-encoded)", "path", r.URL.Path)
		h.elbv2Handler.ServeHTTP(w, r)
		return
	}

	// Step 2: Check X-Amz-Target header for service routing
	target := r.Header.Get("X-Amz-Target")
	if target != "" {
		// ECS requests
		if strings.HasPrefix(target, "AmazonEC2ContainerServiceV") {
			logging.Debug("Routing to ECS handler", "target", target)
			h.ecsHandler.ServeHTTP(w, r)
			return
		}

		// Service Discovery requests
		if strings.HasPrefix(target, "Route53AutoNaming_") {
			logging.Debug("Routing to Service Discovery handler", "target", target)
			h.sdHandler.ServeHTTP(w, r)
			return
		}
	}

	// Step 3: Check path-based routing for non-root paths
	path := r.URL.Path
	if strings.HasPrefix(path, "/v1/") {
		// ECS v1 API endpoints
		logging.Debug("Routing to ECS handler (v1 path)", "path", path)
		h.ecsHandler.ServeHTTP(w, r)
		return
	}

	// Default: proxy to LocalStack
	logging.Debug("Proxying to LocalStack", "path", r.URL.Path)
	h.localStackProxy.ServeHTTP(w, r)
}

// HealthCheck performs health check on LocalStack connection
func (h *ProxyHandler) HealthCheck(ctx context.Context) error {
	// Create a health check request
	req, err := http.NewRequestWithContext(ctx, "GET", h.localStackURL.String()+"/_localstack/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	// Send request with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}
