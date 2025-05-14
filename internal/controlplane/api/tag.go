package api

import (
	"encoding/json"
	"net/http"
)

// TagResourceRequest represents the request to tag a resource
type TagResourceRequest struct {
	ResourceArn string `json:"resourceArn"`
	Tags        []Tag  `json:"tags"`
}

// TagResourceResponse represents the response from tagging a resource
type TagResourceResponse struct {
}

// UntagResourceRequest represents the request to untag a resource
type UntagResourceRequest struct {
	ResourceArn string   `json:"resourceArn"`
	TagKeys     []string `json:"tagKeys"`
}

// UntagResourceResponse represents the response from untagging a resource
type UntagResourceResponse struct {
}

// ListTagsForResourceRequest represents the request to list tags for a resource
type ListTagsForResourceRequest struct {
	ResourceArn string `json:"resourceArn"`
}

// ListTagsForResourceResponse represents the response from listing tags for a resource
type ListTagsForResourceResponse struct {
	Tags []Tag `json:"tags"`
}

// registerTagEndpoints registers all tag-related API endpoints
func (s *Server) registerTagEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/v1/tagresource", s.handleTagResource)
	mux.HandleFunc("/v1/untagresource", s.handleUntagResource)
	mux.HandleFunc("/v1/listtagsforresource", s.handleListTagsForResource)
}

// handleTagResource handles the TagResource API endpoint
func (s *Server) handleTagResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TagResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual resource tagging logic

	// For now, return a mock response
	resp := TagResourceResponse{}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleUntagResource handles the UntagResource API endpoint
func (s *Server) handleUntagResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UntagResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual resource untagging logic

	// For now, return a mock response
	resp := UntagResourceResponse{}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleListTagsForResource handles the ListTagsForResource API endpoint
func (s *Server) handleListTagsForResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ListTagsForResourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual resource tag listing logic

	// For now, return a mock response
	resp := ListTagsForResourceResponse{
		Tags: []Tag{
			{
				Key:   "Name",
				Value: "Sample Resource",
			},
			{
				Key:   "Environment",
				Value: "Development",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
