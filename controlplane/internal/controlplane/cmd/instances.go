package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
)

var (
	instancesConfigFile string
)

var instancesCmd = &cobra.Command{
	Use:   "instances",
	Short: "Manage multiple KECS instances",
	Long:  `List and manage multiple KECS server instances running in containers.`,
}

var instancesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all KECS instances",
	Long:  `List all KECS instances defined in configuration or currently running.`,
	RunE:  runInstancesList,
}

var instancesStartAllCmd = &cobra.Command{
	Use:   "start-all",
	Short: "Start all configured KECS instances",
	Long:  `Start all KECS instances defined in the configuration file with autoStart enabled.`,
	RunE:  runInstancesStartAll,
}

var instancesStopAllCmd = &cobra.Command{
	Use:   "stop-all",
	Short: "Stop all running KECS instances",
	Long:  `Stop all currently running KECS instances.`,
	RunE:  runInstancesStopAll,
}

func init() {
	RootCmd.AddCommand(instancesCmd)
	instancesCmd.AddCommand(instancesListCmd)
	instancesCmd.AddCommand(instancesStartAllCmd)
	instancesCmd.AddCommand(instancesStopAllCmd)

	instancesCmd.PersistentFlags().StringVar(&instancesConfigFile, "config", "", "Path to instances configuration file")
}

func runInstancesList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// Get running containers
	filters := filters.NewArgs()
	filters.Add("label", "app=kecs")
	
	runningContainers, err := cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Create a map of running containers
	runningMap := make(map[string]types.Container)
	for _, c := range runningContainers {
		name := strings.TrimPrefix(c.Names[0], "/")
		runningMap[name] = c
	}

	// Load configuration if specified
	var instancesConfig *InstancesConfig
	if instancesConfigFile != "" {
		config, err := LoadContainerConfig(instancesConfigFile)
		if err != nil {
			fmt.Printf("Warning: Failed to load config file: %v\n", err)
		} else {
			instancesConfig = config
		}
	}

	// Print header
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "INSTANCE\tSTATUS\tAPI PORT\tADMIN PORT\tIMAGE\tDATA DIR")
	fmt.Fprintln(w, strings.Repeat("-", 80))

	// Print configured instances
	if instancesConfig != nil {
		for _, inst := range instancesConfig.Instances {
			status := "configured"
			apiPort := fmt.Sprintf("%d", inst.Ports.API)
			adminPort := fmt.Sprintf("%d", inst.Ports.Admin)
			
			if running, ok := runningMap[inst.Name]; ok {
				status = running.State
				// Get actual ports from running container
				for _, p := range running.Ports {
					if p.PrivatePort == 8080 && p.PublicPort != 0 {
						apiPort = fmt.Sprintf("%d", p.PublicPort)
					} else if p.PrivatePort == 8081 && p.PublicPort != 0 {
						adminPort = fmt.Sprintf("%d", p.PublicPort)
					}
				}
				delete(runningMap, inst.Name) // Remove from map to avoid duplicate
			}
			
			defaultMark := ""
			if inst.Name == instancesConfig.DefaultInstance {
				defaultMark = " *"
			}
			
			fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\t%s\n",
				inst.Name, defaultMark, status, apiPort, adminPort, 
				truncate(inst.Image, 30), truncate(inst.DataDir, 30))
		}
	}

	// Print running containers not in configuration
	for name, c := range runningMap {
		apiPort := "-"
		adminPort := "-"
		for _, p := range c.Ports {
			if p.PrivatePort == 8080 && p.PublicPort != 0 {
				apiPort = fmt.Sprintf("%d", p.PublicPort)
			} else if p.PrivatePort == 8081 && p.PublicPort != 0 {
				adminPort = fmt.Sprintf("%d", p.PublicPort)
			}
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			name, c.State, apiPort, adminPort, 
			truncate(c.Image, 30), "-")
	}

	w.Flush()
	
	if instancesConfig != nil && instancesConfig.DefaultInstance != "" {
		fmt.Printf("\n* Default instance: %s\n", instancesConfig.DefaultInstance)
	}
	
	return nil
}

func runInstancesStartAll(cmd *cobra.Command, args []string) error {
	if instancesConfigFile == "" {
		instancesConfigFile = GetDefaultConfigPath()
	}

	config, err := LoadContainerConfig(instancesConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	var startedCount int
	var errors []string

	for _, inst := range config.Instances {
		if !inst.AutoStart {
			continue
		}

		fmt.Printf("Starting instance '%s'...\n", inst.Name)
		
		// Build start command arguments
		args := []string{
			"start",
			"--name", inst.Name,
			"--image", inst.Image,
			"--api-port", fmt.Sprintf("%d", inst.Ports.API),
			"--admin-port", fmt.Sprintf("%d", inst.Ports.Admin),
		}
		
		if inst.DataDir != "" {
			args = append(args, "--data-dir", inst.DataDir)
		}

		// Execute start command
		startCmd := &cobra.Command{}
		startCmd.SetArgs(args)
		
		if err := runStart(startCmd, []string{}); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", inst.Name, err))
		} else {
			startedCount++
		}
	}

	fmt.Printf("\nStarted %d instances\n", startedCount)
	
	if len(errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
		return fmt.Errorf("some instances failed to start")
	}

	return nil
}

func runInstancesStopAll(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// Get all running KECS containers
	containers, err := getRunningKECSContainers(ctx, cli)
	if err != nil {
		return fmt.Errorf("failed to get running containers: %w", err)
	}

	if len(containers) == 0 {
		fmt.Println("No running KECS instances found")
		return nil
	}

	var stoppedCount int
	var errors []string

	for _, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")

		fmt.Printf("Stopping instance '%s'...\n", name)
		
		// Stop container
		timeout := 10
		if err := cli.ContainerStop(ctx, c.ID, container.StopOptions{
			Timeout: &timeout,
		}); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))
			continue
		}

		// Remove container
		if err := cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{}); err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to remove: %v", name, err))
		} else {
			stoppedCount++
		}
	}

	fmt.Printf("\nStopped %d instances\n", stoppedCount)
	
	if len(errors) > 0 {
		fmt.Println("\nErrors:")
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
		return fmt.Errorf("some instances failed to stop")
	}

	return nil
}