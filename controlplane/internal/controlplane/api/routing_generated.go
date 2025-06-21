package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	generated_v2 "github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_v2"
)

// ECSAPIGenerated defines the interface for ECS API operations using generated types
type ECSAPIGenerated interface {
	// Cluster operations
	ListClusters(ctx context.Context, req *generated_v2.ListClustersRequest) (*generated_v2.ListClustersResponse, error)
	CreateCluster(ctx context.Context, req *generated_v2.CreateClusterRequest) (*generated_v2.CreateClusterResponse, error)
	DescribeClusters(ctx context.Context, req *generated_v2.DescribeClustersRequest) (*generated_v2.DescribeClustersResponse, error)
	DeleteCluster(ctx context.Context, req *generated_v2.DeleteClusterRequest) (*generated_v2.DeleteClusterResponse, error)
	UpdateCluster(ctx context.Context, req *generated_v2.UpdateClusterRequest) (*generated_v2.UpdateClusterResponse, error)
	
	// Service operations
	CreateService(ctx context.Context, req *generated_v2.CreateServiceRequest) (*generated_v2.CreateServiceResponse, error)
	ListServices(ctx context.Context, req *generated_v2.ListServicesRequest) (*generated_v2.ListServicesResponse, error)
	DescribeServices(ctx context.Context, req *generated_v2.DescribeServicesRequest) (*generated_v2.DescribeServicesResponse, error)
	UpdateService(ctx context.Context, req *generated_v2.UpdateServiceRequest) (*generated_v2.UpdateServiceResponse, error)
	DeleteService(ctx context.Context, req *generated_v2.DeleteServiceRequest) (*generated_v2.DeleteServiceResponse, error)
	
	// Task operations
	RunTask(ctx context.Context, req *generated_v2.RunTaskRequest) (*generated_v2.RunTaskResponse, error)
	StopTask(ctx context.Context, req *generated_v2.StopTaskRequest) (*generated_v2.StopTaskResponse, error)
	DescribeTasks(ctx context.Context, req *generated_v2.DescribeTasksRequest) (*generated_v2.DescribeTasksResponse, error)
	ListTasks(ctx context.Context, req *generated_v2.ListTasksRequest) (*generated_v2.ListTasksResponse, error)
	
	// TaskDefinition operations
	RegisterTaskDefinition(ctx context.Context, req *generated_v2.RegisterTaskDefinitionRequest) (*generated_v2.RegisterTaskDefinitionResponse, error)
	DeregisterTaskDefinition(ctx context.Context, req *generated_v2.DeregisterTaskDefinitionRequest) (*generated_v2.DeregisterTaskDefinitionResponse, error)
	DescribeTaskDefinition(ctx context.Context, req *generated_v2.DescribeTaskDefinitionRequest) (*generated_v2.DescribeTaskDefinitionResponse, error)
	ListTaskDefinitionFamilies(ctx context.Context, req *generated_v2.ListTaskDefinitionFamiliesRequest) (*generated_v2.ListTaskDefinitionFamiliesResponse, error)
	ListTaskDefinitions(ctx context.Context, req *generated_v2.ListTaskDefinitionsRequest) (*generated_v2.ListTaskDefinitionsResponse, error)
	
	// Tag operations
	TagResource(ctx context.Context, req *generated_v2.TagResourceRequest) (*generated_v2.TagResourceResponse, error)
	UntagResource(ctx context.Context, req *generated_v2.UntagResourceRequest) (*generated_v2.UntagResourceResponse, error)
	ListTagsForResource(ctx context.Context, req *generated_v2.ListTagsForResourceRequest) (*generated_v2.ListTagsForResourceResponse, error)
}

