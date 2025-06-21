package main

import (
	"context"
	"fmt"
	"log"
	"os"

	generated_v2 "github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_v2"
)

// This example demonstrates using the generated types with our custom AWS client
// instead of the AWS SDK v2
func main() {
	// Get endpoint from environment or use default
	endpoint := os.Getenv("KECS_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8080"
	}

	// Create ECS client using generated client
	ecsClient := generated_v2.NewClient(generated_v2.ClientOptions{
		Endpoint: endpoint,
	})

	ctx := context.Background()

	// Example 1: List clusters
	fmt.Println("=== Listing Clusters ===")
	listReq := &generated_v2.ListClustersRequest{}
	
	listResp, err := ecsClient.ListClusters(ctx, listReq)
	if err != nil {
		log.Fatalf("Failed to list clusters: %v", err)
	}

	fmt.Printf("Found %d clusters\n", len(listResp.ClusterArns))
	for _, arn := range listResp.ClusterArns {
		fmt.Printf("  - %s\n", arn)
	}

	// Example 2: Create a cluster
	fmt.Println("\n=== Creating Cluster ===")
	clusterName := "test-cluster"
	createReq := &generated_v2.CreateClusterRequest{
		ClusterName: &clusterName,
		Settings: []generated_v2.ClusterSetting{
			{
				Name:  interfacePtr("containerInsights"),
				Value: stringPtr("enabled"),
			},
		},
		Tags: []generated_v2.Tag{
			{
				Key:   stringPtr("Environment"),
				Value: stringPtr("test"),
			},
		},
	}
	
	createResp, err := ecsClient.CreateCluster(ctx, createReq)
	if err != nil {
		log.Fatalf("Failed to create cluster: %v", err)
	}

	if createResp.Cluster != nil {
		fmt.Printf("Created cluster: %s\n", *createResp.Cluster.ClusterName)
		fmt.Printf("  ARN: %s\n", *createResp.Cluster.ClusterArn)
		fmt.Printf("  Status: %s\n", *createResp.Cluster.Status)
	}

	// Example 3: Describe the cluster
	fmt.Println("\n=== Describing Cluster ===")
	describeReq := &generated_v2.DescribeClustersRequest{
		Clusters: []string{clusterName},
	}
	
	describeResp, err := ecsClient.DescribeClusters(ctx, describeReq)
	if err != nil {
		log.Fatalf("Failed to describe clusters: %v", err)
	}

	for _, cluster := range describeResp.Clusters {
		fmt.Printf("Cluster: %s\n", *cluster.ClusterName)
		fmt.Printf("  Status: %s\n", *cluster.Status)
		if cluster.Settings != nil {
			fmt.Println("  Settings:")
			for _, setting := range cluster.Settings {
				fmt.Printf("    - %v: %s\n", *setting.Name, *setting.Value)
			}
		}
		if cluster.Tags != nil {
			fmt.Println("  Tags:")
			for _, tag := range cluster.Tags {
				fmt.Printf("    - %s: %s\n", *tag.Key, *tag.Value)
			}
		}
	}

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("This demonstrates using generated types without AWS SDK v2 dependencies")
}

func stringPtr(s string) *string {
	return &s
}

func interfacePtr(v interface{}) *interface{} {
	return &v
}