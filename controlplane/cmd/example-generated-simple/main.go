package main

import (
	"fmt"
	"os"

	"github.com/nandemo-ya/kecs/controlplane/internal/awsclient"
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

	// Create AWS client with our custom implementation
	client := awsclient.NewClient(awsclient.Config{
		Endpoint: endpoint,
		Region:   "ap-northeast-1",
		Credentials: awsclient.Credentials{
			AccessKeyID:     "test",
			SecretAccessKey: "test",
		},
	})

	// Example: Create and use ECS client methods directly
	fmt.Println("=== Using Generated Types ===")

	// List clusters using generated request type
	listReq := &generated_v2.ListClustersRequest{}
	fmt.Printf("Request type: %T\n", listReq)

	// This demonstrates how the generated types would be used
	// In a real implementation, you would call:
	// resp, err := ecsClient.ListClusters(ctx, listReq)

	// Create cluster request with generated types
	clusterName := "test-cluster"
	createReq := &generated_v2.CreateClusterRequest{
		ClusterName: &clusterName,
	}
	fmt.Printf("Create request type: %T\n", createReq)

	// Demonstrate the custom client can be used for raw AWS API calls
	signerOpts := awsclient.SignerOptions{
		Service: "ecs",
		Region:  "ap-northeast-1",
	}
	signer := awsclient.NewSigner(client.GetCredentials(), signerOpts)
	fmt.Printf("Signer created: %T\n", signer)

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("This demonstrates:")
	fmt.Println("1. Generated types are available and can be instantiated")
	fmt.Println("2. Custom AWS client can be created without SDK dependencies")
	fmt.Println("3. AWS Signature V4 signer is available for authentication")
}