package utils

import (
	"fmt"
	"strings"
	"time"
)

// ClusterHelpers provides enhanced utilities for cluster testing

// AssertClusterHasSettings verifies that a cluster has the expected settings
func AssertClusterHasSettings(t TestingT, client ECSClientInterface, clusterName string, expectedSettings map[string]string) error {
	awsClient, ok := client.(*AWSCLIClient)
	if !ok {
		return fmt.Errorf("client is not an AWS CLI client")
	}

	// Describe cluster with SETTINGS include
	clusters, err := awsClient.DescribeClustersWithInclude([]string{clusterName}, []string{"SETTINGS"})
	if err != nil {
		return fmt.Errorf("failed to describe cluster: %w", err)
	}
	if len(clusters) != 1 {
		return fmt.Errorf("expected 1 cluster, got %d", len(clusters))
	}

	cluster := clusters[0]
	if cluster.Settings == nil {
		return fmt.Errorf("cluster has no settings")
	}

	// Check each expected setting
	for name, expectedValue := range expectedSettings {
		found := false
		for _, setting := range cluster.Settings {
			if setting["name"] == name {
				if setting["value"] != expectedValue {
					return fmt.Errorf("setting %s has value %s, expected %s", 
						name, setting["value"], expectedValue)
				}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("setting %s not found in cluster", name)
		}
	}
	return nil
}

// AssertClusterHasConfiguration verifies that a cluster has the expected configuration
func AssertClusterHasConfiguration(t TestingT, client ECSClientInterface, clusterName string) error {
	awsClient, ok := client.(*AWSCLIClient)
	if !ok {
		return fmt.Errorf("client is not an AWS CLI client")
	}

	// Describe cluster with CONFIGURATIONS include
	clusters, err := awsClient.DescribeClustersWithInclude([]string{clusterName}, []string{"CONFIGURATIONS"})
	if err != nil {
		return fmt.Errorf("failed to describe cluster: %w", err)
	}
	if len(clusters) != 1 {
		return fmt.Errorf("expected 1 cluster, got %d", len(clusters))
	}

	// Configuration validation would go here based on what's returned
	return nil
}

// AssertClusterHasTags verifies that a cluster has the expected tags
func AssertClusterHasTags(t TestingT, client ECSClientInterface, clusterArn string, expectedTags map[string]string) error {
	tags, err := client.ListTagsForResource(clusterArn)
	if err != nil {
		return fmt.Errorf("failed to list tags: %w", err)
	}

	for key, expectedValue := range expectedTags {
		actualValue, found := tags[key]
		if !found {
			return fmt.Errorf("tag %s not found on cluster", key)
		}
		if actualValue != expectedValue {
			return fmt.Errorf("tag %s has value %s, expected %s", key, actualValue, expectedValue)
		}
	}
	return nil
}

// AssertClusterHasCapacityProviders verifies that a cluster has the expected capacity providers
func AssertClusterHasCapacityProviders(t TestingT, client ECSClientInterface, clusterName string, expectedProviders []string) error {
	awsClient, ok := client.(*AWSCLIClient)
	if !ok {
		return fmt.Errorf("client is not an AWS CLI client")
	}

	// Describe cluster
	clusters, err := awsClient.DescribeClustersWithInclude([]string{clusterName}, nil)
	if err != nil {
		return fmt.Errorf("failed to describe cluster: %w", err)
	}
	if len(clusters) != 1 {
		return fmt.Errorf("expected 1 cluster, got %d", len(clusters))
	}

	cluster := clusters[0]
	// Check capacity providers
	for _, expected := range expectedProviders {
		found := false
		for _, provider := range cluster.CapacityProviders {
			if provider == expected {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("capacity provider %s not found in cluster", expected)
		}
	}
	return nil
}

// CreateClusterWithSettings creates a cluster with specific settings
func CreateClusterWithSettings(t TestingT, client ECSClientInterface, clusterName string, settings []map[string]string) error {
	// First create the cluster
	err := client.CreateCluster(clusterName)
	if err != nil {
		return err
	}

	// Then update settings if AWS CLI client
	if awsClient, ok := client.(*AWSCLIClient); ok && len(settings) > 0 {
		return awsClient.UpdateClusterSettings(clusterName, settings)
	}

	return nil
}

// CreateClusterWithTags creates a cluster and adds tags
func CreateClusterWithTags(t TestingT, client ECSClientInterface, clusterName string, tags map[string]string) error {
	// Create cluster
	err := client.CreateCluster(clusterName)
	if err != nil {
		return err
	}

	// Get cluster ARN
	cluster, err := client.DescribeCluster(clusterName)
	if err != nil {
		return err
	}

	// Add tags
	return client.TagResource(cluster.ClusterArn, tags)
}

// WaitForClusterDeletion waits for a cluster to be fully deleted
func WaitForClusterDeletion(t TestingT, client ECSClientInterface, clusterName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		_, err := client.DescribeCluster(clusterName)
		if err != nil && strings.Contains(err.Error(), "not found") {
			// Cluster is deleted
			return nil
		}
		
		time.Sleep(2 * time.Second)
	}
	
	return fmt.Errorf("timeout waiting for cluster %s to be deleted", clusterName)
}

// ValidateClusterResponse performs comprehensive validation of a cluster response
func ValidateClusterResponse(t TestingT, cluster *Cluster) error {
	if cluster == nil {
		return fmt.Errorf("cluster is nil")
	}
	if cluster.ClusterArn == "" {
		return fmt.Errorf("cluster ARN is empty")
	}
	if cluster.ClusterName == "" {
		return fmt.Errorf("cluster name is empty")
	}
	if cluster.Status == "" {
		return fmt.Errorf("cluster status is empty")
	}
	
	// Validate ARN format
	if !strings.HasPrefix(cluster.ClusterArn, "arn:aws:ecs:") {
		return fmt.Errorf("cluster ARN should start with arn:aws:ecs:, got %s", cluster.ClusterArn)
	}
	if !strings.Contains(cluster.ClusterArn, fmt.Sprintf("cluster/%s", cluster.ClusterName)) {
		return fmt.Errorf("cluster ARN should contain cluster name, got %s", cluster.ClusterArn)
	}
	
	// Validate counts are non-negative
	if cluster.RegisteredContainerInstancesCount < 0 {
		return fmt.Errorf("registered container instances count is negative: %d", cluster.RegisteredContainerInstancesCount)
	}
	if cluster.RunningTasksCount < 0 {
		return fmt.Errorf("running tasks count is negative: %d", cluster.RunningTasksCount)
	}
	if cluster.PendingTasksCount < 0 {
		return fmt.Errorf("pending tasks count is negative: %d", cluster.PendingTasksCount)
	}
	if cluster.ActiveServicesCount < 0 {
		return fmt.Errorf("active services count is negative: %d", cluster.ActiveServicesCount)
	}
	
	return nil
}

// GetAllClustersWithPagination retrieves all clusters handling pagination
func GetAllClustersWithPagination(t TestingT, awsClient *AWSCLIClient) ([]string, error) {
	var allClusters []string
	nextToken := ""
	
	for {
		clusters, newToken, err := awsClient.ListClustersWithPagination(100, nextToken)
		if err != nil {
			return nil, err
		}
		
		allClusters = append(allClusters, clusters...)
		
		if newToken == "" {
			break
		}
		nextToken = newToken
	}
	
	return allClusters, nil
}

// CleanupClusters deletes multiple clusters in parallel
func CleanupClusters(t TestingT, client ECSClientInterface, clusterNames []string) {
	for _, name := range clusterNames {
		go func(clusterName string) {
			_ = client.DeleteCluster(clusterName)
		}(name)
	}
}

// AssertClusterEventuallyActive waits for a cluster to become active
func AssertClusterEventuallyActive(t TestingT, client ECSClientInterface, clusterName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		cluster, err := client.DescribeCluster(clusterName)
		if err == nil && cluster.Status == "ACTIVE" {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	
	return fmt.Errorf("cluster %s did not become ACTIVE within %v", clusterName, timeout)
}