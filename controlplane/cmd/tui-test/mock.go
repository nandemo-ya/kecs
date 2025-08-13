// Test program for TUI API with mock client
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
)

func testMockClientBasic() {
	fmt.Println("Testing TUI API with mock client")

	// Create mock client
	client := api.NewMockClient()

	// Test context
	ctx := context.Background()

	// Test 1: List instances
	fmt.Println("\n1. Testing ListInstances:")
	instances, err := client.ListInstances(ctx)
	if err != nil {
		log.Printf("Error listing instances: %v", err)
	} else {
		fmt.Printf("Found %d instances:\n", len(instances))
		for _, inst := range instances {
			fmt.Printf("  - %s: %s (clusters=%d, services=%d, tasks=%d, age=%s)\n",
				inst.Name, inst.Status, inst.Clusters, inst.Services, inst.Tasks,
				time.Since(inst.CreatedAt).Round(time.Hour))
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
		}
	}

	// Test 3: Create instance
	fmt.Println("\n3. Testing CreateInstance:")
	newInst, err := client.CreateInstance(ctx, api.CreateInstanceOptions{
		Name:       "test-instance",
		APIPort:    8090,
		AdminPort:  8091,
		LocalStack: true,
		Traefik:    false,
		DevMode:    false,
	})
	if err != nil {
		log.Printf("Error creating instance: %v", err)
	} else {
		fmt.Printf("Created instance: %s (status=%s)\n", newInst.Name, newInst.Status)
	}

	// Wait a bit for status to change
	time.Sleep(3 * time.Second)

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
	} else {
		fmt.Println("Instance deleted successfully")
	}

	// Test 6: List clusters
	if len(instances) > 0 {
		fmt.Printf("\n6. Testing ListClusters for instance '%s':\n", instances[0].Name)
		clusterArns, err := client.ListClusters(ctx, instances[0].Name)
		if err != nil {
			log.Printf("Error listing clusters: %v", err)
		} else {
			fmt.Printf("Found %d clusters:\n", len(clusterArns))
			for _, arn := range clusterArns {
				fmt.Printf("  - %s\n", arn)
			}

			// Describe clusters
			if len(clusterArns) > 0 {
				clusters, err := client.DescribeClusters(ctx, instances[0].Name, clusterArns)
				if err != nil {
					log.Printf("Error describing clusters: %v", err)
				} else {
					for _, cluster := range clusters {
						fmt.Printf("    %s: %s (services=%d, tasks=%d)\n",
							cluster.ClusterName, cluster.Status,
							cluster.ActiveServicesCount, cluster.RunningTasksCount)
					}
				}
			}
		}
	}

	fmt.Println("\nMock test completed!")
}

func testMockClient() {
	testMockClientBasic()
}

func init() {
	testFuncs["mock"] = testMockClient
}
