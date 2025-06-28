// Copyright 2025 The KECS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import "time"

// Cluster represents an ECS cluster
type Cluster struct {
	ClusterArn                     string    `json:"clusterArn"`
	ClusterName                    string    `json:"clusterName"`
	Status                         string    `json:"status"`
	RegisteredContainerInstancesCount int     `json:"registeredContainerInstancesCount"`
	RunningTasksCount              int       `json:"runningTasksCount"`
	PendingTasksCount              int       `json:"pendingTasksCount"`
	ActiveServicesCount            int       `json:"activeServicesCount"`
	Tags                           []Tag     `json:"tags,omitempty"`
	CreatedAt                      time.Time `json:"createdAt,omitempty"`
}

// Tag represents a key-value tag
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ListClustersRequest represents a request to list clusters
type ListClustersRequest struct {
	NextToken  string `json:"nextToken,omitempty"`
	MaxResults int    `json:"maxResults,omitempty"`
}

// ListClustersResponse represents a response from ListClusters
type ListClustersResponse struct {
	ClusterArns []string `json:"clusterArns"`
	NextToken   string   `json:"nextToken,omitempty"`
}

// DescribeClustersRequest represents a request to describe clusters
type DescribeClustersRequest struct {
	Clusters []string `json:"clusters"`
	Include  []string `json:"include,omitempty"`
}

// DescribeClustersResponse represents a response from DescribeClusters
type DescribeClustersResponse struct {
	Clusters []Cluster `json:"clusters"`
	Failures []Failure `json:"failures,omitempty"`
}

// CreateClusterRequest represents a request to create a cluster
type CreateClusterRequest struct {
	ClusterName string `json:"clusterName"`
	Tags        []Tag  `json:"tags,omitempty"`
}

// CreateClusterResponse represents a response from CreateCluster
type CreateClusterResponse struct {
	Cluster Cluster `json:"cluster"`
}

// DeleteClusterRequest represents a request to delete a cluster
type DeleteClusterRequest struct {
	Cluster string `json:"cluster"`
}

// DeleteClusterResponse represents a response from DeleteCluster
type DeleteClusterResponse struct {
	Cluster Cluster `json:"cluster"`
}

// Failure represents an API failure
type Failure struct {
	Arn    string `json:"arn"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}