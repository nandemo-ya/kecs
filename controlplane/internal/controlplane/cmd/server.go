package cmd

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	apiconfig "github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/admin"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/restoration"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/cache"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/duckdb"
	"github.com/nandemo-ya/kecs/controlplane/internal/webhook"
)

var (
	// Additional server-specific flags
	kubeconfig        string
	adminPort         int
	dataDir           string
	localstackEnabled bool
	configFile        string
	serverMode        string

	// serverCmd represents the server command
	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Start the KECS Control Plane server",
		Long: `Start the KECS Control Plane server that provides Amazon ECS compatible APIs.
The server connects to a Kubernetes cluster and translates ECS API calls to Kubernetes resources.`,
		Run: func(cmd *cobra.Command, args []string) {
			runServer(cmd)
		},
	}
)

// getDefaultDataDir returns the default data directory path
func getDefaultDataDir() string {
	// Use config package to get data directory
	cfg := config.DefaultConfig()
	return cfg.Server.DataDir
}

func init() {
	// Add server-specific flags
	serverCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig file (default is $HOME/.kube/config)")
	serverCmd.Flags().IntVar(&adminPort, "admin-port", 5374, "Port for the admin server")
	serverCmd.Flags().StringVar(&dataDir, "data-dir", getDefaultDataDir(), "Directory for storing persistent data")
	serverCmd.Flags().BoolVar(&localstackEnabled, "localstack-enabled", false, "Enable LocalStack integration for AWS service emulation")
	serverCmd.Flags().StringVar(&configFile, "config", "", "Path to configuration file")
	serverCmd.Flags().StringVar(&serverMode, "mode", "standalone", "Server mode: standalone or in-cluster")
}

