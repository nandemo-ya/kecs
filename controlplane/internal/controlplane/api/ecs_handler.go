package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
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
		s.handleRunTaskECS(w, body)
	case "StartTask":
		s.handleStartTaskECS(w, body)
	case "DescribeTasks":
		s.handleDescribeTasksECS(w, body)
	case "ListTasks":
		s.handleListTasksECS(w, body)
	case "RegisterTaskDefinition":
		s.handleECSRegisterTaskDefinition(w, body)
	case "DescribeTaskDefinition":
		s.handleECSDescribeTaskDefinition(w, body)
	case "DeregisterTaskDefinition":
		s.handleECSDeregisterTaskDefinition(w, body)
	case "ListTaskDefinitions":
		s.handleECSListTaskDefinitions(w, body)
	case "DeleteTaskDefinitions":
		s.handleECSDeleteTaskDefinitions(w, body)
	case "CreateService":
		s.handleECSCreateService(w, body)
	case "UpdateService":
		s.handleECSUpdateService(w, body)
	case "DeleteService":
		s.handleECSDeleteService(w, body)
	case "DescribeServices":
		s.handleECSDescribeServices(w, body)
	case "ListServices":
		s.handleECSListServices(w, body)
	case "StopTask":
		s.handleStopTaskECS(w, body)
	case "PutAttributes":
		s.handleECSPutAttributes(w, body)
	case "DeleteAttributes":
		s.handleECSDeleteAttributes(w, body)
	case "ListAttributes":
		s.handleECSListAttributes(w, body)
	case "CreateCapacityProvider":
		s.handleECSCreateCapacityProvider(w, body)
	case "UpdateCapacityProvider":
		s.handleECSUpdateCapacityProvider(w, body)
	case "DeleteCapacityProvider":
		s.handleECSDeleteCapacityProvider(w, body)
	case "DescribeCapacityProviders":
		s.handleECSDescribeCapacityProviders(w, body)
	case "RegisterContainerInstance":
		s.handleECSRegisterContainerInstance(w, body)
	case "DeregisterContainerInstance":
		s.handleECSDeregisterContainerInstance(w, body)
	case "DescribeContainerInstances":
		s.handleECSDescribeContainerInstances(w, body)
	case "ListContainerInstances":
		s.handleECSListContainerInstances(w, body)
	case "CreateTaskSet":
		s.handleECSCreateTaskSet(w, body)
	case "DeleteTaskSet":
		s.handleECSDeleteTaskSet(w, body)
	case "DescribeTaskSets":
		s.handleECSDescribeTaskSets(w, body)
	case "UpdateTaskSet":
		s.handleECSUpdateTaskSet(w, body)
	case "TagResource":
		s.handleECSTagResource(w, body)
	case "UntagResource":
		s.handleECSUntagResource(w, body)
	case "ListTagsForResource":
		s.handleECSListTagsForResource(w, body)
	default:
		// Return a basic empty response for unsupported operations
		s.handleUnsupportedOperation(w, operation)
	}
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

