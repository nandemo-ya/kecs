package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// RegisterTaskDefinition implements the RegisterTaskDefinition operation
func (api *DefaultECSAPI) RegisterTaskDefinition(ctx context.Context, req *generated.RegisterTaskDefinitionRequest) (*generated.RegisterTaskDefinitionResponse, error) {
	// Validate required fields
	if req.Family == nil || *req.Family == "" {
		return nil, fmt.Errorf("family is required")
	}
	if req.ContainerDefinitions == nil || len(req.ContainerDefinitions) == 0 {
		return nil, fmt.Errorf("containerDefinitions is required")
	}

	// Set default values
	networkMode := generated.NetworkModeBridge
	if req.NetworkMode != nil {
		networkMode = *req.NetworkMode
	}

	// Marshal complex fields to JSON
	containerDefsJSON, err := json.Marshal(req.ContainerDefinitions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal container definitions: %w", err)
	}

	volumesJSON := "[]"
	if req.Volumes != nil && len(req.Volumes) > 0 {
		volumesData, err := json.Marshal(req.Volumes)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal volumes: %w", err)
		}
		volumesJSON = string(volumesData)
	}

	placementConstraintsJSON := "[]"
	if req.PlacementConstraints != nil && len(req.PlacementConstraints) > 0 {
		placementData, err := json.Marshal(req.PlacementConstraints)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal placement constraints: %w", err)
		}
		placementConstraintsJSON = string(placementData)
	}

	requiresCompatibilitiesJSON := "[]"
	if req.RequiresCompatibilities != nil && len(req.RequiresCompatibilities) > 0 {
		compatData, err := json.Marshal(req.RequiresCompatibilities)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal requires compatibilities: %w", err)
		}
		requiresCompatibilitiesJSON = string(compatData)
	}

	tagsJSON := "[]"
	if req.Tags != nil && len(req.Tags) > 0 {
		tagsData, err := json.Marshal(req.Tags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tags: %w", err)
		}
		tagsJSON = string(tagsData)
	}

	proxyConfigJSON := ""
	if req.ProxyConfiguration != nil {
		proxyData, err := json.Marshal(req.ProxyConfiguration)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal proxy configuration: %w", err)
		}
		proxyConfigJSON = string(proxyData)
	}

	inferenceAcceleratorsJSON := ""
	if req.InferenceAccelerators != nil && len(req.InferenceAccelerators) > 0 {
		acceleratorData, err := json.Marshal(req.InferenceAccelerators)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal inference accelerators: %w", err)
		}
		inferenceAcceleratorsJSON = string(acceleratorData)
	}

	runtimePlatformJSON := ""
	if req.RuntimePlatform != nil {
		platformData, err := json.Marshal(req.RuntimePlatform)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal runtime platform: %w", err)
		}
		runtimePlatformJSON = string(platformData)
	}

	// Extract optional string values
	var taskRoleARN, executionRoleARN, cpu, memory, pidMode, ipcMode string
	if req.TaskRoleArn != nil {
		taskRoleARN = *req.TaskRoleArn
	}
	if req.ExecutionRoleArn != nil {
		executionRoleARN = *req.ExecutionRoleArn
	}
	if req.Cpu != nil {
		cpu = *req.Cpu
	}
	if req.Memory != nil {
		memory = *req.Memory
	}
	if req.PidMode != nil {
		pidMode = string(*req.PidMode)
	}
	if req.IpcMode != nil {
		ipcMode = string(*req.IpcMode)
	}

	// Create storage task definition
	storageTaskDef := &storage.TaskDefinition{
		ID:                       uuid.New().String(),
		Family:                   *req.Family,
		TaskRoleARN:              taskRoleARN,
		ExecutionRoleARN:         executionRoleARN,
		NetworkMode:              string(networkMode),
		ContainerDefinitions:     string(containerDefsJSON),
		Volumes:                  volumesJSON,
		PlacementConstraints:     placementConstraintsJSON,
		RequiresCompatibilities:  requiresCompatibilitiesJSON,
		CPU:                      cpu,
		Memory:                   memory,
		Tags:                     tagsJSON,
		PidMode:                  pidMode,
		IpcMode:                  ipcMode,
		ProxyConfiguration:       proxyConfigJSON,
		InferenceAccelerators:    inferenceAcceleratorsJSON,
		RuntimePlatform:          runtimePlatformJSON,
		Region:                   api.region,
		AccountID:                api.accountID,
	}

	// Register the task definition
	registeredTaskDef, err := api.storage.TaskDefinitionStore().Register(ctx, storageTaskDef)
	if err != nil {
		return nil, fmt.Errorf("failed to register task definition: %w", err)
	}

	// Convert storage task definition to generated response
	responseTaskDef := storageTaskDefinitionToGenerated(registeredTaskDef)

	return &generated.RegisterTaskDefinitionResponse{
		TaskDefinition: responseTaskDef,
		Tags:           req.Tags,
	}, nil
}

