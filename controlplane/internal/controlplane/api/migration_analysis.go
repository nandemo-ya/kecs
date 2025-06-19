package api

// This file is for analyzing differences between generated types and AWS SDK types
// It will be removed after migration is complete

import (
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)

// Type comparison for ListClusters
type MigrationAnalysis struct {
	// Generated types
	GeneratedListClustersInput  generated.ListClustersRequest
	GeneratedListClustersOutput generated.ListClustersResponse
	
	// AWS SDK types
	SDKListClustersInput  ecs.ListClustersInput
	SDKListClustersOutput ecs.ListClustersOutput
}

// This file helps us understand the structural differences between:
// 1. Our generated types from api-models/ecs.json
// 2. AWS SDK official types from github.com/aws/aws-sdk-go-v2/service/ecs