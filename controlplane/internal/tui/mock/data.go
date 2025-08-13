package mock

import (
	"fmt"
	"math/rand"
	"time"
)

// InstanceData holds mock instance data
type InstanceData struct {
	Name       string
	Status     string
	Clusters   int
	Services   int
	Tasks      int
	APIPort    int
	AdminPort  int
	LocalStack bool
	Traefik    bool
	DevMode    bool
	Age        time.Duration
}

// ClusterData holds mock cluster data
type ClusterData struct {
	Name        string
	Status      string
	Services    int
	Tasks       int
	CPUUsed     float64
	CPUTotal    float64
	MemoryUsed  string
	MemoryTotal string
	Namespace   string
	Age         time.Duration
}

// ServiceData holds mock service data
type ServiceData struct {
	Name    string
	Desired int
	Running int
	Pending int
	Status  string
	TaskDef string
	Age     time.Duration
}

// TaskData holds mock task data
type TaskData struct {
	ID      string
	Service string
	Status  string
	Health  string
	CPU     float64
	Memory  string
	IP      string
	Age     time.Duration
}

// LogData holds mock log data
type LogData struct {
	Timestamp time.Time
	Level     string
	Message   string
}

// Predefined mock data
var (
	mockInstances = []InstanceData{
		{Name: "development", Status: "ACTIVE", Clusters: 3, Services: 12, Tasks: 28, APIPort: 8080, AdminPort: 8081, LocalStack: true, Traefik: true, DevMode: true, Age: 5 * 24 * time.Hour},
		{Name: "staging", Status: "ACTIVE", Clusters: 2, Services: 8, Tasks: 18, APIPort: 8090, AdminPort: 8091, LocalStack: true, Traefik: true, DevMode: false, Age: 3 * 24 * time.Hour},
		{Name: "testing", Status: "STOPPED", Clusters: 1, Services: 0, Tasks: 0, APIPort: 8100, AdminPort: 8101, LocalStack: false, Traefik: false, DevMode: false, Age: 7 * 24 * time.Hour},
		{Name: "production", Status: "ACTIVE", Clusters: 5, Services: 25, Tasks: 62, APIPort: 8110, AdminPort: 8111, LocalStack: false, Traefik: true, DevMode: false, Age: 24 * time.Hour},
		{Name: "local", Status: "ACTIVE", Clusters: 1, Services: 3, Tasks: 5, APIPort: 8120, AdminPort: 8121, LocalStack: true, Traefik: false, DevMode: true, Age: 2 * time.Hour},
	}

	mockClusters = map[string][]ClusterData{
		"development": {
			{Name: "default", Status: "ACTIVE", Services: 5, Tasks: 12, CPUUsed: 2.4, CPUTotal: 8.0, MemoryUsed: "3.2G", MemoryTotal: "16G", Namespace: "default.us-east-1", Age: 2 * 24 * time.Hour},
			{Name: "staging", Status: "ACTIVE", Services: 4, Tasks: 10, CPUUsed: 1.8, CPUTotal: 6.0, MemoryUsed: "2.8G", MemoryTotal: "12G", Namespace: "staging.us-east-1", Age: 24 * time.Hour},
			{Name: "development", Status: "ACTIVE", Services: 3, Tasks: 6, CPUUsed: 0.6, CPUTotal: 4.0, MemoryUsed: "1.5G", MemoryTotal: "8G", Namespace: "development.us-east-1", Age: 5 * time.Hour},
		},
		"staging": {
			{Name: "default", Status: "ACTIVE", Services: 5, Tasks: 10, CPUUsed: 3.2, CPUTotal: 8.0, MemoryUsed: "4.1G", MemoryTotal: "16G", Namespace: "default.us-west-2", Age: 24 * time.Hour},
			{Name: "api", Status: "ACTIVE", Services: 3, Tasks: 8, CPUUsed: 2.1, CPUTotal: 6.0, MemoryUsed: "3.5G", MemoryTotal: "12G", Namespace: "api.us-west-2", Age: 12 * time.Hour},
		},
		"production": {
			{Name: "default", Status: "ACTIVE", Services: 8, Tasks: 20, CPUUsed: 6.5, CPUTotal: 16.0, MemoryUsed: "12.3G", MemoryTotal: "32G", Namespace: "default.us-east-1", Age: 10 * 24 * time.Hour},
			{Name: "api", Status: "ACTIVE", Services: 6, Tasks: 15, CPUUsed: 4.2, CPUTotal: 12.0, MemoryUsed: "8.7G", MemoryTotal: "24G", Namespace: "api.us-east-1", Age: 7 * 24 * time.Hour},
			{Name: "worker", Status: "ACTIVE", Services: 4, Tasks: 12, CPUUsed: 3.8, CPUTotal: 8.0, MemoryUsed: "6.2G", MemoryTotal: "16G", Namespace: "worker.us-east-1", Age: 5 * 24 * time.Hour},
			{Name: "batch", Status: "ACTIVE", Services: 3, Tasks: 8, CPUUsed: 2.1, CPUTotal: 6.0, MemoryUsed: "4.5G", MemoryTotal: "12G", Namespace: "batch.us-east-1", Age: 3 * 24 * time.Hour},
			{Name: "analytics", Status: "ACTIVE", Services: 4, Tasks: 7, CPUUsed: 3.2, CPUTotal: 8.0, MemoryUsed: "5.8G", MemoryTotal: "16G", Namespace: "analytics.us-east-1", Age: 2 * 24 * time.Hour},
		},
		"local": {
			{Name: "default", Status: "ACTIVE", Services: 3, Tasks: 5, CPUUsed: 0.8, CPUTotal: 4.0, MemoryUsed: "1.2G", MemoryTotal: "8G", Namespace: "default.local", Age: 2 * time.Hour},
		},
	}

	serviceTemplates = []string{
		"web-service", "api-service", "worker", "db-migrate", "cache-service",
		"auth-service", "notification-service", "search-service", "analytics-worker",
		"batch-processor", "stream-processor", "scheduler", "monitoring-agent",
	}

	taskDefVersions = map[string]int{
		"web-app":        5,
		"api":            12,
		"worker":         3,
		"db-migrate":     1,
		"cache":          2,
		"auth":           8,
		"notification":   4,
		"search":         6,
		"analytics":      2,
		"batch":          1,
		"stream":         3,
		"scheduler":      2,
		"monitoring":     1,
	}

	logTemplates = []string{
		"Server started on port %d",
		"Connected to database",
		"GET %s %d %dms",
		"POST %s %d %dms",
		"Slow query detected: %dms",
		"Cache hit for key: %s",
		"Cache miss for key: %s",
		"Background job completed: %s",
		"Error processing request: %s",
		"Retry attempt %d of %d",
		"Connection pool size: %d/%d",
		"Memory usage: %.2fMB",
		"Active connections: %d",
		"Request rate: %.2f req/s",
		"Health check passed",
		"Configuration reloaded",
		"Shutting down gracefully",
	}

	endpoints = []string{
		"/health", "/api/users", "/api/products", "/api/orders",
		"/metrics", "/status", "/api/auth/login", "/api/auth/logout",
		"/api/search", "/api/recommendations", "/api/analytics",
	}

	errorMessages = []string{
		"connection timeout", "invalid credentials", "resource not found",
		"rate limit exceeded", "internal server error", "bad request",
		"unauthorized access", "service unavailable",
	}
)

