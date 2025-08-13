package api

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

var (
	// Global JSON cache for API responses
	jsonCache     *utils.JSONCache
	jsonCacheOnce sync.Once
)

// getJSONCache returns the global JSON cache instance
func getJSONCache() *utils.JSONCache {
	jsonCacheOnce.Do(func() {
		// Cache up to 1000 objects for 5 minutes
		jsonCache = utils.NewJSONCache(1000, 5*time.Minute)
	})
	return jsonCache
}

// marshalCachedJSON marshals an object to JSON with caching
func marshalCachedJSON(key string, v interface{}) (string, error) {
	cache := getJSONCache()
	data, err := cache.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// unmarshalCachedJSON unmarshals JSON with potential caching of the result
func unmarshalCachedJSON(data string, v interface{}) error {
	if data == "" {
		return nil
	}
	return json.Unmarshal([]byte(data), v)
}

// parseClusterSettings parses and caches cluster settings
func parseClusterSettings(clusterName, settingsJSON string) ([]generated.ClusterSetting, error) {
	if settingsJSON == "" {
		return nil, nil
	}

	var settings []generated.ClusterSetting

	// Parse
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// parseClusterConfiguration parses and caches cluster configuration
func parseClusterConfiguration(clusterName, configJSON string) (*generated.ClusterConfiguration, error) {
	if configJSON == "" {
		return nil, nil
	}

	var config generated.ClusterConfiguration

	// Parse
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// parseTags parses and caches tags
func parseTags(resourceID, tagsJSON string) ([]generated.Tag, error) {
	if tagsJSON == "" {
		return nil, nil
	}

	var tags []generated.Tag

	// Parse
	if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
		return nil, err
	}

	return tags, nil
}

// invalidateClusterCache invalidates all cached data for a cluster
func invalidateClusterCache(clusterName string) {
	cache := getJSONCache()
	cache.Invalidate(fmt.Sprintf("cluster_settings:%s", clusterName))
	cache.Invalidate(fmt.Sprintf("cluster_config:%s", clusterName))
	cache.Invalidate(fmt.Sprintf("tags:cluster:%s", clusterName))
}

// invalidateServiceCache invalidates all cached data for a service
func invalidateServiceCache(clusterARN, serviceName string) {
	cache := getJSONCache()
	serviceID := fmt.Sprintf("%s:%s", clusterARN, serviceName)
	cache.Invalidate(fmt.Sprintf("service_config:%s", serviceID))
	cache.Invalidate(fmt.Sprintf("tags:service:%s", serviceID))
}

// invalidateTaskDefinitionCache invalidates all cached data for a task definition
func invalidateTaskDefinitionCache(taskDefARN string) {
	cache := getJSONCache()
	cache.Invalidate(fmt.Sprintf("taskdef_containers:%s", taskDefARN))
	cache.Invalidate(fmt.Sprintf("tags:taskdef:%s", taskDefARN))
}
