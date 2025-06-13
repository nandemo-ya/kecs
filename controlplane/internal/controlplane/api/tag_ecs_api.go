package api

import (
	"context"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)

// TagResource implements the TagResource operation
func (api *DefaultECSAPI) TagResource(ctx context.Context, req *generated.TagResourceRequest) (*generated.TagResourceResponse, error) {
	// TODO: Implement TagResource
	return nil, fmt.Errorf("TagResource not implemented")
}

// UntagResource implements the UntagResource operation
func (api *DefaultECSAPI) UntagResource(ctx context.Context, req *generated.UntagResourceRequest) (*generated.UntagResourceResponse, error) {
	// TODO: Implement UntagResource
	return nil, fmt.Errorf("UntagResource not implemented")
}

// ListTagsForResource implements the ListTagsForResource operation
func (api *DefaultECSAPI) ListTagsForResource(ctx context.Context, req *generated.ListTagsForResourceRequest) (*generated.ListTagsForResourceResponse, error) {
	// TODO: Implement ListTagsForResource
	return nil, fmt.Errorf("ListTagsForResource not implemented")
}