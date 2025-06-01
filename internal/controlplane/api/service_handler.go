package api

import (
	"encoding/json"
	"net/http"
)

// HTTP Handlers for ECS Service operations

// handleECSCreateService handles the CreateService operation
func (s *Server) handleECSCreateService(w http.ResponseWriter, body []byte) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

// handleECSDescribeServices handles the DescribeServices operation
func (s *Server) handleECSDescribeServices(w http.ResponseWriter, body []byte) {
	response := map[string]interface{}{
		"services": []interface{}{},
		"failures": []interface{}{},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleECSListServices handles the ListServices operation
func (s *Server) handleECSListServices(w http.ResponseWriter, body []byte) {
	response := map[string]interface{}{
		"serviceArns": []interface{}{},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}