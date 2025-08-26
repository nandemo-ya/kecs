package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui"
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
It allows you to run ECS workloads locally or in any Kubernetes cluster without AWS dependencies.

When run without arguments, launches the interactive terminal user interface (TUI).`,
		// The default command when no subcommands are specified
		RunE: func(cmd *cobra.Command, args []string) error {
			// Launch TUI by default when no subcommands are provided
			// Initialize config if not already done
			if err := config.InitConfig(); err != nil {
				return err
			}

			// Run the TUI
			return tui.Run()
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
	RootCmd.PersistentFlags().IntVarP(&port, "port", "p", 5373, "Port to run the control plane server on")
	RootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")

	// Add subcommands
	RootCmd.AddCommand(serverCmd)
	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(localstackCmd)
}
