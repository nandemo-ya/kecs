package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// AWSProxy handles proxying AWS SDK requests to LocalStack
type AWSProxy struct {
	localStackEndpoint string
	httpClient         *http.Client
	reverseProxy       *httputil.ReverseProxy
	debug              bool
}

// NewAWSProxy creates a new AWS proxy instance
func NewAWSProxy(localStackEndpoint string, debug bool) (*AWSProxy, error) {
	targetURL, err := url.Parse(localStackEndpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid LocalStack endpoint: %w", err)
	}

	// Validate that the URL has a valid scheme
	if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
		return nil, fmt.Errorf("invalid LocalStack endpoint scheme: %s", targetURL.Scheme)
	}

	// Create a reverse proxy
	reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Customize the director to handle AWS service routing
	originalDirector := reverseProxy.Director
	reverseProxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Preserve the original host header for AWS signature validation
		req.Host = targetURL.Host
		req.URL.Host = targetURL.Host
		req.URL.Scheme = targetURL.Scheme

		// Log the request if debug is enabled
		if debug {
			log.Printf("Proxying request: %s %s -> %s", req.Method, req.URL.Path, targetURL.String())
		}
	}

	// Custom error handler
	reverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	// Modify response if needed
	reverseProxy.ModifyResponse = func(resp *http.Response) error {
		if debug {
			log.Printf("Response: %d %s", resp.StatusCode, resp.Status)
		}
		return nil
	}

	return &AWSProxy{
		localStackEndpoint: localStackEndpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		reverseProxy: reverseProxy,
		debug:        debug,
	}, nil
}

// ServeHTTP implements the http.Handler interface
func (p *AWSProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle health check
	if r.URL.Path == "/health" {
		p.handleHealth(w, r)
		return
	}

	// Extract AWS service from the request
	service := p.extractAWSService(r)
	if p.debug && service != "" {
		log.Printf("Detected AWS service: %s", service)
	}

	// Add LocalStack-specific headers if needed
	r.Header.Set("X-Forwarded-For", r.RemoteAddr)
	r.Header.Set("X-Forwarded-Proto", "http")

	// Proxy the request
	p.reverseProxy.ServeHTTP(w, r)
}

// extractAWSService attempts to identify the AWS service from the request
func (p *AWSProxy) extractAWSService(r *http.Request) string {
	// Check Authorization header for service hints (most reliable)
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// AWS Signature Version 4 format: AWS4-HMAC-SHA256 Credential=.../20230101/us-east-1/s3/aws4_request
		if strings.HasPrefix(authHeader, "AWS4-HMAC-SHA256") {
			parts := strings.Split(authHeader, "/")
			if len(parts) >= 5 {
				return parts[3] // Service name is typically the 4th component
			}
		}
	}

	// Check X-Amz-Target header (used by some services like DynamoDB)
	target := r.Header.Get("X-Amz-Target")
	if target != "" {
		// Target format is like "DynamoDB_20120810.ListTables"
		parts := strings.Split(target, ".")
		if len(parts) > 0 {
			// Extract service and version (e.g., "DynamoDB_20120810" -> "dynamodb_20120810")
			servicePart := strings.ToLower(parts[0])
			// Replace DynamoDB with dynamodb for consistency
			return servicePart
		}
	}

	// Check User-Agent for SDK hints
	userAgent := r.Header.Get("User-Agent")
	if strings.Contains(userAgent, "aws-sdk-go") {
		// Try to extract service from the user agent
		// Format: aws-sdk-go/1.44.122 (go1.19.2; linux; amd64) S3Manager
		parts := strings.Fields(userAgent)
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			// Check if it's a service name (not a version or parenthetical)
			if !strings.Contains(lastPart, "/") && !strings.Contains(lastPart, "(") && !strings.Contains(lastPart, ")") {
				return strings.ToLower(lastPart)
			}
		}
	}

	// Check host header last (least reliable due to various formats)
	host := r.Host
	if host != "" {
		// Extract service from host (e.g., s3.localhost.localstack.cloud:4566)
		parts := strings.Split(host, ".")
		if len(parts) > 0 && parts[0] != "localhost" && parts[0] != "localstack" && parts[0] != "example" {
			return parts[0]
		}
	}

	return ""
}

// handleHealth handles the health check endpoint
func (p *AWSProxy) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check if LocalStack is reachable
	healthURL := fmt.Sprintf("%s/_localstack/health", p.localStackEndpoint)
	resp, err := p.httpClient.Get(healthURL)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"unhealthy","message":"Cannot reach LocalStack: %s"}`, err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"unhealthy","message":"LocalStack returned status %d"}`, resp.StatusCode)
		return
	}

	// Return healthy status
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","localstack_endpoint":"%s","timestamp":"%s"}`,
		p.localStackEndpoint, time.Now().UTC().Format(time.RFC3339))
}

// copyHeaders copies headers from source to destination
func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// copyResponse copies the response body and headers
func copyResponse(w http.ResponseWriter, resp *http.Response) error {
	// Copy headers
	copyHeaders(w.Header(), resp.Header)

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy body
	_, err := io.Copy(w, resp.Body)
	return err
}
