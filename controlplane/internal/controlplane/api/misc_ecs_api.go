package api

import (
	"context"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)

// DiscoverPollEndpoint implements the DiscoverPollEndpoint operation
func (api *DefaultECSAPI) DiscoverPollEndpoint(ctx context.Context, req *generated.DiscoverPollEndpointRequest) (*generated.DiscoverPollEndpointResponse, error) {
	// TODO: Implement DiscoverPollEndpoint
	return nil, fmt.Errorf("DiscoverPollEndpoint not implemented")
}

// ExecuteCommand implements the ExecuteCommand operation
func (api *DefaultECSAPI) ExecuteCommand(ctx context.Context, req *generated.ExecuteCommandRequest) (*generated.ExecuteCommandResponse, error) {
	// TODO: Implement ExecuteCommand
	return nil, fmt.Errorf("ExecuteCommand not implemented")
}

// SubmitAttachmentStateChanges implements the SubmitAttachmentStateChanges operation
func (api *DefaultECSAPI) SubmitAttachmentStateChanges(ctx context.Context, req *generated.SubmitAttachmentStateChangesRequest) (*generated.SubmitAttachmentStateChangesResponse, error) {
	// TODO: Implement SubmitAttachmentStateChanges
	return nil, fmt.Errorf("SubmitAttachmentStateChanges not implemented")
}
