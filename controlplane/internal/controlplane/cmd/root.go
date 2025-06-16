package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Used for flags
	port     int
	logLevel string

	// RootCmd represents the base command when called without any subcommands
	RootCmd = &cobra.Command{
		Use:   "controlplane",
		Short: "KECS Control Plane - Kubernetes-based ECS Compatible Service",
		Long: `KECS Control Plane provides Amazon ECS compatible APIs running on Kubernetes.
It allows you to run ECS workloads locally or in any Kubernetes cluster without AWS dependencies.`,
		// The default command when no subcommands are specified
		Run: func(cmd *cobra.Command, args []string) {
			// This will start the server by default
			startServer()
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Define persistent flags that will be inherited by all subcommands
	RootCmd.PersistentFlags().IntVarP(&port, "port", "p", 8080, "Port to run the control plane server on")
	RootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")

	// Add subcommands
	RootCmd.AddCommand(serverCmd)
	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(localstackCmd)
}

// startServer is the default action when no subcommands are specified
func startServer() {
	fmt.Printf("Starting KECS Control Plane on port %d with log level %s\n", port, logLevel)
	// TODO: Implement actual control plane server startup logic
}
