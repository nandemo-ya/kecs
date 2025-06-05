package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleECSTagResource handles the TagResource API endpoint in AWS ECS format
func (s *Server) handleECSTagResource(w http.ResponseWriter, body []byte) {
	var req TagResourceRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate resource ARN format
	if !strings.HasPrefix(req.ResourceArn, "arn:aws:ecs:") {
		http.Error(w, "Invalid resource ARN format", http.StatusBadRequest)
		return
	}

	// Validate tags
	if len(req.Tags) == 0 {
		http.Error(w, "At least one tag must be specified", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual resource tagging logic
	// In a real implementation, we would:
	// 1. Parse the resource ARN to determine resource type
	// 2. Validate the resource exists
	// 3. Store the tags in the database
	// 4. Apply AWS tag limits (50 tags per resource, key/value length limits)

	// For now, return an empty successful response
	resp := TagResourceResponse{}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleECSUntagResource handles the UntagResource API endpoint in AWS ECS format
func (s *Server) handleECSUntagResource(w http.ResponseWriter, body []byte) {
	var req UntagResourceRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate resource ARN format
	if !strings.HasPrefix(req.ResourceArn, "arn:aws:ecs:") {
		http.Error(w, "Invalid resource ARN format", http.StatusBadRequest)
		return
	}

	// Validate tag keys
	if len(req.TagKeys) == 0 {
		http.Error(w, "At least one tag key must be specified", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual resource untagging logic
	// In a real implementation, we would:
	// 1. Parse the resource ARN to determine resource type
	// 2. Validate the resource exists
	// 3. Remove the specified tags from the database
	// 4. Handle non-existent tag keys gracefully

	// For now, return an empty successful response
	resp := UntagResourceResponse{}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleECSListTagsForResource handles the ListTagsForResource API endpoint in AWS ECS format
func (s *Server) handleECSListTagsForResource(w http.ResponseWriter, body []byte) {
	var req ListTagsForResourceRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validate resource ARN format
	if !strings.HasPrefix(req.ResourceArn, "arn:aws:ecs:") {
		http.Error(w, "Invalid resource ARN format", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual tag listing logic
	// In a real implementation, we would:
	// 1. Parse the resource ARN to determine resource type
	// 2. Validate the resource exists
	// 3. Retrieve tags from the database
	// 4. Return appropriate error if resource not found

	// For now, return mock tags based on resource type
	tags := []Tag{}
	
	// Determine resource type from ARN
	if strings.Contains(req.ResourceArn, ":cluster/") {
		tags = append(tags, Tag{
			Key:   "Environment",
			Value: "Development",
		})
		tags = append(tags, Tag{
			Key:   "Team",
			Value: "Platform",
		})
	} else if strings.Contains(req.ResourceArn, ":service/") {
		tags = append(tags, Tag{
			Key:   "Application",
			Value: "WebApp",
		})
		tags = append(tags, Tag{
			Key:   "Version",
			Value: "1.0.0",
		})
	} else if strings.Contains(req.ResourceArn, ":task/") {
		tags = append(tags, Tag{
			Key:   "Purpose",
			Value: "Testing",
		})
		tags = append(tags, Tag{
			Key:   "Owner",
			Value: "DevOps",
		})
	} else if strings.Contains(req.ResourceArn, ":task-definition/") {
		tags = append(tags, Tag{
			Key:   "Component",
			Value: "Backend",
		})
		tags = append(tags, Tag{
			Key:   "Language",
			Value: "Go",
		})
	} else if strings.Contains(req.ResourceArn, ":container-instance/") {
		tags = append(tags, Tag{
			Key:   "InstanceType",
			Value: "t3.medium",
		})
		tags = append(tags, Tag{
			Key:   "AZ",
			Value: "us-east-1a",
		})
	} else if strings.Contains(req.ResourceArn, ":capacity-provider/") {
		tags = append(tags, Tag{
			Key:   "Type",
			Value: "AutoScaling",
		})
		tags = append(tags, Tag{
			Key:   "ManagedBy",
			Value: "ECS",
		})
	}

	resp := ListTagsForResourceResponse{
		Tags: tags,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}