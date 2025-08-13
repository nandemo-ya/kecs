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

import (
	"context"
)

// Client defines the interface for KECS API operations
type Client interface {
	// Instance operations
	ListInstances(ctx context.Context) ([]Instance, error)
	GetInstance(ctx context.Context, name string) (*Instance, error)
	CreateInstance(ctx context.Context, opts CreateInstanceOptions) (*Instance, error)
	DeleteInstance(ctx context.Context, name string) error
	GetInstanceLogs(ctx context.Context, name string, follow bool) (<-chan LogEntry, error)
	GetInstanceCreationStatus(ctx context.Context, name string) (*CreationStatus, error)

	// ECS Cluster operations
	ListClusters(ctx context.Context, instanceName string) ([]string, error)
	DescribeClusters(ctx context.Context, instanceName string, clusterNames []string) ([]Cluster, error)
	CreateCluster(ctx context.Context, instanceName, clusterName string) (*Cluster, error)
	DeleteCluster(ctx context.Context, instanceName, clusterName string) error

	// ECS Service operations
	ListServices(ctx context.Context, instanceName, clusterName string) ([]string, error)
	DescribeServices(ctx context.Context, instanceName, clusterName string, serviceNames []string) ([]Service, error)
	CreateService(ctx context.Context, instanceName, clusterName string, service Service) (*Service, error)
	UpdateService(ctx context.Context, instanceName, clusterName string, service Service) (*Service, error)
	DeleteService(ctx context.Context, instanceName, clusterName, serviceName string) error

	// ECS Task operations
	ListTasks(ctx context.Context, instanceName, clusterName string, serviceName string) ([]string, error)
	DescribeTasks(ctx context.Context, instanceName, clusterName string, taskArns []string) ([]Task, error)
	RunTask(ctx context.Context, instanceName, clusterName string, taskDef string) (*Task, error)
	StopTask(ctx context.Context, instanceName, clusterName, taskArn string) error
	GetTaskLogs(ctx context.Context, instanceName, clusterName, taskArn string, tail int64) ([]LogEntry, error)

	// Task Definition operations
	ListTaskDefinitions(ctx context.Context, instanceName string) ([]string, error)
	ListTaskDefinitionFamilies(ctx context.Context, instanceName string) ([]string, error)
	ListTaskDefinitionRevisions(ctx context.Context, instanceName string, family string) ([]TaskDefinitionRevision, error)
	DescribeTaskDefinition(ctx context.Context, instanceName string, taskDefArn string) (*TaskDefinition, error)
	RegisterTaskDefinition(ctx context.Context, instanceName string, taskDef interface{}) (string, error)
	DeregisterTaskDefinition(ctx context.Context, instanceName string, taskDefArn string) error

	// Health check
	HealthCheck(ctx context.Context, instanceName string) error

	// Cleanup
	Close() error
}

// StreamingClient defines the interface for real-time updates
type StreamingClient interface {
	// Subscribe to real-time updates
	Subscribe(ctx context.Context, instanceName string) (<-chan Event, error)
	Unsubscribe(ctx context.Context, instanceName string) error
}

// Event represents a real-time update event
type Event struct {
	Type      EventType   `json:"type"`
	Resource  string      `json:"resource"`
	Action    string      `json:"action"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// EventType represents the type of event
type EventType string

const (
	EventTypeCluster EventType = "cluster"
	EventTypeService EventType = "service"
	EventTypeTask    EventType = "task"
)
