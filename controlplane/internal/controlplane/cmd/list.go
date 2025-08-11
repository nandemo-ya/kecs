package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/instance"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all KECS instances",
	Long:  `List all KECS instances with their status and configuration.`,
	RunE:  runList,
}

func init() {
	RootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Create instance manager
	manager, err := instance.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create instance manager: %w", err)
	}

	// Get list of instances
	instances, err := manager.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}
	
	if len(instances) == 0 {
		fmt.Println("No KECS instances found")
		fmt.Println("\nCreate a new instance with: kecs start")
		return nil
	}
	
	// Display instances in a table format
	fmt.Println("KECS Instances:")
	fmt.Println("===============================================================================")
	fmt.Printf("%-20s %-10s %-10s %-10s %-8s %-10s %-8s\n", "NAME", "STATUS", "API PORT", "ADMIN PORT", "DEV MODE", "LOCALSTACK", "TRAEFIK")
	fmt.Println("-------------------------------------------------------------------------------")
	
	for _, inst := range instances {
		status := strings.ToLower(inst.Status)
		devMode := "no"
		if inst.DevMode {
			devMode = "yes"
		}
		localStack := "no"
		if inst.LocalStack {
			localStack = "yes"
		}
		traefik := "no"
		if inst.Traefik {
			traefik = "yes"
		}
		
		fmt.Printf("%-20s %-10s %-10d %-10d %-8s %-10s %-8s\n",
			inst.Name,
			status,
			inst.ApiPort,
			inst.AdminPort,
			devMode,
			localStack,
			traefik,
		)
	}
	fmt.Println("===============================================================================")
	
	// Show helpful commands
	fmt.Println("\nCommands:")
	fmt.Println("  Start instance:   kecs start --instance <name>")
	fmt.Println("  Stop instance:    kecs stop --instance <name>")
	fmt.Println("  Destroy instance: kecs destroy --instance <name>")
	fmt.Println("  Get kubeconfig:   kecs kubeconfig get <name>")
	
	return nil
}