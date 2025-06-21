package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/nandemo-ya/kecs/controlplane/internal/awsclient"
	"github.com/nandemo-ya/kecs/controlplane/internal/awsclient/services/ecs"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)

// This example demonstrates using the generated types with our custom AWS client
// instead of the AWS SDK v2
func main() {
	// Get endpoint from environment or use default
	endpoint := os.Getenv("KECS_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8080"
	}

	// Create ECS client using our custom AWS client
	ecsClient := ecs.NewClient(awsclient.Config{
		Endpoint: endpoint,
		Region:   "ap-northeast-1",
		Credentials: awsclient.Credentials{
			AccessKeyID:     "test",
			SecretAccessKey: "test",
		},
	})

	ctx := context.Background()

	// Example 1: List clusters
	fmt.Println("=== Listing Clusters ===")
	listReq := &generated.ListClustersRequest{}
	
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
	settingName := generated.ClusterSettingName("containerInsights")
	tagKey := generated.TagKey("Environment")
	tagValue := generated.TagValue("test")
	createReq := &generated.CreateClusterRequest{
		ClusterName: &clusterName,
		Settings: []generated.ClusterSetting{
			{
				Name:  &settingName,
				Value: stringPtr("enabled"),
			},
		},
		Tags: []generated.Tag{
			{
				Key:   &tagKey,
				Value: &tagValue,
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
	describeReq := &generated.DescribeClustersRequest{
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
				if setting.Name != nil && setting.Value != nil {
					fmt.Printf("    - %s: %s\n", string(*setting.Name), *setting.Value)
				}
			}
		}
		if cluster.Tags != nil {
			fmt.Println("  Tags:")
			for _, tag := range cluster.Tags {
				if tag.Key != nil && tag.Value != nil {
					fmt.Printf("    - %s: %s\n", string(*tag.Key), string(*tag.Value))
				}
			}
		}
	}

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("This demonstrates using generated types without AWS SDK v2 dependencies")
}

func stringPtr(s string) *string {
	return &s
}

