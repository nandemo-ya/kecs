package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/portforward"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	localPort   int
	targetPort  int
	allForwards bool
)

var portForwardCmd = &cobra.Command{
	Use:   "port-forward",
	Short: "Manage port forwarding for ECS services and tasks",
	Long: `Manage port forwarding from your local machine to ECS services and tasks running in KECS.

This command provides background port forwarding with automatic management and configuration file support.`,
}

// getKubeconfigPath returns the kubeconfig path for the given instance
func getKubeconfigPath(instanceName string) string {
	return fmt.Sprintf("/tmp/kecs-%s.config", instanceName)
}

// getInstanceName returns the instance name from environment or default
func getInstanceName() string {
	instanceName := config.GetString("kubernetes.instanceName")
	if instanceName == "" {
		instanceName = "default"
	}
	return instanceName
}

var portForwardStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start port forwarding",
}

var portForwardStartServiceCmd = &cobra.Command{
	Use:   "service <cluster>/<service-name>",
	Short: "Start port forwarding to an ECS service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		parts := strings.Split(args[0], "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid format: expected <cluster>/<service-name>")
		}

		cluster := parts[0]
		serviceName := parts[1]

		// Get instance name and kubeconfig path
		instanceName := getInstanceName()
		kubeconfigPath := getKubeconfigPath(instanceName)
		if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
			return fmt.Errorf("kubeconfig not found for instance %s. Is KECS running?", instanceName)
		}

		// Create Kubernetes client
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return fmt.Errorf("failed to build config: %w", err)
		}

		k8sClient, err := kubernetes.NewClient(config)
		if err != nil {
			return fmt.Errorf("failed to create Kubernetes client: %w", err)
		}

		// Create port forward manager
		manager := portforward.NewManager(instanceName, k8sClient)

		// Start port forwarding for the service
		forwardID, assignedPort, err := manager.StartServiceForward(context.Background(), cluster, serviceName, localPort, targetPort)
		if err != nil {
			return fmt.Errorf("failed to start port forward: %w", err)
		}

		fmt.Printf("Port forwarding started successfully\n")
		fmt.Printf("Forward ID: %s\n", forwardID)
		fmt.Printf("Forwarding localhost:%d -> %s/%s\n", assignedPort, cluster, serviceName)

		return nil
	},
}

var portForwardStartTaskCmd = &cobra.Command{
	Use:   "task <cluster>/<task-id>",
	Short: "Start port forwarding to an ECS task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		parts := strings.Split(args[0], "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid format: expected <cluster>/<task-id>")
		}

		cluster := parts[0]
		taskID := parts[1]

		// Get instance name and kubeconfig path
		instanceName := getInstanceName()
		kubeconfigPath := getKubeconfigPath(instanceName)
		if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
			return fmt.Errorf("kubeconfig not found for instance %s. Is KECS running?", instanceName)
		}

		// Create Kubernetes client
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return fmt.Errorf("failed to build config: %w", err)
		}

		k8sClient, err := kubernetes.NewClient(config)
		if err != nil {
			return fmt.Errorf("failed to create Kubernetes client: %w", err)
		}

		// Create port forward manager
		manager := portforward.NewManager(instanceName, k8sClient)

		// Start port forwarding for the task
		forwardID, assignedPort, err := manager.StartTaskForward(context.Background(), cluster, taskID, localPort, targetPort)
		if err != nil {
			return fmt.Errorf("failed to start port forward: %w", err)
		}

		fmt.Printf("Port forwarding started successfully\n")
		fmt.Printf("Forward ID: %s\n", forwardID)
		fmt.Printf("Forwarding localhost:%d -> %s/%s\n", assignedPort, cluster, taskID)

		return nil
	},
}

var portForwardListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active port forwards",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get instance name from environment or flag
		instanceName := config.GetString("kubernetes.instanceName")
		if instanceName == "" {
			instanceName = "default"
		}

		// Create port forward manager (k8s client not needed for list)
		manager := portforward.NewManager(instanceName, nil)

		// List all active port forwards
		forwards, err := manager.ListForwards()
		if err != nil {
			return fmt.Errorf("failed to list port forwards: %w", err)
		}

		if len(forwards) == 0 {
			fmt.Println("No active port forwards")
			return nil
		}

		// Display forwards in a table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTYPE\tTARGET\tLOCAL PORT\tSTATUS")
		fmt.Fprintln(w, "--\t----\t------\t----------\t------")

		for _, fwd := range forwards {
			target := fmt.Sprintf("%s/%s", fwd.Cluster, fwd.TargetName)
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
				fwd.ID, fwd.Type, target, fwd.LocalPort, fwd.Status)
		}

		w.Flush()
		return nil
	},
}

var portForwardStopCmd = &cobra.Command{
	Use:   "stop [forward-id]",
	Short: "Stop port forwarding",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get instance name from environment or flag
		instanceName := config.GetString("kubernetes.instanceName")
		if instanceName == "" {
			instanceName = "default"
		}

		// Create port forward manager (k8s client not needed for stop)
		manager := portforward.NewManager(instanceName, nil)

		if allForwards {
			// Stop all forwards
			err := manager.StopAllForwards()
			if err != nil {
				return fmt.Errorf("failed to stop all port forwards: %w", err)
			}
			fmt.Println("All port forwards stopped")
		} else {
			// Stop specific forward
			if len(args) == 0 {
				return fmt.Errorf("forward-id required when not using --all")
			}

			forwardID := args[0]
			err := manager.StopForward(forwardID)
			if err != nil {
				return fmt.Errorf("failed to stop port forward: %w", err)
			}
			fmt.Printf("Port forward %s stopped\n", forwardID)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(portForwardCmd)

	// Add subcommands
	portForwardCmd.AddCommand(portForwardStartCmd)
	portForwardCmd.AddCommand(portForwardListCmd)
	portForwardCmd.AddCommand(portForwardStopCmd)

	// Add start subcommands
	portForwardStartCmd.AddCommand(portForwardStartServiceCmd)
	portForwardStartCmd.AddCommand(portForwardStartTaskCmd)

	// Add flags
	portForwardStartServiceCmd.Flags().IntVar(&localPort, "local-port", 0, "Local port to forward (0 for auto-assign)")
	portForwardStartServiceCmd.Flags().IntVar(&targetPort, "target-port", 0, "Target port on the service")

	portForwardStartTaskCmd.Flags().IntVar(&localPort, "local-port", 0, "Local port to forward (0 for auto-assign)")
	portForwardStartTaskCmd.Flags().IntVar(&targetPort, "target-port", 0, "Target port on the task")

	portForwardStopCmd.Flags().BoolVar(&allForwards, "all", false, "Stop all port forwards")
}
