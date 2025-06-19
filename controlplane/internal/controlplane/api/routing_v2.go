package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)

// ECSAPIV2 defines the interface for ECS API v2 operations using AWS SDK types
type ECSAPIV2 interface {
	ListClustersV2(ctx context.Context, req *ecs.ListClustersInput) (*ecs.ListClustersOutput, error)
	CreateClusterV2(ctx context.Context, req *ecs.CreateClusterInput) (*ecs.CreateClusterOutput, error)
	DescribeClustersV2(ctx context.Context, req *ecs.DescribeClustersInput) (*ecs.DescribeClustersOutput, error)
	DeleteClusterV2(ctx context.Context, req *ecs.DeleteClusterInput) (*ecs.DeleteClusterOutput, error)
	UpdateClusterV2(ctx context.Context, req *ecs.UpdateClusterInput) (*ecs.UpdateClusterOutput, error)
}

// handleRequestV2 is a generic handler for ECS operations using AWS SDK types
func handleRequestV2[TReq any, TResp any](
	serviceFunc func(context.Context, *TReq) (*TResp, error),
	w http.ResponseWriter,
	r *http.Request,
) {
	var req TReq
	if r.Body != nil {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		if len(body) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				http.Error(w, "Invalid request format", http.StatusBadRequest)
				return
			}
		}
	}

	resp, err := serviceFunc(r.Context(), &req)
	if err != nil {
		// TODO: Handle specific error types
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HandleECSRequestV2 routes specific ECS API requests to v2 handlers using AWS SDK types
func HandleECSRequestV2(api ECSAPIV2, mux *http.ServeMux) {
	// Register v2 handlers for migrated operations
	mux.HandleFunc("/v2/listclusters", func(w http.ResponseWriter, r *http.Request) {
		handleRequestV2(api.ListClustersV2, w, r)
	})
	mux.HandleFunc("/v2/createcluster", func(w http.ResponseWriter, r *http.Request) {
		handleRequestV2(api.CreateClusterV2, w, r)
	})
}

// AdapterMiddleware adapts requests to use either v1 (generated) or v2 (AWS SDK) handlers
func AdapterMiddleware(v1API generated.ECSAPIInterface, v2API ECSAPIV2) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get X-Amz-Target header
		target := r.Header.Get("X-Amz-Target")
		if target == "" {
			http.Error(w, "Missing X-Amz-Target header", http.StatusBadRequest)
			return
		}

		// Extract operation name (e.g., "AmazonEC2ContainerServiceV20141113.ListClusters" -> "ListClusters")
		parts := strings.Split(target, ".")
		if len(parts) != 2 {
			http.Error(w, "Invalid X-Amz-Target format", http.StatusBadRequest)
			return
		}
		operation := parts[1]

		// Route to v2 handlers for migrated operations
		switch operation {
		case "ListClusters":
			handleRequestV2(v2API.ListClustersV2, w, r)
		case "CreateCluster":
			handleRequestV2(v2API.CreateClusterV2, w, r)
		case "DescribeClusters":
			handleRequestV2(v2API.DescribeClustersV2, w, r)
		case "DeleteCluster":
			handleRequestV2(v2API.DeleteClusterV2, w, r)
		case "UpdateCluster":
			handleRequestV2(v2API.UpdateClusterV2, w, r)
		default:
			// Fall back to v1 handler for non-migrated operations
			generated.HandleECSRequest(v1API)(w, r)
		}
	}
}