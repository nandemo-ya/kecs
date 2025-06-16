package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	localstackVersion string
	localstackImage   string
)

var localstackCmd = &cobra.Command{
	Use:   "localstack",
	Short: "Manage LocalStack integration",
	Long:  `Manage LocalStack integration for AWS service emulation`,
}

var localstackStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start LocalStack",
	Long:  `Start LocalStack with the configured services`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		
		manager, err := getLocalStackManager()
		if err != nil {
			return err
		}

		fmt.Println("Starting LocalStack...")
		if err := manager.Start(ctx); err != nil {
			return fmt.Errorf("failed to start LocalStack: %w", err)
		}

		fmt.Println("LocalStack started successfully")
		return nil
	},
}

var localstackStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop LocalStack",
	Long:  `Stop LocalStack and clean up resources`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		
		manager, err := getLocalStackManager()
		if err != nil {
			return err
		}

		fmt.Println("Stopping LocalStack...")
		if err := manager.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop LocalStack: %w", err)
		}

		fmt.Println("LocalStack stopped successfully")
		return nil
	},
}

var localstackStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get LocalStack status",
	Long:  `Get the current status of LocalStack`,
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := getLocalStackManager()
		if err != nil {
			return err
		}

		status, err := manager.GetStatus()
		if err != nil {
			return fmt.Errorf("failed to get status: %w", err)
		}

		// Print status
		fmt.Printf("LocalStack Status:\n")
		fmt.Printf("  Running: %v\n", status.Running)
		fmt.Printf("  Healthy: %v\n", status.Healthy)
		if status.Running {
			fmt.Printf("  Endpoint: %s\n", status.Endpoint)
			fmt.Printf("  Uptime: %s\n", status.Uptime)
			if status.Version != "" {
				fmt.Printf("  Version: %s\n", status.Version)
			}
		}

		// Print enabled services
		if len(status.EnabledServices) > 0 {
			fmt.Printf("\nEnabled Services:\n")
			for _, service := range status.EnabledServices {
				fmt.Printf("  - %s\n", service)
			}
		}

		// Print service health
		if len(status.ServiceStatus) > 0 {
			fmt.Printf("\nService Health:\n")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "  SERVICE\tHEALTHY\tENDPOINT\n")
			for _, service := range status.ServiceStatus {
				fmt.Fprintf(w, "  %s\t%v\t%s\n", service.Name, service.Healthy, service.Endpoint)
			}
			w.Flush()
		}

		return nil
	},
}

var localstackRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart LocalStack",
	Long:  `Restart LocalStack with the current configuration`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		
		manager, err := getLocalStackManager()
		if err != nil {
			return err
		}

		fmt.Println("Restarting LocalStack...")
		if err := manager.Restart(ctx); err != nil {
			return fmt.Errorf("failed to restart LocalStack: %w", err)
		}

		fmt.Println("LocalStack restarted successfully")
		return nil
	},
}

var localstackEnableCmd = &cobra.Command{
	Use:   "enable [services...]",
	Short: "Enable LocalStack services",
	Long:  `Enable additional LocalStack services (e.g., s3, dynamodb, rds)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("no services specified")
		}

		manager, err := getLocalStackManager()
		if err != nil {
			return err
		}

		// Get current services
		currentServices, err := manager.GetEnabledServices()
		if err != nil {
			return fmt.Errorf("failed to get current services: %w", err)
		}

		// Add new services
		serviceMap := make(map[string]bool)
		for _, service := range currentServices {
			serviceMap[service] = true
		}

		for _, service := range args {
			service = strings.ToLower(strings.TrimSpace(service))
			if !localstack.IsValidService(service) {
				return fmt.Errorf("invalid service: %s", service)
			}
			serviceMap[service] = true
		}

		// Convert back to slice
		services := make([]string, 0, len(serviceMap))
		for service := range serviceMap {
			services = append(services, service)
		}

		// Update services
		if err := manager.UpdateServices(services); err != nil {
			return fmt.Errorf("failed to update services: %w", err)
		}

		fmt.Printf("Enabled services: %s\n", strings.Join(args, ", "))
		return nil
	},
}

var localstackDisableCmd = &cobra.Command{
	Use:   "disable [services...]",
	Short: "Disable LocalStack services",
	Long:  `Disable LocalStack services`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("no services specified")
		}

		manager, err := getLocalStackManager()
		if err != nil {
			return err
		}

		// Get current services
		currentServices, err := manager.GetEnabledServices()
		if err != nil {
			return fmt.Errorf("failed to get current services: %w", err)
		}

		// Remove services
		serviceMap := make(map[string]bool)
		for _, service := range currentServices {
			serviceMap[service] = true
		}

		for _, service := range args {
			service = strings.ToLower(strings.TrimSpace(service))
			delete(serviceMap, service)
		}

		// Convert back to slice
		services := make([]string, 0, len(serviceMap))
		for service := range serviceMap {
			services = append(services, service)
		}

		// Update services
		if err := manager.UpdateServices(services); err != nil {
			return fmt.Errorf("failed to update services: %w", err)
		}

		fmt.Printf("Disabled services: %s\n", strings.Join(args, ", "))
		return nil
	},
}

var localstackServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "List available LocalStack services",
	Long:  `List all available LocalStack services`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Available LocalStack services:")
		
		services := []string{
			"s3", "iam", "logs", "ssm", "secretsmanager", 
			"elbv2", "rds", "dynamodb",
		}

		for _, service := range services {
			fmt.Printf("  - %s\n", service)
		}

		return nil
	},
}

func init() {
	// Add persistent flags
	localstackCmd.PersistentFlags().StringVar(&localstackVersion, "version", "", "LocalStack version to use (default: latest)")
	localstackCmd.PersistentFlags().StringVar(&localstackImage, "image", "", "LocalStack image to use (default: localstack/localstack)")
	
	// Add subcommands
	localstackCmd.AddCommand(localstackStartCmd)
	localstackCmd.AddCommand(localstackStopCmd)
	localstackCmd.AddCommand(localstackStatusCmd)
	localstackCmd.AddCommand(localstackRestartCmd)
	localstackCmd.AddCommand(localstackEnableCmd)
	localstackCmd.AddCommand(localstackDisableCmd)
	localstackCmd.AddCommand(localstackServicesCmd)
}

// getLocalStackManager creates a LocalStack manager instance
func getLocalStackManager() (localstack.Manager, error) {
	// Get Kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Create LocalStack configuration
	lsConfig := localstack.DefaultConfig()
	lsConfig.Enabled = true
	
	// Override version if specified
	if localstackVersion != "" {
		lsConfig.Version = localstackVersion
	}
	
	// Override image if specified
	if localstackImage != "" {
		lsConfig.Image = localstackImage
	}

	// Create and return manager
	manager, err := localstack.NewManager(lsConfig, clientset)
	if err != nil {
		return nil, fmt.Errorf("failed to create LocalStack manager: %w", err)
	}

	return manager, nil
}