package api

import (
	"context"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)

// RunTask implements the RunTask operation
func (api *DefaultECSAPI) RunTask(ctx context.Context, req *generated.RunTaskRequest) (*generated.RunTaskResponse, error) {
	// TODO: Implement RunTask
	return nil, fmt.Errorf("RunTask not implemented")
}

// StartTask implements the StartTask operation
func (api *DefaultECSAPI) StartTask(ctx context.Context, req *generated.StartTaskRequest) (*generated.StartTaskResponse, error) {
	// TODO: Implement StartTask
	return nil, fmt.Errorf("StartTask not implemented")
}

// StopTask implements the StopTask operation
func (api *DefaultECSAPI) StopTask(ctx context.Context, req *generated.StopTaskRequest) (*generated.StopTaskResponse, error) {
	// TODO: Implement StopTask
	return nil, fmt.Errorf("StopTask not implemented")
}

// DescribeTasks implements the DescribeTasks operation
func (api *DefaultECSAPI) DescribeTasks(ctx context.Context, req *generated.DescribeTasksRequest) (*generated.DescribeTasksResponse, error) {
	// TODO: Implement DescribeTasks
	return nil, fmt.Errorf("DescribeTasks not implemented")
}

// ListTasks implements the ListTasks operation
func (api *DefaultECSAPI) ListTasks(ctx context.Context, req *generated.ListTasksRequest) (*generated.ListTasksResponse, error) {
	// TODO: Implement ListTasks
	return nil, fmt.Errorf("ListTasks not implemented")
}

// GetTaskProtection implements the GetTaskProtection operation
func (api *DefaultECSAPI) GetTaskProtection(ctx context.Context, req *generated.GetTaskProtectionRequest) (*generated.GetTaskProtectionResponse, error) {
	// TODO: Implement GetTaskProtection
	return nil, fmt.Errorf("GetTaskProtection not implemented")
}

// UpdateTaskProtection implements the UpdateTaskProtection operation
func (api *DefaultECSAPI) UpdateTaskProtection(ctx context.Context, req *generated.UpdateTaskProtectionRequest) (*generated.UpdateTaskProtectionResponse, error) {
	// TODO: Implement UpdateTaskProtection
	return nil, fmt.Errorf("UpdateTaskProtection not implemented")
}

// SubmitTaskStateChange implements the SubmitTaskStateChange operation
func (api *DefaultECSAPI) SubmitTaskStateChange(ctx context.Context, req *generated.SubmitTaskStateChangeRequest) (*generated.SubmitTaskStateChangeResponse, error) {
	// TODO: Implement SubmitTaskStateChange
	return nil, fmt.Errorf("SubmitTaskStateChange not implemented")
}