package ecs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nandemo-ya/kecs/controlplane/internal/awsclient"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_v2"
)

func TestClient_ListClusters(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/x-amz-json-1.1", r.Header.Get("Content-Type"))
		assert.Equal(t, "AmazonEC2ContainerServiceV20141113.ListClusters", r.Header.Get("X-Amz-Target"))

		// Send response
		response := api.ListClustersResponse{
			ClusterArns: []string{
				"arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster-1",
				"arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster-2",
			},
		}

		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewClient(awsclient.Config{
		Endpoint: server.URL,
		Credentials: awsclient.Credentials{
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
		},
		Region: "us-east-1",
	})

	// Make request
	input := &api.ListClustersRequest{}
	output, err := client.ListClusters(context.Background(), input)

	// Verify response
	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output.ClusterArns, 2)
	assert.Contains(t, output.ClusterArns[0], "test-cluster-1")
	assert.Contains(t, output.ClusterArns[1], "test-cluster-2")
}

func TestClient_CreateCluster(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "AmazonEC2ContainerServiceV20141113.CreateCluster", r.Header.Get("X-Amz-Target"))

		// Parse request
		var req api.CreateClusterRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "test-cluster", *req.ClusterName)

		// Send response
		clusterName := "test-cluster"
		clusterArn := "arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster"
		status := "ACTIVE"
		
		response := api.CreateClusterResponse{
			Cluster: &api.Cluster{
				ClusterName: &clusterName,
				ClusterArn:  &clusterArn,
				Status:      &status,
			},
		}

		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	client := NewClient(awsclient.Config{
		Endpoint: server.URL,
		Credentials: awsclient.Credentials{
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
		},
		Region: "us-east-1",
	})

	// Make request
	clusterName := "test-cluster"
	input := &api.CreateClusterRequest{
		ClusterName: &clusterName,
	}
	output, err := client.CreateCluster(context.Background(), input)

	// Verify response
	require.NoError(t, err)
	require.NotNil(t, output)
	require.NotNil(t, output.Cluster)
	assert.Equal(t, "test-cluster", *output.Cluster.ClusterName)
	assert.Equal(t, "ACTIVE", *output.Cluster.Status)
}

func TestClient_ErrorHandling(t *testing.T) {
	// Create test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Send error response
		errorResponse := map[string]string{
			"__type": "ClusterNotFoundException",
			"message": "The referenced cluster was not found",
		}

		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
	}))
	defer server.Close()

	// Create client
	client := NewClient(awsclient.Config{
		Endpoint: server.URL,
		Credentials: awsclient.Credentials{
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
		},
		Region: "us-east-1",
	})

	// Make request
	clusterName := "non-existent-cluster"
	input := &api.DeleteClusterRequest{
		Cluster: clusterName,
	}
	_, err := client.DeleteCluster(context.Background(), input)

	// Verify error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ClusterNotFoundException")
	assert.Contains(t, err.Error(), "The referenced cluster was not found")
}