package api

import (
	"context"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)

// RegisterContainerInstance implements the RegisterContainerInstance operation
func (api *DefaultECSAPI) RegisterContainerInstance(ctx context.Context, req *generated.RegisterContainerInstanceRequest) (*generated.RegisterContainerInstanceResponse, error) {
	// TODO: Implement RegisterContainerInstance
	return nil, fmt.Errorf("RegisterContainerInstance not implemented")
}

// DeregisterContainerInstance implements the DeregisterContainerInstance operation
func (api *DefaultECSAPI) DeregisterContainerInstance(ctx context.Context, req *generated.DeregisterContainerInstanceRequest) (*generated.DeregisterContainerInstanceResponse, error) {
	// TODO: Implement DeregisterContainerInstance
	return nil, fmt.Errorf("DeregisterContainerInstance not implemented")
}

// DescribeContainerInstances implements the DescribeContainerInstances operation
func (api *DefaultECSAPI) DescribeContainerInstances(ctx context.Context, req *generated.DescribeContainerInstancesRequest) (*generated.DescribeContainerInstancesResponse, error) {
	// TODO: Implement DescribeContainerInstances
	return nil, fmt.Errorf("DescribeContainerInstances not implemented")
}

// ListContainerInstances implements the ListContainerInstances operation
func (api *DefaultECSAPI) ListContainerInstances(ctx context.Context, req *generated.ListContainerInstancesRequest) (*generated.ListContainerInstancesResponse, error) {
	// TODO: Implement ListContainerInstances
	return nil, fmt.Errorf("ListContainerInstances not implemented")
}

// UpdateContainerAgent implements the UpdateContainerAgent operation
func (api *DefaultECSAPI) UpdateContainerAgent(ctx context.Context, req *generated.UpdateContainerAgentRequest) (*generated.UpdateContainerAgentResponse, error) {
	// TODO: Implement UpdateContainerAgent
	return nil, fmt.Errorf("UpdateContainerAgent not implemented")
}

// UpdateContainerInstancesState implements the UpdateContainerInstancesState operation
func (api *DefaultECSAPI) UpdateContainerInstancesState(ctx context.Context, req *generated.UpdateContainerInstancesStateRequest) (*generated.UpdateContainerInstancesStateResponse, error) {
	// TODO: Implement UpdateContainerInstancesState
	return nil, fmt.Errorf("UpdateContainerInstancesState not implemented")
}

// SubmitContainerStateChange implements the SubmitContainerStateChange operation
func (api *DefaultECSAPI) SubmitContainerStateChange(ctx context.Context, req *generated.SubmitContainerStateChangeRequest) (*generated.SubmitContainerStateChangeResponse, error) {
	// TODO: Implement SubmitContainerStateChange
	return nil, fmt.Errorf("SubmitContainerStateChange not implemented")
}