package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/nandemo-ya/kecs/controlplane/internal/host/instance"
)

var (
	listFormat string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all KECS instances",
	Long:  `List all KECS instances with their status and configuration.`,
	RunE:  runList,
}

func init() {
	RootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&listFormat, "format", "f", "table", "Output format: table, json, yaml")
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

	if len(instances) == 0 && listFormat == "table" {
		fmt.Println("No KECS instances found")
		fmt.Println("\nCreate a new instance with: kecs start")
		return nil
	}

	// Output based on format
	switch strings.ToLower(listFormat) {
	case "json":
		return outputJSON(instances)
	case "yaml":
		return outputYAML(instances)
	case "table":
		return outputTable(instances)
	default:
		return fmt.Errorf("unsupported format: %s (supported: table, json, yaml)", listFormat)
	}
}

func outputJSON(instances []instance.InstanceInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(instances); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

func outputYAML(instances []instance.InstanceInfo) error {
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	if err := encoder.Encode(instances); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}
	return nil
}

func outputTable(instances []instance.InstanceInfo) error {
	if len(instances) == 0 {
		return nil
	}

	// Display instances in a table format
	fmt.Println("KECS Instances:")
	fmt.Println("=========================================================")
	fmt.Printf("%-20s %-10s %-10s %-10s %-10s\n", "NAME", "STATUS", "API PORT", "ADMIN PORT", "LOCALSTACK")
	fmt.Println("---------------------------------------------------------")

	for _, inst := range instances {
		status := strings.ToLower(inst.Status)
		localStack := "no"
		if inst.LocalStack {
			localStack = "yes"
		}

		fmt.Printf("%-20s %-10s %-10d %-10d %-10s\n",
			inst.Name,
			status,
			inst.ApiPort,
			inst.AdminPort,
			localStack,
		)
	}
	fmt.Println("=========================================================")

	// Show helpful commands
	fmt.Println("\nCommands:")
	fmt.Println("  Start instance:   kecs start --instance <name>")
	fmt.Println("  Stop instance:    kecs stop --instance <name>")
	fmt.Println("  Destroy instance: kecs destroy --instance <name>")
	fmt.Println("  Get kubeconfig:   kecs kubeconfig get <name>")

	return nil
}
