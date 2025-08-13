package storage

import (
	"encoding/json"
	"time"
)

// LocalStackState represents the state of LocalStack deployment in a cluster
type LocalStackState struct {
	// Whether LocalStack is deployed
	Deployed bool `json:"deployed"`

	// Deployment status (pending, running, failed, stopped)
	Status string `json:"status"`

	// LocalStack version
	Version string `json:"version,omitempty"`

	// LocalStack namespace
	Namespace string `json:"namespace,omitempty"`

	// LocalStack pod name
	PodName string `json:"podName,omitempty"`

	// LocalStack service endpoint
	Endpoint string `json:"endpoint,omitempty"`

	// Deployment timestamp
	DeployedAt *time.Time `json:"deployedAt,omitempty"`

	// Last health check timestamp
	LastHealthCheck *time.Time `json:"lastHealthCheck,omitempty"`

	// Health status
	HealthStatus string `json:"healthStatus,omitempty"`
}

// SerializeLocalStackState converts LocalStackState to JSON string
func SerializeLocalStackState(state *LocalStackState) (string, error) {
	if state == nil {
		return "", nil
	}
	data, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DeserializeLocalStackState converts JSON string to LocalStackState
func DeserializeLocalStackState(data string) (*LocalStackState, error) {
	if data == "" {
		return nil, nil
	}
	var state LocalStackState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, err
	}
	return &state, nil
}
