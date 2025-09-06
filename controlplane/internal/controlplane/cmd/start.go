package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/host/instance"
	"github.com/nandemo-ya/kecs/controlplane/internal/host/k3d"
	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

// Error messages
const (
	errCreateClusterManager   = "failed to create cluster manager: %w"
	errCreateInstanceManager  = "failed to create instance manager: %w"
	errCheckInstanceExistence = "failed to check instance existence: %w"
	errCheckInstanceStatus    = "failed to check instance status: %w"
	errListInstances          = "failed to list instances: %w"
	errGenerateName           = "failed to generate instance name: %w"
)

// Display messages
const (
	msgInstanceAlreadyRunning = "⚠️  Instance '%s' is already running\n"
	msgRestartingInstance     = "Restarting stopped instance: %s\n"
	msgCreatingInstance       = "\n=== Creating KECS instance '%s' ===\n"
	msgInstanceReady          = "\n🎉 KECS instance '%s' is ready!\n"
	msgFetchingInstances      = "Fetching KECS instances..."
	msgCreatingNewInstance    = "Creating new KECS instance: %s\n"
	msgExistingInstances      = "\nExisting KECS instances:"
	msgUseExistingHint        = "\nTo use an existing instance, specify it with --instance flag"
	msgNextSteps              = "\n=== Next steps ==="
)

var (
	startInstanceName            string
	startDataDir                 string
	startApiPort                 int
	startAdminPort               int
	startConfigFile              string
	startAdditionalLocalServices string
	startTimeout                 time.Duration
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start KECS with control plane in k3d cluster",
	Long: `Start KECS by creating a k3d cluster and deploying the control plane inside it.
This provides a unified AWS API endpoint accessible from all containers.`,
	RunE: runStart,
}

func init() {
	RootCmd.AddCommand(startCmd)

	startCmd.Flags().StringVar(&startInstanceName, "instance", "", "KECS instance name (auto-generated if not specified)")
	startCmd.Flags().StringVar(&startDataDir, "data-dir", "", "Data directory (default: ~/.kecs/data)")
	startCmd.Flags().IntVar(&startApiPort, "api-port", 5373, "AWS API port")
	startCmd.Flags().IntVar(&startAdminPort, "admin-port", 5374, "Admin API port")
	startCmd.Flags().StringVar(&startConfigFile, "config", "", "Configuration file path")
	startCmd.Flags().StringVar(&startAdditionalLocalServices, "additional-localstack-services", "", "Additional LocalStack services (comma-separated, e.g., s3,dynamodb,sqs)")
	startCmd.Flags().DurationVar(&startTimeout, "timeout", 10*time.Minute, "Timeout for cluster creation")
}

func runStart(cmd *cobra.Command, args []string) error {
	// Create k3d cluster manager to check existing instances
	manager, err := k3d.NewK3dClusterManager(nil)
	if err != nil {
		return fmt.Errorf(errCreateClusterManager, err)
	}

	// Determine instance name and check its status
	instanceName, shouldStart, err := determineInstanceToStart(manager)
	if err != nil {
		return err
	}
	if !shouldStart {
		return nil // Instance is already running or user canceled
	}

	startInstanceName = instanceName

	// Show header
	fmt.Printf(msgCreatingInstance, startInstanceName)

	// Create instance manager
	instanceManager, err := instance.NewManager()
	if err != nil {
		return fmt.Errorf(errCreateInstanceManager, err)
	}

	// Set up start options
	opts := instance.StartOptions{
		InstanceName:                 startInstanceName,
		DataDir:                      startDataDir,
		ConfigFile:                   startConfigFile,
		AdditionalLocalStackServices: startAdditionalLocalServices,
		ApiPort:                      startApiPort,
		AdminPort:                    startAdminPort,
	}

	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	// Start the instance using the shared manager
	if err := instanceManager.Start(ctx, opts); err != nil {
		return err
	}

	// Show completion message
	showStartCompletionMessage(opts)

	return nil
}

