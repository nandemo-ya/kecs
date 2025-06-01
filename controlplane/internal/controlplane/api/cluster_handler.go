package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

// HTTP Handlers for ECS Cluster operations

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

// CreateClusterWithStorage implements the CreateCluster operation with storage
func (s *Server) CreateClusterWithStorage(ctx context.Context, req *generated.CreateClusterRequest) (*generated.CreateClusterResponse, error) {
	log.Printf("Creating cluster: %v", req)

	// Default cluster name if not provided
	clusterName := "default"
	if req != nil && (*req)["clusterName"] != nil {
		if name, ok := (*req)["clusterName"].(string); ok && name != "" {
			clusterName = name
		}
	}

	// Generate ARN (simplified for now)
	region := "ap-northeast-1" // TODO: Get from config
	accountID := "123456789012" // TODO: Get from actual account
	arn := fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", region, accountID, clusterName)

	// Generate a random kind cluster name
	kindClusterName, err := utils.GenerateClusterName()
	if err != nil {
		// Fallback to deterministic name if random generation fails
		log.Printf("Failed to generate random cluster name: %v, using fallback", err)
		kindClusterName = fmt.Sprintf("kecs-%s", clusterName)
	}

	// Create cluster object
	cluster := &storage.Cluster{
		ID:              uuid.New().String(),
		ARN:             arn,
		Name:            clusterName,
		Status:          "ACTIVE",
		Region:          region,
		AccountID:       accountID,
		KindClusterName: kindClusterName,
		RegisteredContainerInstancesCount: 0,
		RunningTasksCount:                 0,
		PendingTasksCount:                 0,
		ActiveServicesCount:               0,
	}

	// Extract settings and configuration from request
	if req != nil {
		if settings, ok := (*req)["settings"].(map[string]interface{}); ok {
			settingsJSON, _ := json.Marshal(settings)
			cluster.Settings = string(settingsJSON)
		}
		if config, ok := (*req)["configuration"].(map[string]interface{}); ok {
			configJSON, _ := json.Marshal(config)
			cluster.Configuration = string(configJSON)
		}
		if tags, ok := (*req)["tags"].([]interface{}); ok {
			tagsJSON, _ := json.Marshal(tags)
			cluster.Tags = string(tagsJSON)
		}
	}

	// Save to storage
	if err := s.storage.ClusterStore().Create(ctx, cluster); err != nil {
		log.Printf("Failed to create cluster: %v", err)
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	// Create kind cluster and namespace
	go func() {
		ctx := context.Background()
		
		// Create kind cluster using the generated kind cluster name
		if err := s.kindManager.CreateCluster(ctx, cluster.KindClusterName); err != nil {
			log.Printf("Failed to create kind cluster %s for ECS cluster %s: %v", cluster.KindClusterName, clusterName, err)
			return
		}
		
		// Get Kubernetes client
		kubeClient, err := s.kindManager.GetKubeClient(cluster.KindClusterName)
		if err != nil {
			log.Printf("Failed to get kubernetes client for %s: %v", cluster.KindClusterName, err)
			return
		}
		
		// Create namespace
		namespaceManager := kubernetes.NewNamespaceManager(kubeClient)
		if err := namespaceManager.CreateNamespace(ctx, clusterName, region); err != nil {
			log.Printf("Failed to create namespace for %s: %v", clusterName, err)
			return
		}
		
		log.Printf("Successfully created kind cluster %s and namespace for ECS cluster %s", cluster.KindClusterName, clusterName)
	}()

	// Build response
	response := &generated.CreateClusterResponse{
		"cluster": map[string]interface{}{
			"clusterArn":                        cluster.ARN,
			"clusterName":                       cluster.Name,
			"status":                            cluster.Status,
			"registeredContainerInstancesCount": cluster.RegisteredContainerInstancesCount,
			"runningTasksCount":                 cluster.RunningTasksCount,
			"pendingTasksCount":                 cluster.PendingTasksCount,
			"activeServicesCount":               cluster.ActiveServicesCount,
		},
	}

	// Add optional fields
	if cluster.Settings != "" {
		var settings interface{}
		json.Unmarshal([]byte(cluster.Settings), &settings)
		(*response)["cluster"].(map[string]interface{})["settings"] = settings
	}
	if cluster.Configuration != "" {
		var config interface{}
		json.Unmarshal([]byte(cluster.Configuration), &config)
		(*response)["cluster"].(map[string]interface{})["configuration"] = config
	}
	if cluster.Tags != "" {
		var tags interface{}
		json.Unmarshal([]byte(cluster.Tags), &tags)
		(*response)["cluster"].(map[string]interface{})["tags"] = tags
	}

	return response, nil
}

// ListClustersWithStorage implements the ListClusters operation with storage
func (s *Server) ListClustersWithStorage(ctx context.Context, req *generated.ListClustersRequest) (*generated.ListClustersResponse, error) {
	log.Printf("Listing clusters")

	// Get all clusters from storage
	clusters, err := s.storage.ClusterStore().List(ctx)
	if err != nil {
		log.Printf("Failed to list clusters: %v", err)
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	// Build response
	clusterArns := make([]string, len(clusters))
	for i, cluster := range clusters {
		clusterArns[i] = cluster.ARN
	}

	response := &generated.ListClustersResponse{
		"clusterArns": clusterArns,
	}

	// TODO: Implement pagination with nextToken
	return response, nil
}

// DescribeClustersWithStorage implements the DescribeClusters operation with storage
func (s *Server) DescribeClustersWithStorage(ctx context.Context, req *generated.DescribeClustersRequest) (*generated.DescribeClustersResponse, error) {
	log.Printf("Describing clusters: %v", req)

	var clusterNames []string
	if req != nil && (*req)["clusters"] != nil {
		if clusters, ok := (*req)["clusters"].([]interface{}); ok {
			for _, c := range clusters {
				if name, ok := c.(string); ok {
					// Extract cluster name from ARN or use as-is
					clusterNames = append(clusterNames, extractClusterName(name))
				}
			}
		}
	}

	// If no clusters specified, describe all
	if len(clusterNames) == 0 {
		clusters, err := s.storage.ClusterStore().List(ctx)
		if err != nil {
			log.Printf("Failed to list clusters: %v", err)
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}
		
		response := buildDescribeClustersResponse(clusters, nil)
		return response, nil
	}

	// Get specific clusters
	var clusters []*storage.Cluster
	var failures []map[string]interface{}

	for _, name := range clusterNames {
		cluster, err := s.storage.ClusterStore().Get(ctx, name)
		if err != nil {
			failures = append(failures, map[string]interface{}{
				"arn":    fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:cluster/%s", name),
				"reason": "MISSING",
			})
			continue
		}
		clusters = append(clusters, cluster)
	}

	response := buildDescribeClustersResponse(clusters, failures)
	return response, nil
}

// DeleteClusterWithStorage implements the DeleteCluster operation with storage
func (s *Server) DeleteClusterWithStorage(ctx context.Context, req *generated.DeleteClusterRequest) (*generated.DeleteClusterResponse, error) {
	log.Printf("Deleting cluster: %v", req)

	var clusterName string
	if req != nil && (*req)["cluster"] != nil {
		if name, ok := (*req)["cluster"].(string); ok {
			clusterName = extractClusterName(name)
		}
	}

	if clusterName == "" {
		return nil, fmt.Errorf("cluster name is required")
	}

	// Get cluster before deletion for response
	cluster, err := s.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil {
		log.Printf("Failed to get cluster: %v", err)
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Delete from storage
	if err := s.storage.ClusterStore().Delete(ctx, clusterName); err != nil {
		log.Printf("Failed to delete cluster: %v", err)
		return nil, fmt.Errorf("failed to delete cluster: %w", err)
	}

	// Delete kind cluster and namespace
	go func() {
		ctx := context.Background()
		
		// Get Kubernetes client before deleting cluster
		kubeClient, err := s.kindManager.GetKubeClient(cluster.KindClusterName)
		if err == nil {
			// Delete namespace
			namespaceManager := kubernetes.NewNamespaceManager(kubeClient)
			if err := namespaceManager.DeleteNamespace(ctx, clusterName, cluster.Region); err != nil {
				log.Printf("Failed to delete namespace for %s: %v", clusterName, err)
			}
		}
		
		// Delete kind cluster
		if err := s.kindManager.DeleteCluster(ctx, cluster.KindClusterName); err != nil {
			log.Printf("Failed to delete kind cluster %s for ECS cluster %s: %v", cluster.KindClusterName, clusterName, err)
			return
		}
		
		log.Printf("Successfully deleted kind cluster %s and namespace for ECS cluster %s", cluster.KindClusterName, clusterName)
	}()

	// Build response
	response := &generated.DeleteClusterResponse{
		"cluster": map[string]interface{}{
			"clusterArn":                        cluster.ARN,
			"clusterName":                       cluster.Name,
			"status":                            "INACTIVE",
			"registeredContainerInstancesCount": cluster.RegisteredContainerInstancesCount,
			"runningTasksCount":                 cluster.RunningTasksCount,
			"pendingTasksCount":                 cluster.PendingTasksCount,
			"activeServicesCount":               cluster.ActiveServicesCount,
		},
	}

	return response, nil
}

// Helper functions

func extractClusterName(nameOrArn string) string {
	// If it's an ARN, extract the cluster name
	// Format: arn:aws:ecs:region:account-id:cluster/cluster-name
	if len(nameOrArn) > 0 && nameOrArn[:3] == "arn" {
		parts := splitARN(nameOrArn)
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return nameOrArn
}

func splitARN(arn string) []string {
	// Simple ARN parser
	parts := []string{}
	segments := []string{}
	current := ""
	
	for _, ch := range arn {
		if ch == ':' || ch == '/' {
			if current != "" {
				segments = append(segments, current)
			}
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		segments = append(segments, current)
	}
	
	if len(segments) >= 6 {
		// Get the resource type/name part
		if len(segments) > 6 {
			parts = segments[6:]
		}
	}
	
	return parts
}

func buildDescribeClustersResponse(clusters []*storage.Cluster, failures []map[string]interface{}) *generated.DescribeClustersResponse {
	clusterList := make([]map[string]interface{}, len(clusters))
	
	for i, cluster := range clusters {
		clusterMap := map[string]interface{}{
			"clusterArn":                        cluster.ARN,
			"clusterName":                       cluster.Name,
			"status":                            cluster.Status,
			"registeredContainerInstancesCount": cluster.RegisteredContainerInstancesCount,
			"runningTasksCount":                 cluster.RunningTasksCount,
			"pendingTasksCount":                 cluster.PendingTasksCount,
			"activeServicesCount":               cluster.ActiveServicesCount,
		}

		// Add optional fields
		if cluster.Settings != "" {
			var settings interface{}
			json.Unmarshal([]byte(cluster.Settings), &settings)
			clusterMap["settings"] = settings
		}
		if cluster.Configuration != "" {
			var config interface{}
			json.Unmarshal([]byte(cluster.Configuration), &config)
			clusterMap["configuration"] = config
		}
		if cluster.Tags != "" {
			var tags interface{}
			json.Unmarshal([]byte(cluster.Tags), &tags)
			clusterMap["tags"] = tags
		}

		clusterList[i] = clusterMap
	}

	response := &generated.DescribeClustersResponse{
		"clusters": clusterList,
	}

	if len(failures) > 0 {
		(*response)["failures"] = failures
	} else {
		(*response)["failures"] = []interface{}{}
	}

	return response
}