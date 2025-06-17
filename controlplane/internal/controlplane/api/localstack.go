package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// LocalStackStatus represents the status of LocalStack integration
type LocalStackStatus struct {
	Enabled         bool                       `json:"enabled"`
	Running         bool                       `json:"running"`
	Endpoint        string                     `json:"endpoint"`
	Services        []string                   `json:"services"`
	ServiceStatuses map[string]ServiceStatus   `json:"serviceStatuses"`
	ProxyMode       string                     `json:"proxyMode"`
	ProxyEnabled    bool                       `json:"proxyEnabled"`
	ProxyEndpoint   string                     `json:"proxyEndpoint,omitempty"`
	Version         string                     `json:"version,omitempty"`
	Timestamp       string                     `json:"timestamp"`
}

// ServiceStatus represents the status of a LocalStack service
type ServiceStatus struct {
	Available bool   `json:"available"`
	Endpoint  string `json:"endpoint"`
	Error     string `json:"error,omitempty"`
}

// LocalStackService represents a LocalStack service in the dashboard
type LocalStackService struct {
	Name         string    `json:"name"`
	Available    bool      `json:"available"`
	Endpoint     string    `json:"endpoint"`
	LastChecked  time.Time `json:"lastChecked"`
	LastError    string    `json:"lastError,omitempty"`
}

// LocalStackDashboardResponse represents the LocalStack dashboard response
type LocalStackDashboardResponse struct {
	Running              bool                 `json:"running"`
	Services             []LocalStackService  `json:"services"`
	ActiveServicesCount  int                  `json:"activeServicesCount"`
	TasksUsingLocalStack int                  `json:"tasksUsingLocalStack"`
	ResourceUsage        []string             `json:"resourceUsage"`
	LastUpdated          time.Time            `json:"lastUpdated"`
}

// GetLocalStackStatus handles the LocalStack status API endpoint
func (s *Server) GetLocalStackStatus(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := LocalStackStatus{
		Enabled:         false,
		Running:         false,
		Services:        []string{},
		ServiceStatuses: make(map[string]ServiceStatus),
		ProxyMode:       "disabled",
		ProxyEnabled:    false,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}

	// Check if LocalStack manager is available
	if s.localStackManager != nil {
		status.Enabled = true
		
		// Get LocalStack status
		if s.localStackManager.IsRunning() {
			status.Running = true
			endpoint, err := s.localStackManager.GetEndpoint()
			if err == nil {
				status.Endpoint = endpoint
			}
			
			// Get configured services
			config := s.localStackManager.GetConfig()
			if config != nil {
				status.Services = config.Services
				status.Version = config.Version
			}
			
			// Check individual service statuses
			for _, service := range status.Services {
				serviceStatus := ServiceStatus{
					Available: false,
					Endpoint:  status.Endpoint,
				}
				
				// Check if service is healthy
				if err := s.localStackManager.CheckServiceHealth(service); err != nil {
					serviceStatus.Error = err.Error()
				} else {
					serviceStatus.Available = true
				}
				
				status.ServiceStatuses[service] = serviceStatus
			}
		}
		
		// Get proxy configuration from AWS proxy router if available
		if s.awsProxyRouter != nil {
			// For now, set proxy as enabled if AWS proxy router is available
			status.ProxyEnabled = true
			status.ProxyMode = "environment" // Default mode
			if status.Endpoint != "" {
				status.ProxyEndpoint = status.Endpoint
			}
		}
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetLocalStackDashboard handles the LocalStack dashboard API endpoint
func (s *Server) GetLocalStackDashboard(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := LocalStackDashboardResponse{
		Running:              false,
		Services:             []LocalStackService{},
		ActiveServicesCount:  0,
		TasksUsingLocalStack: 0,
		ResourceUsage:        []string{},
		LastUpdated:          time.Now(),
	}

	// Check if LocalStack manager is available
	if s.localStackManager != nil && s.localStackManager.IsRunning() {
		response.Running = true
		
		// Get LocalStack status
		status, _ := s.localStackManager.GetStatus()
		if status != nil {
			// Convert enabled services to dashboard format
			for serviceName, serviceInfo := range status.ServiceStatus {
				service := LocalStackService{
					Name:        serviceName,
					Available:   serviceInfo.Healthy,
					Endpoint:    serviceInfo.Endpoint,
					LastChecked: time.Now(),
				}
				response.Services = append(response.Services, service)
				
				if serviceInfo.Healthy {
					response.ActiveServicesCount++
				}
			}
		}
	}

	// Count tasks using LocalStack
	if s.storage != nil {
		services, _, err := s.storage.ServiceStore().List(r.Context(), "", "", "", 0, "")
		if err == nil {
			resourceMap := make(map[string]bool)
			
			for _, service := range services {
				if service.TaskDefinitionARN != "" {
					// Extract task definition family and revision from ARN
					parts := strings.Split(service.TaskDefinitionARN, ":")
					if len(parts) >= 6 {
						taskDefID := parts[5]
						if strings.Contains(taskDefID, "/") {
							taskDefID = strings.Split(taskDefID, "/")[1]
						}
						
						// Get task definition
						taskDef, err := s.storage.TaskDefinitionStore().GetByARN(r.Context(), service.TaskDefinitionARN)
						if err == nil && taskDef.ContainerDefinitions != "" {
							// Parse container definitions
							var containers []map[string]interface{}
							if err := json.Unmarshal([]byte(taskDef.ContainerDefinitions), &containers); err == nil {
								hasLocalStack := false
								
								for _, container := range containers {
									// Check environment variables
									if envVars, ok := container["environment"].([]interface{}); ok {
										for _, envVar := range envVars {
											if env, ok := envVar.(map[string]interface{}); ok {
												if name, ok := env["name"].(string); ok {
													// Check for AWS service usage
													if strings.HasPrefix(name, "AWS_") || 
														strings.Contains(name, "_BUCKET") ||
														strings.Contains(name, "_TABLE") ||
														strings.Contains(name, "_QUEUE") ||
														strings.Contains(name, "_TOPIC") {
														hasLocalStack = true
														
														// Detect resource types
														if strings.Contains(name, "S3") || strings.Contains(name, "_BUCKET") {
															resourceMap["s3"] = true
														}
														if strings.Contains(name, "DYNAMODB") || strings.Contains(name, "_TABLE") {
															resourceMap["dynamodb"] = true
														}
														if strings.Contains(name, "SQS") || strings.Contains(name, "_QUEUE") {
															resourceMap["sqs"] = true
														}
														if strings.Contains(name, "SNS") || strings.Contains(name, "_TOPIC") {
															resourceMap["sns"] = true
														}
														if strings.Contains(name, "AWS_") {
															resourceMap["aws"] = true
														}
													}
												}
											}
										}
									}
								}
								
								if hasLocalStack {
									response.TasksUsingLocalStack++
								}
							}
						}
					}
				}
			}
			
			// Convert resource map to slice
			for resource := range resourceMap {
				response.ResourceUsage = append(response.ResourceUsage, resource)
			}
		}
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}