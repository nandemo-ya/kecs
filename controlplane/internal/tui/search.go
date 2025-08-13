package tui

import (
	"fmt"
	"strings"
)

// filterInstances filters instances based on search query
func (m Model) filterInstances(instances []Instance) []Instance {
	if m.searchQuery == "" {
		return instances
	}

	query := strings.ToLower(m.searchQuery)
	filtered := make([]Instance, 0)

	for _, instance := range instances {
		if matchesSearch(instance.Name, query) ||
			matchesSearch(instance.Status, query) ||
			matchesSearch(fmt.Sprintf("%d", instance.APIPort), query) {
			filtered = append(filtered, instance)
		}
	}

	return filtered
}

// filterClusters filters clusters based on search query
func (m Model) filterClusters(clusters []Cluster) []Cluster {
	if m.searchQuery == "" {
		return clusters
	}

	query := strings.ToLower(m.searchQuery)
	filtered := make([]Cluster, 0)

	for _, cluster := range clusters {
		if matchesSearch(cluster.Name, query) ||
			matchesSearch(cluster.Status, query) {
			filtered = append(filtered, cluster)
		}
	}

	return filtered
}

// filterServices filters services based on search query
func (m Model) filterServices(services []Service) []Service {
	if m.searchQuery == "" {
		return services
	}

	query := strings.ToLower(m.searchQuery)
	filtered := make([]Service, 0)

	for _, service := range services {
		if matchesSearch(service.Name, query) ||
			matchesSearch(service.Status, query) ||
			matchesSearch(service.TaskDef, query) {
			filtered = append(filtered, service)
		}
	}

	return filtered
}

// filterTasks filters tasks based on search query
func (m Model) filterTasks(tasks []Task) []Task {
	if m.searchQuery == "" {
		return tasks
	}

	query := strings.ToLower(m.searchQuery)
	filtered := make([]Task, 0)

	for _, task := range tasks {
		if matchesSearch(task.ID, query) ||
			matchesSearch(task.Service, query) ||
			matchesSearch(task.Status, query) ||
			matchesSearch(task.Health, query) ||
			matchesSearch(task.IP, query) {
			filtered = append(filtered, task)
		}
	}

	return filtered
}

// filterLogs filters logs based on search query
func (m Model) filterLogs(logs []LogEntry) []LogEntry {
	if m.searchQuery == "" {
		return logs
	}

	query := strings.ToLower(m.searchQuery)
	filtered := make([]LogEntry, 0)

	for _, log := range logs {
		if matchesSearch(log.Level, query) ||
			matchesSearch(log.Message, query) {
			filtered = append(filtered, log)
		}
	}

	return filtered
}

// filterTaskDefFamilies filters task definition families based on search query
func (m Model) filterTaskDefFamilies(families []TaskDefinitionFamily) []TaskDefinitionFamily {
	if m.searchQuery == "" {
		return families
	}
	
	query := strings.ToLower(m.searchQuery)
	filtered := make([]TaskDefinitionFamily, 0)
	
	for _, family := range families {
		if matchesSearch(family.Family, query) {
			filtered = append(filtered, family)
		}
	}
	
	return filtered
}

// matchesSearch checks if a field contains the search query
func matchesSearch(field, query string) bool {
	return strings.Contains(strings.ToLower(field), query)
}

// getFilteredData returns filtered data based on current view and search query
func (m Model) getFilteredData() interface{} {
	switch m.currentView {
	case ViewInstances:
		return m.filterInstances(m.instances)
	case ViewClusters:
		return m.filterClusters(m.clusters)
	case ViewServices:
		return m.filterServices(m.services)
	case ViewTasks:
		return m.filterTasks(m.tasks)
	case ViewLogs:
		return m.filterLogs(m.logs)
	case ViewTaskDefinitionFamilies:
		return m.filterTaskDefFamilies(m.taskDefFamilies)
	default:
		return nil
	}
}

// resetCursorAfterSearch resets cursor position when search results change
func (m *Model) resetCursorAfterSearch() {
	switch m.currentView {
	case ViewInstances:
		m.instanceCursor = 0
	case ViewClusters:
		m.clusterCursor = 0
	case ViewServices:
		m.serviceCursor = 0
	case ViewTasks:
		m.taskCursor = 0
	case ViewLogs:
		m.logCursor = 0
	case ViewTaskDefinitionFamilies:
		m.taskDefFamilyCursor = 0
	case ViewTaskDefinitionRevisions:
		m.taskDefRevisionCursor = 0
	}
}