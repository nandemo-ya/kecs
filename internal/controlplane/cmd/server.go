package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nandemo-ya/kecs/internal/controlplane/api"
	"github.com/spf13/cobra"
)

var (
	// Additional server-specific flags
	kubeconfig string

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
}

func runServer() {
	fmt.Printf("Starting KECS Control Plane server on port %d with log level %s\n", port, logLevel)
	if kubeconfig != "" {
		fmt.Printf("Using kubeconfig: %s\n", kubeconfig)
	} else {
		fmt.Println("Using in-cluster configuration or default kubeconfig")
	}

	// Initialize and start the API server
	apiServer := api.NewServer(port, kubeconfig)
	
	// Set up graceful shutdown
	_, cancel := context.WithCancel(context.Background())
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
		
		if err := apiServer.Stop(shutdownCtx); err != nil {
			fmt.Printf("Error during server shutdown: %v\n", err)
		}
	}()
	
	// Start the server
	if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("Server error: %v\n", err)
		os.Exit(1)
	}
}
