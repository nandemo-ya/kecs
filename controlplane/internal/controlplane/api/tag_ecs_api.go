package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)

// TagResource implements the TagResource operation
func (api *DefaultECSAPI) TagResource(ctx context.Context, req *generated.TagResourceRequest) (*generated.TagResourceResponse, error) {
	// Validate resource ARN
	if err := ValidateResourceARN(req.ResourceArn); err != nil {
		return nil, err
	}

	// Validate tags
	if len(req.Tags) == 0 {
		return nil, fmt.Errorf("Invalid parameter: At least one tag must be specified")
	}

	// Parse resource ARN to determine resource type
	resourceArn := req.ResourceArn
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

		// Parse existing tags
		existingTags, err := parseTags(clusterName, cluster.Tags)
		if err != nil {
			return nil, fmt.Errorf("failed to parse existing tags: %w", err)
		}

		// Convert existing tags to a map for easier manipulation
		tagMap := make(map[string]string)
		for _, tag := range existingTags {
			if tag.Key != nil && tag.Value != nil {
				tagMap[*tag.Key] = *tag.Value
			}
		}

		// Add/update new tags
		for _, tag := range req.Tags {
			if tag.Key != nil && tag.Value != nil {
				tagMap[*tag.Key] = *tag.Value
			}
		}

		// Convert back to tag array
		var updatedTags []generated.Tag
		for k, v := range tagMap {
			key := k
			value := v
			updatedTags = append(updatedTags, generated.Tag{
				Key:   &key,
				Value: &value,
			})
		}

		// Marshal tags to JSON
		tagsJSON, err := json.Marshal(updatedTags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tags: %w", err)
		}
		cluster.Tags = string(tagsJSON)

		// Update cluster in storage
		if err := api.storage.ClusterStore().Update(ctx, cluster); err != nil {
			return nil, fmt.Errorf("failed to update cluster: %w", err)
		}

		// Invalidate cache
		invalidateClusterCache(clusterName)
	} else {
		// For other resource types, just validate they could exist
		// In a full implementation, we'd check each resource type
		return nil, fmt.Errorf("Resource type not supported yet")
	}

	// Return successful response
	resp := &generated.TagResourceResponse{}

	return resp, nil
}

// UntagResource implements the UntagResource operation
func (api *DefaultECSAPI) UntagResource(ctx context.Context, req *generated.UntagResourceRequest) (*generated.UntagResourceResponse, error) {
	// Validate resource ARN
	if err := ValidateResourceARN(req.ResourceArn); err != nil {
		return nil, err
	}

	// Validate tag keys
	if len(req.TagKeys) == 0 {
		return nil, fmt.Errorf("Invalid parameter: At least one tag key must be specified")
	}

	// Parse resource ARN to determine resource type
	resourceArn := req.ResourceArn
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

		// Parse existing tags
		existingTags, err := parseTags(clusterName, cluster.Tags)
		if err != nil {
			return nil, fmt.Errorf("failed to parse existing tags: %w", err)
		}

		// Convert existing tags to a map for easier manipulation
		tagMap := make(map[string]string)
		for _, tag := range existingTags {
			if tag.Key != nil && tag.Value != nil {
				tagMap[*tag.Key] = *tag.Value
			}
		}

		// Remove specified tag keys
		for _, tagKey := range req.TagKeys {
			delete(tagMap, tagKey)
		}

		// Convert back to tag array
		var updatedTags []generated.Tag
		for k, v := range tagMap {
			key := k
			value := v
			updatedTags = append(updatedTags, generated.Tag{
				Key:   &key,
				Value: &value,
			})
		}

		// Marshal tags to JSON (or empty string if no tags)
		if len(updatedTags) > 0 {
			tagsJSON, err := json.Marshal(updatedTags)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal tags: %w", err)
			}
			cluster.Tags = string(tagsJSON)
		} else {
			cluster.Tags = ""
		}

		// Update cluster in storage
		if err := api.storage.ClusterStore().Update(ctx, cluster); err != nil {
			return nil, fmt.Errorf("failed to update cluster: %w", err)
		}

		// Invalidate cache
		invalidateClusterCache(clusterName)
	} else {
		// For other resource types, just validate they could exist
		// In a full implementation, we'd check each resource type
		return nil, fmt.Errorf("Resource type not supported yet")
	}

	// Return successful response
	resp := &generated.UntagResourceResponse{}

	return resp, nil
}

// ListTagsForResource implements the ListTagsForResource operation
func (api *DefaultECSAPI) ListTagsForResource(ctx context.Context, req *generated.ListTagsForResourceRequest) (*generated.ListTagsForResourceResponse, error) {
	// Validate resource ARN
	if err := ValidateResourceARN(req.ResourceArn); err != nil {
		return nil, err
	}

	tags := []generated.Tag{}

	// Parse resource ARN to determine resource type
	resourceArn := req.ResourceArn
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

		// Parse tags from storage
		tags, err = parseTags(clusterName, cluster.Tags)
		if err != nil {
			return nil, fmt.Errorf("failed to parse tags: %w", err)
		}
	} else if strings.Contains(resourceArn, ":service/") {
		// For now, return empty tags for other resource types
		// In a full implementation, we'd retrieve from appropriate storage
	} else if strings.Contains(resourceArn, ":task/") {
		// Empty tags for tasks
	} else if strings.Contains(resourceArn, ":task-definition/") {
		// Empty tags for task definitions
	} else if strings.Contains(resourceArn, ":container-instance/") {
		// Empty tags for container instances
	} else if strings.Contains(resourceArn, ":capacity-provider/") {
		// Empty tags for capacity providers
	} else {
		return nil, fmt.Errorf("Invalid parameter: Unknown resource type in ARN")
	}

	resp := &generated.ListTagsForResourceResponse{
		Tags: tags,
	}

	return resp, nil
}
