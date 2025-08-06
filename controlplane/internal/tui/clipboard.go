package tui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
)

// copyToClipboard copies the given text to the system clipboard
func copyToClipboard(text string) error {
	if text == "" {
		return fmt.Errorf("nothing to copy")
	}
	return clipboard.WriteAll(text)
}

// copyInstanceInfo copies instance information to clipboard
func copyInstanceInfo(instance Instance) string {
	parts := []string{
		fmt.Sprintf("Name: %s", instance.Name),
		fmt.Sprintf("API Port: %d", instance.APIPort),
		fmt.Sprintf("Admin Port: %d", instance.AdminPort),
		fmt.Sprintf("Status: %s", instance.Status),
	}
	return strings.Join(parts, "\n")
}

// copyClusterInfo copies cluster information to clipboard
func copyClusterInfo(cluster Cluster) string {
	parts := []string{
		fmt.Sprintf("Name: %s", cluster.Name),
		fmt.Sprintf("Status: %s", cluster.Status),
		fmt.Sprintf("Services: %d", cluster.Services),
		fmt.Sprintf("Tasks: %d", cluster.Tasks),
	}
	return strings.Join(parts, "\n")
}

// copyServiceInfo copies service information to clipboard
func copyServiceInfo(service Service) string {
	parts := []string{
		fmt.Sprintf("Name: %s", service.Name),
		fmt.Sprintf("Status: %s", service.Status),
		fmt.Sprintf("Desired: %d", service.Desired),
		fmt.Sprintf("Running: %d", service.Running),
		fmt.Sprintf("Task Definition: %s", service.TaskDef),
	}
	return strings.Join(parts, "\n")
}

// copyTaskInfo copies task information to clipboard
func copyTaskInfo(task Task) string {
	parts := []string{
		fmt.Sprintf("ID: %s", task.ID),
		fmt.Sprintf("Status: %s", task.Status),
		fmt.Sprintf("Service: %s", task.Service),
		fmt.Sprintf("Health: %s", task.Health),
		fmt.Sprintf("IP: %s", task.IP),
	}
	return strings.Join(parts, "\n")
}