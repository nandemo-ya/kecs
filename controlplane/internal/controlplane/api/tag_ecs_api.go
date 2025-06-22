package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
)

// TagResource implements the TagResource operation
func (api *DefaultECSAPI) TagResource(ctx context.Context, req *generated.TagResourceRequest) (*generated.TagResourceResponse, error) {
	// Validate resource ARN
	if req.ResourceArn == nil {
		return nil, fmt.Errorf("Invalid parameter: Resource ARN is required")
	}
	if err := ValidateResourceARN(*req.ResourceArn); err != nil {
		return nil, err
	}

	// Validate tags
	if len(req.Tags) == 0 {
		return nil, fmt.Errorf("Invalid parameter: At least one tag must be specified")
	}

	// Parse resource ARN to determine resource type
	resourceArn := *req.ResourceArn
	if strings.Contains(resourceArn, ":cluster/") {
		// Extract cluster name from ARN
		parts := strings.Split(resourceArn, "/")
		if len(parts) != 2 {
			return nil, fmt.Errorf("Invalid parameter: Invalid cluster ARN format")
		}
		clusterName := parts[1]

		// Check if cluster exists
		cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
		if err != nil {
			return nil, fmt.Errorf("The cluster '%s' does not exist", clusterName)
		}

		// TODO: Actually update cluster tags in storage
		_ = cluster
	} else {
		// For other resource types, just validate they could exist
		// In a full implementation, we'd check each resource type
	}

	// For now, return an empty successful response
	resp := &generated.TagResourceResponse{}

	return resp, nil
}

// UntagResource implements the UntagResource operation
func (api *DefaultECSAPI) UntagResource(ctx context.Context, req *generated.UntagResourceRequest) (*generated.UntagResourceResponse, error) {
	// Validate resource ARN
	if req.ResourceArn == nil {
		return nil, fmt.Errorf("Invalid parameter: Resource ARN is required")
	}
	if err := ValidateResourceARN(*req.ResourceArn); err != nil {
		return nil, err
	}

	// Validate tag keys
	if len(req.TagKeys) == 0 {
		return nil, fmt.Errorf("Invalid parameter: At least one tag key must be specified")
	}

	// TODO: Implement actual resource untagging logic
	// In a real implementation, we would:
	// 1. Parse the resource ARN to determine resource type
	// 2. Validate the resource exists
	// 3. Remove the specified tags from the database
	// 4. Handle non-existent tag keys gracefully

	// For now, return an empty successful response
	resp := &generated.UntagResourceResponse{}

	return resp, nil
}

// ListTagsForResource implements the ListTagsForResource operation
func (api *DefaultECSAPI) ListTagsForResource(ctx context.Context, req *generated.ListTagsForResourceRequest) (*generated.ListTagsForResourceResponse, error) {
	// Validate resource ARN
	if req.ResourceArn == nil {
		return nil, fmt.Errorf("Invalid parameter: Resource ARN is required")
	}
	if err := ValidateResourceARN(*req.ResourceArn); err != nil {
		return nil, err
	}

	// TODO: Implement actual tag listing logic
	// In a real implementation, we would:
	// 1. Parse the resource ARN to determine resource type
	// 2. Validate the resource exists
	// 3. Retrieve tags from the database
	// 4. Return appropriate error if resource not found

	// For now, return mock tags based on resource type
	tags := []generated.Tag{}

	// Determine resource type from ARN
	resourceArn := *req.ResourceArn
	if strings.Contains(resourceArn, ":cluster/") {
		tags = append(tags, generated.Tag{
			Key:   (*generated.TagKey)(ptr.String("Environment")),
			Value: (*generated.TagValue)(ptr.String("Development")),
		})
		tags = append(tags, generated.Tag{
			Key:   (*generated.TagKey)(ptr.String("Team")),
			Value: (*generated.TagValue)(ptr.String("Platform")),
		})
	} else if strings.Contains(resourceArn, ":service/") {
		tags = append(tags, generated.Tag{
			Key:   (*generated.TagKey)(ptr.String("Application")),
			Value: (*generated.TagValue)(ptr.String("WebApp")),
		})
		tags = append(tags, generated.Tag{
			Key:   (*generated.TagKey)(ptr.String("Version")),
			Value: (*generated.TagValue)(ptr.String("1.0.0")),
		})
	} else if strings.Contains(resourceArn, ":task/") {
		tags = append(tags, generated.Tag{
			Key:   (*generated.TagKey)(ptr.String("Purpose")),
			Value: (*generated.TagValue)(ptr.String("Testing")),
		})
		tags = append(tags, generated.Tag{
			Key:   (*generated.TagKey)(ptr.String("Owner")),
			Value: (*generated.TagValue)(ptr.String("DevOps")),
		})
	} else if strings.Contains(resourceArn, ":task-definition/") {
		tags = append(tags, generated.Tag{
			Key:   (*generated.TagKey)(ptr.String("Component")),
			Value: (*generated.TagValue)(ptr.String("Backend")),
		})
		tags = append(tags, generated.Tag{
			Key:   (*generated.TagKey)(ptr.String("Language")),
			Value: (*generated.TagValue)(ptr.String("Go")),
		})
	} else if strings.Contains(resourceArn, ":container-instance/") {
		tags = append(tags, generated.Tag{
			Key:   (*generated.TagKey)(ptr.String("InstanceType")),
			Value: (*generated.TagValue)(ptr.String("t3.medium")),
		})
		tags = append(tags, generated.Tag{
			Key:   (*generated.TagKey)(ptr.String("AZ")),
			Value: (*generated.TagValue)(ptr.String("us-east-1a")),
		})
	} else if strings.Contains(resourceArn, ":capacity-provider/") {
		tags = append(tags, generated.Tag{
			Key:   (*generated.TagKey)(ptr.String("Type")),
			Value: (*generated.TagValue)(ptr.String("AutoScaling")),
		})
		tags = append(tags, generated.Tag{
			Key:   (*generated.TagKey)(ptr.String("ManagedBy")),
			Value: (*generated.TagValue)(ptr.String("ECS")),
		})
	}

	resp := &generated.ListTagsForResourceResponse{
		Tags: tags,
	}

	return resp, nil
}
