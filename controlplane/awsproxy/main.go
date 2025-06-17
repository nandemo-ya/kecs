package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	// Version is set during build
	Version = "dev"
	// CommitHash is set during build
	CommitHash = "unknown"
)

func main() {
	// Parse command line flags
	var (
		port               = flag.Int("port", 4566, "Port to listen on")
		localStackEndpoint = flag.String("localstack-endpoint", getEnvOrDefault("LOCALSTACK_ENDPOINT", "http://localstack.aws-services.svc.cluster.local:4566"), "LocalStack endpoint URL")
		debug              = flag.Bool("debug", getEnvOrDefaultBool("DEBUG", false), "Enable debug logging")
		version            = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	// Show version if requested
	if *version {
		fmt.Printf("kecs-aws-proxy version %s (commit: %s)\n", Version, CommitHash)
		os.Exit(0)
	}

	// Configure logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	if *debug {
		log.Printf("Debug mode enabled")
	}

	// Create AWS proxy
	proxy, err := NewAWSProxy(*localStackEndpoint, *debug)
	if err != nil {
		log.Fatalf("Failed to create AWS proxy: %v", err)
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      proxy,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting AWS proxy server on port %d", *port)
		log.Printf("Proxying to LocalStack at %s", *localStackEndpoint)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

// getEnvOrDefault returns the value of an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvOrDefaultBool returns the boolean value of an environment variable or a default value
func getEnvOrDefaultBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes" || value == "on"
}