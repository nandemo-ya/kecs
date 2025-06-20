package api

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// PutAttributesV2 implements the PutAttributes operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) PutAttributesV2(ctx context.Context, req *ecs.PutAttributesInput) (*ecs.PutAttributesOutput, error) {
	// TODO: Implement PutAttributes
	return nil, fmt.Errorf("PutAttributes not implemented")
}

// DeleteAttributesV2 implements the DeleteAttributes operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) DeleteAttributesV2(ctx context.Context, req *ecs.DeleteAttributesInput) (*ecs.DeleteAttributesOutput, error) {
	// TODO: Implement DeleteAttributes
	return nil, fmt.Errorf("DeleteAttributes not implemented")
}

// ListAttributesV2 implements the ListAttributes operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) ListAttributesV2(ctx context.Context, req *ecs.ListAttributesInput) (*ecs.ListAttributesOutput, error) {
	// Get target type
	targetType := types.TargetTypeContainerInstance // Default
	if req.TargetType != "" {
		targetType = req.TargetType
	}

	// Get cluster
	cluster := ""
	if req.Cluster != nil {
		cluster = extractClusterNameFromARN(*req.Cluster)
	}

	// Set default limit if not specified
	limit := 100
	if req.MaxResults != nil && *req.MaxResults > 0 {
		limit = int(*req.MaxResults)
		// AWS ECS has a maximum of 100 results per page
		if limit > 100 {
			limit = 100
		}
	}

	// Extract next token
	var nextToken string
	if req.NextToken != nil {
		nextToken = *req.NextToken
	}

	// Get attributes with pagination
	attributes, newNextToken, err := api.storage.AttributeStore().ListWithPagination(ctx, string(targetType), cluster, limit, nextToken)
	if err != nil {
		return nil, fmt.Errorf("failed to list attributes: %w", err)
	}

	// Convert to API response format
	// Initialize with empty slice to ensure it's not nil
	apiAttributes := make([]types.Attribute, 0)
	for _, attr := range attributes {
		apiAttr := types.Attribute{
			Name:       aws.String(attr.Name),
			TargetType: types.TargetType(attr.TargetType),
			TargetId:   aws.String(attr.TargetID),
		}
		if attr.Value != "" {
			apiAttr.Value = aws.String(attr.Value)
		}
		apiAttributes = append(apiAttributes, apiAttr)
	}

	resp := &ecs.ListAttributesOutput{
		Attributes: apiAttributes,
	}

	// Add next token if there are more results
	if newNextToken != "" {
		resp.NextToken = aws.String(newNextToken)
	}

	return resp, nil
}