package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// RegisterTaskDefinitionV2 implements the RegisterTaskDefinition operation using AWS SDK types
func (api *DefaultECSAPIV2) RegisterTaskDefinitionV2(ctx context.Context, req *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	// Validate required fields
	if req.Family == nil || *req.Family == "" {
		return nil, fmt.Errorf("family is required")
	}
	if req.ContainerDefinitions == nil || len(req.ContainerDefinitions) == 0 {
		return nil, fmt.Errorf("containerDefinitions is required")
	}

	// Convert container definitions to JSON for storage
	containerDefsJSON, err := json.Marshal(req.ContainerDefinitions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal container definitions: %w", err)
	}

	// Create storage task definition
	taskDef := &storage.TaskDefinition{
		Family:               *req.Family,
		ContainerDefinitions: string(containerDefsJSON),
		NetworkMode:          "bridge", // default
		Region:               api.region,
		AccountID:            api.accountID,
	}

	// Set optional fields
	if req.TaskRoleArn != nil {
		taskDef.TaskRoleARN = *req.TaskRoleArn
	}
	if req.ExecutionRoleArn != nil {
		taskDef.ExecutionRoleARN = *req.ExecutionRoleArn
	}
	if req.NetworkMode != "" {
		taskDef.NetworkMode = string(req.NetworkMode)
	}
	if req.Cpu != nil {
		taskDef.CPU = *req.Cpu
	}
	if req.Memory != nil {
		taskDef.Memory = *req.Memory
	}
	if req.PidMode != "" {
		taskDef.PidMode = string(req.PidMode)
	}
	if req.IpcMode != "" {
		taskDef.IpcMode = string(req.IpcMode)
	}

	// Convert complex objects to JSON
	if len(req.Volumes) > 0 {
		volumesJSON, _ := json.Marshal(req.Volumes)
		taskDef.Volumes = string(volumesJSON)
	}
	if len(req.PlacementConstraints) > 0 {
		constraintsJSON, _ := json.Marshal(req.PlacementConstraints)
		taskDef.PlacementConstraints = string(constraintsJSON)
	}
	if len(req.RequiresCompatibilities) > 0 {
		compatibilities := make([]string, len(req.RequiresCompatibilities))
		for i, c := range req.RequiresCompatibilities {
			compatibilities[i] = string(c)
		}
		taskDef.RequiresCompatibilities = strings.Join(compatibilities, ",")
	}
	if len(req.Tags) > 0 {
		tagsJSON, _ := json.Marshal(req.Tags)
		taskDef.Tags = string(tagsJSON)
	}
	if req.ProxyConfiguration != nil {
		proxyJSON, _ := json.Marshal(req.ProxyConfiguration)
		taskDef.ProxyConfiguration = string(proxyJSON)
	}
	if len(req.InferenceAccelerators) > 0 {
		accelJSON, _ := json.Marshal(req.InferenceAccelerators)
		taskDef.InferenceAccelerators = string(accelJSON)
	}
	if req.RuntimePlatform != nil {
		platformJSON, _ := json.Marshal(req.RuntimePlatform)
		taskDef.RuntimePlatform = string(platformJSON)
	}

	// Register task definition
	registeredTaskDef, err := api.storage.TaskDefinitionStore().Register(ctx, taskDef)
	if err != nil {
		return nil, fmt.Errorf("failed to register task definition: %w", err)
	}

	// Build response
	describeResp, err := api.buildTaskDefinitionResponse(registeredTaskDef)
	if err != nil {
		return nil, err
	}
	
	return &ecs.RegisterTaskDefinitionOutput{
		TaskDefinition: describeResp.TaskDefinition,
		Tags:           describeResp.Tags,
	}, nil
}

