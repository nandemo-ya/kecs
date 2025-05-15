package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nandemo-ya/kecs/internal/controlplane/admin"
	"github.com/nandemo-ya/kecs/internal/controlplane/api"
	"github.com/spf13/cobra"
)

var (
	// Additional server-specific flags
	kubeconfig string
	adminPort  int

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
}

func runServer() {
	fmt.Printf("Starting KECS Control Plane server on port %d with log level %s\n", port, logLevel)
	fmt.Printf("Starting KECS Admin server on port %d\n", adminPort)
	if kubeconfig != "" {
		fmt.Printf("Using kubeconfig: %s\n", kubeconfig)
	} else {
		fmt.Println("Using in-cluster configuration or default kubeconfig")
	}

	// Initialize the API and Admin servers
	apiServer := api.NewServer(port, kubeconfig)
	adminServer := admin.NewServer(adminPort)
	
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
