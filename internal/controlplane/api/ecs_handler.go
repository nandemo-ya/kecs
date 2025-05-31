package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nandemo-ya/kecs/internal/controlplane/api/generated"
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
	s.writeEmptyResponse(w)
}

func (s *Server) handleECSDescribeTaskDefinition(w http.ResponseWriter, body []byte) {
	s.writeEmptyResponse(w)
}

func (s *Server) handleECSListTaskDefinitions(w http.ResponseWriter, body []byte) {
	response := map[string]interface{}{
		"taskDefinitionArns": []interface{}{},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
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