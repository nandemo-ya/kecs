package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nandemo-ya/kecs/controlplane/internal/instance"
)

var (
	healthInstance string
	healthTimeout  time.Duration
	healthDetailed bool
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check the health of KECS instances",
	Long:  `Check the health status of all KECS instances or a specific instance.`,
	RunE:  runHealth,
}

func init() {
	RootCmd.AddCommand(healthCmd)

	healthCmd.Flags().StringVar(&healthInstance, "instance", "", "Check health of specific instance (default: all instances)")
	healthCmd.Flags().DurationVar(&healthTimeout, "timeout", 5*time.Second, "Request timeout")
	healthCmd.Flags().BoolVar(&healthDetailed, "detailed", false, "Show detailed health information")
}

func runHealth(cmd *cobra.Command, args []string) error {
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

	// Filter by instance name if specified
	if healthInstance != "" {
		var filtered []instance.InstanceInfo
		for _, inst := range instances {
			if inst.Name == healthInstance {
				filtered = append(filtered, inst)
				break
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("instance %q not found", healthInstance)
		}
		instances = filtered
	}

	if len(instances) == 0 {
		fmt.Println("No KECS instances found")
		return nil
	}

	// Check health of each instance
	type instanceHealth struct {
		Name      string                 `json:"name"`
		Status    string                 `json:"status"`
		ApiPort   int                    `json:"api_port"`
		AdminPort int                    `json:"admin_port"`
		Health    map[string]interface{} `json:"health,omitempty"`
		Error     string                 `json:"error,omitempty"`
	}

	var results []instanceHealth

	client := &http.Client{
		Timeout: healthTimeout,
	}

	for _, inst := range instances {
		result := instanceHealth{
			Name:      inst.Name,
			Status:    strings.ToLower(inst.Status),
			ApiPort:   inst.ApiPort,
			AdminPort: inst.AdminPort,
		}

		// Only check health for running instances
		if strings.ToLower(inst.Status) == "running" {
			// Choose endpoint based on detailed flag
			endpoint := "/health"
			if healthDetailed {
				endpoint = "/health/detailed"
			}

			// Build health check URL for this instance
			instanceURL := fmt.Sprintf("http://localhost:%d%s", inst.AdminPort, endpoint)

			resp, err := client.Get(instanceURL)
			if err != nil {
				result.Error = fmt.Sprintf("Failed to connect: %v", err)
			} else {
				defer resp.Body.Close()

				// Parse response
				var healthData map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&healthData); err != nil {
					result.Error = fmt.Sprintf("Failed to parse response: %v", err)
				} else {
					result.Health = healthData
				}
			}
		}

		results = append(results, result)
	}

	// Pretty print the results
	output, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format response: %w", err)
	}

	fmt.Println(string(output))

	// Check if any instance is unhealthy
	hasUnhealthy := false
	for _, result := range results {
		if result.Error != "" || (result.Health != nil && result.Health["status"] != "ok") {
			hasUnhealthy = true
			break
		}
	}

	if hasUnhealthy {
		os.Exit(1)
	}

	return nil
}
