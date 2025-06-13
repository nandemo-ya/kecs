package api

import (
	"context"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
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
	// TODO: Implement ListAttributes
	return nil, fmt.Errorf("ListAttributes not implemented")
}