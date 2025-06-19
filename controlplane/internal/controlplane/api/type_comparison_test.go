package api

import (
	"testing"
	
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
)

// TestTypeComparison demonstrates the differences between generated and SDK types
func TestTypeComparison(t *testing.T) {
	// Example 1: ListClusters Request
	// Generated version
	genListReq := &generated.ListClustersRequest{
		MaxResults: ptr.Int32(10),
		NextToken:  ptr.String("next-token"),
	}
	
	// AWS SDK version
	sdkListReq := &ecs.ListClustersInput{
		MaxResults: aws.Int32(10),
		NextToken:  aws.String("next-token"),
	}
	
	t.Logf("Generated ListClustersRequest: %+v", genListReq)
	t.Logf("SDK ListClustersInput: %+v", sdkListReq)
	
	// Example 2: Cluster type
	// Generated version
	genCluster := &generated.Cluster{
		ClusterName: ptr.String("test-cluster"),
		ClusterArn:  ptr.String("arn:aws:ecs:region:account:cluster/test-cluster"),
		Status:      ptr.String("ACTIVE"),
		Tags:        []generated.Tag{},
	}
	
	// AWS SDK version
	sdkCluster := &ecstypes.Cluster{
		ClusterName: aws.String("test-cluster"),
		ClusterArn:  aws.String("arn:aws:ecs:region:account:cluster/test-cluster"),
		Status:      aws.String("ACTIVE"),
		Tags:        []ecstypes.Tag{},
	}
	
	t.Logf("Generated Cluster: %+v", genCluster)
	t.Logf("SDK Cluster: %+v", sdkCluster)
	
	// Key differences observed:
	// 1. Package naming: generated vs ecs/types
	// 2. Input/Output suffix: Request/Response vs Input/Output
	// 3. Pointer helpers: ptr.String() vs aws.String()
	// 4. Otherwise, the structures are very similar
}