// RegisterECSRoutesGenerated registers HTTP routes for ECS API using generated types
func RegisterECSRoutesGenerated(mux *http.ServeMux, api ECSAPIGenerated) {
	// Register handler for all ECS operations
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get the action from X-Amz-Target header
		target := r.Header.Get("X-Amz-Target")
		if target == "" {
			http.Error(w, "Missing X-Amz-Target header", http.StatusBadRequest)
			return
		}

		// Extract action name
		parts := strings.Split(target, ".")
		if len(parts) != 2 {
			http.Error(w, "Invalid X-Amz-Target header", http.StatusBadRequest)
			return
		}

		action := parts[1]

		// Route to appropriate handler
		switch action {
		// Cluster operations
		case "ListClusters":
			handleListClustersGenerated(w, r, api)
		case "CreateCluster":
			handleCreateClusterGenerated(w, r, api)
		case "DescribeClusters":
			handleDescribeClustersGenerated(w, r, api)
		case "DeleteCluster":
			handleDeleteClusterGenerated(w, r, api)
		case "UpdateCluster":
			handleUpdateClusterGenerated(w, r, api)
			
		// Service operations
		case "CreateService":
			handleCreateServiceGenerated(w, r, api)
		case "ListServices":
			handleListServicesGenerated(w, r, api)
		case "DescribeServices":
			handleDescribeServicesGenerated(w, r, api)
		case "UpdateService":
			handleUpdateServiceGenerated(w, r, api)
		case "DeleteService":
			handleDeleteServiceGenerated(w, r, api)
			
		// Task operations
		case "RunTask":
			handleRunTaskGenerated(w, r, api)
		case "StopTask":
			handleStopTaskGenerated(w, r, api)
		case "DescribeTasks":
			handleDescribeTasksGenerated(w, r, api)
		case "ListTasks":
			handleListTasksGenerated(w, r, api)
			
		// TaskDefinition operations
		case "RegisterTaskDefinition":
			handleRegisterTaskDefinitionGenerated(w, r, api)
		case "DeregisterTaskDefinition":
			handleDeregisterTaskDefinitionGenerated(w, r, api)
		case "DescribeTaskDefinition":
			handleDescribeTaskDefinitionGenerated(w, r, api)
		case "ListTaskDefinitionFamilies":
			handleListTaskDefinitionFamiliesGenerated(w, r, api)
		case "ListTaskDefinitions":
			handleListTaskDefinitionsGenerated(w, r, api)
			
		// Tag operations
		case "TagResource":
			handleTagResourceGenerated(w, r, api)
		case "UntagResource":
			handleUntagResourceGenerated(w, r, api)
		case "ListTagsForResource":
			handleListTagsForResourceGenerated(w, r, api)
			
		default:
			http.Error(w, "Unknown action: "+action, http.StatusBadRequest)
		}
	})
}

// Handler functions for cluster operations

func handleListClustersGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.ListClustersRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.ListClusters(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleCreateClusterGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.CreateClusterRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.CreateCluster(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleDescribeClustersGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.DescribeClustersRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.DescribeClusters(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleDeleteClusterGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.DeleteClusterRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.DeleteCluster(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleUpdateClusterGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.UpdateClusterRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.UpdateCluster(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

// Handler functions for service operations

func handleCreateServiceGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.CreateServiceRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.CreateService(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleListServicesGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.ListServicesRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.ListServices(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleDescribeServicesGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.DescribeServicesRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.DescribeServices(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleUpdateServiceGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.UpdateServiceRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.UpdateService(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleDeleteServiceGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.DeleteServiceRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.DeleteService(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

// Handler functions for task operations

func handleRunTaskGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.RunTaskRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.RunTask(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleStopTaskGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.StopTaskRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.StopTask(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleDescribeTasksGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.DescribeTasksRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.DescribeTasks(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleListTasksGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.ListTasksRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.ListTasks(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

// Handler functions for task definition operations

func handleRegisterTaskDefinitionGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.RegisterTaskDefinitionRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.RegisterTaskDefinition(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleDeregisterTaskDefinitionGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.DeregisterTaskDefinitionRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.DeregisterTaskDefinition(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleDescribeTaskDefinitionGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.DescribeTaskDefinitionRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.DescribeTaskDefinition(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleListTaskDefinitionFamiliesGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.ListTaskDefinitionFamiliesRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.ListTaskDefinitionFamilies(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleListTaskDefinitionsGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.ListTaskDefinitionsRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.ListTaskDefinitions(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

// Handler functions for tag operations

func handleTagResourceGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.TagResourceRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.TagResource(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleUntagResourceGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.UntagResourceRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.UntagResource(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

func handleListTagsForResourceGenerated(w http.ResponseWriter, r *http.Request, api ECSAPIGenerated) {
	var req generated_v2.ListTagsForResourceRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidParameterValue", err.Error())
		return
	}

	resp, err := api.ListTagsForResource(r.Context(), &req)
	if err != nil {
		writeAPIError(w, err)
		return
	}

	writeJSON(w, resp)
}

// Helper functions

func decodeJSONBody(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()
	
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	
	if len(body) == 0 {
		return nil
	}
	
	return json.Unmarshal(body, v)
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, statusCode int, errorType string, message string) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"__type":  errorType,
		"message": message,
	})
}

func writeAPIError(w http.ResponseWriter, err error) {
	// Check for specific error types
	switch err.(type) {
	case *ClusterNotFoundException:
		writeError(w, http.StatusBadRequest, "ClusterNotFoundException", err.Error())
	case *ClusterContainsServicesException:
		writeError(w, http.StatusBadRequest, "ClusterContainsServicesException", err.Error())
	case *ClusterContainsTasksException:
		writeError(w, http.StatusBadRequest, "ClusterContainsTasksException", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "InternalError", err.Error())
	}
}