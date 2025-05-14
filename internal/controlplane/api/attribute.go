package api

import (
	"encoding/json"
	"net/http"
)

// PutAttributesRequest represents the request to put attributes
type PutAttributesRequest struct {
	Cluster     string      `json:"cluster,omitempty"`
	Attributes  []Attribute `json:"attributes"`
}

// PutAttributesResponse represents the response from putting attributes
type PutAttributesResponse struct {
	Attributes []Attribute `json:"attributes"`
}

// DeleteAttributesRequest represents the request to delete attributes
type DeleteAttributesRequest struct {
	Cluster     string      `json:"cluster,omitempty"`
	Attributes  []Attribute `json:"attributes"`
}

// DeleteAttributesResponse represents the response from deleting attributes
type DeleteAttributesResponse struct {
	Attributes []Attribute `json:"attributes"`
}

// ListAttributesRequest represents the request to list attributes
type ListAttributesRequest struct {
	Cluster            string `json:"cluster,omitempty"`
	TargetType         string `json:"targetType,omitempty"`
	AttributeName      string `json:"attributeName,omitempty"`
	AttributeValue     string `json:"attributeValue,omitempty"`
	NextToken          string `json:"nextToken,omitempty"`
	MaxResults         int    `json:"maxResults,omitempty"`
}

// ListAttributesResponse represents the response from listing attributes
type ListAttributesResponse struct {
	Attributes []Attribute `json:"attributes"`
	NextToken  string      `json:"nextToken,omitempty"`
}

// registerAttributeEndpoints registers all attribute-related API endpoints
func (s *Server) registerAttributeEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/v1/putattributes", s.handlePutAttributes)
	mux.HandleFunc("/v1/deleteattributes", s.handleDeleteAttributes)
	mux.HandleFunc("/v1/listattributes", s.handleListAttributes)
}

// handlePutAttributes handles the PutAttributes API endpoint
func (s *Server) handlePutAttributes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PutAttributesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual attribute creation logic

	// For now, return a mock response
	resp := PutAttributesResponse{
		Attributes: req.Attributes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDeleteAttributes handles the DeleteAttributes API endpoint
func (s *Server) handleDeleteAttributes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeleteAttributesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual attribute deletion logic

	// For now, return a mock response
	resp := DeleteAttributesResponse{
		Attributes: req.Attributes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleListAttributes handles the ListAttributes API endpoint
func (s *Server) handleListAttributes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ListAttributesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual attribute listing logic

	// For now, return a mock response
	resp := ListAttributesResponse{
		Attributes: []Attribute{
			{
				Name:  "ecs.instance-type",
				Value: "t3.medium",
			},
			{
				Name:  "ecs.availability-zone",
				Value: "us-west-2a",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
