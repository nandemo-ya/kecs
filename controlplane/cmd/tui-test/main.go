// Test program for TUI API integration
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui"
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
var testFuncs = map[string]func(){
	"mock":        testMockAPI,
	"interactive": runInteractiveTUI,
}

func init() {
	// Register test functions
}

func testMockAPI() {
	fmt.Println("Testing TUI API with mock client")

	// Create mock client
	client := api.NewMockClient()

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
	fmt.Println("\n2. Testing GetInstance for 'dev':")
	inst, err := client.GetInstance(ctx, "dev")
	if err != nil {
		log.Printf("Error getting instance: %v", err)
	} else {
		fmt.Printf("  Name: %s\n", inst.Name)
		fmt.Printf("  Status: %s\n", inst.Status)
		fmt.Printf("  API Port: %d\n", inst.APIPort)
		fmt.Printf("  Admin Port: %d\n", inst.AdminPort)
	}

	// Test 3: Create instance
	fmt.Println("\n3. Testing CreateInstance:")
	opts := api.CreateInstanceOptions{
		Name:      "test-instance",
		APIPort:   8090,
		AdminPort: 8091,
	}
	_, err = client.CreateInstance(ctx, opts)
	if err != nil {
		log.Printf("Error creating instance: %v", err)
	}

	// Test 4: List instances again
	fmt.Println("\n4. Listing instances after creation:")
	instances, err = client.ListInstances(ctx)
	if err != nil {
		log.Printf("Error listing instances: %v", err)
	} else {
		fmt.Printf("Found %d instances:\n", len(instances))
		for _, inst := range instances {
			fmt.Printf("  - %s: %s\n", inst.Name, inst.Status)
		}
	}

	// Test 5: Delete instance
	fmt.Println("\n5. Testing DeleteInstance:")
	err = client.DeleteInstance(ctx, "test-instance")
	if err != nil {
		log.Printf("Error deleting instance: %v", err)
	}

	// Test 6: List clusters
	fmt.Println("\n6. Testing ListClusters for instance 'dev':")
	clusterArns, err := client.ListClusters(ctx, "dev")
	if err != nil {
		log.Printf("Error listing clusters: %v", err)
	} else {
		fmt.Printf("Found %d clusters:\n", len(clusterArns))
		for _, arn := range clusterArns {
			fmt.Printf("  - %s\n", arn)
		}
	}

	fmt.Println("\nMock test completed!")
}

func runInteractiveTUI() {
	// Get API endpoint from environment
	endpoint := os.Getenv("KECS_API_ENDPOINT")
	useMock := endpoint == ""

	var model tui.Model
	if useMock {
		fmt.Println("Starting interactive TUI with mock data...")
		model = tui.NewModel()
	} else {
		fmt.Printf("Starting interactive TUI connected to: %s\n", endpoint)
		// Create HTTP client for real API
		client := api.NewHTTPClient(endpoint)
		model = tui.NewModelWithClient(client)
	}

	// Create and run the Bubble Tea program
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