// DeregisterTaskDefinitionV2 implements the DeregisterTaskDefinition operation using AWS SDK types
func (api *DefaultECSAPIV2) DeregisterTaskDefinitionV2(ctx context.Context, req *ecs.DeregisterTaskDefinitionInput) (*ecs.DeregisterTaskDefinitionOutput, error) {
	// Validate required fields
	if req.TaskDefinition == nil || *req.TaskDefinition == "" {
		return nil, fmt.Errorf("taskDefinition is required")
	}

	// Parse task definition ARN
	taskDefArn := *req.TaskDefinition
	family, revision, err := parseTaskDefinitionArn(taskDefArn)
	if err != nil {
		return nil, fmt.Errorf("invalid task definition ARN: %s", taskDefArn)
	}

	// Deregister task definition
	if err := api.storage.TaskDefinitionStore().Deregister(ctx, family, revision); err != nil {
		return nil, fmt.Errorf("failed to deregister task definition: %w", err)
	}

	// Get the deregistered task definition to return
	taskDef, err := api.storage.TaskDefinitionStore().Get(ctx, family, revision)
	if err != nil {
		return nil, fmt.Errorf("failed to get deregistered task definition: %w", err)
	}

	// Build response
	resp, err := api.buildTaskDefinitionResponse(taskDef)
	if err != nil {
		return nil, err
	}

	return &ecs.DeregisterTaskDefinitionOutput{
		TaskDefinition: resp.TaskDefinition,
	}, nil
}

// DescribeTaskDefinitionV2 implements the DescribeTaskDefinition operation using AWS SDK types
func (api *DefaultECSAPIV2) DescribeTaskDefinitionV2(ctx context.Context, req *ecs.DescribeTaskDefinitionInput) (*ecs.DescribeTaskDefinitionOutput, error) {
	// Validate required fields
	if req.TaskDefinition == nil || *req.TaskDefinition == "" {
		return nil, fmt.Errorf("taskDefinition is required")
	}

	var taskDef *storage.TaskDefinition
	taskDefIdentifier := *req.TaskDefinition

	// Check if it's an ARN or family:revision format
	if strings.Contains(taskDefIdentifier, ":") {
		if strings.HasPrefix(taskDefIdentifier, "arn:aws:ecs:") {
			// Full ARN
			family, revision, err := parseTaskDefinitionArn(taskDefIdentifier)
			if err != nil {
				return nil, fmt.Errorf("invalid task definition ARN: %s", taskDefIdentifier)
			}
			taskDef, err = api.storage.TaskDefinitionStore().Get(ctx, family, revision)
			if err != nil {
				return nil, fmt.Errorf("task definition not found: %s", taskDefIdentifier)
			}
		} else {
			// family:revision format
			parts := strings.Split(taskDefIdentifier, ":")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid task definition format: %s", taskDefIdentifier)
			}
			revision, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid revision number: %s", parts[1])
			}
			taskDef, err = api.storage.TaskDefinitionStore().Get(ctx, parts[0], revision)
			if err != nil {
				return nil, fmt.Errorf("task definition not found: %s", taskDefIdentifier)
			}
		}
	} else {
		// Just family - get latest
		var err error
		taskDef, err = api.storage.TaskDefinitionStore().GetLatest(ctx, taskDefIdentifier)
		if err != nil {
			return nil, fmt.Errorf("task definition not found: %s", taskDefIdentifier)
		}
	}

	if taskDef == nil {
		return nil, fmt.Errorf("task definition not found: %s", taskDefIdentifier)
	}

	// Build response
	return api.buildTaskDefinitionResponse(taskDef)
}

// ListTaskDefinitionFamiliesV2 implements the ListTaskDefinitionFamilies operation using AWS SDK types
func (api *DefaultECSAPIV2) ListTaskDefinitionFamiliesV2(ctx context.Context, req *ecs.ListTaskDefinitionFamiliesInput) (*ecs.ListTaskDefinitionFamiliesOutput, error) {
	// Set defaults
	familyPrefix := ""
	if req.FamilyPrefix != nil {
		familyPrefix = *req.FamilyPrefix
	}

	status := "ACTIVE"
	if req.Status != "" {
		status = string(req.Status)
	}

	limit := 100
	if req.MaxResults != nil && *req.MaxResults > 0 {
		limit = int(*req.MaxResults)
		if limit > 100 {
			limit = 100
		}
	}

	nextToken := ""
	if req.NextToken != nil {
		nextToken = *req.NextToken
	}

	// List families
	families, newNextToken, err := api.storage.TaskDefinitionStore().ListFamilies(ctx, familyPrefix, status, limit, nextToken)
	if err != nil {
		return nil, fmt.Errorf("failed to list task definition families: %w", err)
	}

	// Build response
	familyNames := make([]string, len(families))
	for i, family := range families {
		familyNames[i] = family.Family
	}

	resp := &ecs.ListTaskDefinitionFamiliesOutput{
		Families: familyNames,
	}

	if newNextToken != "" {
		resp.NextToken = aws.String(newNextToken)
	}

	return resp, nil
}