// DeregisterTaskDefinition implements the DeregisterTaskDefinition operation
func (api *DefaultECSAPI) DeregisterTaskDefinition(ctx context.Context, req *generated.DeregisterTaskDefinitionRequest) (*generated.DeregisterTaskDefinitionResponse, error) {
	if req.TaskDefinition == nil || *req.TaskDefinition == "" {
		return nil, fmt.Errorf("taskDefinition is required")
	}

	taskDefIdentifier := *req.TaskDefinition
	var family string
	var revision int
	var err error

	// Check if it's an ARN or family:revision format
	if strings.HasPrefix(taskDefIdentifier, "arn:aws:ecs:") {
		// Parse ARN to get family and revision
		taskDef, err := api.storage.TaskDefinitionStore().GetByARN(ctx, taskDefIdentifier)
		if err != nil {
			return nil, fmt.Errorf("task definition not found: %s", taskDefIdentifier)
		}
		family = taskDef.Family
		revision = taskDef.Revision
	} else if strings.Contains(taskDefIdentifier, ":") {
		// family:revision format
		parts := strings.SplitN(taskDefIdentifier, ":", 2)
		family = parts[0]
		revision, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid task definition revision: %s", parts[1])
		}
	} else {
		return nil, fmt.Errorf("task definition must include revision number")
	}

	// Deregister the task definition
	if err := api.storage.TaskDefinitionStore().Deregister(ctx, family, revision); err != nil {
		// Check if it's already inactive (for idempotency)
		if strings.Contains(err.Error(), "already inactive") {
			// Get the task definition to return it
			taskDef, getErr := api.storage.TaskDefinitionStore().Get(ctx, family, revision)
			if getErr != nil {
				return nil, fmt.Errorf("failed to deregister task definition: %w", err)
			}
			responseTaskDef := storageTaskDefinitionToGenerated(taskDef)
			return &generated.DeregisterTaskDefinitionResponse{
				TaskDefinition: responseTaskDef,
			}, nil
		}
		return nil, fmt.Errorf("failed to deregister task definition: %w", err)
	}

	// Get the deregistered task definition to return
	taskDef, err := api.storage.TaskDefinitionStore().Get(ctx, family, revision)
	if err != nil {
		return nil, fmt.Errorf("failed to get deregistered task definition: %w", err)
	}

	responseTaskDef := storageTaskDefinitionToGenerated(taskDef)
	return &generated.DeregisterTaskDefinitionResponse{
		TaskDefinition: responseTaskDef,
	}, nil
}

