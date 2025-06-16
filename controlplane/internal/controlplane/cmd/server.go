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

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/admin"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/duckdb"
	"github.com/spf13/cobra"
)

var (
	// Additional server-specific flags
	kubeconfig string
	adminPort  int
	dataDir    string

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

func init() {
	// Add server-specific flags
	serverCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig file (default is $HOME/.kube/config)")
	serverCmd.Flags().IntVar(&adminPort, "admin-port", 8081, "Port for the admin server")
	// Default to user's home directory
	defaultDataDir := "~/.kecs/data"
	if home, err := os.UserHomeDir(); err == nil {
		defaultDataDir = filepath.Join(home, ".kecs", "data")
	}
	serverCmd.Flags().StringVar(&dataDir, "data-dir", defaultDataDir, "Directory for storing persistent data")
}

func runServer() {
	// Log test mode status for debugging
	if os.Getenv("KECS_TEST_MODE") == "true" {
		fmt.Println("KECS_TEST_MODE is enabled - running in test mode")
	}

	fmt.Printf("Starting KECS Control Plane server on port %d with log level %s\n", port, logLevel)
	fmt.Printf("Starting KECS Admin server on port %d\n", adminPort)
	if kubeconfig != "" {
		fmt.Printf("Using kubeconfig: %s\n", kubeconfig)
	} else {
		fmt.Println("Using in-cluster configuration or default kubeconfig")
	}

	// Initialize storage
	dbPath := filepath.Join(dataDir, "kecs.db")
	fmt.Printf("Using database: %s\n", dbPath)

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	storage, err := duckdb.NewDuckDBStorage(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer func() {
		if err := storage.Close(); err != nil {
			log.Printf("Error closing storage: %v", err)
		}
	}()

	// Initialize storage tables
	ctx := context.Background()
	if err := storage.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize storage tables: %v", err)
	}

	// Initialize the API and Admin servers
	// TODO: Load LocalStack configuration from config file or environment
	apiServer, err := api.NewServer(port, kubeconfig, storage, nil)
	if err != nil {
		log.Fatalf("Failed to initialize API server: %v", err)
	}
	adminServer := admin.NewServer(adminPort, storage)

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
