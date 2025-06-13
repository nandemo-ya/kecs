package api

import (
	"context"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)

// RegisterTaskDefinition implements the RegisterTaskDefinition operation
func (api *DefaultECSAPI) RegisterTaskDefinition(ctx context.Context, req *generated.RegisterTaskDefinitionRequest) (*generated.RegisterTaskDefinitionResponse, error) {
	// TODO: Implement RegisterTaskDefinition
	return nil, fmt.Errorf("RegisterTaskDefinition not implemented")
}

// DeregisterTaskDefinition implements the DeregisterTaskDefinition operation
func (api *DefaultECSAPI) DeregisterTaskDefinition(ctx context.Context, req *generated.DeregisterTaskDefinitionRequest) (*generated.DeregisterTaskDefinitionResponse, error) {
	// TODO: Implement DeregisterTaskDefinition
	return nil, fmt.Errorf("DeregisterTaskDefinition not implemented")
}

// DescribeTaskDefinition implements the DescribeTaskDefinition operation
func (api *DefaultECSAPI) DescribeTaskDefinition(ctx context.Context, req *generated.DescribeTaskDefinitionRequest) (*generated.DescribeTaskDefinitionResponse, error) {
	// TODO: Implement DescribeTaskDefinition
	return nil, fmt.Errorf("DescribeTaskDefinition not implemented")
}

// DeleteTaskDefinitions implements the DeleteTaskDefinitions operation
func (api *DefaultECSAPI) DeleteTaskDefinitions(ctx context.Context, req *generated.DeleteTaskDefinitionsRequest) (*generated.DeleteTaskDefinitionsResponse, error) {
	// TODO: Implement DeleteTaskDefinitions
	return nil, fmt.Errorf("DeleteTaskDefinitions not implemented")
}

// ListTaskDefinitionFamilies implements the ListTaskDefinitionFamilies operation
func (api *DefaultECSAPI) ListTaskDefinitionFamilies(ctx context.Context, req *generated.ListTaskDefinitionFamiliesRequest) (*generated.ListTaskDefinitionFamiliesResponse, error) {
	// TODO: Implement ListTaskDefinitionFamilies
	return nil, fmt.Errorf("ListTaskDefinitionFamilies not implemented")
}

// ListTaskDefinitions implements the ListTaskDefinitions operation
func (api *DefaultECSAPI) ListTaskDefinitions(ctx context.Context, req *generated.ListTaskDefinitionsRequest) (*generated.ListTaskDefinitionsResponse, error) {
	// TODO: Implement ListTaskDefinitions
	return nil, fmt.Errorf("ListTaskDefinitions not implemented")
}