// DescribeTaskDefinition implements the DescribeTaskDefinition operation
func (api *DefaultECSAPI) DescribeTaskDefinition(ctx context.Context, req *generated.DescribeTaskDefinitionRequest) (*generated.DescribeTaskDefinitionResponse, error) {
	if req.TaskDefinition == nil || *req.TaskDefinition == "" {
		return nil, fmt.Errorf("taskDefinition is required")
	}

	taskDefIdentifier := *req.TaskDefinition
	var taskDef *storage.TaskDefinition
	var err error

	// Check if it's an ARN or family:revision format
	if strings.HasPrefix(taskDefIdentifier, "arn:aws:ecs:") {
		// ARN format
		taskDef, err = api.storage.TaskDefinitionStore().GetByARN(ctx, taskDefIdentifier)
	} else if strings.Contains(taskDefIdentifier, ":") {
		// family:revision format
		parts := strings.SplitN(taskDefIdentifier, ":", 2)
		family := parts[0]
		revision, parseErr := strconv.Atoi(parts[1])
		if parseErr != nil {
			return nil, fmt.Errorf("invalid task definition revision: %s", parts[1])
		}
		taskDef, err = api.storage.TaskDefinitionStore().Get(ctx, family, revision)
	} else {
		// Just family name - get latest
		taskDef, err = api.storage.TaskDefinitionStore().GetLatest(ctx, taskDefIdentifier)
	}

	if err != nil {
		return nil, fmt.Errorf("task definition not found: %s", taskDefIdentifier)
	}

	// Convert to generated response
	responseTaskDef := storageTaskDefinitionToGenerated(taskDef)

	// Note: Include parameter handling would be here if needed
	// Currently, generated.TaskDefinition doesn't have a Tags field

	// Note: generated.TaskDefinition doesn't have a Tags field
	// Tags are handled separately in the API response

	return &generated.DescribeTaskDefinitionResponse{
		TaskDefinition: responseTaskDef,
	}, nil
}

// DeleteTaskDefinitions implements the DeleteTaskDefinitions operation
func (api *DefaultECSAPI) DeleteTaskDefinitions(ctx context.Context, req *generated.DeleteTaskDefinitionsRequest) (*generated.DeleteTaskDefinitionsResponse, error) {
	if req.TaskDefinitions == nil || len(req.TaskDefinitions) == 0 {
		return nil, fmt.Errorf("taskDefinitions is required")
	}

	var deletedTaskDefs []generated.TaskDefinition
	var failures []generated.Failure

	// Process each task definition
	for _, taskDefIdentifier := range req.TaskDefinitions {
		var family string
		var revision int
		var err error

		// Parse the identifier
		if strings.HasPrefix(taskDefIdentifier, "arn:aws:ecs:") {
			// Parse ARN to get family and revision
			taskDef, err := api.storage.TaskDefinitionStore().GetByARN(ctx, taskDefIdentifier)
			if err != nil {
				failures = append(failures, generated.Failure{
					Arn:    ptr.String(taskDefIdentifier),
					Reason: ptr.String("MISSING"),
					Detail: ptr.String(fmt.Sprintf("Task definition not found: %s", taskDefIdentifier)),
				})
				continue
			}
			family = taskDef.Family
			revision = taskDef.Revision
		} else if strings.Contains(taskDefIdentifier, ":") {
			// family:revision format
			parts := strings.SplitN(taskDefIdentifier, ":", 2)
			family = parts[0]
			revision, err = strconv.Atoi(parts[1])
			if err != nil {
				failures = append(failures, generated.Failure{
					Arn:    ptr.String(taskDefIdentifier),
					Reason: ptr.String("INVALID"),
					Detail: ptr.String(fmt.Sprintf("Invalid task definition revision: %s", parts[1])),
				})
				continue
			}
		} else {
			failures = append(failures, generated.Failure{
				Arn:    ptr.String(taskDefIdentifier),
				Reason: ptr.String("INVALID"),
				Detail: ptr.String("Task definition must include revision number"),
			})
			continue
		}

		// Deregister the task definition
		if err := api.storage.TaskDefinitionStore().Deregister(ctx, family, revision); err != nil {
			reason := "FAILED"
			if strings.Contains(err.Error(), "not found") {
				reason = "MISSING"
			}
			failures = append(failures, generated.Failure{
				Arn:    ptr.String(taskDefIdentifier),
				Reason: ptr.String(reason),
				Detail: ptr.String(fmt.Sprintf("Failed to delete task definition: %v", err)),
			})
			continue
		}

		// Get the deregistered task definition to add to response
		taskDef, err := api.storage.TaskDefinitionStore().Get(ctx, family, revision)
		if err != nil {
			// Log but don't fail - the deletion succeeded
			log.Printf("Failed to get deleted task definition %s:%d: %v", family, revision, err)
			continue
		}

		deletedTaskDef := storageTaskDefinitionToGenerated(taskDef)
		if deletedTaskDef != nil {
			deletedTaskDefs = append(deletedTaskDefs, *deletedTaskDef)
		}
	}

	return &generated.DeleteTaskDefinitionsResponse{
		TaskDefinitions: deletedTaskDefs,
		Failures:        failures,
	}, nil
}

