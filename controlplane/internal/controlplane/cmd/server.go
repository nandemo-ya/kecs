//go:build !nocli
// +build !nocli

package cmd

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	apiconfig "github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/admin"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/restoration"
	storageTypes "github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/cache"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/postgres"
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
	// Check environment variable if configFile is not set via flag
	if configFile == "" {
		if envConfig := os.Getenv("KECS_CONFIG_PATH"); envConfig != "" {
			configFile = envConfig
		}
	}
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

	// Initialize storage backend - PostgreSQL as default
	var dbStorage storageTypes.Storage
	ctx := context.Background()

	// PostgreSQL storage (default) - runs as sidecar at localhost:5432
	// Use fixed connection string for sidecar pattern
	databaseURL := "postgres://kecs:kecs-postgres-2024@localhost:5432/kecs?sslmode=disable"

	// Allow override from configuration for backwards compatibility
	if configURL := config.GetString("database.url"); configURL != "" {
		databaseURL = configURL
	}

	// Mask password in logs using proper URL parsing
	maskedURL := databaseURL
	if parsed, err := url.Parse(databaseURL); err == nil && parsed.User != nil {
		if _, hasPassword := parsed.User.Password(); hasPassword {
			// Create a copy with masked password
			userInfo := url.UserPassword(parsed.User.Username(), "***")
			parsed.User = userInfo
			maskedURL = parsed.String()
		}
	}
	logging.Info("Using PostgreSQL storage", "url", maskedURL)

	dbStorage = postgres.NewPostgreSQLStorage(databaseURL)
	if err := dbStorage.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize PostgreSQL storage: %v", err)
	}

	// Wrap storage with cache layer
	// Default cache settings: 5 minute TTL, 10000 max items
	cacheTTL := 5 * time.Minute
	cacheSize := 10000
	cachedStorage := cache.NewCachedStorage(dbStorage, cacheSize, cacheTTL)

	defer func() {
		if err := cachedStorage.Close(); err != nil {
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

	apiServer, err := api.NewServer(cfg.Server.Port, kubeconfig, cachedStorage, localstackConfig)
	if err != nil {
		log.Fatalf("Failed to initialize API server: %v", err)
	}
	adminServer := admin.NewServer(cfg.Server.AdminPort, cachedStorage)

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
			Storage:   cachedStorage,
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

	// Deploy global Traefik if Kubernetes client is available
	if apiServer != nil && apiServer.GetKubeClient() != nil {
		logging.Info("Deploying global Traefik for ALB support...")

		traefikManager := kubernetes.NewTraefikManager(apiServer.GetKubeClient())

		// Check if already deployed
		if !traefikManager.IsDeployed(context.Background()) {
			if err := traefikManager.DeployGlobalTraefik(context.Background()); err != nil {
				logging.Error("Failed to deploy global Traefik", "error", err)
				// Don't fail startup, continue without Traefik
			} else {
				logging.Info("Global Traefik deployed successfully")
			}
		} else {
			logging.Info("Global Traefik is already deployed")
		}
	}

	// Perform state restoration from DuckDB if available
	if apiServer != nil && !apiconfig.GetBool("features.testMode") {
		logging.Info("Checking for state restoration from DuckDB...")

		// Create restoration service
		restorationService := restoration.NewService(
			cachedStorage,
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
						// Check if webhook configuration is registered
						if !webhookRegistrar.IsRegistered(ctx) {
							logging.Debug("Webhook configuration not yet registered")
							continue
						}

						// Check if webhook server reports ready
						if !webhookServer.IsReady() {
							logging.Debug("Webhook server not yet ready")
							continue
						}

						// Check if webhook service endpoints are available
						if apiServer.GetKubeClient() != nil {
							endpoints, err := apiServer.GetKubeClient().CoreV1().Endpoints("kecs-system").Get(ctx, "kecs-webhook", metav1.GetOptions{})
							if err != nil || endpoints == nil || len(endpoints.Subsets) == 0 {
								logging.Debug("Webhook service endpoints not yet available")
								continue
							}

							// Check if endpoints have addresses
							hasAddresses := false
							for _, subset := range endpoints.Subsets {
								if len(subset.Addresses) > 0 {
									hasAddresses = true
									break
								}
							}
							if !hasAddresses {
								logging.Debug("Webhook service endpoints have no addresses")
								continue
							}
						}

						// All checks passed, add a small delay for Kubernetes API server sync
						logging.Info("Webhook is registered and endpoints are ready, waiting for API server sync...")
						time.Sleep(1 * time.Second)
						logging.Info("Starting restoration")
						goto startRestoration
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
		if cachedStorage != nil {
			logging.Info("Persisting state before shutdown...")

			// Mark all running/pending tasks as stopped before shutdown
			logging.Info("Marking running tasks as stopped...")
			clusters, err := cachedStorage.ClusterStore().List(shutdownCtx)
			if err != nil {
				logging.Warn("Failed to list clusters for shutdown cleanup", "error", err)
			} else if len(clusters) > 0 {
				for _, cluster := range clusters {
					// Get all running or pending tasks
					tasks, err := cachedStorage.TaskStore().List(shutdownCtx, cluster.ARN, storageTypes.TaskFilters{
						MaxResults: 1000,
					})
					if err != nil {
						logging.Warn("Failed to list tasks for cluster",
							"cluster", cluster.Name, "error", err)
						continue
					}

					// Update tasks that are still running or pending
					now := time.Now()
					updatedCount := 0
					for _, task := range tasks {
						if task.LastStatus == "RUNNING" || task.LastStatus == "PENDING" {
							task.LastStatus = "STOPPED"
							task.DesiredStatus = "STOPPED"
							task.StoppedAt = &now
							task.StoppedReason = "KECS instance shutdown"
							task.Version++

							if err := cachedStorage.TaskStore().Update(shutdownCtx, task); err != nil {
								logging.Warn("Failed to update task status to STOPPED",
									"task", task.ARN, "error", err)
							} else {
								updatedCount++
							}
						}
					}

					if updatedCount > 0 {
						logging.Info("Marked tasks as stopped",
							"cluster", cluster.Name,
							"count", updatedCount)
					}
				}
			}
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

		// LocalStack persists across controlplane restarts
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
