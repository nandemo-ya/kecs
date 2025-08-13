// Test program for TUI API integration
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
)

func main() {
	// Check if a specific test was requested
	if len(os.Args) > 1 {
		testName := os.Args[1]
		if testFunc, ok := testFuncs[testName]; ok {
			testFunc()
			return
		} else {
			fmt.Printf("Unknown test: %s\n", testName)
			fmt.Println("Available tests:")
			for name := range testFuncs {
				fmt.Printf("  - %s\n", name)
			}
			return
		}
	}

	// Get API endpoint from environment
	endpoint := os.Getenv("KECS_API_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8081"
	}

	fmt.Printf("Testing TUI API client with endpoint: %s\n", endpoint)

	// Create HTTP client
	client := api.NewHTTPClient(endpoint)

	// Test context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test 1: List instances
	fmt.Println("\n1. Testing ListInstances:")
	instances, err := client.ListInstances(ctx)
	if err != nil {
		log.Printf("Error listing instances: %v", err)
	} else {
		fmt.Printf("Found %d instances:\n", len(instances))
		for _, inst := range instances {
			fmt.Printf("  - %s: %s (clusters=%d, services=%d, tasks=%d)\n",
				inst.Name, inst.Status, inst.Clusters, inst.Services, inst.Tasks)
		}
	}

	// Test 2: Get specific instance
	if len(instances) > 0 {
		fmt.Printf("\n2. Testing GetInstance for '%s':\n", instances[0].Name)
		inst, err := client.GetInstance(ctx, instances[0].Name)
		if err != nil {
			log.Printf("Error getting instance: %v", err)
		} else {
			fmt.Printf("  Name: %s\n", inst.Name)
			fmt.Printf("  Status: %s\n", inst.Status)
			fmt.Printf("  API Port: %d\n", inst.APIPort)
			fmt.Printf("  Admin Port: %d\n", inst.AdminPort)
			fmt.Printf("  Created: %s\n", inst.CreatedAt.Format(time.RFC3339))
		}
	}

	// Test 3: List clusters
	if len(instances) > 0 {
		fmt.Printf("\n3. Testing ListClusters for instance '%s':\n", instances[0].Name)
		clusterArns, err := client.ListClusters(ctx, instances[0].Name)
		if err != nil {
			log.Printf("Error listing clusters: %v", err)
		} else {
			fmt.Printf("Found %d clusters:\n", len(clusterArns))
			for _, arn := range clusterArns {
				fmt.Printf("  - %s\n", arn)
			}
		}
	}

	fmt.Println("\nTest completed!")
}

// Test function registry
var testFuncs = map[string]func(){}

func init() {
	// No longer auto-run mock test
}