// ListTaskDefinitionFamilies implements the ListTaskDefinitionFamilies operation
func (api *DefaultECSAPI) ListTaskDefinitionFamilies(ctx context.Context, req *generated.ListTaskDefinitionFamiliesRequest) (*generated.ListTaskDefinitionFamiliesResponse, error) {
	// Extract parameters
	var familyPrefix string
	if req.FamilyPrefix != nil {
		familyPrefix = *req.FamilyPrefix
	}

	var status string
	if req.Status != nil {
		status = string(*req.Status)
	}

	limit := 100 // Default limit
	if req.MaxResults != nil && *req.MaxResults > 0 {
		limit = int(*req.MaxResults)
	}

	var nextToken string
	if req.NextToken != nil {
		nextToken = *req.NextToken
	}

	// Get families from storage
	families, newNextToken, err := api.storage.TaskDefinitionStore().ListFamilies(ctx, familyPrefix, status, limit, nextToken)
	if err != nil {
		return nil, fmt.Errorf("failed to list task definition families: %w", err)
	}

	// Convert to response format
	familyNames := make([]string, 0, len(families))
	for _, family := range families {
		familyNames = append(familyNames, family.Family)
	}

	response := &generated.ListTaskDefinitionFamiliesResponse{
		Families: familyNames,
	}

	if newNextToken != "" {
		response.NextToken = ptr.String(newNextToken)
	}

	return response, nil
}

// ListTaskDefinitions implements the ListTaskDefinitions operation
func (api *DefaultECSAPI) ListTaskDefinitions(ctx context.Context, req *generated.ListTaskDefinitionsRequest) (*generated.ListTaskDefinitionsResponse, error) {
	// Extract parameters
	var familyPrefix string
	if req.FamilyPrefix != nil {
		familyPrefix = *req.FamilyPrefix
	}

	var status string
	if req.Status != nil {
		status = string(*req.Status)
	}

	limit := 100 // Default limit
	if req.MaxResults != nil && *req.MaxResults > 0 {
		limit = int(*req.MaxResults)
	}

	var nextToken string
	if req.NextToken != nil {
		nextToken = *req.NextToken
	}

	// Get all families that match the criteria
	families, _, err := api.storage.TaskDefinitionStore().ListFamilies(ctx, familyPrefix, status, 0, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list task definition families: %w", err)
	}

	// Collect all revisions for matching families
	var allRevisions []string
	for _, family := range families {
		revisions, _, err := api.storage.TaskDefinitionStore().ListRevisions(ctx, family.Family, status, 0, "")
		if err != nil {
			log.Printf("Failed to list revisions for family %s: %v", family.Family, err)
			continue
		}
		for _, rev := range revisions {
			allRevisions = append(allRevisions, rev.ARN)
		}
	}

	// Sort by ARN (which includes family and revision)
	// Apply pagination
	start := 0
	if nextToken != "" {
		for i, arn := range allRevisions {
			if arn > nextToken {
				start = i
				break
			}
		}
	}

	end := start + limit
	if end > len(allRevisions) {
		end = len(allRevisions)
	}

	taskDefinitionArns := allRevisions[start:end]

	response := &generated.ListTaskDefinitionsResponse{
		TaskDefinitionArns: taskDefinitionArns,
	}

	// Set next token if there are more results
	if end < len(allRevisions) {
		response.NextToken = ptr.String(allRevisions[end-1])
	}

	return response, nil
}