// GetInstances returns mock instance data
func GetInstances() []InstanceData {
	// Simulate some dynamic changes
	instances := make([]InstanceData, len(mockInstances))
	copy(instances, mockInstances)
	
	// Randomly update some metrics
	for i := range instances {
		if instances[i].Status == "ACTIVE" {
			// Add some variance to task counts
			instances[i].Tasks += rand.Intn(5) - 2
			if instances[i].Tasks < 0 {
				instances[i].Tasks = 0
			}
		}
	}
	
	return instances
}

// GetClusters returns mock cluster data for an instance
func GetClusters(instanceName string) []ClusterData {
	clusters, ok := mockClusters[instanceName]
	if !ok {
		return []ClusterData{}
	}
	
	// Simulate dynamic CPU/Memory changes
	result := make([]ClusterData, len(clusters))
	for i, cluster := range clusters {
		result[i] = cluster
		// Add some variance
		variance := (rand.Float64() - 0.5) * 0.2
		result[i].CPUUsed = cluster.CPUUsed * (1 + variance)
		if result[i].CPUUsed > cluster.CPUTotal {
			result[i].CPUUsed = cluster.CPUTotal * 0.95
		}
	}
	
	return result
}

// GetServices returns mock service data for a cluster
func GetServices(instanceName, clusterName string) []ServiceData {
	services := []ServiceData{}
	
	// Generate services based on cluster
	numServices := 3 + rand.Intn(8)
	for i := 0; i < numServices; i++ {
		template := serviceTemplates[rand.Intn(len(serviceTemplates))]
		taskDefBase := template
		if rand.Float64() > 0.5 {
			taskDefBase = serviceTemplates[rand.Intn(len(serviceTemplates))]
		}
		
		version := 1
		if v, ok := taskDefVersions[taskDefBase]; ok {
			version = v
		}
		
		desired := 1 + rand.Intn(5)
		running := desired
		pending := 0
		status := "ACTIVE"
		
		// Simulate some services with issues
		roll := rand.Float64()
		if roll < 0.1 {
			running = desired - 1
			pending = 1
			status = "UPDATING"
		} else if roll < 0.15 {
			running = 0
			pending = 0
			status = "INACTIVE"
			desired = 0
		} else if roll < 0.2 {
			running = desired - rand.Intn(desired+1)
			pending = desired - running
			status = "PROVISIONING"
		}
		
		services = append(services, ServiceData{
			Name:    fmt.Sprintf("%s-%d", template, i+1),
			Desired: desired,
			Running: running,
			Pending: pending,
			Status:  status,
			TaskDef: fmt.Sprintf("%s:%d", taskDefBase, version),
			Age:     time.Duration(rand.Intn(30*24)) * time.Hour,
		})
	}
	
	return services
}