// ListTaskDefinitionsV2 implements the ListTaskDefinitions operation using AWS SDK types
func (api *DefaultECSAPIV2) ListTaskDefinitionsV2(ctx context.Context, req *ecs.ListTaskDefinitionsInput) (*ecs.ListTaskDefinitionsOutput, error) {
	// Parse family prefix if provided
	family := ""
	if req.FamilyPrefix != nil {
		family = *req.FamilyPrefix
	}

	status := "ACTIVE"
	if req.Status != "" {
		status = string(req.Status)
	}

	limit := 100
	if req.MaxResults != nil && *req.MaxResults > 0 {
		limit = int(*req.MaxResults)
		if limit > 100 {
			limit = 100
		}
	}

	nextToken := ""
	if req.NextToken != nil {
		nextToken = *req.NextToken
	}

	// If family is specified, list revisions for that family
	if family != "" {
		revisions, newNextToken, err := api.storage.TaskDefinitionStore().ListRevisions(ctx, family, status, limit, nextToken)
		if err != nil {
			return nil, fmt.Errorf("failed to list task definition revisions: %w", err)
		}

		// Build ARN list
		arns := make([]string, len(revisions))
		for i, rev := range revisions {
			arns[i] = rev.ARN
		}

		resp := &ecs.ListTaskDefinitionsOutput{
			TaskDefinitionArns: arns,
		}

		if newNextToken != "" {
			resp.NextToken = aws.String(newNextToken)
		}

		return resp, nil
	}

	// List all task definitions
	families, newNextToken, err := api.storage.TaskDefinitionStore().ListFamilies(ctx, "", status, limit, nextToken)
	if err != nil {
		return nil, fmt.Errorf("failed to list task definition families: %w", err)
	}

	// Get all revisions for each family
	var arns []string
	for _, family := range families {
		revisions, _, err := api.storage.TaskDefinitionStore().ListRevisions(ctx, family.Family, status, 100, "")
		if err != nil {
			log.Printf("Failed to list revisions for family %s: %v", family.Family, err)
			continue
		}
		for _, rev := range revisions {
			arns = append(arns, rev.ARN)
		}
	}

	resp := &ecs.ListTaskDefinitionsOutput{
		TaskDefinitionArns: arns,
	}

	if newNextToken != "" {
		resp.NextToken = aws.String(newNextToken)
	}

	return resp, nil
}

