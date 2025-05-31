package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/internal/storage"
)

// handleECSRequest handles AWS ECS API requests in the AWS format
func (s *Server) handleECSRequest(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the target operation from X-Amz-Target header
	target := r.Header.Get("X-Amz-Target")
	if target == "" {
		http.Error(w, "Missing X-Amz-Target header", http.StatusBadRequest)
		return
	}

	// Parse the operation name from the target header
	// Format: "AmazonEC2ContainerServiceV20141113.OperationName"
	parts := strings.Split(target, ".")
	if len(parts) != 2 {
		http.Error(w, "Invalid X-Amz-Target format", http.StatusBadRequest)
		return
	}
	operation := parts[1]

	// Set appropriate headers
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Route to appropriate handler based on operation
	switch operation {
	case "ListClusters":
		s.handleECSListClusters(w, body)
	case "CreateCluster":
		s.handleECSCreateCluster(w, body)
	case "DescribeClusters":
		s.handleECSDescribeClusters(w, body)
	case "DeleteCluster":
		s.handleECSDeleteCluster(w, body)
	case "RunTask":
		s.handleECSRunTask(w, body)
	case "DescribeTasks":
		s.handleECSDescribeTasks(w, body)
	case "ListTasks":
		s.handleECSListTasks(w, body)
	case "RegisterTaskDefinition":
		s.handleECSRegisterTaskDefinition(w, body)
	case "DescribeTaskDefinition":
		s.handleECSDescribeTaskDefinition(w, body)
	case "ListTaskDefinitions":
		s.handleECSListTaskDefinitions(w, body)
	case "CreateService":
		s.handleECSCreateService(w, body)
	case "DescribeServices":
		s.handleECSDescribeServices(w, body)
	case "ListServices":
		s.handleECSListServices(w, body)
	default:
		// Return a basic empty response for unsupported operations
		s.handleUnsupportedOperation(w, operation)
	}
}

