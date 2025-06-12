package api

import (
	"context"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)

// HTTP Handlers for ECS Cluster operations have been moved to generated/routing.go
// and the implementation is now in service_impl.go

// DEPRECATED: CreateClusterWithStorage - use DefaultECSAPI.CreateCluster instead
// This method is kept temporarily for compatibility but will be removed
func (s *Server) CreateClusterWithStorage(ctx context.Context, req *generated.CreateClusterRequest) (*generated.CreateClusterResponse, error) {
	return s.ecsAPI.CreateCluster(ctx, req)
}

// DEPRECATED: ListClustersWithStorage - use DefaultECSAPI.ListClusters instead
func (s *Server) ListClustersWithStorage(ctx context.Context, req *generated.ListClustersRequest) (*generated.ListClustersResponse, error) {
	return s.ecsAPI.ListClusters(ctx, req)
}

// DEPRECATED: DescribeClustersWithStorage - use DefaultECSAPI.DescribeClusters instead
func (s *Server) DescribeClustersWithStorage(ctx context.Context, req *generated.DescribeClustersRequest) (*generated.DescribeClustersResponse, error) {
	return s.ecsAPI.DescribeClusters(ctx, req)
}

// DEPRECATED: DeleteClusterWithStorage - use DefaultECSAPI.DeleteCluster instead
func (s *Server) DeleteClusterWithStorage(ctx context.Context, req *generated.DeleteClusterRequest) (*generated.DeleteClusterResponse, error) {
	return s.ecsAPI.DeleteCluster(ctx, req)
}

// Helper functions

func extractClusterName(nameOrArn string) string {
	// If it's an ARN, extract the cluster name
	// Format: arn:aws:ecs:region:account-id:cluster/cluster-name
	if len(nameOrArn) > 0 && nameOrArn[:3] == "arn" {
		parts := splitARN(nameOrArn)
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return nameOrArn
}

func splitARN(arn string) []string {
	// Simple ARN parser
	parts := []string{}
	segments := []string{}
	current := ""
	
	for _, ch := range arn {
		if ch == ':' || ch == '/' {
			if current != "" {
				segments = append(segments, current)
			}
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		segments = append(segments, current)
	}
	
	if len(segments) >= 6 {
		// Get the resource type/name part
		if len(segments) > 6 {
			parts = segments[6:]
		}
	}
	
	return parts
}

// DEPRECATED: buildDescribeClustersResponse is no longer used