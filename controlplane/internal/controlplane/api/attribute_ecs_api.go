package api

import (
	"context"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
)

// PutAttributes implements the PutAttributes operation
func (api *DefaultECSAPI) PutAttributes(ctx context.Context, req *generated.PutAttributesRequest) (*generated.PutAttributesResponse, error) {
	// TODO: Implement PutAttributes
	return nil, fmt.Errorf("PutAttributes not implemented")
}

// DeleteAttributes implements the DeleteAttributes operation
func (api *DefaultECSAPI) DeleteAttributes(ctx context.Context, req *generated.DeleteAttributesRequest) (*generated.DeleteAttributesResponse, error) {
	// TODO: Implement DeleteAttributes
	return nil, fmt.Errorf("DeleteAttributes not implemented")
}

// ListAttributes implements the ListAttributes operation
func (api *DefaultECSAPI) ListAttributes(ctx context.Context, req *generated.ListAttributesRequest) (*generated.ListAttributesResponse, error) {
	// Get target type
	targetType := "container-instance" // Default
	if req.TargetType != nil {
		targetType = string(*req.TargetType)
	}

	// Get cluster
	cluster := ""
	if req.Cluster != nil {
		cluster = *req.Cluster
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
	attributes, newNextToken, err := api.storage.AttributeStore().ListWithPagination(ctx, targetType, cluster, limit, nextToken)
	if err != nil {
		return nil, fmt.Errorf("failed to list attributes: %w", err)
	}

	// Convert to API response format
	// Initialize with empty slice to ensure it's not nil
	apiAttributes := make([]generated.Attribute, 0)
	for _, attr := range attributes {
		apiAttr := generated.Attribute{
			Name:       ptr.String(attr.Name),
			TargetType: (*generated.TargetType)(ptr.String(attr.TargetType)),
			TargetId:   ptr.String(attr.TargetID),
		}
		if attr.Value != "" {
			apiAttr.Value = ptr.String(attr.Value)
		}
		apiAttributes = append(apiAttributes, apiAttr)
	}

	resp := &generated.ListAttributesResponse{
		Attributes: apiAttributes,
	}

	// Add next token if there are more results
	if newNextToken != "" {
		resp.NextToken = ptr.String(newNextToken)
	}

	return resp, nil
}