// handleECSListClusters handles the ListClusters operation
func (s *Server) handleECSListClusters(w http.ResponseWriter, body []byte) {
	var req generated.ListClustersRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}
	}

	ctx := context.Background()
	response, err := s.ListClustersWithStorage(ctx, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleECSCreateCluster handles the CreateCluster operation
func (s *Server) handleECSCreateCluster(w http.ResponseWriter, body []byte) {
	var req generated.CreateClusterRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}
	}

	ctx := context.Background()
	response, err := s.CreateClusterWithStorage(ctx, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleECSDescribeClusters handles the DescribeClusters operation
func (s *Server) handleECSDescribeClusters(w http.ResponseWriter, body []byte) {
	var req generated.DescribeClustersRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}
	}

	ctx := context.Background()
	response, err := s.DescribeClustersWithStorage(ctx, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleECSDeleteCluster handles the DeleteCluster operation
func (s *Server) handleECSDeleteCluster(w http.ResponseWriter, body []byte) {
	var req generated.DeleteClusterRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}
	}

	ctx := context.Background()
	response, err := s.DeleteClusterWithStorage(ctx, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleECSRunTask(w http.ResponseWriter, body []byte) {
	response := map[string]interface{}{
		"tasks":    []interface{}{},
		"failures": []interface{}{},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleECSDescribeTasks(w http.ResponseWriter, body []byte) {
	response := map[string]interface{}{
		"tasks":    []interface{}{},
		"failures": []interface{}{},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleECSListTasks(w http.ResponseWriter, body []byte) {
	response := map[string]interface{}{
		"taskArns": []interface{}{},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleECSRegisterTaskDefinition(w http.ResponseWriter, body []byte) {
	fmt.Printf("RegisterTaskDefinition called with body: %s\n", string(body))
	
	// Parse body as a generic map to handle generated type limitations
	var requestData map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &requestData); err != nil {
			fmt.Printf("Failed to unmarshal request: %v\n", err)
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}
	}

	fmt.Printf("Parsed request data: %+v\n", requestData)

	if s.storage == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
		return
	}

	ctx := context.Background()
	
	// Manually call storage service with converted data
	storageTaskDef, err := s.convertMapToStorageTaskDefinition(requestData)
	if err != nil {
		fmt.Printf("Failed to convert request: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	registered, err := s.storage.TaskDefinitionStore().Register(ctx, storageTaskDef)
	if err != nil {
		fmt.Printf("RegisterTaskDefinition error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert back to API format
	apiTaskDef, err := s.convertFromStorageTaskDefinitionToMap(registered)
	if err != nil {
		fmt.Printf("Failed to convert response: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responseMap := map[string]interface{}{
		"taskDefinition": apiTaskDef,
	}
	
	if tags, ok := requestData["tags"]; ok && tags != nil {
		responseMap["tags"] = tags
	}

	fmt.Printf("RegisterTaskDefinition response: %+v\n", responseMap)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responseMap)
}

func (s *Server) handleECSDescribeTaskDefinition(w http.ResponseWriter, body []byte) {
	s.writeEmptyResponse(w)
}

func (s *Server) handleECSListTaskDefinitions(w http.ResponseWriter, body []byte) {
	fmt.Printf("ListTaskDefinitions called with body: %s\n", string(body))
	
	// Parse body as a generic map
	var requestData map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &requestData); err != nil {
			fmt.Printf("Failed to unmarshal request: %v\n", err)
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}
	}

	fmt.Printf("Parsed request data: %+v\n", requestData)

	if s.storage == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"taskDefinitionArns": []string{},
		})
		return
	}

	ctx := context.Background()

	// Extract parameters from request
	familyPrefix := ""
	if fp, ok := requestData["familyPrefix"].(string); ok {
		familyPrefix = fp
	}

	status := ""
	if st, ok := requestData["status"].(string); ok {
		status = st
	}

	limit := 0
	if mr, ok := requestData["maxResults"].(float64); ok {
		limit = int(mr)
	}

	nextToken := ""
	if nt, ok := requestData["nextToken"].(string); ok {
		nextToken = nt
	}

	families, newNextToken, err := s.storage.TaskDefinitionStore().ListFamilies(ctx, familyPrefix, status, limit, nextToken)
	if err != nil {
		fmt.Printf("Storage error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var arns []string
	for _, family := range families {
		revisions, _, err := s.storage.TaskDefinitionStore().ListRevisions(ctx, family.Family, status, 0, "")
		if err != nil {
			continue
		}
		for _, rev := range revisions {
			arns = append(arns, rev.ARN)
		}
	}

	responseMap := map[string]interface{}{
		"taskDefinitionArns": arns,
	}
	
	if newNextToken != "" {
		responseMap["nextToken"] = newNextToken
	}

	fmt.Printf("ListTaskDefinitions response: %+v\n", responseMap)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responseMap)
}

func (s *Server) handleECSCreateService(w http.ResponseWriter, body []byte) {
	s.writeEmptyResponse(w)
}

func (s *Server) handleECSDescribeServices(w http.ResponseWriter, body []byte) {
	response := map[string]interface{}{
		"services": []interface{}{},
		"failures": []interface{}{},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleECSListServices(w http.ResponseWriter, body []byte) {
	response := map[string]interface{}{
		"serviceArns": []interface{}{},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleUnsupportedOperation returns a basic response for unsupported operations
func (s *Server) handleUnsupportedOperation(w http.ResponseWriter, operation string) {
	fmt.Printf("Unsupported operation: %s\n", operation)
	s.writeEmptyResponse(w)
}

// writeEmptyResponse writes an empty JSON response
func (s *Server) writeEmptyResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

// convertMapToStorageTaskDefinition converts a map to storage.TaskDefinition
func (s *Server) convertMapToStorageTaskDefinition(requestData map[string]interface{}) (*storage.TaskDefinition, error) {
	// Get family
	family := ""
	if f, ok := requestData["family"].(string); ok {
		family = f
	}

	// Marshal container definitions
	containerDefs, err := json.Marshal(requestData["containerDefinitions"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal container definitions: %w", err)
	}

	// Optional fields
	var volumesJSON, tagsJSON, requiresCompatibilitiesJSON, placementConstraintsJSON string
	var inferenceAcceleratorsJSON, proxyConfigJSON, runtimePlatformJSON string

	if volumes := requestData["volumes"]; volumes != nil {
		v, _ := json.Marshal(volumes)
		volumesJSON = string(v)
	}

	if tags := requestData["tags"]; tags != nil {
		t, _ := json.Marshal(tags)
		tagsJSON = string(t)
	}

	if reqCompat := requestData["requiresCompatibilities"]; reqCompat != nil {
		rc, _ := json.Marshal(reqCompat)
		requiresCompatibilitiesJSON = string(rc)
	}

	if placementConstraints := requestData["placementConstraints"]; placementConstraints != nil {
		pc, _ := json.Marshal(placementConstraints)
		placementConstraintsJSON = string(pc)
	}

	if inferenceAccelerators := requestData["inferenceAccelerators"]; inferenceAccelerators != nil {
		ia, _ := json.Marshal(inferenceAccelerators)
		inferenceAcceleratorsJSON = string(ia)
	}

	if proxyConfig := requestData["proxyConfiguration"]; proxyConfig != nil {
		p, _ := json.Marshal(proxyConfig)
		proxyConfigJSON = string(p)
	}

	if runtimePlatform := requestData["runtimePlatform"]; runtimePlatform != nil {
		rp, _ := json.Marshal(runtimePlatform)
		runtimePlatformJSON = string(rp)
	}

	// Get string values
	taskRoleArn, _ := requestData["taskRoleArn"].(string)
	executionRoleArn, _ := requestData["executionRoleArn"].(string)
	networkMode, _ := requestData["networkMode"].(string)
	cpu, _ := requestData["cpu"].(string)
	memory, _ := requestData["memory"].(string)
	pidMode, _ := requestData["pidMode"].(string)
	ipcMode, _ := requestData["ipcMode"].(string)

	if networkMode == "" {
		networkMode = "bridge"
	}

	return &storage.TaskDefinition{
		ID:                       fmt.Sprintf("td-%d", time.Now().UnixNano()),
		Family:                   family,
		TaskRoleARN:              taskRoleArn,
		ExecutionRoleARN:         executionRoleArn,
		NetworkMode:              networkMode,
		ContainerDefinitions:     string(containerDefs),
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
		Region:                   "us-east-1",
		AccountID:                "123456789012",
	}, nil
}

// convertFromStorageTaskDefinitionToMap converts storage.TaskDefinition to a map
func (s *Server) convertFromStorageTaskDefinitionToMap(stored *storage.TaskDefinition) (map[string]interface{}, error) {
	// Parse JSON fields back to interface{}
	var containerDefs interface{}
	if err := json.Unmarshal([]byte(stored.ContainerDefinitions), &containerDefs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal container definitions: %w", err)
	}

	taskDef := map[string]interface{}{
		"taskDefinitionArn":      stored.ARN,
		"family":                 stored.Family,
		"revision":               stored.Revision,
		"status":                 stored.Status,
		"networkMode":            stored.NetworkMode,
		"containerDefinitions":   containerDefs,
		"registeredAt":           stored.RegisteredAt.Format(time.RFC3339),
		"registeredBy":           "kecs",
	}

	// Add optional fields if they exist
	if stored.TaskRoleARN != "" {
		taskDef["taskRoleArn"] = stored.TaskRoleARN
	}
	if stored.ExecutionRoleARN != "" {
		taskDef["executionRoleArn"] = stored.ExecutionRoleARN
	}
	if stored.CPU != "" {
		taskDef["cpu"] = stored.CPU
	}
	if stored.Memory != "" {
		taskDef["memory"] = stored.Memory
	}

	// Parse and add JSON fields
	if stored.Volumes != "" {
		var volumes interface{}
		if err := json.Unmarshal([]byte(stored.Volumes), &volumes); err == nil {
			taskDef["volumes"] = volumes
		}
	}

	if stored.RequiresCompatibilities != "" {
		var reqCompat interface{}
		if err := json.Unmarshal([]byte(stored.RequiresCompatibilities), &reqCompat); err == nil {
			taskDef["requiresCompatibilities"] = reqCompat
			taskDef["compatibilities"] = reqCompat
		}
	}

	if stored.PlacementConstraints != "" {
		var placementConstraints interface{}
		if err := json.Unmarshal([]byte(stored.PlacementConstraints), &placementConstraints); err == nil {
			taskDef["placementConstraints"] = placementConstraints
		}
	}

	if stored.PidMode != "" {
		taskDef["pidMode"] = stored.PidMode
	}
	if stored.IpcMode != "" {
		taskDef["ipcMode"] = stored.IpcMode
	}

	if stored.DeregisteredAt != nil {
		taskDef["deregisteredAt"] = stored.DeregisteredAt.Format(time.RFC3339)
	}

	return taskDef, nil
}