// GetTasks returns mock task data for a service
func GetTasks(instanceName, clusterName, serviceName string) []TaskData {
	tasks := []TaskData{}
	
	// Generate tasks based on service
	numTasks := 1 + rand.Intn(5)
	for i := 0; i < numTasks; i++ {
		status := "RUNNING"
		health := "HEALTHY"
		cpu := rand.Float64() * 2.0
		memory := fmt.Sprintf("%dM", 128+rand.Intn(896))
		ip := fmt.Sprintf("10.0.%d.%d", rand.Intn(10), rand.Intn(255))
		
		// Simulate different task states
		roll := rand.Float64()
		if roll < 0.1 {
			status = "PENDING"
			health = "-"
			cpu = 0
			memory = "-"
			ip = "-"
		} else if roll < 0.15 {
			status = "STOPPING"
			health = "UNKNOWN"
		} else if roll < 0.2 {
			health = "UNHEALTHY"
		} else if roll < 0.3 {
			health = "UNKNOWN"
		}
		
		tasks = append(tasks, TaskData{
			ID:      fmt.Sprintf("%08x-%04x-%04x", rand.Uint32(), rand.Intn(65536), rand.Intn(65536)),
			Service: serviceName,
			Status:  status,
			Health:  health,
			CPU:     cpu,
			Memory:  memory,
			IP:      ip,
			Age:     time.Duration(rand.Intn(48)) * time.Hour,
		})
	}
	
	return tasks
}

// GetLogs returns mock log entries
func GetLogs(taskID string, count int) []LogData {
	logs := []LogData{}
	
	now := time.Now()
	for i := 0; i < count; i++ {
		level := "INFO"
		roll := rand.Float64()
		if roll < 0.1 {
			level = "ERROR"
		} else if roll < 0.2 {
			level = "WARN"
		} else if roll < 0.05 {
			level = "DEBUG"
		}
		
		var message string
		template := logTemplates[rand.Intn(len(logTemplates))]
		
		switch template {
		case "Server started on port %d":
			message = fmt.Sprintf(template, 8080+rand.Intn(20))
		case "GET %s %d %dms", "POST %s %d %dms":
			endpoint := endpoints[rand.Intn(len(endpoints))]
			status := 200
			if level == "ERROR" {
				status = 400 + rand.Intn(100)
			}
			duration := 5 + rand.Intn(200)
			message = fmt.Sprintf(template, endpoint, status, duration)
		case "Slow query detected: %dms":
			message = fmt.Sprintf(template, 100+rand.Intn(900))
		case "Cache hit for key: %s", "Cache miss for key: %s":
			message = fmt.Sprintf(template, fmt.Sprintf("user:%d", rand.Intn(10000)))
		case "Background job completed: %s":
			jobs := []string{"email-sender", "report-generator", "data-sync", "cleanup"}
			message = fmt.Sprintf(template, jobs[rand.Intn(len(jobs))])
		case "Error processing request: %s":
			if level != "ERROR" {
				level = "ERROR"
			}
			message = fmt.Sprintf(template, errorMessages[rand.Intn(len(errorMessages))])
		case "Retry attempt %d of %d":
			message = fmt.Sprintf(template, 1+rand.Intn(3), 3)
		case "Connection pool size: %d/%d":
			used := rand.Intn(50)
			total := used + rand.Intn(50)
			message = fmt.Sprintf(template, used, total)
		case "Memory usage: %.2fMB":
			message = fmt.Sprintf(template, 100.0+rand.Float64()*400.0)
		case "Active connections: %d":
			message = fmt.Sprintf(template, rand.Intn(100))
		case "Request rate: %.2f req/s":
			message = fmt.Sprintf(template, rand.Float64()*100.0)
		default:
			message = template
		}
		
		logs = append(logs, LogData{
			Timestamp: now.Add(-time.Duration(i) * time.Second),
			Level:     level,
			Message:   message,
		})
	}
	
	return logs
}

// StreamLogs generates new log entries continuously
func StreamLogs(taskID string) <-chan LogData {
	ch := make(chan LogData)
	
	go func() {
		ticker := time.NewTicker(time.Duration(500+rand.Intn(2000)) * time.Millisecond)
		defer ticker.Stop()
		defer close(ch)
		
		for range ticker.C {
			logs := GetLogs(taskID, 1)
			if len(logs) > 0 {
				ch <- logs[0]
			}
		}
	}()
	
	return ch
}