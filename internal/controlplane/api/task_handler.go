package api

import (
	"encoding/json"
	"net/http"
)

// HTTP Handlers for ECS Task operations

// handleECSRunTask handles the RunTask operation
func (s *Server) handleECSRunTask(w http.ResponseWriter, body []byte) {
	response := map[string]interface{}{
		"tasks":    []interface{}{},
		"failures": []interface{}{},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleECSDescribeTasks handles the DescribeTasks operation
func (s *Server) handleECSDescribeTasks(w http.ResponseWriter, body []byte) {
	response := map[string]interface{}{
		"tasks":    []interface{}{},
		"failures": []interface{}{},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleECSListTasks handles the ListTasks operation
func (s *Server) handleECSListTasks(w http.ResponseWriter, body []byte) {
	response := map[string]interface{}{
		"taskArns": []interface{}{},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}