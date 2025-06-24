package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/admin"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/cache"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/duckdb"
)

var (
	// Additional server-specific flags
	kubeconfig        string
	adminPort         int
	dataDir           string
	localstackEnabled bool
	configFile        string

	// serverCmd represents the server command
	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Start the KECS Control Plane server",
		Long: `Start the KECS Control Plane server that provides Amazon ECS compatible APIs.
The server connects to a Kubernetes cluster and translates ECS API calls to Kubernetes resources.`,
		Run: func(cmd *cobra.Command, args []string) {
			runServer()
		},
	}
)

// getDefaultDataDir returns the default data directory path
func getDefaultDataDir() string {
	// Check KECS_DATA_DIR env var first
	if dataDir := os.Getenv("KECS_DATA_DIR"); dataDir != "" {
		return dataDir
	}

	// Fall back to home directory
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".kecs", "data")
	}

	return "~/.kecs/data"
}

func init() {
	// Add server-specific flags
	serverCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig file (default is $HOME/.kube/config)")
	serverCmd.Flags().IntVar(&adminPort, "admin-port", 8081, "Port for the admin server")
	serverCmd.Flags().StringVar(&dataDir, "data-dir", getDefaultDataDir(), "Directory for storing persistent data")
	serverCmd.Flags().BoolVar(&localstackEnabled, "localstack-enabled", false, "Enable LocalStack integration for AWS service emulation")
	serverCmd.Flags().StringVar(&configFile, "config", "", "Path to configuration file")
}

func runServer() {
	// Log mode status for debugging
	if os.Getenv("KECS_TEST_MODE") == "true" {
		fmt.Println("KECS_TEST_MODE is enabled - running in test mode")
	}
	if os.Getenv("KECS_CONTAINER_MODE") == "true" {
		fmt.Println("KECS_CONTAINER_MODE is enabled - running in container mode")
	}

	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override config with command line flags
	if port != 0 {
		cfg.Server.Port = port
	}
	if adminPort != 0 {
		cfg.Server.AdminPort = adminPort
	}
	if dataDir != "" {
		cfg.Server.DataDir = dataDir
	}
	if logLevel != "" {
		cfg.Server.LogLevel = logLevel
	}
	if localstackEnabled {
		cfg.LocalStack.Enabled = true
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	fmt.Printf("Starting KECS Control Plane server on port %d with log level %s\n", cfg.Server.Port, cfg.Server.LogLevel)
	fmt.Printf("Starting KECS Admin server on port %d\n", cfg.Server.AdminPort)
	if kubeconfig != "" {
		fmt.Printf("Using kubeconfig: %s\n", kubeconfig)
	} else {
		fmt.Println("Using in-cluster configuration or default kubeconfig")
	}

	// Initialize storage
	dbPath := filepath.Join(cfg.Server.DataDir, "kecs.db")
	fmt.Printf("Using database: %s\n", dbPath)

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(cfg.Server.DataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize DuckDB storage
	dbStorage, err := duckdb.NewDuckDBStorage(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize storage tables
	ctx := context.Background()
	if err := dbStorage.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize storage tables: %v", err)
	}

	// Wrap storage with cache layer
	// Default cache settings: 5 minute TTL, 10000 max items
	cacheTTL := 5 * time.Minute
	cacheSize := 10000
	storage := cache.NewCachedStorage(dbStorage, cacheSize, cacheTTL)

	defer func() {
		if err := storage.Close(); err != nil {
			log.Printf("Error closing storage: %v", err)
		}
	}()

	fmt.Printf("Initialized in-memory cache (TTL: %v, Max Size: %d)\n", cacheTTL, cacheSize)

	// Initialize the API server with LocalStack configuration
	var localstackConfig *localstack.Config
	if cfg.LocalStack.Enabled {
		fmt.Println("LocalStack integration is enabled")
		localstackConfig = &cfg.LocalStack
	}

	apiServer, err := api.NewServer(cfg.Server.Port, kubeconfig, storage, localstackConfig)
	if err != nil {
		log.Fatalf("Failed to initialize API server: %v", err)
	}
	adminServer := admin.NewServer(cfg.Server.AdminPort, storage)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		fmt.Printf("Received signal %s, shutting down...\n", sig)
		cancel()

		// Allow some time for graceful shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		// Stop both servers
		if err := apiServer.Stop(shutdownCtx); err != nil {
			fmt.Printf("Error during API server shutdown: %v\n", err)
		}

		if err := adminServer.Stop(shutdownCtx); err != nil {
			fmt.Printf("Error during Admin server shutdown: %v\n", err)
		}

		// LocalStack will be stopped by the API server during shutdown
	}()

	// Start the admin server in a goroutine
	go func() {
		if err := adminServer.Start(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Admin server error: %v\n", err)
			cancel() // Cancel context to trigger shutdown of API server
		}
	}()

	// Start the API server in the main goroutine
	if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("API server error: %v\n", err)
		os.Exit(1)
	}

	// Wait for context cancellation
	<-ctx.Done()
}