func runServer(cmd *cobra.Command) {
	// Log mode status for debugging
	if config.GetBool("features.testMode") {
		logging.Info("KECS_TEST_MODE is enabled - running in test mode")
	}
	if config.GetBool("features.containerMode") {
		logging.Info("KECS_CONTAINER_MODE is enabled - running in container mode")
	}

	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override config with command line flags
	if portFlag, _ := cmd.Parent().PersistentFlags().GetInt("port"); portFlag != 0 {
		cfg.Server.Port = portFlag
	}
	if adminPort != 0 {
		cfg.Server.AdminPort = adminPort
	}
	if dataDir != "" {
		cfg.Server.DataDir = dataDir
	}
	if logLevelFlag, _ := cmd.Parent().PersistentFlags().GetString("log-level"); logLevelFlag != "" {
		cfg.Server.LogLevel = logLevelFlag
	}
	if localstackEnabled {
		cfg.LocalStack.Enabled = true
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize logging with configured level
	logging.SetLevel(logging.ParseLevel(cfg.Server.LogLevel))

	logging.Info("Starting KECS Control Plane server",
		"port", cfg.Server.Port,
		"logLevel", cfg.Server.LogLevel)
	logging.Info("Starting KECS Admin server",
		"port", cfg.Server.AdminPort)
	if kubeconfig != "" {
		logging.Info("Using kubeconfig",
			"path", kubeconfig)
	} else {
		logging.Info("Using in-cluster configuration or default kubeconfig")
	}

	// Initialize storage
	dbPath := filepath.Join(cfg.Server.DataDir, "kecs.db")
	logging.Info("Using database",
		"path", dbPath)

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(cfg.Server.DataDir, 0o755); err != nil {
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
			logging.Error("Error closing storage",
				"error", err)
		}
	}()

	logging.Info("Initialized in-memory cache",
		"ttl", cacheTTL,
		"maxSize", cacheSize)

	// Initialize the API server with LocalStack configuration
	var localstackConfig *localstack.Config
	if cfg.LocalStack.Enabled {
		logging.Info("LocalStack integration is enabled")
		localstackConfig = &cfg.LocalStack
	}

	apiServer, err := api.NewServer(cfg.Server.Port, kubeconfig, storage, localstackConfig)
	if err != nil {
		log.Fatalf("Failed to initialize API server: %v", err)
	}
	adminServer := admin.NewServer(cfg.Server.AdminPort, storage)

	// Set Kubernetes client for admin server if available
	if apiServer != nil && apiServer.GetKubeClient() != nil {
		adminServer.SetKubeClient(apiServer.GetKubeClient())
		logging.Info("Kubernetes client is available for admin server")
	} else {
		logging.Info("Kubernetes client is not available",
			"apiServerNil", apiServer == nil,
			"kubeClientNil", apiServer != nil && apiServer.GetKubeClient() == nil)
	}

	// Start webhook server if Kubernetes client is available
	var webhookServer *webhook.Server
	var webhookRegistrar *webhook.WebhookRegistrar
	if apiServer != nil && apiServer.GetKubeClient() != nil {
		logging.Info("Starting webhook server for pod mutation")

		// Create webhook configuration
		webhookConfig := webhook.Config{
			Port:      9443, // Standard webhook port
			Storage:   storage,
			Region:    cfg.AWS.DefaultRegion,
			AccountID: cfg.AWS.AccountID,
		}

		// Generate self-signed certificates for the webhook
		// In production, proper certificate management would be needed
		certPath := "/tmp/webhook-cert.pem"
		keyPath := "/tmp/webhook-key.pem"
		if err := webhook.GenerateSelfSignedCert(certPath, keyPath, "kecs-webhook.kecs-system.svc"); err != nil {
			logging.Warn("Failed to generate webhook certificates", "error", err)
			// Continue without TLS for development
			webhookConfig.CertFile = ""
			webhookConfig.KeyFile = ""
		} else {
			webhookConfig.CertFile = certPath
			webhookConfig.KeyFile = keyPath
			logging.Info("Generated self-signed certificates for webhook")
		}

		// Create webhook server
		var err error
		webhookServer, err = webhook.NewServer(webhookConfig)
		if err != nil {
			logging.Error("Failed to create webhook server", "error", err)
			// Don't fail startup, just log the error
		} else {
			// Start webhook server in goroutine
			go func() {
				webhookCtx := context.Background()
				if err := webhookServer.Start(webhookCtx); err != nil {
					logging.Error("Webhook server error", "error", err)
				}
			}()

			logging.Info("Webhook server started successfully on port", "port", webhookConfig.Port)

			// Register the webhook with Kubernetes
			if webhookConfig.CertFile != "" {
				// Read the certificate for CA bundle
				certData, err := os.ReadFile(webhookConfig.CertFile)
				if err != nil {
					logging.Error("Failed to read webhook certificate", "error", err)
				} else {
					// Create webhook registrar
					// Note: Service port is 443, which maps to targetPort 9443
					webhookRegistrar = webhook.NewWebhookRegistrar(
						apiServer.GetKubeClient(),
						"kecs-system",
						"kecs-webhook",
						443, // Service port, not the actual webhook port
					)

					// Register the webhook configuration
					webhookCtx := context.Background()
					if err := webhookRegistrar.Register(webhookCtx, certData); err != nil {
						logging.Error("Failed to register webhook configuration", "error", err)
						webhookRegistrar = nil // Clear registrar on failure
					} else {
						logging.Info("Webhook configuration registered successfully")
						webhookServer.SetReady()
					}
				}
			}
		}
	}

	// Perform state restoration from DuckDB if available
	if apiServer != nil && !apiconfig.GetBool("features.testMode") {
		logging.Info("Checking for state restoration from DuckDB...")

		// Create restoration service
		restorationService := restoration.NewService(
			storage,
			apiServer.GetTaskManager(),
			apiServer.GetServiceManager(),
			apiServer.GetLocalStackManager(),
		)

		// Perform restoration (non-blocking, best effort)
		go func() {
			// Wait for webhook to be fully registered if it's enabled
			if webhookServer != nil && webhookRegistrar != nil {
				logging.Info("Waiting for webhook registration to complete before restoration...")

				// Wait up to 10 seconds for webhook to be registered
				timeout := time.After(10 * time.Second)
				ticker := time.NewTicker(500 * time.Millisecond)
				defer ticker.Stop()

				for {
					select {
					case <-timeout:
						logging.Warn("Timeout waiting for webhook registration, proceeding with restoration")
						goto startRestoration
					case <-ticker.C:
						ctx := context.Background()
						if webhookRegistrar.IsRegistered(ctx) && webhookServer.IsReady() {
							logging.Info("Webhook is registered and ready, starting restoration")
							goto startRestoration
						}
					}
				}
			}

		startRestoration:
			restorationCtx, restorationCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer restorationCancel()

			if err := restorationService.RestoreAll(restorationCtx); err != nil {
				logging.Error("Failed to restore state from DuckDB", "error", err)
			} else {
				logging.Info("State restoration from DuckDB completed successfully")
			}
		}()
	}

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logging.Info("Received signal, shutting down gracefully...",
			"signal", sig)

		// For in-cluster mode, we need to handle preStop hook gracefully
		if serverMode == "in-cluster" {
			logging.Info("Running in-cluster shutdown sequence...")
			// Give time for endpoints to be removed from service
			time.Sleep(5 * time.Second)
		}

		cancel()

		// Allow some time for graceful shutdown
		shutdownTimeout := 10 * time.Second
		if serverMode == "in-cluster" {
			// Longer timeout for in-cluster mode to handle state persistence
			shutdownTimeout = 20 * time.Second
		}
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()

		// Persist state before shutdown
		if storage != nil {
			logging.Info("Persisting state before shutdown...")
			// Storage should handle its own graceful shutdown
		}

		// Stop all servers
		if err := apiServer.Stop(shutdownCtx); err != nil {
			logging.Error("Error during API server shutdown",
				"error", err)
		}

		if err := adminServer.Stop(shutdownCtx); err != nil {
			logging.Error("Error during Admin server shutdown",
				"error", err)
		}

		// Stop webhook server if running
		if webhookServer != nil {
			if err := webhookServer.Shutdown(); err != nil {
				logging.Error("Error during webhook server shutdown",
					"error", err)
			}
		}

		// LocalStack will be stopped by the API server during shutdown
		logging.Info("Shutdown complete")
	}()

	// Start the admin server in a goroutine
	go func() {
		if err := adminServer.Start(); err != nil && err != http.ErrServerClosed {
			logging.Error("Admin server error",
				"error", err)
			cancel() // Cancel context to trigger shutdown of API server
		}
	}()

	// Start the API server in the main goroutine
	if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
		logging.Error("API server error",
			"error", err)
		os.Exit(1)
	}

	// Wait for context cancellation
	<-ctx.Done()
}