// storageTaskDefinitionToGenerated converts a storage.TaskDefinition to generated.TaskDefinition
func storageTaskDefinitionToGenerated(taskDef *storage.TaskDefinition) *generated.TaskDefinition {
	if taskDef == nil {
		return nil
	}

	// Create the response
	response := &generated.TaskDefinition{
		TaskDefinitionArn: ptr.String(taskDef.ARN),
		Family:            ptr.String(taskDef.Family),
		Revision:          ptr.Int32(int32(taskDef.Revision)),
		Status:            (*generated.TaskDefinitionStatus)(ptr.String(taskDef.Status)),
		RegisteredAt:      ptr.Time(taskDef.RegisteredAt),
	}

	// Set optional string fields
	if taskDef.TaskRoleARN != "" {
		response.TaskRoleArn = ptr.String(taskDef.TaskRoleARN)
	}
	if taskDef.ExecutionRoleARN != "" {
		response.ExecutionRoleArn = ptr.String(taskDef.ExecutionRoleARN)
	}
	if taskDef.CPU != "" {
		response.Cpu = ptr.String(taskDef.CPU)
	}
	if taskDef.Memory != "" {
		response.Memory = ptr.String(taskDef.Memory)
	}
	if taskDef.PidMode != "" {
		response.PidMode = (*generated.PidMode)(ptr.String(taskDef.PidMode))
	}
	if taskDef.IpcMode != "" {
		response.IpcMode = (*generated.IpcMode)(ptr.String(taskDef.IpcMode))
	}
	if taskDef.NetworkMode != "" {
		response.NetworkMode = (*generated.NetworkMode)(ptr.String(taskDef.NetworkMode))
	}
	if taskDef.DeregisteredAt != nil {
		response.DeregisteredAt = ptr.Time(*taskDef.DeregisteredAt)
	}

	// Parse JSON fields
	if taskDef.ContainerDefinitions != "" {
		var containerDefs []generated.ContainerDefinition
		if err := json.Unmarshal([]byte(taskDef.ContainerDefinitions), &containerDefs); err == nil {
			response.ContainerDefinitions = containerDefs
		}
	}
	if taskDef.Volumes != "" && taskDef.Volumes != "[]" {
		var volumes []generated.Volume
		if err := json.Unmarshal([]byte(taskDef.Volumes), &volumes); err == nil {
			response.Volumes = volumes
		}
	}
	if taskDef.PlacementConstraints != "" && taskDef.PlacementConstraints != "[]" {
		var constraints []generated.TaskDefinitionPlacementConstraint
		if err := json.Unmarshal([]byte(taskDef.PlacementConstraints), &constraints); err == nil {
			response.PlacementConstraints = constraints
		}
	}
	if taskDef.RequiresCompatibilities != "" && taskDef.RequiresCompatibilities != "[]" {
		var compatibilities []generated.Compatibility
		if err := json.Unmarshal([]byte(taskDef.RequiresCompatibilities), &compatibilities); err == nil {
			response.RequiresCompatibilities = compatibilities
			// Also set compatibilities field for backward compatibility
			response.Compatibilities = compatibilities
		}
	}
	// Note: generated.TaskDefinition doesn't have a Tags field
	// Tags are handled separately in the API
	if taskDef.ProxyConfiguration != "" {
		var proxyConfig generated.ProxyConfiguration
		if err := json.Unmarshal([]byte(taskDef.ProxyConfiguration), &proxyConfig); err == nil {
			response.ProxyConfiguration = &proxyConfig
		}
	}
	if taskDef.InferenceAccelerators != "" {
		var accelerators []generated.InferenceAccelerator
		if err := json.Unmarshal([]byte(taskDef.InferenceAccelerators), &accelerators); err == nil {
			response.InferenceAccelerators = accelerators
		}
	}
	if taskDef.RuntimePlatform != "" {
		var platform generated.RuntimePlatform
		if err := json.Unmarshal([]byte(taskDef.RuntimePlatform), &platform); err == nil {
			response.RuntimePlatform = &platform
		}
	}

	return response
}