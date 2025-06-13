package api

import (
	"context"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)

// CreateTaskSet implements the CreateTaskSet operation
func (api *DefaultECSAPI) CreateTaskSet(ctx context.Context, req *generated.CreateTaskSetRequest) (*generated.CreateTaskSetResponse, error) {
	// TODO: Implement CreateTaskSet
	return nil, fmt.Errorf("CreateTaskSet not implemented")
}

// DeleteTaskSet implements the DeleteTaskSet operation
func (api *DefaultECSAPI) DeleteTaskSet(ctx context.Context, req *generated.DeleteTaskSetRequest) (*generated.DeleteTaskSetResponse, error) {
	// TODO: Implement DeleteTaskSet
	return nil, fmt.Errorf("DeleteTaskSet not implemented")
}

// DescribeTaskSets implements the DescribeTaskSets operation
func (api *DefaultECSAPI) DescribeTaskSets(ctx context.Context, req *generated.DescribeTaskSetsRequest) (*generated.DescribeTaskSetsResponse, error) {
	// TODO: Implement DescribeTaskSets
	return nil, fmt.Errorf("DescribeTaskSets not implemented")
}

// UpdateTaskSet implements the UpdateTaskSet operation
func (api *DefaultECSAPI) UpdateTaskSet(ctx context.Context, req *generated.UpdateTaskSetRequest) (*generated.UpdateTaskSetResponse, error) {
	// TODO: Implement UpdateTaskSet
	return nil, fmt.Errorf("UpdateTaskSet not implemented")
}