// determineInstanceToStart handles instance selection and status checking
// Returns: (instanceName, shouldStart, error)
func determineInstanceToStart(manager *k3d.K3dClusterManager) (string, bool, error) {
	var instanceName string
	var isNew bool

	// If instance name is not provided, show selection
	if startInstanceName == "" {
		name, new, err := selectOrCreateInstance(manager)
		if err != nil {
			return "", false, err
		}
		instanceName = name
		isNew = new
	} else {
		// Check if specified instance exists
		exists, err := manager.ClusterExists(context.Background(), startInstanceName)
		if err != nil {
			return "", false, fmt.Errorf(errCheckInstanceExistence, err)
		}
		instanceName = startInstanceName
		isNew = !exists
	}

	// Check if instance is already running (only for existing instances)
	if !isNew {
		running, err := checkInstanceRunning(manager, instanceName)
		if err != nil {
			return "", false, fmt.Errorf(errCheckInstanceStatus, err)
		}
		if running {
			fmt.Printf(msgInstanceAlreadyRunning, instanceName)
			return instanceName, false, nil
		}
		// For stopped instances, we'll restart them
		fmt.Printf(msgRestartingInstance, instanceName)
	}

	return instanceName, true, nil
}

// showStartCompletionMessage displays the completion message after successful start
func showStartCompletionMessage(opts instance.StartOptions) {
	fmt.Printf(msgInstanceReady, opts.InstanceName)
	fmt.Println(msgNextSteps)
	fmt.Printf("AWS API: http://localhost:%d\n", opts.ApiPort)
	fmt.Printf("Admin API: http://localhost:%d\n", opts.AdminPort)
	fmt.Printf("Data directory: %s\n", opts.DataDir)
	fmt.Println()
	fmt.Printf("To stop this instance: kecs stop --instance %s\n", opts.InstanceName)
	fmt.Printf("To get kubeconfig: kecs kubeconfig get %s\n", opts.InstanceName)
}

// selectOrCreateInstance shows an interactive selection for existing instances or creates a new one
func selectOrCreateInstance(manager *k3d.K3dClusterManager) (string, bool, error) {
	ctx := context.Background()

	fmt.Println(msgFetchingInstances)

	// Get list of clusters
	clusters, err := manager.ListClusters(ctx)
	if err != nil {
		return "", false, fmt.Errorf(errListInstances, err)
	}

	if len(clusters) == 0 {
		return createNewInstance()
	}

	// Display existing instances
	displayExistingInstances(manager, clusters)

	// Since we can't do interactive selection without pterm,
	// we'll auto-generate a new instance name
	return createNewInstance()
}

// createNewInstance generates a new instance name
func createNewInstance() (string, bool, error) {
	generatedName, err := utils.GenerateRandomName()
	if err != nil {
		return "", false, fmt.Errorf(errGenerateName, err)
	}
	fmt.Printf(msgCreatingNewInstance, generatedName)
	return generatedName, true, nil
}

// displayExistingInstances shows the list of existing KECS instances
func displayExistingInstances(manager *k3d.K3dClusterManager, clusters []k3d.ClusterInfo) {
	fmt.Println(msgExistingInstances)
	for i, cluster := range clusters {
		status := getInstanceStatus(manager, cluster.Name)
		dataInfo := getInstanceDataInfo(cluster.Name)

		fmt.Printf("  %d. %s (%s%s)\n", i+1, cluster.Name, status, dataInfo)
	}

	fmt.Println(msgUseExistingHint)
}

// getInstanceStatus returns the status string for an instance
func getInstanceStatus(manager *k3d.K3dClusterManager, instanceName string) string {
	running, _ := checkInstanceRunning(manager, instanceName)
	if running {
		return "running"
	}
	return "stopped"
}

// getInstanceDataInfo returns data directory information for display
func getInstanceDataInfo(instanceName string) string {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".kecs", "instances", instanceName, "data")
	if _, err := os.Stat(dataDir); err == nil {
		return ", has data"
	}
	return ""
}

// checkInstanceRunning checks if a KECS instance is currently running
func checkInstanceRunning(manager *k3d.K3dClusterManager, instanceName string) (bool, error) {
	ctx := context.Background()

	// Use the new IsClusterRunning method to check status without triggering warnings
	return manager.IsClusterRunning(ctx, instanceName)
}
