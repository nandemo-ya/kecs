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
	// Cluster operations
	ListClustersV2(ctx context.Context, req *ecs.ListClustersInput) (*ecs.ListClustersOutput, error)
	CreateClusterV2(ctx context.Context, req *ecs.CreateClusterInput) (*ecs.CreateClusterOutput, error)
	DescribeClustersV2(ctx context.Context, req *ecs.DescribeClustersInput) (*ecs.DescribeClustersOutput, error)
	DeleteClusterV2(ctx context.Context, req *ecs.DeleteClusterInput) (*ecs.DeleteClusterOutput, error)
	UpdateClusterV2(ctx context.Context, req *ecs.UpdateClusterInput) (*ecs.UpdateClusterOutput, error)
	
	// Service operations
	CreateServiceV2(ctx context.Context, req *ecs.CreateServiceInput) (*ecs.CreateServiceOutput, error)
	ListServicesV2(ctx context.Context, req *ecs.ListServicesInput) (*ecs.ListServicesOutput, error)
	DescribeServicesV2(ctx context.Context, req *ecs.DescribeServicesInput) (*ecs.DescribeServicesOutput, error)
	UpdateServiceV2(ctx context.Context, req *ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error)
	DeleteServiceV2(ctx context.Context, req *ecs.DeleteServiceInput) (*ecs.DeleteServiceOutput, error)
	
	// Task operations
	RunTaskV2(ctx context.Context, req *ecs.RunTaskInput) (*ecs.RunTaskOutput, error)
	StopTaskV2(ctx context.Context, req *ecs.StopTaskInput) (*ecs.StopTaskOutput, error)
	DescribeTasksV2(ctx context.Context, req *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error)
	ListTasksV2(ctx context.Context, req *ecs.ListTasksInput) (*ecs.ListTasksOutput, error)
	
	// TaskDefinition operations
	RegisterTaskDefinitionV2(ctx context.Context, req *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error)
	DeregisterTaskDefinitionV2(ctx context.Context, req *ecs.DeregisterTaskDefinitionInput) (*ecs.DeregisterTaskDefinitionOutput, error)
	DescribeTaskDefinitionV2(ctx context.Context, req *ecs.DescribeTaskDefinitionInput) (*ecs.DescribeTaskDefinitionOutput, error)
	ListTaskDefinitionFamiliesV2(ctx context.Context, req *ecs.ListTaskDefinitionFamiliesInput) (*ecs.ListTaskDefinitionFamiliesOutput, error)
	ListTaskDefinitionsV2(ctx context.Context, req *ecs.ListTaskDefinitionsInput) (*ecs.ListTaskDefinitionsOutput, error)
	
	// Tag operations
	TagResourceV2(ctx context.Context, req *ecs.TagResourceInput) (*ecs.TagResourceOutput, error)
	UntagResourceV2(ctx context.Context, req *ecs.UntagResourceInput) (*ecs.UntagResourceOutput, error)
	ListTagsForResourceV2(ctx context.Context, req *ecs.ListTagsForResourceInput) (*ecs.ListTagsForResourceOutput, error)
	
	// Container Instance operations
	RegisterContainerInstanceV2(ctx context.Context, req *ecs.RegisterContainerInstanceInput) (*ecs.RegisterContainerInstanceOutput, error)
	DeregisterContainerInstanceV2(ctx context.Context, req *ecs.DeregisterContainerInstanceInput) (*ecs.DeregisterContainerInstanceOutput, error)
	DescribeContainerInstancesV2(ctx context.Context, req *ecs.DescribeContainerInstancesInput) (*ecs.DescribeContainerInstancesOutput, error)
	ListContainerInstancesV2(ctx context.Context, req *ecs.ListContainerInstancesInput) (*ecs.ListContainerInstancesOutput, error)
	UpdateContainerAgentV2(ctx context.Context, req *ecs.UpdateContainerAgentInput) (*ecs.UpdateContainerAgentOutput, error)
	UpdateContainerInstancesStateV2(ctx context.Context, req *ecs.UpdateContainerInstancesStateInput) (*ecs.UpdateContainerInstancesStateOutput, error)
	SubmitContainerStateChangeV2(ctx context.Context, req *ecs.SubmitContainerStateChangeInput) (*ecs.SubmitContainerStateChangeOutput, error)
	
	// Capacity Provider operations
	CreateCapacityProviderV2(ctx context.Context, req *ecs.CreateCapacityProviderInput) (*ecs.CreateCapacityProviderOutput, error)
	DeleteCapacityProviderV2(ctx context.Context, req *ecs.DeleteCapacityProviderInput) (*ecs.DeleteCapacityProviderOutput, error)
	DescribeCapacityProvidersV2(ctx context.Context, req *ecs.DescribeCapacityProvidersInput) (*ecs.DescribeCapacityProvidersOutput, error)
	UpdateCapacityProviderV2(ctx context.Context, req *ecs.UpdateCapacityProviderInput) (*ecs.UpdateCapacityProviderOutput, error)
	
	// Task Set operations
	CreateTaskSetV2(ctx context.Context, req *ecs.CreateTaskSetInput) (*ecs.CreateTaskSetOutput, error)
	DeleteTaskSetV2(ctx context.Context, req *ecs.DeleteTaskSetInput) (*ecs.DeleteTaskSetOutput, error)
	DescribeTaskSetsV2(ctx context.Context, req *ecs.DescribeTaskSetsInput) (*ecs.DescribeTaskSetsOutput, error)
	UpdateTaskSetV2(ctx context.Context, req *ecs.UpdateTaskSetInput) (*ecs.UpdateTaskSetOutput, error)
	UpdateServicePrimaryTaskSetV2(ctx context.Context, req *ecs.UpdateServicePrimaryTaskSetInput) (*ecs.UpdateServicePrimaryTaskSetOutput, error)
	
	// Account Settings operations
	PutAccountSettingV2(ctx context.Context, req *ecs.PutAccountSettingInput) (*ecs.PutAccountSettingOutput, error)
	PutAccountSettingDefaultV2(ctx context.Context, req *ecs.PutAccountSettingDefaultInput) (*ecs.PutAccountSettingDefaultOutput, error)
	DeleteAccountSettingV2(ctx context.Context, req *ecs.DeleteAccountSettingInput) (*ecs.DeleteAccountSettingOutput, error)
	ListAccountSettingsV2(ctx context.Context, req *ecs.ListAccountSettingsInput) (*ecs.ListAccountSettingsOutput, error)
	
	// Attributes operations
	PutAttributesV2(ctx context.Context, req *ecs.PutAttributesInput) (*ecs.PutAttributesOutput, error)
	DeleteAttributesV2(ctx context.Context, req *ecs.DeleteAttributesInput) (*ecs.DeleteAttributesOutput, error)
	ListAttributesV2(ctx context.Context, req *ecs.ListAttributesInput) (*ecs.ListAttributesOutput, error)
	
	// Misc operations
	DiscoverPollEndpointV2(ctx context.Context, req *ecs.DiscoverPollEndpointInput) (*ecs.DiscoverPollEndpointOutput, error)
	ExecuteCommandV2(ctx context.Context, req *ecs.ExecuteCommandInput) (*ecs.ExecuteCommandOutput, error)
	SubmitAttachmentStateChangesV2(ctx context.Context, req *ecs.SubmitAttachmentStateChangesInput) (*ecs.SubmitAttachmentStateChangesOutput, error)
	SubmitTaskStateChangeV2(ctx context.Context, req *ecs.SubmitTaskStateChangeInput) (*ecs.SubmitTaskStateChangeOutput, error)
	DeleteTaskDefinitionsV2(ctx context.Context, req *ecs.DeleteTaskDefinitionsInput) (*ecs.DeleteTaskDefinitionsOutput, error)
	GetTaskProtectionV2(ctx context.Context, req *ecs.GetTaskProtectionInput) (*ecs.GetTaskProtectionOutput, error)
	UpdateTaskProtectionV2(ctx context.Context, req *ecs.UpdateTaskProtectionInput) (*ecs.UpdateTaskProtectionOutput, error)
	StartTaskV2(ctx context.Context, req *ecs.StartTaskInput) (*ecs.StartTaskOutput, error)
	PutClusterCapacityProvidersV2(ctx context.Context, req *ecs.PutClusterCapacityProvidersInput) (*ecs.PutClusterCapacityProvidersOutput, error)
	UpdateClusterSettingsV2(ctx context.Context, req *ecs.UpdateClusterSettingsInput) (*ecs.UpdateClusterSettingsOutput, error)
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
		// Cluster operations
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
		// Service operations
		case "CreateService":
			handleRequestV2(v2API.CreateServiceV2, w, r)
		case "ListServices":
			handleRequestV2(v2API.ListServicesV2, w, r)
		case "DescribeServices":
			handleRequestV2(v2API.DescribeServicesV2, w, r)
		case "UpdateService":
			handleRequestV2(v2API.UpdateServiceV2, w, r)
		case "DeleteService":
			handleRequestV2(v2API.DeleteServiceV2, w, r)
		// Task operations
		case "RunTask":
			handleRequestV2(v2API.RunTaskV2, w, r)
		case "StopTask":
			handleRequestV2(v2API.StopTaskV2, w, r)
		case "DescribeTasks":
			handleRequestV2(v2API.DescribeTasksV2, w, r)
		case "ListTasks":
			handleRequestV2(v2API.ListTasksV2, w, r)
		// TaskDefinition operations
		case "RegisterTaskDefinition":
			handleRequestV2(v2API.RegisterTaskDefinitionV2, w, r)
		case "DeregisterTaskDefinition":
			handleRequestV2(v2API.DeregisterTaskDefinitionV2, w, r)
		case "DescribeTaskDefinition":
			handleRequestV2(v2API.DescribeTaskDefinitionV2, w, r)
		case "ListTaskDefinitionFamilies":
			handleRequestV2(v2API.ListTaskDefinitionFamiliesV2, w, r)
		case "ListTaskDefinitions":
			handleRequestV2(v2API.ListTaskDefinitionsV2, w, r)
		// Tag operations
		case "TagResource":
			handleRequestV2(v2API.TagResourceV2, w, r)
		case "UntagResource":
			handleRequestV2(v2API.UntagResourceV2, w, r)
		case "ListTagsForResource":
			handleRequestV2(v2API.ListTagsForResourceV2, w, r)
		// Container Instance operations
		case "RegisterContainerInstance":
			handleRequestV2(v2API.RegisterContainerInstanceV2, w, r)
		case "DeregisterContainerInstance":
			handleRequestV2(v2API.DeregisterContainerInstanceV2, w, r)
		case "DescribeContainerInstances":
			handleRequestV2(v2API.DescribeContainerInstancesV2, w, r)
		case "ListContainerInstances":
			handleRequestV2(v2API.ListContainerInstancesV2, w, r)
		case "UpdateContainerAgent":
			handleRequestV2(v2API.UpdateContainerAgentV2, w, r)
		case "UpdateContainerInstancesState":
			handleRequestV2(v2API.UpdateContainerInstancesStateV2, w, r)
		case "SubmitContainerStateChange":
			handleRequestV2(v2API.SubmitContainerStateChangeV2, w, r)
		// Capacity Provider operations
		case "CreateCapacityProvider":
			handleRequestV2(v2API.CreateCapacityProviderV2, w, r)
		case "DeleteCapacityProvider":
			handleRequestV2(v2API.DeleteCapacityProviderV2, w, r)
		case "DescribeCapacityProviders":
			handleRequestV2(v2API.DescribeCapacityProvidersV2, w, r)
		case "UpdateCapacityProvider":
			handleRequestV2(v2API.UpdateCapacityProviderV2, w, r)
		// Task Set operations
		case "CreateTaskSet":
			handleRequestV2(v2API.CreateTaskSetV2, w, r)
		case "DeleteTaskSet":
			handleRequestV2(v2API.DeleteTaskSetV2, w, r)
		case "DescribeTaskSets":
			handleRequestV2(v2API.DescribeTaskSetsV2, w, r)
		case "UpdateTaskSet":
			handleRequestV2(v2API.UpdateTaskSetV2, w, r)
		case "UpdateServicePrimaryTaskSet":
			handleRequestV2(v2API.UpdateServicePrimaryTaskSetV2, w, r)
		// Account Settings operations
		case "PutAccountSetting":
			handleRequestV2(v2API.PutAccountSettingV2, w, r)
		case "PutAccountSettingDefault":
			handleRequestV2(v2API.PutAccountSettingDefaultV2, w, r)
		case "DeleteAccountSetting":
			handleRequestV2(v2API.DeleteAccountSettingV2, w, r)
		case "ListAccountSettings":
			handleRequestV2(v2API.ListAccountSettingsV2, w, r)
		// Attributes operations
		case "PutAttributes":
			handleRequestV2(v2API.PutAttributesV2, w, r)
		case "DeleteAttributes":
			handleRequestV2(v2API.DeleteAttributesV2, w, r)
		case "ListAttributes":
			handleRequestV2(v2API.ListAttributesV2, w, r)
		// Misc operations
		case "DiscoverPollEndpoint":
			handleRequestV2(v2API.DiscoverPollEndpointV2, w, r)
		case "ExecuteCommand":
			handleRequestV2(v2API.ExecuteCommandV2, w, r)
		case "SubmitAttachmentStateChanges":
			handleRequestV2(v2API.SubmitAttachmentStateChangesV2, w, r)
		case "SubmitTaskStateChange":
			handleRequestV2(v2API.SubmitTaskStateChangeV2, w, r)
		case "DeleteTaskDefinitions":
			handleRequestV2(v2API.DeleteTaskDefinitionsV2, w, r)
		case "GetTaskProtection":
			handleRequestV2(v2API.GetTaskProtectionV2, w, r)
		case "UpdateTaskProtection":
			handleRequestV2(v2API.UpdateTaskProtectionV2, w, r)
		case "StartTask":
			handleRequestV2(v2API.StartTaskV2, w, r)
		case "PutClusterCapacityProviders":
			handleRequestV2(v2API.PutClusterCapacityProvidersV2, w, r)
		case "UpdateClusterSettings":
			handleRequestV2(v2API.UpdateClusterSettingsV2, w, r)
		default:
			// Fall back to v1 handler for non-migrated operations
			generated.HandleECSRequest(v1API)(w, r)
		}
	}
}