// Helper function to build task definition response
func (api *DefaultECSAPIV2) buildTaskDefinitionResponse(taskDef *storage.TaskDefinition) (*ecs.DescribeTaskDefinitionOutput, error) {
	// Parse container definitions
	var containerDefs []types.ContainerDefinition
	if err := json.Unmarshal([]byte(taskDef.ContainerDefinitions), &containerDefs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal container definitions: %w", err)
	}

	// Build task definition response
	respTaskDef := &types.TaskDefinition{
		TaskDefinitionArn: aws.String(taskDef.ARN),
		Family:            aws.String(taskDef.Family),
		Revision:          int32(taskDef.Revision),
		ContainerDefinitions: containerDefs,
		NetworkMode:       types.NetworkMode(taskDef.NetworkMode),
		Status:            types.TaskDefinitionStatus(taskDef.Status),
		RegisteredAt:      aws.Time(taskDef.RegisteredAt),
	}

	// Set optional fields
	if taskDef.TaskRoleARN != "" {
		respTaskDef.TaskRoleArn = aws.String(taskDef.TaskRoleARN)
	}
	if taskDef.ExecutionRoleARN != "" {
		respTaskDef.ExecutionRoleArn = aws.String(taskDef.ExecutionRoleARN)
	}
	if taskDef.CPU != "" {
		respTaskDef.Cpu = aws.String(taskDef.CPU)
	}
	if taskDef.Memory != "" {
		respTaskDef.Memory = aws.String(taskDef.Memory)
	}
	if taskDef.PidMode != "" {
		respTaskDef.PidMode = types.PidMode(taskDef.PidMode)
	}
	if taskDef.IpcMode != "" {
		respTaskDef.IpcMode = types.IpcMode(taskDef.IpcMode)
	}
	if taskDef.DeregisteredAt != nil {
		respTaskDef.DeregisteredAt = taskDef.DeregisteredAt
	}

	// Parse JSON fields
	if taskDef.Volumes != "" {
		var volumes []types.Volume
		if err := json.Unmarshal([]byte(taskDef.Volumes), &volumes); err == nil {
			respTaskDef.Volumes = volumes
		}
	}
	if taskDef.PlacementConstraints != "" {
		var constraints []types.TaskDefinitionPlacementConstraint
		if err := json.Unmarshal([]byte(taskDef.PlacementConstraints), &constraints); err == nil {
			respTaskDef.PlacementConstraints = constraints
		}
	}
	if taskDef.RequiresCompatibilities != "" {
		compatibilities := strings.Split(taskDef.RequiresCompatibilities, ",")
		respTaskDef.RequiresCompatibilities = make([]types.Compatibility, len(compatibilities))
		for i, c := range compatibilities {
			respTaskDef.RequiresCompatibilities[i] = types.Compatibility(c)
		}
	}
	if taskDef.ProxyConfiguration != "" {
		var proxyConfig types.ProxyConfiguration
		if err := json.Unmarshal([]byte(taskDef.ProxyConfiguration), &proxyConfig); err == nil {
			respTaskDef.ProxyConfiguration = &proxyConfig
		}
	}
	if taskDef.InferenceAccelerators != "" {
		var accelerators []types.InferenceAccelerator
		if err := json.Unmarshal([]byte(taskDef.InferenceAccelerators), &accelerators); err == nil {
			respTaskDef.InferenceAccelerators = accelerators
		}
	}
	if taskDef.RuntimePlatform != "" {
		var platform types.RuntimePlatform
		if err := json.Unmarshal([]byte(taskDef.RuntimePlatform), &platform); err == nil {
			respTaskDef.RuntimePlatform = &platform
		}
	}

	// Calculate compatibilities if not set
	if len(respTaskDef.RequiresCompatibilities) == 0 {
		compatibilities := []types.Compatibility{types.CompatibilityEc2}
		if taskDef.NetworkMode == "awsvpc" && taskDef.CPU != "" && taskDef.Memory != "" {
			compatibilities = append(compatibilities, types.CompatibilityFargate)
		}
		respTaskDef.Compatibilities = compatibilities
	} else {
		respTaskDef.Compatibilities = respTaskDef.RequiresCompatibilities
	}

	// Parse tags
	var tags []types.Tag
	if taskDef.Tags != "" {
		if err := json.Unmarshal([]byte(taskDef.Tags), &tags); err == nil {
			// Tags are returned separately in the response
		}
	}

	return &ecs.DescribeTaskDefinitionOutput{
		TaskDefinition: respTaskDef,
		Tags:           tags,
	}, nil
}

// parseTaskDefinitionArn parses a task definition ARN and returns family and revision
func parseTaskDefinitionArn(arn string) (string, int, error) {
	var family string
	var revisionStr string

	if strings.HasPrefix(arn, "arn:aws:ecs:") {
		// ARN format: arn:aws:ecs:region:account:task-definition/family:revision
		parts := strings.Split(arn, "/")
		if len(parts) != 2 {
			return "", 0, fmt.Errorf("invalid ARN format")
		}
		familyRevision := parts[1]
		colonIndex := strings.LastIndex(familyRevision, ":")
		if colonIndex == -1 {
			return "", 0, fmt.Errorf("missing revision in ARN")
		}
		family = familyRevision[:colonIndex]
		revisionStr = familyRevision[colonIndex+1:]
	} else if strings.Contains(arn, ":") {
		// family:revision format
		colonIndex := strings.LastIndex(arn, ":")
		family = arn[:colonIndex]
		revisionStr = arn[colonIndex+1:]
	} else {
		return "", 0, fmt.Errorf("invalid format: must be ARN or family:revision")
	}
	
	revision, err := strconv.Atoi(revisionStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid revision number: %s", revisionStr)
	}

	return family, revision, nil
}