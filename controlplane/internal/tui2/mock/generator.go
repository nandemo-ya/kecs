package mock

import (
	"time"
	tea "github.com/charmbracelet/bubbletea"
)

// DataMsg represents loaded mock data
type DataMsg struct {
	Instances []InstanceMsg
	Clusters  []ClusterMsg
	Services  []ServiceMsg
	Tasks     []TaskMsg
	Logs      []LogMsg
}

// InstanceMsg represents instance data for messaging
type InstanceMsg struct {
	Name      string
	Status    string
	Clusters  int
	Services  int
	Tasks     int
	APIPort   int
	Age       time.Duration
}

// ClusterMsg represents cluster data for messaging
type ClusterMsg struct {
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

// ServiceMsg represents service data for messaging
type ServiceMsg struct {
	Name    string
	Desired int
	Running int
	Pending int
	Status  string
	TaskDef string
	Age     time.Duration
}

// TaskMsg represents task data for messaging
type TaskMsg struct {
	ID      string
	Service string
	Status  string
	Health  string
	CPU     float64
	Memory  string
	IP      string
	Age     time.Duration
}

// LogMsg represents log data for messaging
type LogMsg struct {
	Timestamp time.Time
	Level     string
	Message   string
}

// LoadAllData loads all data based on current selection
func LoadAllData(selectedInstance, selectedCluster, selectedService, selectedTask string) tea.Cmd {
	return func() tea.Msg {
		result := DataMsg{}
		
		// Always load instances
		instanceData := GetInstances()
		result.Instances = make([]InstanceMsg, len(instanceData))
		for i, data := range instanceData {
			result.Instances[i] = InstanceMsg{
				Name:     data.Name,
				Status:   data.Status,
				Clusters: data.Clusters,
				Services: data.Services,
				Tasks:    data.Tasks,
				APIPort:  data.APIPort,
				Age:      data.Age,
			}
		}
		
		// Load clusters if instance is selected
		if selectedInstance != "" {
			clusterData := GetClusters(selectedInstance)
			result.Clusters = make([]ClusterMsg, len(clusterData))
			for i, data := range clusterData {
				result.Clusters[i] = ClusterMsg{
					Name:        data.Name,
					Status:      data.Status,
					Services:    data.Services,
					Tasks:       data.Tasks,
					CPUUsed:     data.CPUUsed,
					CPUTotal:    data.CPUTotal,
					MemoryUsed:  data.MemoryUsed,
					MemoryTotal: data.MemoryTotal,
					Namespace:   data.Namespace,
					Age:         data.Age,
				}
			}
		}
		
		// Load services if cluster is selected
		if selectedCluster != "" {
			serviceData := GetServices(selectedInstance, selectedCluster)
			result.Services = make([]ServiceMsg, len(serviceData))
			for i, data := range serviceData {
				result.Services[i] = ServiceMsg{
					Name:    data.Name,
					Desired: data.Desired,
					Running: data.Running,
					Pending: data.Pending,
					Status:  data.Status,
					TaskDef: data.TaskDef,
					Age:     data.Age,
				}
			}
		}
		
		// Load tasks if service is selected
		if selectedService != "" {
			taskData := GetTasks(selectedInstance, selectedCluster, selectedService)
			result.Tasks = make([]TaskMsg, len(taskData))
			for i, data := range taskData {
				result.Tasks[i] = TaskMsg{
					ID:      data.ID,
					Service: data.Service,
					Status:  data.Status,
					Health:  data.Health,
					CPU:     data.CPU,
					Memory:  data.Memory,
					IP:      data.IP,
					Age:     data.Age,
				}
			}
		}
		
		// Load logs if task is selected
		if selectedTask != "" {
			logData := GetLogs(selectedTask, 100)
			result.Logs = make([]LogMsg, len(logData))
			for i, data := range logData {
				result.Logs[i] = LogMsg{
					Timestamp: data.Timestamp,
					Level:     data.Level,
					Message:   data.Message,
				}
			}
		}
		
		return result
	}
}