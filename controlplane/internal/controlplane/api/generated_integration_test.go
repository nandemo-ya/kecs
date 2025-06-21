package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeneratedTypesIntegration tests that generated types work correctly with the API
func TestGeneratedTypesIntegration(t *testing.T) {
	// Create storage
	storage, err := duckdb.NewDuckDBStorage(":memory:")
	require.NoError(t, err)
	defer storage.Close()

	err = storage.Initialize(context.Background())
	require.NoError(t, err)

	// Create Kind manager
	kindManager := kubernetes.NewKindManager()

	// Create ECS API with generated types
	ecsAPI := api.NewDefaultECSAPI(storage, kindManager)

	// Create mux and register routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", generated.HandleECSRequest(ecsAPI))

	// Create test server
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("CreateCluster", func(t *testing.T) {
		// Create request using generated types
		settingName := generated.ClusterSettingName("containerInsights")
		tagKey := generated.TagKey("Environment")
		tagValue := generated.TagValue("test")
		req := &generated.CreateClusterRequest{
			ClusterName: stringPtr("test-cluster"),
			Settings: []generated.ClusterSetting{
				{
					Name:  &settingName,
					Value: stringPtr("enabled"),
				},
			},
			Tags: []generated.Tag{
				{
					Key:   &tagKey,
					Value: &tagValue,
				},
			},
		}

		// Make HTTP request
		resp, err := makeRequest(server.URL, "CreateCluster", req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response
		var createResp generated.CreateClusterResponse
		err = json.NewDecoder(resp.Body).Decode(&createResp)
		require.NoError(t, err)

		// Verify response
		assert.NotNil(t, createResp.Cluster)
		assert.Equal(t, "test-cluster", *createResp.Cluster.ClusterName)
		assert.Equal(t, "ACTIVE", *createResp.Cluster.Status)
		assert.Contains(t, *createResp.Cluster.ClusterArn, "arn:aws:ecs:")
	})

	t.Run("ListClusters", func(t *testing.T) {
		// Create request
		req := &generated.ListClustersRequest{}

		// Make HTTP request
		resp, err := makeRequest(server.URL, "ListClusters", req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response
		var listResp generated.ListClustersResponse
		err = json.NewDecoder(resp.Body).Decode(&listResp)
		require.NoError(t, err)

		// Verify response
		assert.NotNil(t, listResp.ClusterArns)
		assert.Contains(t, listResp.ClusterArns, "arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster")
	})

	t.Run("DescribeClusters", func(t *testing.T) {
		// Create request
		req := &generated.DescribeClustersRequest{
			Clusters: []string{"test-cluster"},
			Include: []generated.ClusterField{
				generated.ClusterFieldSettings,
				generated.ClusterFieldTags,
			},
		}

		// Make HTTP request
		resp, err := makeRequest(server.URL, "DescribeClusters", req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response
		var describeResp generated.DescribeClustersResponse
		err = json.NewDecoder(resp.Body).Decode(&describeResp)
		require.NoError(t, err)

		// Verify response
		assert.Len(t, describeResp.Clusters, 1)
		cluster := describeResp.Clusters[0]
		assert.Equal(t, "test-cluster", *cluster.ClusterName)
		assert.Equal(t, "ACTIVE", *cluster.Status)

		// Verify settings were saved
		assert.NotNil(t, cluster.Settings)
		assert.Len(t, cluster.Settings, 1)
		// Name is *ClusterSettingName, so we need to dereference and convert to string
		if cluster.Settings[0].Name != nil {
			assert.Equal(t, "containerInsights", string(*cluster.Settings[0].Name))
		}
		assert.Equal(t, "enabled", *cluster.Settings[0].Value)

		// Verify tags were saved
		assert.NotNil(t, cluster.Tags)
		assert.Len(t, cluster.Tags, 1)
		assert.Equal(t, "Environment", string(*cluster.Tags[0].Key))
		assert.Equal(t, "test", string(*cluster.Tags[0].Value))
	})

	t.Run("DeleteCluster", func(t *testing.T) {
		// Create request
		cluster := "test-cluster"
		req := &generated.DeleteClusterRequest{
			Cluster: &cluster,
		}

		// Make HTTP request
		resp, err := makeRequest(server.URL, "DeleteCluster", req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response
		var deleteResp generated.DeleteClusterResponse
		err = json.NewDecoder(resp.Body).Decode(&deleteResp)
		require.NoError(t, err)

		// Verify response
		assert.NotNil(t, deleteResp.Cluster)
		assert.Equal(t, "test-cluster", *deleteResp.Cluster.ClusterName)
		assert.Equal(t, "INACTIVE", *deleteResp.Cluster.Status)
	})
}

// TestGeneratedTypesJSONCompatibility tests JSON marshaling/unmarshaling
func TestGeneratedTypesJSONCompatibility(t *testing.T) {
	t.Run("CreateClusterRequest", func(t *testing.T) {
		settingName := generated.ClusterSettingName("containerInsights")
		req := &generated.CreateClusterRequest{
			ClusterName: stringPtr("test-cluster"),
			Settings: []generated.ClusterSetting{
				{
					Name:  &settingName,
					Value: stringPtr("enabled"),
				},
			},
		}

		// Marshal to JSON
		data, err := json.Marshal(req)
		require.NoError(t, err)

		// Verify JSON has camelCase fields
		var jsonMap map[string]interface{}
		err = json.Unmarshal(data, &jsonMap)
		require.NoError(t, err)

		assert.Equal(t, "test-cluster", jsonMap["clusterName"])
		settings := jsonMap["settings"].([]interface{})
		assert.Len(t, settings, 1)
		setting := settings[0].(map[string]interface{})
		assert.Equal(t, "containerInsights", setting["name"])
		assert.Equal(t, "enabled", setting["value"])
	})

	t.Run("ClusterResponse", func(t *testing.T) {
		cluster := &generated.Cluster{
			ClusterArn:  stringPtr("arn:aws:ecs:us-east-1:123456789012:cluster/test"),
			ClusterName: stringPtr("test"),
			Status:      stringPtr("ACTIVE"),
		}

		// Marshal to JSON
		data, err := json.Marshal(cluster)
		require.NoError(t, err)

		// Verify JSON has camelCase fields
		var jsonMap map[string]interface{}
		err = json.Unmarshal(data, &jsonMap)
		require.NoError(t, err)

		assert.Equal(t, "arn:aws:ecs:us-east-1:123456789012:cluster/test", jsonMap["clusterArn"])
		assert.Equal(t, "test", jsonMap["clusterName"])
		assert.Equal(t, "ACTIVE", jsonMap["status"])
	})
}

// Helper function to make HTTP requests
func makeRequest(baseURL, action string, body interface{}) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", baseURL+"/", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113."+action)

	client := &http.Client{}
	return client.Do(req)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func interfacePtr(i interface{}) *interface{} {
	return &i
}

func int32Ptr(i int32) *int32 {
	return &i
}